package repositories

import (
	"context"
	"database/sql"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
)

type TeamRepository struct {
	db  *sql.DB
	cfg *config.Config
}

func NewTeamRepository(db *sql.DB, cfg *config.Config) *TeamRepository {
	return &TeamRepository{
		db:  db,
		cfg: cfg,
	}
}

func (r *TeamRepository) GetAllTeams(ctx context.Context) ([]*domain.Team, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `SELECT
		fifa_code,
		name_translations,
		flag_url,
		group_code
	FROM team_localized
	ORDER BY group_code, fifa_code`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, handleDBError(err, resourceTeam)
	}
	defer rows.Close()

	teams := []*domain.Team{}
	for rows.Next() {
		var t domain.Team
		if err := rows.Scan(
			&t.FifaCode,
			&t.Name,
			&t.FlagURL,
			&t.GroupCode,
		); err != nil {
			return nil, handleDBError(err, resourceTeam)
		}

		teams = append(teams, &t)
	}

	return teams, rows.Err()
}
