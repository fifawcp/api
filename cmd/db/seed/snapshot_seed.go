package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/fifawcp/api/cmd/db/snapshot"
	"github.com/fifawcp/api/internal/domain"
)

// RunSnapshot replays a snapshot file (exported from prod) into the dev DB.
func (s *Seeder) RunSnapshot(ctx context.Context, path string) error {
	snap, err := loadSnapshotFile(path)
	if err != nil {
		return err
	}
	s.logger.Info("Loaded snapshot file", "path", path)
	return s.RunSnapshotData(ctx, snap)
}

// RunSnapshotData replays an in-memory snapshot through the same engine prod runs,
// decoupled from the source so it works from a file (-snapshot) or live (-from-prod).
func (s *Seeder) RunSnapshotData(ctx context.Context, snap *snapshot.Snapshot) error {
	s.logger.Info("Running snapshot seed",
		"matches", len(snap.Matches),
		"fair_play", len(snap.FairPlay),
	)

	s.Flush()

	// Flush leaves kickoff_at untouched and a prior scenario may have shifted it;
	// re-pin to the real 2026 dates so the calendar reflects reality.
	if err := s.shiftKickoffDates(ctx, Scenario{AnchorMatchID: 1, UseRealDates: true}); err != nil {
		return fmt.Errorf("restore real kickoff dates: %w", err)
	}

	if err := s.loadFairPlay(ctx, snap.FairPlay); err != nil {
		return fmt.Errorf("load fair play: %w", err)
	}

	matchUsers, err := s.seedParticipants(ctx)
	if err != nil {
		return err
	}

	results, stageGroups := snapshotResults(snap.Matches)
	if err := s.replayResults(ctx, stageGroups, results, matchUsers); err != nil {
		return err
	}

	s.logger.Info("Snapshot seeded successfully")
	return nil
}

func loadSnapshotFile(path string) (*snapshot.Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read snapshot %q: %w", path, err)
	}

	var snap snapshot.Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("parse snapshot %q: %w", path, err)
	}

	return &snap, nil
}

func (s *Seeder) loadFairPlay(ctx context.Context, rows []snapshot.FairPlayRow) error {
	if len(rows) == 0 {
		return nil
	}

	records := make([]domain.MatchFairPlay, 0, len(rows))
	for _, row := range rows {
		records = append(records, domain.MatchFairPlay{
			MatchID:                     row.MatchID,
			TeamFIFACode:                row.TeamFIFACode,
			YellowCards:                 row.YellowCards,
			IndirectRedCards:            row.IndirectRedCards,
			DirectRedCards:              row.DirectRedCards,
			YellowCardAndDirectRedCards: row.YellowCardAndDirectRedCards,
		})
	}

	return s.fairPlayRepository.Upsert(ctx, records)
}

func snapshotResults(matches []snapshot.MatchResult) (map[int64]ScriptedResult, [][]int64) {
	results := make(map[int64]ScriptedResult, len(matches))
	for _, match := range matches {
		results[match.ID] = ScriptedResult{
			HomeScore:        match.HomeScore,
			AwayScore:        match.AwayScore,
			HomePenaltyScore: match.HomePenaltyScore,
			AwayPenaltyScore: match.AwayPenaltyScore,
		}
	}

	stageGroups := make([][]int64, 0, len(stageRanges))
	for _, stage := range stageRanges {
		stageMatchIDs := make([]int64, 0, stage.last-stage.first+1)
		for matchID := stage.first; matchID <= stage.last; matchID++ {
			if _, finished := results[matchID]; finished {
				stageMatchIDs = append(stageMatchIDs, matchID)
			}
		}

		if len(stageMatchIDs) > 0 {
			stageGroups = append(stageGroups, stageMatchIDs)
		}
	}

	return results, stageGroups
}
