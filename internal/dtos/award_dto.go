package dtos

type SaveAwardPicksDto struct {
	Picks []AwardPickDto `json:"picks" validate:"required,max=4,unique=AwardType,dive"`
}

type AwardPickDto struct {
	AwardType string `json:"award_type" validate:"required,oneof=golden_boot golden_ball golden_glove young_player" example:"golden_boot"`
	PlayerID  int64  `json:"player_id" validate:"required" example:"276"`
}
