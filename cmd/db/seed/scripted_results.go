package main

import (
	"context"
	"database/sql"
	"fmt"
	"maps"
)

type ScriptedResult struct {
	HomeScore        int
	AwayScore        int
	HomePenaltyScore *int
	AwayPenaltyScore *int
}

// groupFinish defines who finishes in each position per group (index 0 = 1st)
var groupFinish = map[string][4]string{
	"A": {"MEX", "KOR", "CZE", "RSA"},
	"B": {"SUI", "CAN", "BIH", "QAT"},
	"C": {"BRA", "MAR", "SCO", "HAI"},
	"D": {"USA", "TUR", "AUS", "PAR"},
	"E": {"GER", "ECU", "CIV", "CUW"},
	"F": {"NED", "JPN", "SWE", "TUN"},
	"G": {"BEL", "IRN", "EGY", "NZL"},
	"H": {"ESP", "URU", "CPV", "KSA"},
	"I": {"FRA", "SEN", "NOR", "IRQ"},
	"J": {"ARG", "AUT", "ALG", "JOR"},
	"K": {"POR", "COL", "UZB", "COD"},
	"L": {"ENG", "CRO", "GHA", "PAN"},
}

// Groups designated "strong" so their 3rd-place teams advance to R32
var strongThirdGroups = map[string]bool{
	"A": true, "B": true, "C": true, "D": true,
	"E": true, "F": true, "G": true, "H": true,
}

// positionScoreRubric maps (winnerPos, loserPos) to (winnerScore, loserScore)
// for matches that don't involve the per-group-varying 3rd-vs-4th match
var positionScoreRubric = map[[2]int][2]int{
	{1, 2}: {2, 1},
	{1, 3}: {3, 0},
	{1, 4}: {3, 0},
	{2, 3}: {2, 1},
	{2, 4}: {2, 0},
}

// Strong groups: 3rd wins 3-2 → 3rd's stats: 3pts, GF=4, GA=7, GD=-3
// Weak groups:   3rd wins 1-0 → 3rd's stats: 3pts, GF=2, GA=5, GD=-3
// GDs match across both pools (-3); GF separates: strong=4, weak=2, so the
// 8 strong third-placers (GF=4) outrank all 4 weak (GF=2). Strong-pool ties
// are broken at higher levels of the FIFA list by group letter assignment.
var strongThirdVsFourth = [2]int{3, 2}
var weakThirdVsFourth = [2]int{1, 0}

var knockoutResults = map[int64]ScriptedResult{
	// Round of 32 (73-88)
	73: {HomeScore: 2, AwayScore: 1},
	74: {HomeScore: 3, AwayScore: 0},
	75: {HomeScore: 1, AwayScore: 1, HomePenaltyScore: new(5), AwayPenaltyScore: new(4)},
	76: {HomeScore: 2, AwayScore: 0},
	77: {HomeScore: 2, AwayScore: 1},
	78: {HomeScore: 3, AwayScore: 2},
	79: {HomeScore: 1, AwayScore: 0},
	80: {HomeScore: 0, AwayScore: 0, HomePenaltyScore: new(4), AwayPenaltyScore: new(2)},
	81: {HomeScore: 2, AwayScore: 1},
	82: {HomeScore: 3, AwayScore: 1},
	83: {HomeScore: 1, AwayScore: 0},
	84: {HomeScore: 2, AwayScore: 0},
	85: {HomeScore: 3, AwayScore: 2},
	86: {HomeScore: 1, AwayScore: 1, HomePenaltyScore: new(3), AwayPenaltyScore: new(5)},
	87: {HomeScore: 2, AwayScore: 1},
	88: {HomeScore: 1, AwayScore: 0},
	// Round of 16 (89-96)
	89: {HomeScore: 2, AwayScore: 1},
	90: {HomeScore: 1, AwayScore: 0},
	91: {HomeScore: 3, AwayScore: 1},
	92: {HomeScore: 1, AwayScore: 1, HomePenaltyScore: new(4), AwayPenaltyScore: new(2)},
	93: {HomeScore: 2, AwayScore: 0},
	94: {HomeScore: 1, AwayScore: 0},
	95: {HomeScore: 2, AwayScore: 1},
	96: {HomeScore: 0, AwayScore: 1},
	// Quarterfinals (97-100)
	97:  {HomeScore: 2, AwayScore: 1},
	98:  {HomeScore: 1, AwayScore: 0},
	99:  {HomeScore: 1, AwayScore: 1, HomePenaltyScore: new(5), AwayPenaltyScore: new(4)},
	100: {HomeScore: 3, AwayScore: 2},
	// Semifinals (101-102)
	101: {HomeScore: 2, AwayScore: 1},
	102: {HomeScore: 0, AwayScore: 1},
	// Third place (103)
	103: {HomeScore: 3, AwayScore: 1},
	// Final (104)
	104: {HomeScore: 2, AwayScore: 1},
}

// buildScriptedResults derives the full 104-entry results map. Group-stage
// outcomes are computed from groupFinish + the scoring rubric using the
// canonical home/away teams loaded from the DB. Knockout outcomes come from
// the hand-coded knockoutResults table
func buildScriptedResults(ctx context.Context, db *sql.DB) (map[int64]ScriptedResult, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, home_team_fifa_code, away_team_fifa_code, group_code
		FROM matches
		WHERE stage_code = 'group_stage'
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("query group-stage matches: %w", err)
	}
	defer rows.Close()

	results := make(map[int64]ScriptedResult, 104)
	for rows.Next() {
		var (
			matchID   int64
			homeCode  string
			awayCode  string
			groupCode string
		)
		if err := rows.Scan(&matchID, &homeCode, &awayCode, &groupCode); err != nil {
			return nil, err
		}

		result, err := scriptedGroupResult(groupCode, homeCode, awayCode)
		if err != nil {
			return nil, fmt.Errorf("match %d: %w", matchID, err)
		}

		results[matchID] = result
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	maps.Copy(results, knockoutResults)

	return results, nil
}

func scriptedGroupResult(groupCode, homeCode, awayCode string) (ScriptedResult, error) {
	finish := groupFinish[groupCode]
	homePos := positionInFinish(finish, homeCode)
	awayPos := positionInFinish(finish, awayCode)

	// 3rd vs 4th uses a per-group strong/weak score to spread 3rd-place GF stats.
	if (homePos == 3 && awayPos == 4) || (homePos == 4 && awayPos == 3) {
		score := weakThirdVsFourth
		if strongThirdGroups[groupCode] {
			score = strongThirdVsFourth
		}

		if homePos == 3 {
			return ScriptedResult{HomeScore: score[0], AwayScore: score[1]}, nil
		}

		return ScriptedResult{HomeScore: score[1], AwayScore: score[0]}, nil
	}

	winnerPos, loserPos := homePos, awayPos
	if winnerPos > loserPos {
		winnerPos, loserPos = loserPos, winnerPos
	}

	score, ok := positionScoreRubric[[2]int{winnerPos, loserPos}]
	if !ok {
		return ScriptedResult{}, fmt.Errorf("no rubric for (winner=%d, loser=%d)", winnerPos, loserPos)
	}

	if homePos < awayPos {
		return ScriptedResult{HomeScore: score[0], AwayScore: score[1]}, nil
	}

	return ScriptedResult{HomeScore: score[1], AwayScore: score[0]}, nil
}

func positionInFinish(finish [4]string, fifaCode string) int {
	// Find the position of the team in the group
	for index, code := range finish {
		if code == fifaCode {
			return index + 1
		}
	}

	return 0 // This should never happen
}
