package snapshot

import (
	"context"
	"database/sql"
)

type Snapshot struct {
	Matches  []MatchResult `json:"matches"`
	FairPlay []FairPlayRow `json:"fair_play"`
}

type MatchResult struct {
	ID               int64 `json:"id"`
	HomeScore        int   `json:"home_score"`
	AwayScore        int   `json:"away_score"`
	HomePenaltyScore *int  `json:"home_penalty_score"`
	AwayPenaltyScore *int  `json:"away_penalty_score"`
}

type FairPlayRow struct {
	MatchID                     int64  `json:"match_id"`
	TeamFIFACode                string `json:"team_fifa_code"`
	YellowCards                 int    `json:"yellow_cards"`
	IndirectRedCards            int    `json:"indirect_red_cards"`
	DirectRedCards              int    `json:"direct_red_cards"`
	YellowCardAndDirectRedCards int    `json:"yellow_direct_red_cards"`
}

// Export reads the canonical snapshot — finished matches and all fair-play rows —
// from a database (read-only; SELECT only). Shared by export and the -from-prod seed.
func Export(ctx context.Context, pgDB *sql.DB) (*Snapshot, error) {
	matches, err := exportMatches(ctx, pgDB)
	if err != nil {
		return nil, err
	}

	fairPlay, err := exportFairPlay(ctx, pgDB)
	if err != nil {
		return nil, err
	}

	return &Snapshot{Matches: matches, FairPlay: fairPlay}, nil
}

func exportMatches(ctx context.Context, pgDB *sql.DB) ([]MatchResult, error) {
	rows, err := pgDB.QueryContext(ctx, `
		SELECT id, home_score, away_score, home_penalty_score, away_penalty_score
		FROM matches
		WHERE status = 'finished'
		ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	matches := make([]MatchResult, 0)
	for rows.Next() {
		var match MatchResult
		if err := rows.Scan(
			&match.ID, &match.HomeScore, &match.AwayScore,
			&match.HomePenaltyScore, &match.AwayPenaltyScore,
		); err != nil {
			return nil, err
		}
		matches = append(matches, match)
	}

	return matches, rows.Err()
}

func exportFairPlay(ctx context.Context, pgDB *sql.DB) ([]FairPlayRow, error) {
	rows, err := pgDB.QueryContext(ctx, `
		SELECT match_id, team_fifa_code, yellow_cards, indirect_red_cards,
		       direct_red_cards, yellow_direct_red_cards
		FROM match_fair_play
		ORDER BY match_id, team_fifa_code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fairPlay := make([]FairPlayRow, 0)
	for rows.Next() {
		var row FairPlayRow
		if err := rows.Scan(
			&row.MatchID, &row.TeamFIFACode, &row.YellowCards,
			&row.IndirectRedCards, &row.DirectRedCards, &row.YellowCardAndDirectRedCards,
		); err != nil {
			return nil, err
		}
		fairPlay = append(fairPlay, row)
	}

	return fairPlay, rows.Err()
}
