package repositories

import (
	"context"
	"database/sql"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/lib/pq"
)

type CompetitionScoreRepository struct {
	db  *sql.DB
	cfg *config.Config
}

func NewCompetitionScoreRepository(db *sql.DB, cfg *config.Config) *CompetitionScoreRepository {
	return &CompetitionScoreRepository{db: db, cfg: cfg}
}

func (r *CompetitionScoreRepository) FindMatchCompetitionsByMatches(
	ctx context.Context,
	matchIDs []int64,
) ([]int64, error) {
	if len(matchIDs) == 0 {
		return nil, nil
	}

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
		return nil, handleDBError(err, resourceCompetitionScore)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, handleDBError(err, resourceCompetitionScore)
		}
		ids = append(ids, id)
	}

	return ids, rows.Err()
}

func (r *CompetitionScoreRepository) FindPickCompetitionsByMatches(
	ctx context.Context,
	matchIDs []int64,
) ([]int64, error) {
	if len(matchIDs) == 0 {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `
		SELECT id
		FROM competitions
		WHERE type = 'pick' AND match_id = ANY($1::bigint[])
	`

	rows, err := r.db.QueryContext(ctx, query, pq.Array(matchIDs))
	if err != nil {
		return nil, handleDBError(err, resourceCompetitionScore)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, handleDBError(err, resourceCompetitionScore)
		}
		ids = append(ids, id)
	}

	return ids, rows.Err()
}

func (r *CompetitionScoreRepository) BatchUpsertMatchScores(
	ctx context.Context,
	competitionID int64,
	userIDs []string,
	exactScorePts int,
) error {
	if len(userIDs) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `
		WITH scope_matches AS (
			SELECT m.id
			FROM matches m
			INNER JOIN competition_scope_stages css ON css.competition_id = $1 AND css.stage = m.stage_code
			WHERE (
				NOT EXISTS (SELECT 1 FROM competition_scope_teams cst WHERE cst.competition_id = $1)
				OR EXISTS (
					SELECT 1 FROM competition_scope_teams cst
					WHERE cst.competition_id = $1
					  AND cst.team_fifa_code IN (m.home_team_fifa_code, m.away_team_fifa_code)
				)
			)
		),
		computed AS (
			SELECT
				se.user_id,
				COALESCE(SUM(se.points), 0)                                   AS match_score_points,
				COUNT(*) FILTER (WHERE se.points >= $3)                       AS exact_hits_count,
				COUNT(*) FILTER (WHERE se.points > 0 AND se.points < $3)      AS correct_outcomes_count
			FROM score_events se
			INNER JOIN scope_matches sm ON sm.id = se.source_ref::bigint
			WHERE se.user_id = ANY($2::uuid[])
			  AND se.source_type = 'match_score_pick'
			GROUP BY se.user_id
		)
		INSERT INTO competition_match_scores (
			competition_id, user_id, total_points,
			exact_hits, correct_outcomes, updated_at
		)
		SELECT $1, c.user_id, c.match_score_points, c.exact_hits_count, c.correct_outcomes_count, NOW()
		FROM computed c
		WHERE EXISTS (
			SELECT 1 FROM competitions co
			INNER JOIN board_members bm ON bm.board_id = co.board_id AND bm.user_id = c.user_id
			WHERE co.id = $1
		)
		ON CONFLICT (competition_id, user_id) DO UPDATE SET
			total_points     = EXCLUDED.total_points,
			exact_hits       = EXCLUDED.exact_hits,
			correct_outcomes = EXCLUDED.correct_outcomes,
			updated_at       = EXCLUDED.updated_at
	`

	_, err := r.db.ExecContext(ctx, query, competitionID, pq.Array(userIDs), exactScorePts)
	if err != nil {
		return handleDBError(err, resourceCompetitionScore)
	}

	return nil
}

