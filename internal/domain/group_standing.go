package domain

import "context"

type Group struct {
	Code      string           `json:"code"`
	Standings []*GroupStanding `json:"standings"`
}

type GroupStanding struct {
	Position       int  `json:"position"`
	Team           Team `json:"team"`
	MatchesPlayed  int  `json:"matches_played"`
	Wins           int  `json:"wins"`
	Draws          int  `json:"draws"`
	Losses         int  `json:"losses"`
	GoalsFor       int  `json:"goals_for"`
	GoalsAgainst   int  `json:"goals_against"`
	GoalDifference int  `json:"goal_difference"`
	Points         int  `json:"points"`
}

type ThirdPlaceTeam struct {
	FifaCode       string `json:"fifa_code"`
	GroupCode      string `json:"group_code"`
	Points         int    `json:"points"`
	GoalDifference int    `json:"goal_difference"`
	GoalsFor       int    `json:"goals_for"`
}

type SyncGroupStageOutcomes struct {
	IsGroupStageFinished bool                    `json:"is_group_stage_finished"`
	PromotionOutcome     *PromoteThirdPlaceTeams `json:"promotion_outcome,omitempty"`
}

type PromoteThirdPlaceTeams struct {
	Status      PromotionStatus        `json:"status"`
	Assignments []ThirdPlaceAssignment `json:"assignments,omitempty"`
	Candidates  []ThirdPlaceCandidate  `json:"candidates,omitempty"`
}

type ThirdPlaceCandidate struct {
	Position int    `json:"position"`
	FifaCode string `json:"fifa_code"`
	IsTied   bool   `json:"is_tied"`
}

type PromotionStatus string

const (
	PromotionStatusCompleted PromotionStatus = "completed"
	PromotionStatusConflict  PromotionStatus = "conflict"
)

type ThirdPlaceAssignment struct {
	MatchID          int64  `json:"match_id"`
	AwayTeamFifaCode string `json:"away_team_fifa_code"`
}

type GroupStandingRepository interface {
	GetGroupStandings(ctx context.Context, groupCodes []string, position *int64) ([]*GroupStanding, error)
	UpdateGroupStandings(ctx context.Context, standings []*GroupStanding) error
	GetThirdPlaceGroups(ctx context.Context) ([]*ThirdPlaceTeam, error)
}
