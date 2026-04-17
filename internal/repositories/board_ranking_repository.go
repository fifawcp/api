package repositories

import (
	"context"
	"database/sql"

	"github.com/ncondes/fifawcp/internal/domain"
	"github.com/ncondes/fifawcp/internal/infrastructure/config"
)

type BoardRankingRepository struct {
	db  *sql.DB
	cfg *config.Config
}

func NewBoardRankingRepository(
	db *sql.DB,
	cfg *config.Config,
) *BoardRankingRepository {
	return &BoardRankingRepository{
		db:  db,
		cfg: cfg,
	}
}

func (r *BoardRankingRepository) GetBoardRanking(
	ctx context.Context,
	boardID string,
) ([]*domain.BoardRanking, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `SELECT
		board_id,
		user_id,
		total_points,
		global_points,
		detailed_points,
		exact_hits,
		correct_outcomes,
		updated_at
	FROM board_rankings
	WHERE board_id = $1
	ORDER BY total_points DESC
	`

	boardRankings := []*domain.BoardRanking{}

	rows, err := r.db.QueryContext(ctx, query, boardID)
	if err != nil {
		return nil, handleDBError(err, resourceBoardRanking)
	}
	defer rows.Close()

	for rows.Next() {
		var boardRanking domain.BoardRanking
		if err := rows.Scan(
			&boardRanking.BoardID,
			&boardRanking.UserID,
			&boardRanking.TotalPoints,
			&boardRanking.GlobalPoints,
			&boardRanking.DetailedPoints,
			&boardRanking.ExactHits,
			&boardRanking.CorrectOutcomes,
			&boardRanking.UpdatedAt,
		); err != nil {
			return nil, handleDBError(err, resourceBoardRanking)
		}

		boardRankings = append(boardRankings, &boardRanking)
	}

	return boardRankings, nil
}
