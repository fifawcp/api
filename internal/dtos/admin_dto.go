package dtos

type BulkUpdateMatchesResultDto struct {
	Matches []BulkUpdateMatchResultDto `json:"matches" validate:"required,min_array_len=1,dive"`
}

type UpdateMatchResultDto struct {
	HomeScore        *int `json:"home_score" validate:"required,gte=0,lte=20" example:"2"`
	AwayScore        *int `json:"away_score" validate:"required,gte=0,lte=20" example:"2"`
	HomePenaltyScore *int `json:"home_penalty_score" validate:"omitempty,gte=0,lte=20" example:"5"`
	AwayPenaltyScore *int `json:"away_penalty_score" validate:"omitempty,gte=0,lte=20" example:"4"`
}

type BulkUpdateMatchResultDto struct {
	ID int64 `json:"id" validate:"required"`
	UpdateMatchResultDto
}

type ResolveThirdPlaceConflictDto struct {
	TeamFifaCodes []string `json:"team_fifa_codes" validate:"required,len=8,unique,dive,required,fifa_code"`
}
