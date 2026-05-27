package main

import (
	"fmt"
	"time"
)

// [first, last] match ID range for one stage
type stageMatchRange struct {
	first int64
	last  int64
}

var stageRanges = []stageMatchRange{
	{1, 72},    // group stage
	{73, 88},   // round of 32
	{89, 96},   // round of 16
	{97, 100},  // quarterfinals
	{101, 102}, // semifinals
	{103, 103}, // third place
	{104, 104}, // final
}

// Scenario describes the desired tournament snapshot
//
// StageGroups is partitioned by stage in the same order as stageRanges so
// applyScenarioResults can submit one BulkUpdateMatchesResultDto per stage and
// let SyncGroupStageOutcomes / advanceBracket handle promotion in between
type Scenario struct {
	Name        string
	StageGroups [][]int64

	// AnchorMatchID's kickoff_at is shifted so it sits at AnchorOffset from
	// "now" at seeder run time. All other matches receive the same delta to
	// preserve relative spacing across the schedule
	AnchorMatchID int64
	AnchorOffset  time.Duration
}

const (
	scenarioPreTournament  = "pre_tournament"
	scenarioGroupStageDone = "group_stage_done"
	scenarioRoundOf32Done  = "r32_done"
	scenarioRoundOf16Done  = "r16_done"
	scenarioQuarterfinals  = "qf_done"
	scenarioSemifinalsDone = "sf_done"
	scenarioFinalDone      = "final_done"
)

var ScenarioNames = []string{
	scenarioPreTournament,
	scenarioGroupStageDone,
	scenarioRoundOf32Done,
	scenarioRoundOf16Done,
	scenarioQuarterfinals,
	scenarioSemifinalsDone,
	scenarioFinalDone,
}

var scenarios = map[string]Scenario{
	scenarioPreTournament: {
		Name:          scenarioPreTournament,
		StageGroups:   nil,
		AnchorMatchID: 1,
		AnchorOffset:  24 * time.Hour, // opener is 1 day from now
	},
	scenarioGroupStageDone: {
		Name:          scenarioGroupStageDone,
		StageGroups:   matchIDsByStageUpTo(0),
		AnchorMatchID: 72,
		AnchorOffset:  -24 * time.Hour,
	},
	scenarioRoundOf32Done: {
		Name:          scenarioRoundOf32Done,
		StageGroups:   matchIDsByStageUpTo(1),
		AnchorMatchID: 88,
		AnchorOffset:  -24 * time.Hour,
	},
	scenarioRoundOf16Done: {
		Name:          scenarioRoundOf16Done,
		StageGroups:   matchIDsByStageUpTo(2),
		AnchorMatchID: 96,
		AnchorOffset:  -24 * time.Hour,
	},
	scenarioQuarterfinals: {
		Name:          scenarioQuarterfinals,
		StageGroups:   matchIDsByStageUpTo(3),
		AnchorMatchID: 100,
		AnchorOffset:  -24 * time.Hour,
	},
	scenarioSemifinalsDone: {
		Name:          scenarioSemifinalsDone,
		StageGroups:   matchIDsByStageUpTo(4),
		AnchorMatchID: 102,
		AnchorOffset:  -24 * time.Hour,
	},
	scenarioFinalDone: {
		Name:          scenarioFinalDone,
		StageGroups:   matchIDsByStageUpTo(6),
		AnchorMatchID: 104,
		AnchorOffset:  -48 * time.Hour,
	},
}

// matchIDsByStageUpTo expands stageRanges[0 .. lastStageIndex] into the
// concrete match IDs played in each of those stages, keeping them grouped
// per stage so the scenario loop can apply results one stage at a time
//
// Each outer slice element is one stage; each inner slice is the chronological
// list of match IDs for that stage. The grouping matters: bracket teams for
// stage N+1 are only known AFTER stage N's results are applied (via
// advanceBracket), so RunScenario must process stages sequentially rather
// than dumping every ID into a flat list.
//
// Example: matchIDsByStageUpTo(2) for the r16_done scenario returns
//
//	[ [1..72], [73..88], [89..96] ]  // group stage, R32, R16
func matchIDsByStageUpTo(lastStageIndex int) [][]int64 {
	matchIDsByStage := make([][]int64, 0, lastStageIndex+1)

	// Iterate over each stage and collect the match IDs
	for stageIndex := 0; stageIndex <= lastStageIndex; stageIndex++ {
		stage := stageRanges[stageIndex]
		stageMatchIDs := make([]int64, 0, stage.last-stage.first+1)

		// Collect the match IDs for the current stage
		for matchID := stage.first; matchID <= stage.last; matchID++ {
			stageMatchIDs = append(stageMatchIDs, matchID)
		}

		matchIDsByStage = append(matchIDsByStage, stageMatchIDs)
	}

	return matchIDsByStage
}

func resolveScenario(name string) (Scenario, error) {
	scenario, ok := scenarios[name]
	if !ok {
		return Scenario{}, fmt.Errorf("unknown scenario %q (valid: %v)", name, ScenarioNames)
	}
	return scenario, nil
}
