package domain

import "context"

type PlayerPosition string

const (
	PlayerPositionGoalkeeper PlayerPosition = "goalkeeper"
	PlayerPositionDefender   PlayerPosition = "defender"
	PlayerPositionMidfielder PlayerPosition = "midfielder"
	PlayerPositionAttacker   PlayerPosition = "attacker"
)

func (position PlayerPosition) IsValid() bool {
	switch position {
	case PlayerPositionGoalkeeper, PlayerPositionDefender, PlayerPositionMidfielder, PlayerPositionAttacker:
		return true
	}
	return false
}

type PlayerClub struct {
	Name    string `json:"name"`
	LogoURL string `json:"logo_url"`
}

type Player struct {
	ID          int64          `json:"id"`
	Team        *Team          `json:"team"`
	Name        string         `json:"name"`
	FirstName   string         `json:"first_name,omitempty"`
	LastName    string         `json:"last_name,omitempty"`
	Age         *int           `json:"age,omitempty"`
	Nationality string         `json:"nationality,omitempty"`
	Position    PlayerPosition `json:"position"`
	PhotoURL    string         `json:"photo_url,omitempty"`
	Club        *PlayerClub    `json:"club,omitempty"`
}

type PlayerSearchFilters struct {
	Query         string
	TeamFifaCodes []string
	Positions     []PlayerPosition
}

type PlayerPage struct {
	Players    []*Player  `json:"players"`
	Pagination Pagination `json:"-"`
}

type PlayerRepository interface {
	SearchPlayers(ctx context.Context, filters PlayerSearchFilters, page, limit int) (*PlayerPage, error)
	GetPlayersByIDs(ctx context.Context, ids []int64) ([]*Player, error)
	UpsertPlayers(ctx context.Context, players []*Player) error
}
