package domain

import (
	"context"
	"encoding/json"
	"fmt"
)

type Team struct {
	FifaCode  string    `json:"fifa_code"`
	Name      TeamNames `json:"name"`
	FlagURL   string    `json:"flag_url"`
	GroupCode string    `json:"group_code"`
}

type TeamNames map[string]string

func (n *TeamNames) Scan(value any) error {
	if value == nil {
		*n = TeamNames{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("unsupported type for TeamNames: %T", value)
	}

	if len(bytes) == 0 {
		*n = TeamNames{}
		return nil
	}

	var m map[string]string
	if err := json.Unmarshal(bytes, &m); err != nil {
		return fmt.Errorf("TeamNames unmarshal: %w", err)
	}

	*n = TeamNames(m)
	return nil
}

type TeamRepository interface {
	GetAllTeams(ctx context.Context) ([]*Team, error)
}

type TeamLookup struct {
	byCode map[string]*Team
}

func NewTeamLookup(teams []*Team) *TeamLookup {
	lookup := &TeamLookup{}
	lookup.Set(teams)
	return lookup
}

func (lookup *TeamLookup) Set(teams []*Team) {
	byCode := make(map[string]*Team, len(teams))
	for _, team := range teams {
		byCode[team.FifaCode] = team
	}
	lookup.byCode = byCode
}

func (lookup *TeamLookup) Get(fifaCode string) *Team {
	if fifaCode == "" {
		return nil
	}

	return lookup.byCode[fifaCode]
}
