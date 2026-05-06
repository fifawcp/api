package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/lib/pq"
)

type GroupStandingRepository struct {
	db    *sql.DB
	cfg   *config.Config
	teams *domain.TeamLookup
}

func NewGroupStandingRepository(
	db *sql.DB,
	cfg *config.Config,
	teams *domain.TeamLookup,
) *GroupStandingRepository {
	return &GroupStandingRepository{
		db:    db,
		cfg:   cfg,
		teams: teams,
	}
}

func (r *GroupStandingRepository) GetGroupStandings(ctx context.Context, groupCodes []string, position *int64) ([]*domain.GroupStanding, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	var args []any

	baseQuery := `SELECT
    gs.position,
    gs.matches_played,
    gs.wins,
    gs.draws,
    gs.losses,
    gs.goals_for,
    gs.goals_against,
    gs.goal_difference,
    gs.points,
    gs.fifa_code
	FROM group_standings gs`

	query := baseQuery

	if len(groupCodes) > 0 {
		placeholders := make([]string, len(groupCodes))

		for i := range groupCodes {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
			args = append(args, groupCodes[i])
		}

		query += ` WHERE gs.group_code IN (` + strings.Join(placeholders, ",") + `)`
	}

	if position != nil {
		query += ` AND gs.position = $` + strconv.Itoa(len(args)+1)
		args = append(args, position)
	}

	query += ` ORDER BY gs.group_code, gs.position`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, handleDBError(err, resourceGroupStanding)
	}
	defer rows.Close()

	groupStandings := []*domain.GroupStanding{}

	for rows.Next() {
		var groupStanding domain.GroupStanding
		var fifaCode string
		err := rows.Scan(
			&groupStanding.Position,
			&groupStanding.MatchesPlayed,
			&groupStanding.Wins,
			&groupStanding.Draws,
			&groupStanding.Losses,
			&groupStanding.GoalsFor,
			&groupStanding.GoalsAgainst,
			&groupStanding.GoalDifference,
			&groupStanding.Points,
			&fifaCode,
		)
		if err != nil {
			return nil, handleDBError(err, resourceGroupStanding)
		}

		if team := r.teams.Get(fifaCode); team != nil {
			groupStanding.Team = *team
		}

		groupStandings = append(groupStandings, &groupStanding)
	}

	return groupStandings, nil
}

func (r *GroupStandingRepository) UpdateGroupStandings(ctx context.Context, standings []*domain.GroupStanding) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return handleDBError(err, resourceGroupStanding)
	}
	defer tx.Rollback()

	query := `UPDATE group_standings SET 
		position = $2,
		matches_played = $3,
		wins = $4,
		draws = $5,
		losses = $6,
		goals_for = $7,
		goals_against = $8,
		goal_difference = $9,
		points = $10,
		updated_at = NOW()
	WHERE fifa_code = $1`

	statement, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return handleDBError(err, resourceGroupStanding)
	}
	defer statement.Close()

	for _, standing := range standings {
		_, err := statement.ExecContext(ctx,
			standing.Team.FifaCode,
			standing.Position,
			standing.MatchesPlayed,
			standing.Wins,
			standing.Draws,
			standing.Losses,
			standing.GoalsFor,
			standing.GoalsAgainst,
			standing.GoalDifference,
			standing.Points,
		)
		if err != nil {
			return handleDBError(err, resourceGroupStanding)
		}
	}

	return tx.Commit()
}

func (r *GroupStandingRepository) GetThirdPlaceGroups(ctx context.Context) ([]*domain.ThirdPlaceTeam, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `SELECT
		t.fifa_code,
		t.group_code,
		gs.points,
		gs.goal_difference,
		gs.goals_for
	FROM group_standings gs
	JOIN teams t ON gs.fifa_code = t.fifa_code
	WHERE gs.position = 3
	ORDER BY
		gs.points DESC,
		gs.goal_difference DESC,
		gs.goals_for DESC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, handleDBError(err, resourceGroupStanding)
	}
	defer rows.Close()

	teams := []*domain.ThirdPlaceTeam{}

	for rows.Next() {
		var team domain.ThirdPlaceTeam

		err := rows.Scan(
			&team.FifaCode,
			&team.GroupCode,
			&team.Points,
			&team.GoalDifference,
			&team.GoalsFor,
		)
		if err != nil {
			return nil, handleDBError(err, resourceGroupStanding)
		}

		teams = append(teams, &team)
	}

	return teams, nil
}

func (r *GroupStandingRepository) GetBestThirdPlaceTeamFromSlot(ctx context.Context, slotGroupCodes []string) (*string, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `SELECT t.fifa_code
  FROM group_standings gs
  JOIN teams t ON gs.fifa_code = t.fifa_code
  WHERE gs.group_code = ANY($1) AND gs.position = 3
  ORDER BY gs.points DESC, gs.goal_difference DESC, gs.goals_for DESC
  LIMIT 1`

	var fifaCode *string

	err := r.db.QueryRowContext(
		ctx,
		query,
		pq.Array(slotGroupCodes),
	).Scan(&fifaCode)
	if err != nil {
		return nil, handleDBError(err, resourceGroupStanding)
	}

	return fifaCode, nil
}
