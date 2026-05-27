package dtos

import "github.com/fifawcp/api/internal/domain"

type MatchResponseDto struct {
	*domain.Match
	UserScorePick *UserScorePickDto `json:"user_score_pick,omitempty"`
}

type UserScorePickDto struct {
	HomeScore int `json:"home_score"`
	AwayScore int `json:"away_score"`
}

type SaveMatchScorePickDto struct {
	HomeScore *int `json:"home_score" validate:"required,gte=0,lte=20" example:"2"`
	AwayScore *int `json:"away_score" validate:"required,gte=0,lte=20" example:"1"`
}
