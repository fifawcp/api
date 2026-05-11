package repositories

import (
	"context"
	"database/sql"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
)

type MatchFairPlayRepository struct {
	db  *sql.DB
	cfg *config.Config
}

func NewMatchFairPlayRepository(db *sql.DB, cfg *config.Config) *MatchFairPlayRepository {
	return &MatchFairPlayRepository{db: db, cfg: cfg}
}

func (r *MatchFairPlayRepository) Upsert(ctx context.Context, records []domain.MatchFairPlay) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `
		INSERT INTO match_fair_play
			(match_id, team_fifa_code, yellow_cards, indirect_red_cards, direct_red_cards, yellow_direct_red_cards)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (match_id, team_fifa_code) DO UPDATE SET
			yellow_cards            = EXCLUDED.yellow_cards,
			indirect_red_cards      = EXCLUDED.indirect_red_cards,
			direct_red_cards        = EXCLUDED.direct_red_cards,
			yellow_direct_red_cards = EXCLUDED.yellow_direct_red_cards,
			updated_at              = NOW()`

	for _, record := range records {
		_, err := r.db.ExecContext(ctx, query,
			record.MatchID,
			record.TeamFIFACode,
			record.YellowCards,
			record.IndirectRedCards,
			record.DirectRedCards,
			record.YellowCardAndDirectRedCards,
		)
		if err != nil {
			return handleDBError(err, resourceMatchFairPlay)
		}
	}

	return nil
}

func (r *MatchFairPlayRepository) GetFairPlayTotalsByGroup(ctx context.Context, groupCode string) (map[string]int, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `SELECT
		mfp.team_fifa_code,
		SUM(
			(-1 * mfp.yellow_cards) +
			(-3 * mfp.indirect_red_cards) +
			(-4 * mfp.direct_red_cards) +
			(-5 * mfp.yellow_direct_red_cards)
		) AS fair_play_score
	FROM match_fair_play mfp
	JOIN matches m ON m.id = mfp.match_id
	WHERE m.group_code = $1 AND m.status = 'finished'
	GROUP BY mfp.team_fifa_code`

	rows, err := r.db.QueryContext(ctx, query, groupCode)
	if err != nil {
		return nil, handleDBError(err, resourceMatchFairPlay)
	}
	defer rows.Close()

	// map[teamFIFACode]totalPoints (all values negative or zero)
	totals := make(map[string]int)
	for rows.Next() {
		var teamFIFACode string
		var totalPoints int

		err := rows.Scan(&teamFIFACode, &totalPoints)
		if err != nil {
			return nil, handleDBError(err, resourceMatchFairPlay)
		}

		totals[teamFIFACode] = totalPoints
	}

	return totals, nil
}
