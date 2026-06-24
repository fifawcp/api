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
	db    *sql.DB
	cfg   *config.Config
	teams *domain.TeamLookup
}

func NewMatchRepository(db *sql.DB, cfg *config.Config, teams *domain.TeamLookup) *MatchRepository {
	return &MatchRepository{
		db:    db,
		cfg:   cfg,
		teams: teams,
	}
}

// matchSelectColumns is the shared column projection for every match query, so a
// new column is wired into the scan in exactly one place (scanMatch).
const matchSelectColumns = `
	m.id,
	m.stage_code,
	m.group_code,
	m.venue_name,
	m.venue_city,
	m.home_team_fifa_code,
	m.away_team_fifa_code,
	m.kickoff_at,
	m.status,
	m.home_score,
	m.away_score,
	m.home_penalty_score,
	m.away_penalty_score,
	m.winner_team_fifa_code,
	m.updated_at`

// rowScanner is satisfied by both *sql.Row and *sql.Rows, so scanMatch backs
// single-row and multi-row queries alike.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanMatch reads one matchSelectColumns row into a domain.Match. The raw scan
// error is returned unwrapped so callers can detect sql.ErrNoRows.
func (r *MatchRepository) scanMatch(row rowScanner) (*domain.Match, error) {
	var match domain.Match
	var homeFifa, awayFifa sql.NullString
	var homeScore, awayScore, homePenaltyScore, awayPenaltyScore sql.NullInt64
	var winnerFifaCode sql.NullString

	if err := row.Scan(
		&match.ID,
		&match.StageCode,
		&match.GroupCode,
		&match.Venue.Name,
		&match.Venue.City,
		&homeFifa,
		&awayFifa,
		&match.KickoffAt,
		&match.Status,
		&homeScore,
		&awayScore,
		&homePenaltyScore,
		&awayPenaltyScore,
		&winnerFifaCode,
		&match.UpdatedAt,
	); err != nil {
		return nil, err
	}

	match.Teams = domain.MatchTeams{
		Home: r.teams.Get(homeFifa.String),
		Away: r.teams.Get(awayFifa.String),
	}
	match.Result = buildMatchResult(homeScore, awayScore, homePenaltyScore, awayPenaltyScore, winnerFifaCode)

	return &match, nil
}

func (r *MatchRepository) GetMatches(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	var args []any
	var conditions []string

	baseQuery := `SELECT ` + matchSelectColumns + `
  FROM matches m`

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
		match, err := r.scanMatch(rows)
		if err != nil {
			return nil, handleDBError(err, resourceMatch)
		}
		matches = append(matches, match)
	}

	return matches, nil
}

// GetNextScheduledMatches returns every scheduled match sharing the single
// earliest upcoming kickoff. kickoff_at is TIMESTAMP(0) (second precision), so
// the equality is exact: this yields the simultaneous set (e.g. group-stage
// final matchdays kicking off together), not a time range. An empty slice means
// nothing is scheduled.
func (r *MatchRepository) GetNextScheduledMatches(ctx context.Context) ([]*domain.Match, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `SELECT ` + matchSelectColumns + `
	FROM matches m
	WHERE m.status = 'scheduled'
	  AND m.kickoff_at = (SELECT MIN(kickoff_at) FROM matches WHERE status = 'scheduled')
	ORDER BY m.id ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, handleDBError(err, resourceMatch)
	}
	defer rows.Close()

	matches := []*domain.Match{}
	for rows.Next() {
		match, err := r.scanMatch(rows)
		if err != nil {
			return nil, handleDBError(err, resourceMatch)
		}
		matches = append(matches, match)
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

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func buildMatchResult(
	homeScore, awayScore, homePenalty, awayPenalty sql.NullInt64,
	winnerFifaCode sql.NullString,
) *domain.MatchResult {
	if !homeScore.Valid || !awayScore.Valid {
		return nil
	}

	result := &domain.MatchResult{
		HomeScore: int(homeScore.Int64),
		AwayScore: int(awayScore.Int64),
	}

	if winnerFifaCode.Valid {
		code := winnerFifaCode.String
		result.WinnerTeamFifaCode = &code
	}

	if homePenalty.Valid && awayPenalty.Valid {
		result.Penalties = &domain.Penalties{
			Home: int(homePenalty.Int64),
			Away: int(awayPenalty.Int64),
		}
	}

	return result
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
