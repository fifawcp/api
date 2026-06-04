package dtos

import "github.com/fifawcp/api/internal/domain"

type CreateCompetitionScopeDto struct {
	Stages        []domain.MatchStageCode `json:"stages" validate:"required,min=1,dive,oneof=group_stage round_of_32 round_of_16 quarterfinals semifinals third_place final"`
	TeamFifaCodes []string                `json:"team_fifa_codes" validate:"omitempty,dive,fifa_code"`
}

type CreateCompetitionDto struct {
	Type    domain.CompetitionType     `json:"type" validate:"required,oneof=pickem match pool" example:"pickem"`
	Name    string                     `json:"name" validate:"required,max=20" example:"Pick'em"`
	Scope   *CreateCompetitionScopeDto `json:"scope,omitempty"`
	MatchID *int64                     `json:"match_id,omitempty" example:"42"`
}