func (r *CompetitionScoreRepository) BatchUpsertPickScores(
	ctx context.Context,
	competitionID int64,
	userIDs []string,
	exactScorePts int,
) error {
	if len(userIDs) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `
		WITH scope_matches AS (
			SELECT match_id AS id FROM competitions WHERE id = $1
		),
		computed AS (
			SELECT
				se.user_id,
				COALESCE(SUM(se.points), 0)                                   AS match_score_points,
				COUNT(*) FILTER (WHERE se.points >= $3)                       AS exact_hits_count,
				COUNT(*) FILTER (WHERE se.points > 0 AND se.points < $3)      AS correct_outcomes_count
			FROM score_events se
			INNER JOIN scope_matches sm ON sm.id = se.source_ref::bigint
			WHERE se.user_id = ANY($2::uuid[])
			  AND se.source_type = 'match_score_pick'
			GROUP BY se.user_id
		)
		INSERT INTO competition_match_scores (
			competition_id, user_id, total_points,
			exact_hits, correct_outcomes, updated_at
		)
		SELECT $1, c.user_id, c.match_score_points, c.exact_hits_count, c.correct_outcomes_count, NOW()
		FROM computed c
		WHERE EXISTS (
			SELECT 1 FROM competitions co
			INNER JOIN board_members bm ON bm.board_id = co.board_id AND bm.user_id = c.user_id
			WHERE co.id = $1
		)
		ON CONFLICT (competition_id, user_id) DO UPDATE SET
			total_points     = EXCLUDED.total_points,
			exact_hits       = EXCLUDED.exact_hits,
			correct_outcomes = EXCLUDED.correct_outcomes,
			updated_at       = EXCLUDED.updated_at
	`

	_, err := r.db.ExecContext(ctx, query, competitionID, pq.Array(userIDs), exactScorePts)
	if err != nil {
		return handleDBError(err, resourceCompetitionScore)
	}

	return nil
}

func (r *CompetitionScoreRepository) GetLeaderboard(
	ctx context.Context,
	competitionID int64,
	page, limit int,
	q, sort, dir string,
) (*domain.CompetitionLeaderboardPage, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	var competitionType domain.CompetitionType
	if err := r.db.QueryRowContext(ctx,
		`SELECT type FROM competitions WHERE id = $1`,
		competitionID,
	).Scan(&competitionType); err != nil {
		return nil, handleDBError(err, resourceCompetition)
	}

	switch competitionType {
	case domain.CompetitionTypePickem:
		return r.getPickemLeaderboard(ctx, competitionID, page, limit, q, sort, dir)
	case domain.CompetitionTypeMatch, domain.CompetitionTypePick:
		return r.getMatchLeaderboard(ctx, competitionID, page, limit, q, sort, dir)
	case domain.CompetitionTypeAwards:
		return r.getAwardsLeaderboard(ctx, competitionID, page, limit, q, sort, dir)
	default:
		return nil, domain.ErrCompetitionNotFound
	}
}

// Per-type sort whitelists (frontend column id → output column). These drive the
// row ORDER only; rank() always uses the canonical total-DESC standing, so a
// member's position never changes with the view's sort.
var pickemSortCols = map[string]string{
	"total":           "total_points",
	"groupExact":      "group_exact_positions",
	"groupQualifiers": "group_qualifier_hits",
	"bestThirds":      "best_third_hits",
	"bracket":         "bracket_hits",
}

var awardsSortCols = map[string]string{
	"total":        "total_points",
	"golden_boot":  "golden_boot",
	"golden_ball":  "golden_ball",
	"golden_glove": "golden_glove",
	"young_player": "young_player",
}

var matchSortCols = map[string]string{
	"total":     "total_points",
	"exactHits": "exact_hits",
	"outcomes":  "correct_outcomes",
}

// leaderboardRowOrder builds the final ORDER BY: the chosen column + direction,
// then rank as the stable tiebreak. Whitelisted column → safe to interpolate.
func leaderboardRowOrder(cols map[string]string, sort, dir string) string {
	col, ok := cols[sort]
	if !ok {
		col = cols["total"]
	}
	direction := "DESC"
	if dir == "asc" {
		direction = "ASC"
	}
	return col + " " + direction + ", rank ASC, user_id ASC"
}

