package main

import (
	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
)

var groupStageOnly = []domain.MatchStageCode{
	domain.MatchStageCodeGroupStage,
}

var bracketStages = []domain.MatchStageCode{
	domain.MatchStageCodeRoundOf32,
	domain.MatchStageCodeRoundOf16,
	domain.MatchStageCodeQuarterFinals,
	domain.MatchStageCodeSemiFinals,
	domain.MatchStageCodeThirdPlace,
	domain.MatchStageCodeFinal,
}

var allStages = []domain.MatchStageCode{
	domain.MatchStageCodeGroupStage,
	domain.MatchStageCodeRoundOf32,
	domain.MatchStageCodeRoundOf16,
	domain.MatchStageCodeQuarterFinals,
	domain.MatchStageCodeSemiFinals,
	domain.MatchStageCodeThirdPlace,
	domain.MatchStageCodeFinal,
}

var conmebolTeams = []string{
	"BRA", "ARG", "URU", "COL", "ECU", "PAR",
}

var uefaTeams = []string{
	"ESP", "FRA", "GER", "NED", "ENG", "POR", "BEL", "CRO", "SUI", "AUT", "SCO", "NOR", "CZE", "SWE", "TUR", "BIH",
}

var competitionTemplates = func() []dtos.CreateCompetitionDto {
	templates := []dtos.CreateCompetitionDto{
		// Pure pickem (no scope per validator)
		{Type: domain.CompetitionTypePickem, Name: "World Cup Pick'em"},
		// Stage-only filters
		{Type: domain.CompetitionTypeMatch, Name: "Knockout Stage", Scope: &dtos.CreateCompetitionScopeDto{Stages: bracketStages}},
		{Type: domain.CompetitionTypeMatch, Name: "Group Stage Only", Scope: &dtos.CreateCompetitionScopeDto{Stages: groupStageOnly}},
		// Single-team tours (all stages)
		{Type: domain.CompetitionTypeMatch, Name: "Colombia Tour", Scope: &dtos.CreateCompetitionScopeDto{Stages: allStages, TeamFifaCodes: []string{"COL"}}},
		{Type: domain.CompetitionTypeMatch, Name: "Brazil Run", Scope: &dtos.CreateCompetitionScopeDto{Stages: allStages, TeamFifaCodes: []string{"BRA"}}},
		{Type: domain.CompetitionTypeMatch, Name: "France Campaign", Scope: &dtos.CreateCompetitionScopeDto{Stages: allStages, TeamFifaCodes: []string{"FRA"}}},
		{Type: domain.CompetitionTypeMatch, Name: "Spain Quest", Scope: &dtos.CreateCompetitionScopeDto{Stages: allStages, TeamFifaCodes: []string{"ESP"}}},
		// Multi-team rivalries
		{Type: domain.CompetitionTypeMatch, Name: "Big Three Knockout", Scope: &dtos.CreateCompetitionScopeDto{Stages: bracketStages, TeamFifaCodes: []string{"ESP", "FRA", "BRA"}}},
		{Type: domain.CompetitionTypeMatch, Name: "South Showdown", Scope: &dtos.CreateCompetitionScopeDto{Stages: allStages, TeamFifaCodes: conmebolTeams}},
		{Type: domain.CompetitionTypeMatch, Name: "European Showdown", Scope: &dtos.CreateCompetitionScopeDto{Stages: allStages, TeamFifaCodes: uefaTeams}},
	}

	// 12 "Group X Only" variants, scoped to that group's 4 teams + group-stage
	for groupCode, teams := range teamsByGroup {
		scopeTeams := append([]string{}, teams...)
		templates = append(templates, dtos.CreateCompetitionDto{
			Type: domain.CompetitionTypeMatch,
			Name: "Group " + groupCode + " Only",
			Scope: &dtos.CreateCompetitionScopeDto{
				Stages:        groupStageOnly,
				TeamFifaCodes: scopeTeams,
			},
		})
	}

	return templates
}()
