package repositories

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
)

type AwardPickRepository struct {
	db  *sql.DB
	cfg *config.Config
}

func NewAwardPickRepository(db *sql.DB, cfg *config.Config) *AwardPickRepository {
	return &AwardPickRepository{db: db, cfg: cfg}
}

func (r *AwardPickRepository) GetAwardPicks(ctx context.Context, userID string) ([]*domain.UserAwardPick, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `SELECT user_id, award_type, player_id FROM user_award_picks WHERE user_id = $1`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, handleDBError(err, resourceAwardPick)
	}
	defer rows.Close()

	picks := []*domain.UserAwardPick{}
	for rows.Next() {
		var pick domain.UserAwardPick
		if err := rows.Scan(&pick.UserID, &pick.AwardType, &pick.PlayerID); err != nil {
			return nil, handleDBError(err, resourceAwardPick)
		}
		picks = append(picks, &pick)
	}

	return picks, rows.Err()
}

// UpsertAwardPicks declaratively replaces the user's award picks with the given
// set in a single transaction (delete all, insert new). An empty picks slice
// clears the user's picks. Mirrors PickemRepository.SetGroupLocks.
func (r *AwardPickRepository) UpsertAwardPicks(ctx context.Context, userID string, picks []*domain.UserAwardPick) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return handleDBError(err, resourceAwardPick)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM user_award_picks WHERE user_id = $1`, userID); err != nil {
		return handleDBError(err, resourceAwardPick)
	}

	if len(picks) > 0 {
		values := make([]string, 0, len(picks))
		args := make([]any, 0, len(picks)*3)
		argIndex := 1

		for _, pick := range picks {
			values = append(values, "($"+strconv.Itoa(argIndex)+",$"+strconv.Itoa(argIndex+1)+",$"+strconv.Itoa(argIndex+2)+")")
			args = append(args, userID, string(pick.AwardType), pick.PlayerID)
			argIndex += 3
		}

		query := `INSERT INTO user_award_picks (user_id, award_type, player_id) VALUES ` + strings.Join(values, ",")
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return handleDBError(err, resourceAwardPick)
		}
	}

	return tx.Commit()
}

func (r *AwardPickRepository) GetAwardPicksByPlayer(ctx context.Context, awardType domain.AwardType, playerID int64) ([]*domain.UserAwardPick, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `
		SELECT user_id, award_type, player_id
		FROM user_award_picks
		WHERE award_type = $1 AND player_id = $2`

	rows, err := r.db.QueryContext(ctx, query, string(awardType), playerID)
	if err != nil {
		return nil, handleDBError(err, resourceAwardPick)
	}
	defer rows.Close()

	picks := []*domain.UserAwardPick{}
	for rows.Next() {
		var pick domain.UserAwardPick
		if err := rows.Scan(&pick.UserID, &pick.AwardType, &pick.PlayerID); err != nil {
			return nil, handleDBError(err, resourceAwardPick)
		}
		picks = append(picks, &pick)
	}

	return picks, rows.Err()
}

func (r *AwardPickRepository) UpsertAwardWinners(ctx context.Context, winners []*domain.AwardWinner) error {
	if len(winners) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	values := make([]string, 0, len(winners))
	args := make([]any, 0, len(winners)*2)
	argIndex := 1

	for _, winner := range winners {
		values = append(values, "($"+strconv.Itoa(argIndex)+",$"+strconv.Itoa(argIndex+1)+",NOW())")
		args = append(args, string(winner.AwardType), winner.PlayerID)
		argIndex += 2
	}

	query := `
		INSERT INTO award_winners (award_type, player_id, updated_at)
		VALUES ` + strings.Join(values, ",") + `
		ON CONFLICT (award_type) DO UPDATE SET
			player_id  = EXCLUDED.player_id,
			updated_at = EXCLUDED.updated_at`

	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return handleDBError(err, resourceAwardPick)
	}

	return nil
}

func (r *AwardPickRepository) GetAwardWinners(ctx context.Context) ([]*domain.AwardWinner, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, `SELECT award_type, player_id FROM award_winners`)
	if err != nil {
		return nil, handleDBError(err, resourceAwardPick)
	}
	defer rows.Close()

	winners := []*domain.AwardWinner{}
	for rows.Next() {
		var winner domain.AwardWinner
		if err := rows.Scan(&winner.AwardType, &winner.PlayerID); err != nil {
			return nil, handleDBError(err, resourceAwardPick)
		}
		winners = append(winners, &winner)
	}

	return winners, rows.Err()
}