func (r *CompetitionScoreRepository) getPickemLeaderboard(
	ctx context.Context,
	competitionID int64,
	page, limit int,
	q, sort, dir string,
) (*domain.CompetitionLeaderboardPage, error) {
	offset := (page - 1) * limit
	rowOrder := leaderboardRowOrder(pickemSortCols, sort, dir)

	query := `
		WITH comp AS (
			SELECT id, board_id FROM competitions WHERE id = $1
		),
		scores AS (
			SELECT
				bm.user_id,
				u.username,
				u.first_name,
				u.last_name,
				bm.role,
				bm.created_at AS joined_at,
				COALESCE(SUM(se.points), 0)::int                                                       AS total_points,
				COUNT(*) FILTER (WHERE se.source_type = 'group_standing_pick' AND se.points = $5)::int AS group_exact_positions,
				COUNT(*) FILTER (WHERE se.source_type = 'group_standing_pick' AND se.points = $6)::int AS group_qualifier_hits,
				COUNT(*) FILTER (WHERE se.source_type = 'best_third_pick')::int                        AS best_third_hits,
				COUNT(*) FILTER (WHERE se.source_type = 'bracket_pick')::int                           AS bracket_hits
			FROM comp
			JOIN board_members bm ON bm.board_id = comp.board_id
			JOIN users u          ON u.id = bm.user_id
			LEFT JOIN score_events se
				ON se.user_id = bm.user_id
			   AND se.source_type IN ('group_standing_pick', 'best_third_pick', 'bracket_pick')
			GROUP BY bm.user_id, u.username, u.first_name, u.last_name, bm.role, bm.created_at
		),
		ranked AS (
			SELECT *,
				RANK() OVER (
					ORDER BY
						total_points          DESC,
						bracket_hits          DESC,
						best_third_hits       DESC,
						group_exact_positions DESC,
						group_qualifier_hits  DESC,
						joined_at ASC,
						user_id ASC
				) AS rank
			FROM scores
		),
		filtered AS (
			SELECT *, COUNT(*) OVER () AS total
			FROM ranked
			WHERE $4::text = ''
			   OR username   ILIKE '%' || $4 || '%'
			   OR first_name ILIKE '%' || $4 || '%'
			   OR last_name  ILIKE '%' || $4 || '%'
		)
		SELECT
			user_id, username, first_name, last_name, role, joined_at,
			rank, total_points,
			group_exact_positions, group_qualifier_hits, best_third_hits, bracket_hits,
			total
		FROM filtered
		ORDER BY ` + rowOrder + `
		OFFSET $2 LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, competitionID, offset, limit, q,
		r.cfg.Scoring.GroupPositionExact, r.cfg.Scoring.GroupQualifies,
	)
	if err != nil {
		return nil, handleDBError(err, resourceCompetitionScore)
	}
	defer rows.Close()

	leaderboard := &domain.CompetitionLeaderboardPage{
		Members:    []*domain.CompetitionLeaderboardEntry{},
		Pagination: domain.Pagination{Page: page, Limit: limit},
	}

	for rows.Next() {
		entry := &domain.CompetitionLeaderboardEntry{}
		score := &domain.PickemScore{}
		var total int

		if err := rows.Scan(
			&entry.Member.UserID,
			&entry.Member.UserName,
			&entry.Member.FirstName,
			&entry.Member.LastName,
			&entry.Member.Role,
			&entry.Member.JoinedAt,
			&entry.Rank,
			&score.Total,
			&score.GroupExactPositions,
			&score.GroupQualifierHits,
			&score.BestThirdHits,
			&score.BracketHits,
			&total,
		); err != nil {
			return nil, handleDBError(err, resourceCompetitionScore)
		}

		entry.Score = score
		leaderboard.Pagination.Total = total
		leaderboard.Members = append(leaderboard.Members, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, handleDBError(err, resourceCompetitionScore)
	}

	if len(leaderboard.Members) == 0 {
		leaderboard.Pagination.Total, err = r.countLeaderboardMembers(ctx, competitionID, q)
		if err != nil {
			return nil, err
		}
	}

	leaderboard.Pagination.HasMore = page*limit < leaderboard.Pagination.Total
	return leaderboard, nil
}

// countLeaderboardMembers returns the number of a competition's board members
// matching the search term — used as the total when a page lands past the last row.
func (r *CompetitionScoreRepository) countLeaderboardMembers(ctx context.Context, competitionID int64, q string) (int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*)
		 FROM competitions comp
		 JOIN board_members bm ON bm.board_id = comp.board_id
		 JOIN users u          ON u.id = bm.user_id
		 WHERE comp.id = $1
		   AND ($2::text = ''
		        OR u.username   ILIKE '%' || $2 || '%'
		        OR u.first_name ILIKE '%' || $2 || '%'
		        OR u.last_name  ILIKE '%' || $2 || '%')`,
		competitionID, q,
	).Scan(&total); err != nil && err != sql.ErrNoRows {
		return 0, handleDBError(err, resourceCompetitionScore)
	}
	return total, nil
}

