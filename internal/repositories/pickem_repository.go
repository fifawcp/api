package repositories

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/lib/pq"
)

type PickemRepository struct {
	db  *sql.DB
	cfg *config.Config
}

func NewPickemRepository(db *sql.DB, cfg *config.Config) *PickemRepository {
	return &PickemRepository{
		db:  db,
		cfg: cfg,
	}
}

func (r *PickemRepository) UpsertGroupPicks(
	ctx context.Context,
	userID string,
	picks []*domain.UserGroupPick,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return handleDBError(err, resourcePickem)
	}
	defer tx.Rollback()

	// Delete existing group picks
	if _, err := tx.ExecContext(ctx, `DELETE FROM user_group_picks WHERE user_id = $1`, userID); err != nil {
		return handleDBError(err, resourcePickem)
	}

	// Delete existing best-third picks in cascade
	if _, err := tx.ExecContext(ctx, `DELETE FROM user_best_third_picks WHERE user_id = $1`, userID); err != nil {
		return handleDBError(err, resourcePickem)
	}

	// Delete existing bracket picks in cascade
	if _, err := tx.ExecContext(ctx, `DELETE FROM user_bracket_picks WHERE user_id = $1`, userID); err != nil {
		return handleDBError(err, resourcePickem)
	}

	// Insert new picks
	if len(picks) > 0 {
		var values []string
		var args []any
		argIndex := 1

		for _, p := range picks {
			values = append(values, "($"+strconv.Itoa(argIndex)+",$"+strconv.Itoa(argIndex+1)+",$"+strconv.Itoa(argIndex+2)+",$"+strconv.Itoa(argIndex+3)+")")
			args = append(args, userID, p.TeamFifaCode, p.TeamGroupCode, p.PredictedPosition)
			argIndex += 4
		}

		query := `INSERT INTO user_group_picks (
			user_id,
			team_fifa_code,
			team_group_code,
			predicted_position
		) VALUES ` + strings.Join(values, ",")

		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return handleDBError(err, resourcePickem)
		}
	}

	return tx.Commit()
}

func (r *PickemRepository) UpsertBestThirds(
	ctx context.Context,
	userID string,
	bestThirds []*domain.UserBestThirdPick,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return handleDBError(err, resourcePickem)
	}
	defer tx.Rollback()

	// Delete existing best-third picks
	if _, err := tx.ExecContext(ctx, `DELETE FROM user_best_third_picks WHERE user_id = $1`, userID); err != nil {
		return handleDBError(err, resourcePickem)
	}

	// Delete existing bracket picks in cascade
	if _, err := tx.ExecContext(ctx, `DELETE FROM user_bracket_picks WHERE user_id = $1`, userID); err != nil {
		return handleDBError(err, resourcePickem)
	}

	// Insert new best-third picks
	if len(bestThirds) > 0 {
		var values []string
		var args []any
		argIndex := 1

		for _, b := range bestThirds {
			values = append(values, "($"+strconv.Itoa(argIndex)+",$"+strconv.Itoa(argIndex+1)+")")
			args = append(args, userID, b.TeamFifaCode)
			argIndex += 2
		}

		query := `INSERT INTO user_best_third_picks (user_id, team_fifa_code) VALUES ` + strings.Join(values, ",")

		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return handleDBError(err, resourcePickem)
		}
	}

	return tx.Commit()
}

func (r *PickemRepository) GetGroupPicks(ctx context.Context, userID string) ([]*domain.UserGroupPick, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `SELECT
		user_id,
		team_fifa_code,
		team_group_code,
		predicted_position
	FROM user_group_picks WHERE user_id = $1
	ORDER BY team_group_code, predicted_position`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, handleDBError(err, resourcePickem)
	}
	defer rows.Close()

	picks := []*domain.UserGroupPick{}
	for rows.Next() {
		var p domain.UserGroupPick
		if err := rows.Scan(
			&p.UserID,
			&p.TeamFifaCode,
			&p.TeamGroupCode,
			&p.PredictedPosition,
		); err != nil {
			return nil, handleDBError(err, resourcePickem)
		}

		picks = append(picks, &p)
	}

	return picks, rows.Err()
}

func (r *PickemRepository) GetGroupPicksByGroup(ctx context.Context, groupCode string) ([]*domain.UserGroupPick, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `SELECT
		user_id,
		team_fifa_code,
		team_group_code,
		predicted_position
	FROM user_group_picks WHERE team_group_code = $1`

	rows, err := r.db.QueryContext(ctx, query, groupCode)
	if err != nil {
		return nil, handleDBError(err, resourcePickem)
	}
	defer rows.Close()

	picks := []*domain.UserGroupPick{}
	for rows.Next() {
		var p domain.UserGroupPick
		if err := rows.Scan(&p.UserID, &p.TeamFifaCode, &p.TeamGroupCode, &p.PredictedPosition); err != nil {
			return nil, handleDBError(err, resourcePickem)
		}

		picks = append(picks, &p)
	}

	return picks, rows.Err()
}

