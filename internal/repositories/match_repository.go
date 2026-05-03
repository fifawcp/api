package repositories

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/lib/pq"
)

type MatchRepository struct {
	db  *sql.DB
	cfg *config.Config
}

func NewMatchRepository(db *sql.DB, cfg *config.Config) *MatchRepository {
	return &MatchRepository{
		db:  db,
		cfg: cfg,
	}
}

func (r *MatchRepository) GetMatches(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	var args []any
	var conditions []string

	baseQuery := `SELECT
    m.id,
    m.stage_code,
    m.group_code,
    m.kickoff_at,
    m.status,
    m.home_score,
    m.away_score,
    m.home_penalty_score,
    m.away_penalty_score,
    m.winner_team_fifa_code,
    m.updated_at,
    ht.fifa_code,
    ht.name_translations,
    ht.flag_url,
    at.fifa_code,
    at.name_translations,
    at.flag_url
  FROM matches m
  LEFT JOIN team_localized ht ON m.home_team_fifa_code = ht.fifa_code
  LEFT JOIN team_localized at ON m.away_team_fifa_code = at.fifa_code`

	if len(filters.MatchIDs) > 0 {
		conditions = append(conditions, "m.id = ANY($"+strconv.Itoa(len(args)+1)+")")
		args = append(args, pq.Array(filters.MatchIDs))
	}

	if len(filters.GroupCodes) > 0 {
		placeholders := make([]string, len(filters.GroupCodes))
		for i := range filters.GroupCodes {
			placeholders[i] = "$" + strconv.Itoa(len(args)+i+1)
		}
		conditions = append(conditions, "m.group_code IN ("+strings.Join(placeholders, ",")+")")
		for _, code := range filters.GroupCodes {
			args = append(args, code)
		}
	}

	if len(filters.StageCodes) > 0 {
		placeholders := make([]string, len(filters.StageCodes))
		for i := range filters.StageCodes {
			placeholders[i] = "$" + strconv.Itoa(len(args)+i+1)
		}
		conditions = append(conditions, "m.stage_code IN ("+strings.Join(placeholders, ",")+")")
		for _, code := range filters.StageCodes {
			args = append(args, code)
		}
	}

	if filters.Status != "" {
		conditions = append(conditions, "m.status = $"+strconv.Itoa(len(args)+1))
		args = append(args, filters.Status)
	}

	if len(filters.TeamFifaCodes) > 0 {
		var teamConditions []string
		for _, code := range filters.TeamFifaCodes {
			teamConditions = append(teamConditions, "(m.home_team_fifa_code = $"+strconv.Itoa(len(args)+1)+" OR m.away_team_fifa_code = $"+strconv.Itoa(len(args)+2)+")")
			args = append(args, code, code)
		}
		conditions = append(conditions, "("+strings.Join(teamConditions, " OR ")+")")
	}

	if filters.FromDate != nil {
		conditions = append(conditions, "m.kickoff_at >= $"+strconv.Itoa(len(args)+1))
		args = append(args, *filters.FromDate)
	}

	if filters.ToDate != nil {
		conditions = append(conditions, "m.kickoff_at <= $"+strconv.Itoa(len(args)+1))
		args = append(args, *filters.ToDate)
	}

	query := baseQuery
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY m.kickoff_at ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, handleDBError(err, resourceMatch)
	}
	defer rows.Close()

	matches := []*domain.Match{}

	for rows.Next() {
		var match domain.Match
		var homeFifa, homeFlagURL, awayFifa, awayFlagURL sql.NullString
		var homeNames, awayNames domain.TeamNames
		err := rows.Scan(
			&match.ID,
			&match.StageCode,
			&match.GroupCode,
			&match.KickoffAt,
			&match.Status,
			&match.HomeScore,
			&match.AwayScore,
			&match.HomePenaltyScore,
			&match.AwayPenaltyScore,
			&match.WinnerTeamFifaCode,
			&match.UpdatedAt,
			&homeFifa, &homeNames, &homeFlagURL,
			&awayFifa, &awayNames, &awayFlagURL,
		)
		if err != nil {
			return nil, handleDBError(err, resourceMatch)
		}

		match.HomeTeam = buildMatchTeam(homeFifa, homeNames, homeFlagURL)
		match.AwayTeam = buildMatchTeam(awayFifa, awayNames, awayFlagURL)
		matches = append(matches, &match)
	}

	return matches, nil
}

func (r *MatchRepository) GetFirstGroupStageMatchKickoff(ctx context.Context) (time.Time, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	var kickoffAt time.Time
	if err := r.db.QueryRowContext(ctx,
		`SELECT kickoff_at FROM matches WHERE stage_code = 'group_stage' ORDER BY kickoff_at ASC LIMIT 1`,
	).Scan(&kickoffAt); err != nil {
		return time.Time{}, handleDBError(err, resourceMatch)
	}

	return kickoffAt, nil
}

func nullableInt(value *int) any {
	if value == nil {
		return nil
	}

	return *value
}

func buildMatchTeam(fifaCode sql.NullString, names domain.TeamNames, flagURL sql.NullString) *domain.Team {
	// Match has no assigned team yet (TBD)
	if !fifaCode.Valid {
		return nil
	}

	// Match has defined team
	return &domain.Team{
		FifaCode: fifaCode.String,
		Name:     names,
		FlagURL:  flagURL.String,
	}
}

