package domain

import (
	"context"
	"time"
)

type CompetitionType string

const (
	CompetitionTypePickem CompetitionType = "pickem"
	CompetitionTypeMatch  CompetitionType = "match"
)

type Competition struct {
	ID        int64             `json:"id" example:"1"`
	BoardID   int64             `json:"board_id" example:"1"`
	Type      CompetitionType   `json:"type" example:"pickem"`
	Name      string            `json:"name" example:"Pick'em"`
	CreatedBy *string           `json:"-"`
	CreatedAt time.Time         `json:"created_at" example:"2026-01-15T10:30:00Z"`
	Scope     *CompetitionScope `json:"scope,omitempty"`
}

// CompetitionScope is only populated for match competitions
type CompetitionScope struct {
	Stages        []MatchStageCode `json:"stages"`
	TeamFifaCodes []string         `json:"team_fifa_codes"`
}

type CompetitionListItem struct {
	Competition
	Viewer CompetitionViewer `json:"viewer"`
}

type CompetitionViewer struct {
	Rank        int `json:"rank" example:"3"`
	TotalPoints int `json:"total_points" example:"150"`
}

type CompetitionLeaderboardMember struct {
	UserID    string          `json:"user_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	UserName  string          `json:"username" example:"johndoe"`
	FirstName string          `json:"first_name" example:"John"`
	LastName  string          `json:"last_name" example:"Doe"`
	Role      BoardMemberRole `json:"role" example:"member"`
	JoinedAt  time.Time       `json:"joined_at" example:"2026-01-16T10:30:00Z"`
}

type PickemScore struct {
	Total               int `json:"total" example:"50"`
	GroupExactPositions int `json:"group_exact_positions" example:"4"`
	GroupQualifierHits  int `json:"group_qualifier_hits" example:"3"`
	BestThirdHits       int `json:"best_third_hits" example:"2"`
	BracketHits         int `json:"bracket_hits" example:"5"`
}

type MatchScore struct {
	Total           int `json:"total" example:"12"`
	ExactHits       int `json:"exact_hits" example:"2"`
	CorrectOutcomes int `json:"correct_outcomes" example:"3"`
}

type CompetitionLeaderboardEntry struct {
	Member CompetitionLeaderboardMember `json:"member"`
	Rank   int                          `json:"rank" example:"1"`
	Score  any                          `json:"score"`
}

type CompetitionLeaderboardPage struct {
	Members    []*CompetitionLeaderboardEntry `json:"members"`
	Pagination Pagination                     `json:"-"`
}

type CompetitionRepository interface {
	CreateCompetition(ctx context.Context, competition *Competition) error
	GetBoardCompetitions(ctx context.Context, boardID int64, viewerUserID string) ([]*CompetitionListItem, error)
	DeleteCompetition(ctx context.Context, boardID, competitionID int64) error
	GetAllPickemIDs(ctx context.Context) ([]int64, error)
	FindMatchCompetitionsByMatches(ctx context.Context, matchIDs []int64) ([]int64, error)
	GetGlobalCompetitions(ctx context.Context) (pickem *Competition, match *Competition, err error)
}
