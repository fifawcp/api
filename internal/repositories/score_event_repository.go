package repositories

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
)

type ScoreEventRepository struct {
	db  *sql.DB
	cfg *config.Config
}

func NewScoreEventRepository(db *sql.DB, cfg *config.Config) *ScoreEventRepository {
	return &ScoreEventRepository{
		db:  db,
		cfg: cfg,
	}
}

func (r *ScoreEventRepository) BatchUpsertScoreEvents(ctx context.Context, events []*domain.ScoreEvent) error {
	if len(events) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	var values []string
	var args []any
	argIndex := 1

	for _, e := range events {
		values = append(values, "($"+strconv.Itoa(argIndex)+",$"+strconv.Itoa(argIndex+1)+",$"+strconv.Itoa(argIndex+2)+",$"+strconv.Itoa(argIndex+3)+")")
		args = append(args, e.UserID, string(e.SourceType), e.SourceRef, e.Points)
		argIndex += 4
	}

	// Idempotent upsert - if the score event already exists, it will be updated
	query := `INSERT INTO score_events (user_id, source_type, source_ref, points) VALUES ` +
		strings.Join(values, ",") +
		` ON CONFLICT (user_id, source_type, source_ref) DO UPDATE SET points = EXCLUDED.points`

	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return handleDBError(err, resourceScoreEvent)
	}

	return nil
}
