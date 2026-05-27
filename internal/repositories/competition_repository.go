package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/lib/pq"
)

type CompetitionRepository struct {
	db  *sql.DB
	cfg *config.Config
}

func NewCompetitionRepository(db *sql.DB, cfg *config.Config) *CompetitionRepository {
	return &CompetitionRepository{db: db, cfg: cfg}
}

func (r *CompetitionRepository) CreateCompetition(
	ctx context.Context,
	competition *domain.Competition,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return handleDBError(err, resourceCompetition)
	}
	defer tx.Rollback()

	err = tx.QueryRowContext(ctx,
		`INSERT INTO competitions (board_id, type, name, created_by)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at`,
		competition.BoardID,
		competition.Type,
		competition.Name,
		competition.CreatedBy,
	).Scan(&competition.ID, &competition.CreatedAt)
	if err != nil {
		return handleDBError(err, resourceCompetition)
	}

	if competition.Type == domain.CompetitionTypeMatch && competition.Scope != nil {
		// Get stage codes to be inserted
		stageCodes := make([]string, len(competition.Scope.Stages))
		for i, stage := range competition.Scope.Stages {
			stageCodes[i] = string(stage)
		}

		// Insert stage codes in a batch
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO competition_scope_stages (competition_id, stage)
			 SELECT $1, UNNEST($2::text[])`,
			competition.ID, pq.Array(stageCodes),
		); err != nil {
			return handleDBError(err, resourceCompetition)
		}

		// Insert team fifa codes in a batch
		if teams := competition.Scope.TeamFifaCodes; len(teams) > 0 {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO competition_scope_teams (competition_id, team_fifa_code)
				 SELECT $1, UNNEST($2::text[])`,
				competition.ID, pq.Array(teams),
			); err != nil {
				return handleDBError(err, resourceCompetition)
			}
		}
	}

	if err := r.seedScoresFromEvents(ctx, tx, competition); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *CompetitionRepository) seedScoresFromEvents(
	ctx context.Context,
	tx *sql.Tx,
	competition *domain.Competition,
) error {
	switch competition.Type {
	case domain.CompetitionTypePickem:
		_, err := tx.ExecContext(ctx, `
			INSERT INTO competition_pickem_scores (
				competition_id,
				user_id,
				total_points,
				group_exact_positions,
				group_qualifier_hits,
				best_third_hits,
				bracket_hits
			)
			SELECT
				$1,
				bm.user_id,
				COALESCE(agg.total_points, 0),
				COALESCE(agg.group_exact_count, 0),
				COALESCE(agg.group_qualifier_count, 0),
				COALESCE(agg.best_third_hits_count, 0),
				COALESCE(agg.bracket_hits_count, 0)
			FROM board_members bm
			LEFT JOIN (
				SELECT
					se.user_id,
					SUM(se.points) AS total_points,
					COUNT(*) FILTER (WHERE se.source_type = 'group_standing_pick' AND se.points = $3) AS group_exact_count,
					COUNT(*) FILTER (WHERE se.source_type = 'group_standing_pick' AND se.points = $4) AS group_qualifier_count,
					COUNT(*) FILTER (WHERE se.source_type = 'best_third_pick')                         AS best_third_hits_count,
					COUNT(*) FILTER (WHERE se.source_type = 'bracket_pick')                            AS bracket_hits_count
				FROM score_events se
				WHERE se.source_type IN ('group_standing_pick', 'best_third_pick', 'bracket_pick')
				GROUP BY se.user_id
			) agg ON agg.user_id = bm.user_id
			WHERE bm.board_id = $2`,
			competition.ID,
			competition.BoardID,
			r.cfg.Scoring.GroupPositionExact,
			r.cfg.Scoring.GroupQualifies,
		)
		if err != nil {
			return handleDBError(err, resourceCompetition)
		}

	case domain.CompetitionTypeMatch:
		_, err := tx.ExecContext(ctx, `
			WITH scope_matches AS (
				SELECT m.id
				FROM matches m
				INNER JOIN competition_scope_stages css
					ON css.competition_id = $1 AND css.stage = m.stage_code
				WHERE (
					NOT EXISTS (SELECT 1 FROM competition_scope_teams cst WHERE cst.competition_id = $1)
					OR EXISTS (
						SELECT 1 FROM competition_scope_teams cst
						WHERE cst.competition_id = $1
						  AND cst.team_fifa_code IN (m.home_team_fifa_code, m.away_team_fifa_code)
					)
				)
			),
			user_agg AS (
				SELECT
					se.user_id,
					SUM(se.points)                          AS match_score_points,
					COUNT(*) FILTER (WHERE se.points >= $3)                       AS exact_hits_count,
					COUNT(*) FILTER (WHERE se.points > 0 AND se.points < $3)      AS correct_outcomes_count
				FROM score_events se
				INNER JOIN scope_matches sm ON sm.id = se.source_ref::bigint
				WHERE se.source_type = 'match_score_pick'
				GROUP BY se.user_id
			)
			INSERT INTO competition_match_scores (
				competition_id,
				user_id,
				total_points,
				exact_hits,
				correct_outcomes
			)
			SELECT
				$1,
				bm.user_id,
				COALESCE(agg.match_score_points, 0),
				COALESCE(agg.exact_hits_count, 0),
				COALESCE(agg.correct_outcomes_count, 0)
			FROM board_members bm
			LEFT JOIN user_agg agg ON agg.user_id = bm.user_id
			WHERE bm.board_id = $2`,
			competition.ID,
			competition.BoardID,
			r.cfg.Scoring.MatchScoreExact,
		)
		if err != nil {
			return handleDBError(err, resourceCompetition)
		}
	}

	return nil
}