func (r *CompetitionScoreRepository) getAwardsLeaderboard(
	ctx context.Context,
	competitionID int64,
	page, limit int,
	q, sort, dir string,
) (*domain.CompetitionLeaderboardPage, error) {
	offset := (page - 1) * limit
	rowOrder := leaderboardRowOrder(awardsSortCols, sort, dir)

	query := `
		WITH comp AS (
			SELECT id, board_id FROM competitions WHERE id = $1
		),
		scores AS (
			SELECT
				bm.user_id,
				u.username,
				u.first_name,
				u.last_name,
				bm.role,
				bm.created_at AS joined_at,
				COALESCE(SUM(se.points), 0)::int                            AS total_points,
				COUNT(*) FILTER (WHERE se.source_ref = 'golden_boot')::int  AS golden_boot,
				COUNT(*) FILTER (WHERE se.source_ref = 'golden_ball')::int  AS golden_ball,
				COUNT(*) FILTER (WHERE se.source_ref = 'golden_glove')::int AS golden_glove,
				COUNT(*) FILTER (WHERE se.source_ref = 'young_player')::int AS young_player
			FROM comp
			JOIN board_members bm ON bm.board_id = comp.board_id
			JOIN users u          ON u.id = bm.user_id
			LEFT JOIN score_events se
				ON se.user_id = bm.user_id
			   AND se.source_type = 'award_pick'
			GROUP BY bm.user_id, u.username, u.first_name, u.last_name, bm.role, bm.created_at
		),
		ranked AS (
			SELECT *,
				RANK() OVER (ORDER BY total_points DESC, joined_at ASC, user_id ASC) AS rank
			FROM scores
		),
		filtered AS (
			SELECT *, COUNT(*) OVER () AS total
			FROM ranked
			WHERE $4::text = ''
			   OR username   ILIKE '%' || $4 || '%'
			   OR first_name ILIKE '%' || $4 || '%'
			   OR last_name  ILIKE '%' || $4 || '%'
		)
		SELECT
			user_id, username, first_name, last_name, role, joined_at,
			rank, total_points, golden_boot, golden_ball, golden_glove, young_player,
			total
		FROM filtered
		ORDER BY ` + rowOrder + `
		OFFSET $2 LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, competitionID, offset, limit, q)
	if err != nil {
		return nil, handleDBError(err, resourceCompetitionScore)
	}
	defer rows.Close()

	leaderboard := &domain.CompetitionLeaderboardPage{
		Members:    []*domain.CompetitionLeaderboardEntry{},
		Pagination: domain.Pagination{Page: page, Limit: limit},
	}

	for rows.Next() {
		entry := &domain.CompetitionLeaderboardEntry{}
		score := &domain.AwardsScore{}
		var total int

		if err := rows.Scan(
			&entry.Member.UserID,
			&entry.Member.UserName,
			&entry.Member.FirstName,
			&entry.Member.LastName,
			&entry.Member.Role,
			&entry.Member.JoinedAt,
			&entry.Rank,
			&score.Total,
			&score.GoldenBoot,
			&score.GoldenBall,
			&score.GoldenGlove,
			&score.YoungPlayer,
			&total,
		); err != nil {
			return nil, handleDBError(err, resourceCompetitionScore)
		}

		entry.Score = score
		leaderboard.Pagination.Total = total
		leaderboard.Members = append(leaderboard.Members, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, handleDBError(err, resourceCompetitionScore)
	}

	if len(leaderboard.Members) == 0 {
		leaderboard.Pagination.Total, err = r.countLeaderboardMembers(ctx, competitionID, q)
		if err != nil {
			return nil, err
		}
	}

	leaderboard.Pagination.HasMore = page*limit < leaderboard.Pagination.Total
	return leaderboard, nil
}

// previewScore carries only the total — the card mini-leaderboard shows nothing else.
type previewScore struct {
	Total int `json:"total"`
}

// GetBoardCompetitionPreviews returns the top-`limit` members per competition for
// the whole board in one query, matching each type's leaderboard ordering.
func (r *CompetitionScoreRepository) GetBoardCompetitionPreviews(
	ctx context.Context,
	boardID int64,
	limit int,
) (map[int64][]*domain.CompetitionLeaderboardEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `
		WITH bc AS (
			SELECT id, type, match_id FROM competitions WHERE board_id = $1
		),
		pickem AS (
			SELECT c.id AS competition_id, bm.user_id, agg.total_points,
				RANK() OVER (
					PARTITION BY c.id
					ORDER BY agg.total_points DESC, agg.bracket_hits DESC, agg.best_third_hits DESC,
						agg.group_exact_positions DESC, agg.group_qualifier_hits DESC, bm.created_at ASC, bm.user_id ASC
				) AS rank
			FROM bc c
			JOIN board_members bm ON bm.board_id = $1
			LEFT JOIN LATERAL (
				SELECT
					COALESCE(SUM(se.points), 0)::int                                                       AS total_points,
					COUNT(*) FILTER (WHERE se.source_type = 'group_standing_pick' AND se.points = $3)::int AS group_exact_positions,
					COUNT(*) FILTER (WHERE se.source_type = 'group_standing_pick' AND se.points = $4)::int AS group_qualifier_hits,
					COUNT(*) FILTER (WHERE se.source_type = 'best_third_pick')::int                        AS best_third_hits,
					COUNT(*) FILTER (WHERE se.source_type = 'bracket_pick')::int                           AS bracket_hits
				FROM score_events se
				WHERE se.user_id = bm.user_id AND se.source_type IN ('group_standing_pick', 'best_third_pick', 'bracket_pick')
			) agg ON TRUE
			WHERE c.type = 'pickem'
		),
		awards AS (
			SELECT c.id AS competition_id, bm.user_id, agg.total_points,
				RANK() OVER (PARTITION BY c.id ORDER BY agg.total_points DESC, bm.created_at ASC, bm.user_id ASC) AS rank
			FROM bc c
			JOIN board_members bm ON bm.board_id = $1
			LEFT JOIN LATERAL (
				SELECT COALESCE(SUM(se.points), 0)::int AS total_points
				FROM score_events se WHERE se.user_id = bm.user_id AND se.source_type = 'award_pick'
			) agg ON TRUE
			WHERE c.type = 'awards'
		),
		matchpick AS (
			-- For 'pick' competitions, members who made a prediction rank above those who
			-- didn't within a points tie (match_id IS NOT NULL gates it to single-match picks).
			SELECT cms.competition_id, cms.user_id, cms.total_points,
				RANK() OVER (
					PARTITION BY cms.competition_id
					ORDER BY cms.total_points DESC, cms.exact_hits DESC, cms.correct_outcomes DESC,
						(CASE WHEN c.match_id IS NOT NULL AND ump.match_id IS NULL THEN 1 ELSE 0 END) ASC,
						bm.created_at ASC, cms.user_id ASC
				) AS rank
			FROM competition_match_scores cms
			JOIN bc c ON c.id = cms.competition_id
			JOIN board_members bm ON bm.board_id = $1 AND bm.user_id = cms.user_id
			LEFT JOIN user_match_score_picks ump ON ump.user_id = cms.user_id AND ump.match_id = c.match_id
			WHERE c.type IN ('match', 'pick')
		),
		unioned AS (
			SELECT competition_id, user_id, total_points, rank FROM pickem
			UNION ALL SELECT competition_id, user_id, total_points, rank FROM awards
			UNION ALL SELECT competition_id, user_id, total_points, rank FROM matchpick
		)
		SELECT un.competition_id, un.user_id, u.username, u.first_name, u.last_name, bm.role, bm.created_at, un.rank, un.total_points
		FROM unioned un
		JOIN users u          ON u.id = un.user_id
		JOIN board_members bm ON bm.board_id = $1 AND bm.user_id = un.user_id
		WHERE un.rank <= $2
		ORDER BY un.competition_id, un.rank ASC, bm.created_at ASC, un.user_id ASC
	`

	rows, err := r.db.QueryContext(ctx, query, boardID, limit, r.cfg.Scoring.GroupPositionExact, r.cfg.Scoring.GroupQualifies)
	if err != nil {
		return nil, handleDBError(err, resourceCompetitionScore)
	}
	defer rows.Close()

	previews := map[int64][]*domain.CompetitionLeaderboardEntry{}
	for rows.Next() {
		var competitionID int64
		entry := &domain.CompetitionLeaderboardEntry{}
		score := previewScore{}

		if err := rows.Scan(
			&competitionID,
			&entry.Member.UserID,
			&entry.Member.UserName,
			&entry.Member.FirstName,
			&entry.Member.LastName,
			&entry.Member.Role,
			&entry.Member.JoinedAt,
			&entry.Rank,
			&score.Total,
		); err != nil {
			return nil, handleDBError(err, resourceCompetitionScore)
		}

		entry.Score = score
		previews[competitionID] = append(previews[competitionID], entry)
	}

	return previews, rows.Err()
}

// boardSummarySortCols whitelists the sortable output columns (custom = match).
// Rank is always the canonical total-DESC standing; this only drives row order.
var boardSummarySortCols = map[string]string{
	"total":  "total",
	"pickem": "pickem",
	"custom": "custom",
	"pick":   "pick",
	"awards": "awards",
}

// GetBoardSummary ranks every board member by their raw-sum points across all the
// board's competitions, with per-type subtotals (custom = match competitions).
func (r *CompetitionScoreRepository) GetBoardSummary(
	ctx context.Context,
	boardID int64,
	page, limit int,
	q, sort, dir string,
) (*domain.BoardSummaryPage, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	offset := (page - 1) * limit
	rowOrder := leaderboardRowOrder(boardSummarySortCols, sort, dir)

	query := `
		WITH members AS (
			SELECT bm.user_id, u.username, u.first_name, u.last_name, bm.role, bm.created_at AS joined_at
			FROM board_members bm JOIN users u ON u.id = bm.user_id
			WHERE bm.board_id = $1
		),
		has AS (
			SELECT bool_or(type = 'pickem') AS has_pickem, bool_or(type = 'awards') AS has_awards
			FROM competitions WHERE board_id = $1
		),
		pickem AS (
			SELECT m.user_id, COALESCE(SUM(se.points), 0)::int AS pts
			FROM members m
			LEFT JOIN score_events se ON se.user_id = m.user_id AND se.source_type IN ('group_standing_pick', 'best_third_pick', 'bracket_pick')
			GROUP BY m.user_id
		),
		awards AS (
			SELECT m.user_id, COALESCE(SUM(se.points), 0)::int AS pts
			FROM members m
			LEFT JOIN score_events se ON se.user_id = m.user_id AND se.source_type = 'award_pick'
			GROUP BY m.user_id
		),
		custom AS (
			SELECT cms.user_id, SUM(cms.total_points)::int AS pts
			FROM competition_match_scores cms JOIN competitions c ON c.id = cms.competition_id
			WHERE c.board_id = $1 AND c.type = 'match'
			GROUP BY cms.user_id
		),
		pick AS (
			SELECT cms.user_id, SUM(cms.total_points)::int AS pts
			FROM competition_match_scores cms JOIN competitions c ON c.id = cms.competition_id
			WHERE c.board_id = $1 AND c.type = 'pick'
			GROUP BY cms.user_id
		),
		scored AS (
			SELECT
				m.user_id, m.username, m.first_name, m.last_name, m.role, m.joined_at,
				CASE WHEN h.has_pickem THEN COALESCE(p.pts, 0) ELSE 0 END  AS pickem,
				COALESCE(cu.pts, 0)                                        AS custom,
				COALESCE(pk.pts, 0)                                        AS pick,
				CASE WHEN h.has_awards THEN COALESCE(a.pts, 0) ELSE 0 END  AS awards
			FROM members m
			CROSS JOIN has h
			LEFT JOIN pickem p  ON p.user_id = m.user_id
			LEFT JOIN awards a  ON a.user_id = m.user_id
			LEFT JOIN custom cu ON cu.user_id = m.user_id
			LEFT JOIN pick pk   ON pk.user_id = m.user_id
		),
		ranked AS (
			SELECT *, (pickem + custom + pick + awards) AS total,
				RANK() OVER (ORDER BY (pickem + custom + pick + awards) DESC, joined_at ASC, user_id ASC) AS rank
			FROM scored
		),
		filtered AS (
			SELECT *, COUNT(*) OVER () AS total_count
			FROM ranked
			WHERE $4::text = ''
			   OR username   ILIKE '%' || $4 || '%'
			   OR first_name ILIKE '%' || $4 || '%'
			   OR last_name  ILIKE '%' || $4 || '%'
		)
		SELECT
			user_id, username, first_name, last_name, role, joined_at,
			rank, total, pickem, custom, pick, awards, total_count
		FROM filtered
		ORDER BY ` + rowOrder + `
		OFFSET $2 LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, boardID, offset, limit, q)
	if err != nil {
		return nil, handleDBError(err, resourceCompetitionScore)
	}
	defer rows.Close()

	summary := &domain.BoardSummaryPage{
		Members:    []*domain.BoardSummaryEntry{},
		Pagination: domain.Pagination{Page: page, Limit: limit},
	}

	for rows.Next() {
		entry := &domain.BoardSummaryEntry{}
		var total int

		if err := rows.Scan(
			&entry.Member.UserID,
			&entry.Member.UserName,
			&entry.Member.FirstName,
			&entry.Member.LastName,
			&entry.Member.Role,
			&entry.Member.JoinedAt,
			&entry.Rank,
			&entry.Total,
			&entry.Pickem,
			&entry.Custom,
			&entry.Pick,
			&entry.Awards,
			&total,
		); err != nil {
			return nil, handleDBError(err, resourceCompetitionScore)
		}

		summary.Pagination.Total = total
		summary.Members = append(summary.Members, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, handleDBError(err, resourceCompetitionScore)
	}

	if len(summary.Members) == 0 {
		if err := r.db.QueryRowContext(ctx,
			`SELECT COUNT(*)
			 FROM board_members bm JOIN users u ON u.id = bm.user_id
			 WHERE bm.board_id = $1
			   AND ($2::text = ''
			        OR u.username   ILIKE '%' || $2 || '%'
			        OR u.first_name ILIKE '%' || $2 || '%'
			        OR u.last_name  ILIKE '%' || $2 || '%')`,
			boardID, q,
		).Scan(&summary.Pagination.Total); err != nil && err != sql.ErrNoRows {
			return nil, handleDBError(err, resourceCompetitionScore)
		}
	}

	summary.Pagination.HasMore = page*limit < summary.Pagination.Total
	return summary, nil
}