func (r *MatchRepository) UpdateMatchesResult(ctx context.Context, updates []domain.MatchResultUpdate) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	var matchIDs []int64
	for _, update := range updates {
		matchIDs = append(matchIDs, update.MatchID)
	}

	// Check which match IDs exist in the database
	// This ensures we fail fast if any IDs are invalid (all-or-nothing approach)
	existingIDs, err := r.getExistingMatchIDs(ctx, matchIDs)
	if err != nil {
		return handleDBError(err, resourceMatch)
	}

	// Find missing IDs by comparing requested vs existing
	missingIDs := findMissingIDs(matchIDs, existingIDs)
	if len(missingIDs) > 0 {
		// Return error with list of missing IDs
		return domain.ErrMatchesNotFound(missingIDs)
	}

	// Begin a tx to ensure all updates are atomic
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return handleDBError(err, resourceMatch)
	}
	defer tx.Rollback()

	query := `UPDATE matches SET
		home_score = $1,
		away_score = $2,
		home_penalty_score = $3,
		away_penalty_score = $4,
		status = $5,
		winner_team_fifa_code = CASE
			WHEN $7 > $8  THEN home_team_fifa_code
			WHEN $8 > $7  THEN away_team_fifa_code
			WHEN $9 > $10 THEN home_team_fifa_code
			WHEN $10 > $9 THEN away_team_fifa_code
			ELSE NULL
		END,
		updated_at = NOW()
	WHERE id = $6`

	// Prepare the pre-compiled SQL template
	statement, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return handleDBError(err, resourceMatch)
	}
	defer statement.Close()

	for _, update := range updates {
		homePenalty := nullableInt(update.HomePenaltyScore)
		awayPenalty := nullableInt(update.AwayPenaltyScore)
		if _, err := statement.ExecContext(
			ctx,
			update.HomeScore,
			update.AwayScore,
			homePenalty,
			awayPenalty,
			string(update.Status),
			update.MatchID,
			// pg can't reuse params across CASE branches, so we re-pass the values
			update.HomeScore,
			update.AwayScore,
			homePenalty,
			awayPenalty,
		); err != nil {
			return handleDBError(err, resourceMatch)
		}
	}

	if err := tx.Commit(); err != nil {
		return handleDBError(err, resourceMatch)
	}

	return nil
}

func (r *MatchRepository) UpdateMatchTeams(ctx context.Context, updates []domain.MatchTeamUpdate) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	// Begin a tx to ensure all updates are atomic
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return handleDBError(err, resourceMatch)
	}
	defer tx.Rollback()

	query := `UPDATE matches SET
		home_team_fifa_code = COALESCE($1, home_team_fifa_code),
		away_team_fifa_code = COALESCE($2, away_team_fifa_code),
		updated_at = NOW()
	WHERE id = $3`

	// Prepare the pre-compiled SQL template
	statement, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return handleDBError(err, resourceMatch)
	}
	defer statement.Close()

	for _, update := range updates {
		if _, err := statement.ExecContext(
			ctx,
			update.HomeTeamFifaCode,
			update.AwayTeamFifaCode,
			update.MatchID,
		); err != nil {
			return handleDBError(err, resourceMatch)
		}
	}

	if err := tx.Commit(); err != nil {
		return handleDBError(err, resourceMatch)
	}

	return nil
}

func (r *MatchRepository) ResetMatchResult(ctx context.Context, matchID int64) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `UPDATE matches SET
	  home_score = NULL,
	  away_score = NULL,
	  home_penalty_score = NULL,
	  away_penalty_score = NULL,
	  status = 'scheduled',
	  winner_team_fifa_code = NULL,
	  updated_at = NOW()
	WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, matchID)
	if err != nil {
		return handleDBError(err, resourceMatch)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return handleDBError(err, resourceMatch)
	}

	if rowsAffected == 0 {
		return domain.ErrMatchNotFound
	}

	return nil
}

func (r *MatchRepository) IsGroupFinished(ctx context.Context, groupCode string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `SELECT COUNT(*) FROM matches WHERE group_code = $1 AND status = 'finished'`

	var count int
	err := r.db.QueryRowContext(ctx, query, groupCode).Scan(&count)
	if err != nil {
		return false, handleDBError(err, resourceMatch)
	}

	return count == 6, nil
}

func (r *MatchRepository) IsGroupStageFinished(ctx context.Context) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `SELECT COUNT(*) FROM matches WHERE stage_code = 'group_stage' AND status = 'finished'`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return false, handleDBError(err, resourceMatch)
	}

	return count == 12*6, nil
}

func (r *MatchRepository) getExistingMatchIDs(ctx context.Context, ids []int64) ([]int64, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `SELECT id FROM matches WHERE id = ANY($1)`

	rows, err := r.db.QueryContext(ctx, query, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var existingIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}

		existingIDs = append(existingIDs, id)
	}

	return existingIDs, nil
}

func findMissingIDs(requested []int64, existing []int64) []int64 {
	existingMap := make(map[int64]bool)
	for _, id := range existing {
		existingMap[id] = true
	}

	var missing []int64
	for _, id := range requested {
		if !existingMap[id] {
			missing = append(missing, id)
		}
	}
	return missing
}
