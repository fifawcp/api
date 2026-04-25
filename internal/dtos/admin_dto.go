package dtos

type BulkUpdateMatchesResultDto struct {
	Matches []BulkUpdateMatchResultDto `json:"matches" validate:"required,min_array_len=1,dive"`
}

type UpdateMatchResultDto struct {
	HomeScore *int `json:"home_score" validate:"required,gte=0,lt=100" example:"2"`
	AwayScore *int `json:"away_score" validate:"required,gte=0,lt=100" example:"1"`
}

type BulkUpdateMatchResultDto struct {
	ID int64 `json:"id" validate:"required"`
	UpdateMatchResultDto
}

type ResolveThirdPlaceConflictDto struct {
	TeamFifaCodes []string `json:"team_fifa_codes" validate:"required,min=8,max=8,dive,required"`
}