func (r *CompetitionScoreRepository) GetUserPickemStats(
	ctx context.Context,
	competitionID int64,
	userID string,
) (domain.CompetitionUserStats, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `
		WITH comp AS (
			SELECT id, board_id FROM competitions WHERE id = $1
		),
		scores AS (
			SELECT
				bm.user_id,
				bm.created_at AS joined_at,
				COALESCE(SUM(se.points), 0)::int                                                       AS total_points,
				COUNT(*) FILTER (WHERE se.source_type = 'group_standing_pick' AND se.points = $3)::int AS group_exact_positions,
				COUNT(*) FILTER (WHERE se.source_type = 'group_standing_pick' AND se.points = $4)::int AS group_qualifier_hits,
				COUNT(*) FILTER (WHERE se.source_type = 'best_third_pick')::int                        AS best_third_hits,
				COUNT(*) FILTER (WHERE se.source_type = 'bracket_pick')::int                           AS bracket_hits
			FROM comp
			JOIN board_members bm ON bm.board_id = comp.board_id
			LEFT JOIN score_events se
				ON se.user_id = bm.user_id
			   AND se.source_type IN ('group_standing_pick', 'best_third_pick', 'bracket_pick')
			GROUP BY bm.user_id, bm.created_at
		),
		ranked AS (
			SELECT
				user_id,
				total_points,
				RANK() OVER (
					ORDER BY
						total_points          DESC,
						bracket_hits          DESC,
						best_third_hits       DESC,
						group_exact_positions DESC,
						group_qualifier_hits  DESC,
						joined_at ASC,
						user_id ASC
				) AS rank
			FROM scores
		)
		SELECT rank, total_points
		FROM ranked
		WHERE user_id = $2
	`

	var stats domain.CompetitionUserStats
	err := r.db.QueryRowContext(ctx, query, competitionID, userID,
		r.cfg.Scoring.GroupPositionExact, r.cfg.Scoring.GroupQualifies,
	).Scan(&stats.Rank, &stats.Points)
	if err == sql.ErrNoRows {
		return domain.CompetitionUserStats{}, nil
	}
	if err != nil {
		return domain.CompetitionUserStats{}, handleDBError(err, resourceCompetitionScore)
	}

	return stats, nil
}