func (r *CompetitionRepository) GetBoardCompetitions(
	ctx context.Context,
	boardID int64,
	viewerUserID string,
) ([]*domain.CompetitionListItem, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `
		WITH board_competitions AS (
			SELECT id, board_id, type, name, created_by, created_at
			FROM competitions
			WHERE board_id = $1
		),
		pickem_ranked AS (
			SELECT
				cps.competition_id,
				cps.user_id,
				cps.total_points,
				RANK() OVER (
					PARTITION BY cps.competition_id
					ORDER BY
						cps.total_points          DESC,
						cps.bracket_hits          DESC,
						cps.best_third_hits       DESC,
						cps.group_exact_positions DESC,
						cps.group_qualifier_hits  DESC,
						bm.created_at ASC,
						cps.user_id ASC
				) AS rank
			FROM competition_pickem_scores cps
			INNER JOIN board_members bm ON bm.board_id = $1 AND bm.user_id = cps.user_id
			WHERE cps.competition_id IN (SELECT id FROM board_competitions)
		),
		match_ranked AS (
			SELECT
				cms.competition_id,
				cms.user_id,
				cms.total_points,
				RANK() OVER (
					PARTITION BY cms.competition_id
					ORDER BY
						cms.total_points     DESC,
						cms.exact_hits       DESC,
						cms.correct_outcomes DESC,
						bm.created_at ASC,
						cms.user_id ASC
				) AS rank
			FROM competition_match_scores cms
			INNER JOIN board_members bm ON bm.board_id = $1 AND bm.user_id = cms.user_id
			WHERE cms.competition_id IN (SELECT id FROM board_competitions)
		),
		viewer_ranked AS (
			SELECT competition_id, user_id, total_points, rank FROM pickem_ranked
			UNION ALL
			SELECT competition_id, user_id, total_points, rank FROM match_ranked
		)
		SELECT
			bc.id, bc.board_id, bc.type, bc.name, bc.created_by, bc.created_at,
			COALESCE(vr.rank, 0)   AS viewer_rank,
			COALESCE(vr.total_points, 0) AS viewer_total_points
		FROM board_competitions bc
		LEFT JOIN viewer_ranked vr  ON vr.competition_id = bc.id AND vr.user_id = $2
		ORDER BY bc.created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, boardID, viewerUserID)
	if err != nil {
		return nil, handleDBError(err, resourceCompetition)
	}
	defer rows.Close()

	competitions := []*domain.CompetitionListItem{}

	for rows.Next() {
		competition := &domain.CompetitionListItem{}
		var createdBy sql.NullString

		if err := rows.Scan(
			&competition.ID,
			&competition.BoardID,
			&competition.Type,
			&competition.Name,
			&createdBy,
			&competition.CreatedAt,
			&competition.Viewer.Rank,
			&competition.Viewer.TotalPoints,
		); err != nil {
			return nil, handleDBError(err, resourceCompetition)
		}

		if createdBy.Valid {
			competition.CreatedBy = &createdBy.String
		}

		competitions = append(competitions, competition)
	}

	if err := rows.Err(); err != nil {
		return nil, handleDBError(err, resourceCompetition)
	}

	if err := r.attachScopes(ctx, competitions); err != nil {
		return nil, err
	}

	return competitions, nil
}

func (r *CompetitionRepository) attachScopes(ctx context.Context, competitions []*domain.CompetitionListItem) error {
	scopedIDs := make([]int64, 0, len(competitions))
	scopeByID := make(map[int64]*domain.CompetitionScope, len(competitions))

	for _, competition := range competitions {
		if competition.Type == domain.CompetitionTypeMatch {
			scopedIDs = append(scopedIDs, competition.ID)
			scopeByID[competition.ID] = &domain.CompetitionScope{TeamFifaCodes: []string{}}
		}
	}

	if len(scopedIDs) == 0 {
		return nil
	}

	stageRows, err := r.db.QueryContext(ctx,
		`SELECT competition_id, stage FROM competition_scope_stages WHERE competition_id = ANY($1) ORDER BY stage`,
		pq.Array(scopedIDs),
	)
	if err != nil {
		return handleDBError(err, resourceCompetition)
	}
	defer stageRows.Close()

	for stageRows.Next() {
		var competitionID int64
		var stage domain.MatchStageCode

		if err := stageRows.Scan(&competitionID, &stage); err != nil {
			return handleDBError(err, resourceCompetition)
		}

		scopeByID[competitionID].Stages = append(scopeByID[competitionID].Stages, stage)
	}
	if err := stageRows.Err(); err != nil {
		return handleDBError(err, resourceCompetition)
	}

	teamRows, err := r.db.QueryContext(ctx,
		`SELECT competition_id, team_fifa_code FROM competition_scope_teams WHERE competition_id = ANY($1) ORDER BY team_fifa_code`,
		pq.Array(scopedIDs),
	)
	if err != nil {
		return handleDBError(err, resourceCompetition)
	}
	defer teamRows.Close()

	for teamRows.Next() {
		var competitionID int64
		var code string

		if err := teamRows.Scan(&competitionID, &code); err != nil {
			return handleDBError(err, resourceCompetition)
		}

		scopeByID[competitionID].TeamFifaCodes = append(scopeByID[competitionID].TeamFifaCodes, code)
	}
	if err := teamRows.Err(); err != nil {
		return handleDBError(err, resourceCompetition)
	}

	for _, competition := range competitions {
		if scope, ok := scopeByID[competition.ID]; ok {
			competition.Scope = scope
		}
	}

	return nil
}

func (r *CompetitionRepository) DeleteCompetition(
	ctx context.Context,
	boardID, competitionID int64,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	result, err := r.db.ExecContext(ctx,
		`DELETE FROM competitions WHERE id = $1 AND board_id = $2`,
		competitionID, boardID,
	)
	if err != nil {
		return handleDBError(err, resourceCompetition)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return handleDBError(err, resourceCompetition)
	}

	if rowsAffected == 0 {
		return domain.ErrCompetitionNotFound
	}

	return nil
}

func (r *CompetitionRepository) GetCompetitionByID(
	ctx context.Context,
	boardID, competitionID int64,
) (*domain.Competition, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	competition := &domain.Competition{}
	var createdBy sql.NullString

	err := r.db.QueryRowContext(ctx,
		`SELECT id, board_id, type, name, created_by, created_at
		 FROM competitions
		 WHERE id = $1 AND board_id = $2`,
		competitionID, boardID,
	).Scan(
		&competition.ID,
		&competition.BoardID,
		&competition.Type,
		&competition.Name,
		&createdBy,
		&competition.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrCompetitionNotFound
	}
	if err != nil {
		return nil, handleDBError(err, resourceCompetition)
	}

	if createdBy.Valid {
		competition.CreatedBy = &createdBy.String
	}

	return competition, nil
}

func (r *CompetitionRepository) GetAllPickemIDs(ctx context.Context) ([]int64, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx,
		`SELECT id FROM competitions WHERE type = 'pickem'`,
	)
	if err != nil {
		return nil, handleDBError(err, resourceCompetition)
	}
	defer rows.Close()

	var ids []int64

	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, handleDBError(err, resourceCompetition)
		}
		ids = append(ids, id)
	}

	return ids, rows.Err()
}

func (r *CompetitionRepository) GetGlobalCompetitions(
	ctx context.Context,
) (*domain.Competition, *domain.Competition, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `
		SELECT id, board_id, type, name, created_by, created_at
		FROM competitions
		WHERE board_id = (SELECT id FROM boards WHERE privacy = 'global')
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, handleDBError(err, resourceCompetition)
	}
	defer rows.Close()

	var pickemCompetition, matchCompetition *domain.Competition

	for rows.Next() {
		competition := &domain.Competition{}
		var createdBy sql.NullString

		if err := rows.Scan(
			&competition.ID,
			&competition.BoardID,
			&competition.Type,
			&competition.Name,
			&createdBy,
			&competition.CreatedAt,
		); err != nil {
			return nil, nil, handleDBError(err, resourceCompetition)
		}

		if createdBy.Valid {
			competition.CreatedBy = &createdBy.String
		}

		switch competition.Type {
		case domain.CompetitionTypePickem:
			pickemCompetition = competition
		case domain.CompetitionTypeMatch:
			matchCompetition = competition
		}
	}

	if err := rows.Err(); err != nil {
		return nil, nil, handleDBError(err, resourceCompetition)
	}

	if pickemCompetition == nil || matchCompetition == nil {
		return nil, nil, domain.ErrCompetitionNotFound
	}

	return pickemCompetition, matchCompetition, nil
}

// FindMatchCompetitionsByMatches returns competition IDs whose scope
// includes at least one of the given match IDs
func (r *CompetitionRepository) FindMatchCompetitionsByMatches(ctx context.Context, matchIDs []int64) ([]int64, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `
		SELECT DISTINCT c.id
		FROM competitions c
		INNER JOIN competition_scope_stages css ON css.competition_id = c.id
		INNER JOIN matches m ON m.stage_code = css.stage AND m.id = ANY($1::bigint[])
		WHERE c.type = 'match'
		  AND (
			NOT EXISTS (SELECT 1 FROM competition_scope_teams cst WHERE cst.competition_id = c.id)
			OR EXISTS (
				SELECT 1 FROM competition_scope_teams cst
				WHERE cst.competition_id = c.id
				  AND cst.team_fifa_code IN (m.home_team_fifa_code, m.away_team_fifa_code)
			)
		  )
	`

	rows, err := r.db.QueryContext(ctx, query, pq.Array(matchIDs))
	if err != nil {
		return nil, handleDBError(err, resourceCompetition)
	}
	defer rows.Close()

	var ids []int64

	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, handleDBError(err, resourceCompetition)
		}

		ids = append(ids, id)
	}

	return ids, rows.Err()
}
