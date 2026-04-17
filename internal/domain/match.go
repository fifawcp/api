package domain

import (
	"context"
	"time"
)

type MatchStatus string

const (
	MatchStatusScheduled MatchStatus = "scheduled"
	MatchStatusFinished  MatchStatus = "finished"
)

type MatchStageCode string

const (
	MatchStageCodeGroupStage    MatchStageCode = "group_stage"
	MatchStageCodeRoundOf32     MatchStageCode = "round_of_32"
	MatchStageCodeRoundOf16     MatchStageCode = "round_of_16"
	MatchStageCodeQuarterFinals MatchStageCode = "quarterfinals"
	MatchStageCodeSemiFinals    MatchStageCode = "semifinals"
	MatchStageCodeThirdPlace    MatchStageCode = "third_place"
	MatchStageCodeFinal         MatchStageCode = "final"
)

type Match struct {
	ID                 int64          `json:"id"`
	StageCode          MatchStageCode `json:"stage_code"`
	GroupCode          *string        `json:"group_code"`
	HomeTeam           Team           `json:"home_team"`
	AwayTeam           Team           `json:"away_team"`
	KickoffAt          time.Time      `json:"kickoff_at"`
	Status             MatchStatus    `json:"status"`
	HomeScore          *int           `json:"home_score"`
	AwayScore          *int           `json:"away_score"`
	WinnerTeamFifaCode *string        `json:"winner_team_fifa_code"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

type MatchFilters struct {
	GroupCodes    []string         `json:"group_codes"`
	StageCodes    []MatchStageCode `json:"stage_codes"`
	Status        MatchStatus      `json:"status"`
	TeamFifaCodes []string         `json:"team_fifa_codes"`
	FromDate      *time.Time       `json:"from_date"`
	ToDate        *time.Time       `json:"to_date"`
}

// TODO: think in the future how to update knockout matchs if tied (defined by penalties)
type MatchResultUpdate struct {
	MatchID            int64
	HomeScore          int
	AwayScore          int
	Status             MatchStatus
	WinnerTeamFifaCode *string
}

type MatchTeamUpdate struct {
	MatchID          int64
	HomeTeamFifaCode *string
	AwayTeamFifaCode *string
}

type ThirdPlaceCombination struct {
	QualifyingGroups []string          `json:"qualifying_third_place_groups"`
	Assignments      map[string]string `json:"assignments"`
}

type MatchSlotRule struct {
	Home Source
	Away Source
}

type SourceKind string

const (
	SourceGroupPosition SourceKind = "group_position"
	SourceBestThird     SourceKind = "best_third"
	SourceWinner        SourceKind = "winner"
	SourceLoser         SourceKind = "loser"
)

type Source struct {
	Kind      SourceKind
	Position  int
	GroupCode string
	MatchID   int64
}

var MatchSlotRules = map[int64]MatchSlotRule{
	// Round of 32 - Matches 73-88
	73: {
		Home: Source{Kind: SourceGroupPosition, Position: 2, GroupCode: "A"},
		Away: Source{Kind: SourceGroupPosition, Position: 2, GroupCode: "B"},
	},
	74: {
		Home: Source{Kind: SourceGroupPosition, Position: 1, GroupCode: "E"},
		Away: Source{Kind: SourceBestThird, Position: 3},
	},
	75: {
		Home: Source{Kind: SourceGroupPosition, Position: 1, GroupCode: "F"},
		Away: Source{Kind: SourceGroupPosition, Position: 2, GroupCode: "C"},
	},
	76: {
		Home: Source{Kind: SourceGroupPosition, Position: 1, GroupCode: "C"},
		Away: Source{Kind: SourceGroupPosition, Position: 2, GroupCode: "F"},
	},
	77: {
		Home: Source{Kind: SourceGroupPosition, Position: 1, GroupCode: "I"},
		Away: Source{Kind: SourceBestThird, Position: 3},
	},
	78: {
		Home: Source{Kind: SourceGroupPosition, Position: 2, GroupCode: "E"},
		Away: Source{Kind: SourceGroupPosition, Position: 2, GroupCode: "I"},
	},
	79: {
		Home: Source{Kind: SourceGroupPosition, Position: 1, GroupCode: "A"},
		Away: Source{Kind: SourceBestThird, Position: 3},
	},
	80: {
		Home: Source{Kind: SourceGroupPosition, Position: 1, GroupCode: "L"},
		Away: Source{Kind: SourceBestThird, Position: 3},
	},
	81: {
		Home: Source{Kind: SourceGroupPosition, Position: 1, GroupCode: "D"},
		Away: Source{Kind: SourceBestThird, Position: 3},
	},
	82: {
		Home: Source{Kind: SourceGroupPosition, Position: 1, GroupCode: "G"},
		Away: Source{Kind: SourceBestThird, Position: 3},
	},
	83: {
		Home: Source{Kind: SourceGroupPosition, Position: 2, GroupCode: "K"},
		Away: Source{Kind: SourceGroupPosition, Position: 2, GroupCode: "L"},
	},
	84: {
		Home: Source{Kind: SourceGroupPosition, Position: 1, GroupCode: "H"},
		Away: Source{Kind: SourceGroupPosition, Position: 2, GroupCode: "J"},
	},
	85: {
		Home: Source{Kind: SourceGroupPosition, Position: 1, GroupCode: "B"},
		Away: Source{Kind: SourceBestThird, Position: 3},
	},
	86: {
		Home: Source{Kind: SourceGroupPosition, Position: 1, GroupCode: "J"},
		Away: Source{Kind: SourceGroupPosition, Position: 2, GroupCode: "H"},
	},
	87: {
		Home: Source{Kind: SourceGroupPosition, Position: 1, GroupCode: "K"},
		Away: Source{Kind: SourceBestThird, Position: 3},
	},
	88: {
		Home: Source{Kind: SourceGroupPosition, Position: 2, GroupCode: "D"},
		Away: Source{Kind: SourceGroupPosition, Position: 2, GroupCode: "G"},
	},
	// Round of 16 - Matches 89-96
	89: {
		Home: Source{Kind: SourceWinner, MatchID: 74},
		Away: Source{Kind: SourceWinner, MatchID: 77},
	},
	90: {
		Home: Source{Kind: SourceWinner, MatchID: 73},
		Away: Source{Kind: SourceWinner, MatchID: 75},
	},
	91: {
		Home: Source{Kind: SourceWinner, MatchID: 76},
		Away: Source{Kind: SourceWinner, MatchID: 78},
	},
	92: {
		Home: Source{Kind: SourceWinner, MatchID: 79},
		Away: Source{Kind: SourceWinner, MatchID: 80},
	},
	93: {
		Home: Source{Kind: SourceWinner, MatchID: 83},
		Away: Source{Kind: SourceWinner, MatchID: 84},
	},
	94: {
		Home: Source{Kind: SourceWinner, MatchID: 81},
		Away: Source{Kind: SourceWinner, MatchID: 82},
	},
	95: {
		Home: Source{Kind: SourceWinner, MatchID: 86},
		Away: Source{Kind: SourceWinner, MatchID: 88},
	},
	96: {
		Home: Source{Kind: SourceWinner, MatchID: 85},
		Away: Source{Kind: SourceWinner, MatchID: 87},
	},
	// Quarterfinals - Matches 97-100
	97: {
		Home: Source{Kind: SourceWinner, MatchID: 89},
		Away: Source{Kind: SourceWinner, MatchID: 90},
	},
	98: {
		Home: Source{Kind: SourceWinner, MatchID: 93},
		Away: Source{Kind: SourceWinner, MatchID: 94},
	},
	99: {
		Home: Source{Kind: SourceWinner, MatchID: 91},
		Away: Source{Kind: SourceWinner, MatchID: 92},
	},
	100: {
		Home: Source{Kind: SourceWinner, MatchID: 95},
		Away: Source{Kind: SourceWinner, MatchID: 96},
	},
	// Semifinals - Matches 101-102
	101: {
		Home: Source{Kind: SourceWinner, MatchID: 97},
		Away: Source{Kind: SourceWinner, MatchID: 98},
	},
	102: {
		Home: Source{Kind: SourceWinner, MatchID: 99},
		Away: Source{Kind: SourceWinner, MatchID: 100},
	},
	// Third Place - Match 103
	103: {
		Home: Source{Kind: SourceLoser, MatchID: 101},
		Away: Source{Kind: SourceLoser, MatchID: 102},
	},
	// Final - Match 104
	104: {
		Home: Source{Kind: SourceWinner, MatchID: 101},
		Away: Source{Kind: SourceWinner, MatchID: 102},
	},
}

type MatchRepository interface {
	GetMatches(ctx context.Context, filters MatchFilters) ([]*Match, error)
	UpdateMatchesResult(ctx context.Context, updates []MatchResultUpdate) error
	UpdateMatchTeams(ctx context.Context, updates []MatchTeamUpdate) error
	ResetMatchResult(ctx context.Context, matchID int64) error
	IsGroupFinished(ctx context.Context, groupCode string) (bool, error)
	IsGroupStageFinished(ctx context.Context) (bool, error)
}