func (r *CompetitionScoreRepository) GetUserMatchStats(
	ctx context.Context,
	competitionID int64,
	userID string,
) (domain.CompetitionUserStats, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `
		WITH ranked AS (
			SELECT
				cms.user_id,
				cms.total_points,
				RANK() OVER (
					ORDER BY
						cms.total_points     DESC,
						cms.exact_hits       DESC,
						cms.correct_outcomes DESC,
						bm.created_at ASC,
						cms.user_id ASC
				) AS rank
			FROM competition_match_scores cms
			INNER JOIN competitions comp ON comp.id = cms.competition_id
			INNER JOIN board_members bm  ON bm.board_id = comp.board_id AND bm.user_id = cms.user_id
			WHERE cms.competition_id = $1
		)
		SELECT rank, total_points
		FROM ranked
		WHERE user_id = $2
	`

	var stats domain.CompetitionUserStats
	err := r.db.QueryRowContext(ctx, query, competitionID, userID).Scan(&stats.Rank, &stats.Points)
	if err == sql.ErrNoRows {
		return domain.CompetitionUserStats{}, nil
	}
	if err != nil {
		return domain.CompetitionUserStats{}, handleDBError(err, resourceCompetitionScore)
	}

	return stats, nil
}

func (r *CompetitionScoreRepository) getMatchLeaderboard(
	ctx context.Context,
	competitionID int64,
	page, limit int,
	q, sort, dir string,
) (*domain.CompetitionLeaderboardPage, error) {
	offset := (page - 1) * limit
	rowOrder := leaderboardRowOrder(matchSortCols, sort, dir)

	query := `
		WITH ranked AS (
			SELECT
				cms.user_id,
				u.username,
				u.first_name,
				u.last_name,
				bm.role,
				bm.created_at AS joined_at,
				cms.total_points,
				cms.exact_hits,
				cms.correct_outcomes,
				RANK() OVER (
					ORDER BY
						cms.total_points     DESC,
						cms.exact_hits       DESC,
						cms.correct_outcomes DESC,
						(CASE WHEN comp.match_id IS NOT NULL AND ump.match_id IS NULL THEN 1 ELSE 0 END) ASC,
						bm.created_at ASC,
						cms.user_id ASC
				) AS rank
			FROM competition_match_scores cms
			INNER JOIN competitions comp ON comp.id = cms.competition_id
			INNER JOIN users u           ON u.id = cms.user_id
			INNER JOIN board_members bm  ON bm.board_id = comp.board_id AND bm.user_id = cms.user_id
			LEFT JOIN user_match_score_picks ump ON ump.user_id = cms.user_id AND ump.match_id = comp.match_id
			WHERE cms.competition_id = $1
		),
		filtered AS (
			SELECT *, COUNT(*) OVER () AS total
			FROM ranked
			WHERE $4::text = ''
			   OR username   ILIKE '%' || $4 || '%'
			   OR first_name ILIKE '%' || $4 || '%'
			   OR last_name  ILIKE '%' || $4 || '%'
		)
		SELECT
			user_id, username, first_name, last_name, role, joined_at,
			rank, total_points,
			exact_hits, correct_outcomes,
			total
		FROM filtered
		ORDER BY ` + rowOrder + `
		OFFSET $2 LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, competitionID, offset, limit, q)
	if err != nil {
		return nil, handleDBError(err, resourceCompetitionScore)
	}
	defer rows.Close()

	leaderboard := &domain.CompetitionLeaderboardPage{
		Members:    []*domain.CompetitionLeaderboardEntry{},
		Pagination: domain.Pagination{Page: page, Limit: limit},
	}

	for rows.Next() {
		entry := &domain.CompetitionLeaderboardEntry{}
		score := &domain.MatchScore{}
		var total int

		if err := rows.Scan(
			&entry.Member.UserID,
			&entry.Member.UserName,
			&entry.Member.FirstName,
			&entry.Member.LastName,
			&entry.Member.Role,
			&entry.Member.JoinedAt,
			&entry.Rank,
			&score.Total,
			&score.ExactHits,
			&score.CorrectOutcomes,
			&total,
		); err != nil {
			return nil, handleDBError(err, resourceCompetitionScore)
		}

		entry.Score = score
		leaderboard.Pagination.Total = total
		leaderboard.Members = append(leaderboard.Members, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, handleDBError(err, resourceCompetitionScore)
	}

	if len(leaderboard.Members) == 0 {
		if err := r.db.QueryRowContext(ctx,
			`SELECT COUNT(*)
			 FROM competition_match_scores cms
			 INNER JOIN users u ON u.id = cms.user_id
			 WHERE cms.competition_id = $1
			   AND ($2::text = ''
			        OR u.username   ILIKE '%' || $2 || '%'
			        OR u.first_name ILIKE '%' || $2 || '%'
			        OR u.last_name  ILIKE '%' || $2 || '%')`,
			competitionID, q,
		).Scan(&leaderboard.Pagination.Total); err != nil && err != sql.ErrNoRows {
			return nil, handleDBError(err, resourceCompetitionScore)
		}
	}

	leaderboard.Pagination.HasMore = page*limit < leaderboard.Pagination.Total
	return leaderboard, nil
}