func (r *PickemRepository) GetBestThirdPicks(ctx context.Context, userID string) ([]*domain.UserBestThirdPick, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `SELECT
		user_id,
		team_fifa_code
	FROM user_best_third_picks WHERE user_id = $1`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, handleDBError(err, resourcePickem)
	}
	defer rows.Close()

	picks := []*domain.UserBestThirdPick{}
	for rows.Next() {
		var p domain.UserBestThirdPick
		if err := rows.Scan(&p.UserID, &p.TeamFifaCode); err != nil {
			return nil, handleDBError(err, resourcePickem)
		}

		picks = append(picks, &p)
	}

	return picks, rows.Err()
}

func (r *PickemRepository) GetBestThirdPicksByTeams(ctx context.Context, teamFifaCodes []string) ([]*domain.UserBestThirdPick, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	if len(teamFifaCodes) == 0 {
		return []*domain.UserBestThirdPick{}, nil
	}

	query := `SELECT
		user_id,
		team_fifa_code
	FROM user_best_third_picks WHERE team_fifa_code = ANY($1)`

	rows, err := r.db.QueryContext(ctx, query, pq.Array(teamFifaCodes))
	if err != nil {
		return nil, handleDBError(err, resourcePickem)
	}
	defer rows.Close()

	picks := []*domain.UserBestThirdPick{}
	for rows.Next() {
		var p domain.UserBestThirdPick
		if err := rows.Scan(&p.UserID, &p.TeamFifaCode); err != nil {
			return nil, handleDBError(err, resourcePickem)
		}

		picks = append(picks, &p)
	}

	return picks, rows.Err()
}

func (r *PickemRepository) UpsertBracketPicks(ctx context.Context, userID string, picks []*domain.UserBracketPick) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return handleDBError(err, resourcePickem)
	}
	defer tx.Rollback()

	// Delete existing bracket picks
	if _, err := tx.ExecContext(ctx, `DELETE FROM user_bracket_picks WHERE user_id = $1`, userID); err != nil {
		return handleDBError(err, resourcePickem)
	}

	// Insert new bracket picks
	if len(picks) > 0 {
		var values []string
		var args []any
		argIndex := 1

		for _, p := range picks {
			values = append(values, "($"+strconv.Itoa(argIndex)+",$"+strconv.Itoa(argIndex+1)+",$"+strconv.Itoa(argIndex+2)+")")
			args = append(args, userID, p.MatchID, p.TeamFifaCode)
			argIndex += 3
		}

		query := `INSERT INTO user_bracket_picks (user_id, match_id, team_fifa_code) VALUES ` + strings.Join(values, ",")

		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return handleDBError(err, resourcePickem)
		}
	}

	if err := tx.Commit(); err != nil {
		return handleDBError(err, resourcePickem)
	}

	return nil
}

func (r *PickemRepository) GetBracketPicks(ctx context.Context, userID string) ([]*domain.UserBracketPick, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `SELECT
		user_id,
		match_id,
		team_fifa_code
	FROM user_bracket_picks WHERE user_id = $1 ORDER BY match_id`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, handleDBError(err, resourcePickem)
	}
	defer rows.Close()

	picks := []*domain.UserBracketPick{}
	for rows.Next() {
		var p domain.UserBracketPick
		if err := rows.Scan(&p.UserID, &p.MatchID, &p.TeamFifaCode); err != nil {
			return nil, handleDBError(err, resourcePickem)
		}

		picks = append(picks, &p)
	}

	return picks, rows.Err()
}

func (r *PickemRepository) GetBracketPicksByMatch(ctx context.Context, matchID int64) ([]*domain.UserBracketPick, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `SELECT
		user_id,
		match_id,
		team_fifa_code
	FROM user_bracket_picks WHERE match_id = $1`

	rows, err := r.db.QueryContext(ctx, query, matchID)
	if err != nil {
		return nil, handleDBError(err, resourcePickem)
	}
	defer rows.Close()

	picks := []*domain.UserBracketPick{}
	for rows.Next() {
		var p domain.UserBracketPick
		if err := rows.Scan(&p.UserID, &p.MatchID, &p.TeamFifaCode); err != nil {
			return nil, handleDBError(err, resourcePickem)
		}

		picks = append(picks, &p)
	}

	return picks, rows.Err()
}
