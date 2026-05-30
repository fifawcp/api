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

type PlayerRepository struct {
	db    *sql.DB
	cfg   *config.Config
	teams *domain.TeamLookup
}

func NewPlayerRepository(db *sql.DB, cfg *config.Config, teams *domain.TeamLookup) *PlayerRepository {
	return &PlayerRepository{db: db, cfg: cfg, teams: teams}
}

const playerSelectColumns = `id, team_fifa_code, name, first_name, last_name, age, position, club_name`

const playerSearchHaystack = `unaccent(name || ' ' || COALESCE(first_name, '') || ' ' || COALESCE(last_name, ''))`

func (r *PlayerRepository) SearchPlayers(
	ctx context.Context,
	filters domain.PlayerSearchFilters,
	page, limit int,
) (*domain.PlayerPage, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	conditions := []string{}
	args := []any{}
	argIndex := 1

	for _, token := range strings.Fields(filters.Query) {
		conditions = append(conditions,
			playerSearchHaystack+" ILIKE '%' || unaccent($"+strconv.Itoa(argIndex)+") || '%'")
		args = append(args, token)
		argIndex++
	}
	if len(filters.TeamFifaCodes) > 0 {
		conditions = append(conditions, "team_fifa_code = ANY($"+strconv.Itoa(argIndex)+")")
		args = append(args, pq.Array(filters.TeamFifaCodes))
		argIndex++
	}
	if len(filters.Positions) > 0 {
		positions := make([]string, len(filters.Positions))
		for index, position := range filters.Positions {
			positions[index] = string(position)
		}
		conditions = append(conditions, "position = ANY($"+strconv.Itoa(argIndex)+")")
		args = append(args, pq.Array(positions))
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	offset := (page - 1) * limit
	args = append(args, limit, offset)

	query := `
		SELECT ` + playerSelectColumns + `, COUNT(*) OVER () AS total
		FROM players
		` + whereClause + `
		ORDER BY name ASC, id ASC
		LIMIT $` + strconv.Itoa(argIndex) + ` OFFSET $` + strconv.Itoa(argIndex+1)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, handleDBError(err, resourcePlayer)
	}
	defer rows.Close()

	result := &domain.PlayerPage{
		Players:    []*domain.Player{},
		Pagination: domain.Pagination{Page: page, Limit: limit},
	}

	for rows.Next() {
		var player domain.Player
		var total int
		if err := r.scanPlayerInto(rows, &player, &total); err != nil {
			return nil, handleDBError(err, resourcePlayer)
		}
		result.Pagination.Total = total
		result.Players = append(result.Players, &player)
	}

	if err := rows.Err(); err != nil {
		return nil, handleDBError(err, resourcePlayer)
	}

	result.Pagination.HasMore = page*limit < result.Pagination.Total
	return result, nil
}

func (r *PlayerRepository) GetPlayersByIDs(ctx context.Context, ids []int64) ([]*domain.Player, error) {
	if len(ids) == 0 {
		return []*domain.Player{}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `SELECT ` + playerSelectColumns + ` FROM players WHERE id = ANY($1)`

	rows, err := r.db.QueryContext(ctx, query, pq.Array(ids))
	if err != nil {
		return nil, handleDBError(err, resourcePlayer)
	}
	defer rows.Close()

	players := []*domain.Player{}
	for rows.Next() {
		var player domain.Player
		if err := r.scanPlayerInto(rows, &player, nil); err != nil {
			return nil, handleDBError(err, resourcePlayer)
		}
		players = append(players, &player)
	}

	return players, rows.Err()
}

func (r *PlayerRepository) UpsertPlayers(ctx context.Context, players []*domain.Player) error {
	if len(players) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	const columnsPerRow = 8
	values := make([]string, 0, len(players))
	args := make([]any, 0, len(players)*columnsPerRow)
	argIndex := 1

	for _, player := range players {
		placeholders := make([]string, columnsPerRow)
		for offset := range placeholders {
			placeholders[offset] = "$" + strconv.Itoa(argIndex+offset)
		}
		values = append(values, "("+strings.Join(placeholders, ",")+")")

		clubName := ""
		if player.Club != nil {
			clubName = player.Club.Name
		}

		args = append(args,
			player.ID,
			player.Team.FifaCode,
			player.Name,
			nullableString(player.FirstName),
			nullableString(player.LastName),
			nullableInt(player.Age),
			string(player.Position),
			nullableString(clubName),
		)
		argIndex += columnsPerRow
	}

	query := `
		INSERT INTO players (
			id, team_fifa_code, name, first_name, last_name, age, position, club_name
		) VALUES ` + strings.Join(values, ",") + `
		ON CONFLICT (id) DO UPDATE SET
			team_fifa_code = EXCLUDED.team_fifa_code,
			name           = EXCLUDED.name,
			first_name     = EXCLUDED.first_name,
			last_name      = EXCLUDED.last_name,
			age            = EXCLUDED.age,
			position       = EXCLUDED.position,
			club_name      = EXCLUDED.club_name`

	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return handleDBError(err, resourcePlayer)
	}

	return nil
}

func (r *PlayerRepository) scanPlayerInto(rows *sql.Rows, player *domain.Player, total *int) error {
	var (
		teamFifaCode string
		firstName    sql.NullString
		lastName     sql.NullString
		age          sql.NullInt32
		clubName     sql.NullString
		position     string
	)

	dest := []any{
		&player.ID, &teamFifaCode, &player.Name,
		&firstName, &lastName, &age, &position, &clubName,
	}
	if total != nil {
		dest = append(dest, total)
	}

	if err := rows.Scan(dest...); err != nil {
		return err
	}

	player.Team = r.teams.Get(teamFifaCode)
	player.Position = domain.PlayerPosition(position)
	player.FirstName = firstName.String
	player.LastName = lastName.String
	if age.Valid {
		value := int(age.Int32)
		player.Age = &value
	}
	if clubName.Valid {
		player.Club = &domain.PlayerClub{Name: clubName.String}
	}

	return nil
}
