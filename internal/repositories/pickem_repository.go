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

func (r *PickemRepository) GetChampionPick(ctx context.Context, userID string) (*string, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `SELECT team_fifa_code
		FROM user_bracket_picks
		WHERE user_id = $1
		  AND match_id = (SELECT id FROM matches WHERE stage_code = 'final')
		  AND (SELECT COUNT(*) FROM user_bracket_picks WHERE user_id = $1) = 32`

	var fifaCode string
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&fifaCode)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, handleDBError(err, resourcePickem)
	}

	return &fifaCode, nil
}

func (r *PickemRepository) GetChampionPickCounts(ctx context.Context, limit int) ([]*domain.TitleFavorite, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	// Count final-match winner picks per team and express each as a share of all
	// users who picked a final winner. Aggregate by fifa_code FIRST (a varchar —
	// groupable), then join team_localized; grouping by the team's json
	// name_translations directly errors ("no equality operator for type json").
	query := `
		WITH final_picks AS (
			SELECT team_fifa_code
			FROM user_bracket_picks
			WHERE match_id = (SELECT id FROM matches WHERE stage_code = 'final')
		),
		total AS (SELECT COUNT(*) AS n FROM final_picks),
		counts AS (
			SELECT team_fifa_code, COUNT(*)::int AS pick_count
			FROM final_picks
			GROUP BY team_fifa_code
		)
		SELECT t.fifa_code, t.name_translations, t.flag_url, t.group_code,
			c.pick_count,
			(CASE WHEN (SELECT n FROM total) > 0
				THEN ROUND(100.0 * c.pick_count / (SELECT n FROM total))
				ELSE 0 END)::int AS pick_percent
		FROM counts c
		INNER JOIN team_localized t ON t.fifa_code = c.team_fifa_code
		ORDER BY c.pick_count DESC, t.fifa_code
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, handleDBError(err, resourcePickem)
	}
	defer rows.Close()

	favorites := []*domain.TitleFavorite{}
	for rows.Next() {
		var team domain.Team
		fav := &domain.TitleFavorite{Team: &team}
		if err := rows.Scan(&team.FifaCode, &team.Name, &team.FlagURL, &team.GroupCode, &fav.PickCount, &fav.PickPercent); err != nil {
			return nil, handleDBError(err, resourcePickem)
		}
		favorites = append(favorites, fav)
	}

	return favorites, rows.Err()
}

func (r *PickemRepository) GetUserProgressCounts(ctx context.Context, userID string) (domain.PickemProgressCounts, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	// `groups_completed` is the number of locked groups
	query := `
		SELECT
			(SELECT COUNT(*) FROM user_group_locks WHERE user_id = $1) AS groups_completed,
			(SELECT COUNT(*) FROM user_best_third_picks WHERE user_id = $1) AS best_thirds_completed,
			(SELECT COUNT(*) FROM user_bracket_picks WHERE user_id = $1) AS bracket_completed
	`

	var counts domain.PickemProgressCounts
	if err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&counts.Groups, &counts.BestThirds, &counts.Bracket,
	); err != nil {
		return domain.PickemProgressCounts{}, handleDBError(err, resourcePickem)
	}

	return counts, nil
}

func (r *PickemRepository) GetLockedGroupCodes(ctx context.Context, userID string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, `SELECT group_code FROM user_group_locks WHERE user_id = $1`, userID)
	if err != nil {
		return nil, handleDBError(err, resourcePickem)
	}
	defer rows.Close()

	codes := []string{}
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, handleDBError(err, resourcePickem)
		}
		codes = append(codes, code)
	}

	return codes, rows.Err()
}

// SetGroupLocks replaces the user's locked-group set with exactly lockedCodes in a single
// transaction: every existing lock row is dropped, then one row per locked code is inserted.
// The client sends the lock state for all 12 groups on every save, so a full declarative
// sync keeps the table in step without per-group diffing. Team order lives in
// user_group_picks and is untouched here.
func (r *PickemRepository) SetGroupLocks(ctx context.Context, userID string, lockedCodes []string) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return handleDBError(err, resourcePickem)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM user_group_locks WHERE user_id = $1`, userID); err != nil {
		return handleDBError(err, resourcePickem)
	}

	if len(lockedCodes) > 0 {
		var values []string
		var args []any
		argIndex := 1

		for _, code := range lockedCodes {
			values = append(values, "($"+strconv.Itoa(argIndex)+",$"+strconv.Itoa(argIndex+1)+")")
			args = append(args, userID, code)
			argIndex += 2
		}

		query := `INSERT INTO user_group_locks (user_id, group_code) VALUES ` + strings.Join(values, ",")
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return handleDBError(err, resourcePickem)
		}
	}

	return tx.Commit()
}

func (r *PickemRepository) GetBracketPickUserIDsByTeamAndStage(ctx context.Context, teamFifaCode string, stage domain.MatchStageCode) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `SELECT DISTINCT bp.user_id
		FROM user_bracket_picks bp
		INNER JOIN matches m ON m.id = bp.match_id
		WHERE bp.team_fifa_code = $1 AND m.stage_code = $2`

	rows, err := r.db.QueryContext(ctx, query, teamFifaCode, string(stage))
	if err != nil {
		return nil, handleDBError(err, resourcePickem)
	}
	defer rows.Close()

	userIDs := []string{}
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, handleDBError(err, resourcePickem)
		}

		userIDs = append(userIDs, userID)
	}

	return userIDs, rows.Err()
}
