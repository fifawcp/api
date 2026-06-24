package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
	"github.com/fifawcp/api/internal/infrastructure/football"
	"github.com/fifawcp/api/internal/infrastructure/logging"
)

// MatchResultSyncServiceInterface owns the per-match "fetch the result from the
// football provider and persist it" logic. It is shared by the cron poller
// (which drives it on a timer) and the manual admin sync endpoint (which drives
// it on demand), so both paths resolve fixtures and finalize results identically.
type MatchResultSyncServiceInterface interface {
	// SyncMatch resolves the fixture for a single match, fetches it once, and
	// finalizes the result when the fixture is final. A non-final fixture is a
	// no-op (no error, no DB change) so callers can surface the live status.
	SyncMatch(ctx context.Context, matchID int64) (*domain.MatchSyncResult, error)
	// ResolveFixtureID maps a match to its provider fixture ID, discovering and
	// persisting knockout fixture IDs on demand. Reused by the cron poller's run.
	ResolveFixtureID(ctx context.Context, match *domain.Match) (int64, error)
	// Finalize persists a finished fixture's result (triggering standings sync +
	// scoring) and its fair-play cards. Reused by the cron poller's poll loop.
	Finalize(ctx context.Context, matchID, fixtureID int64, fixture *football.FixtureResponse) (*domain.SyncGroupStageOutcomes, error)
}

type MatchResultSyncService struct {
	matchService   MatchServiceInterface
	fetcher        football.FixtureFetcher
	apiFixtureRepo domain.MatchAPIFixtureRepository
	fairPlayRepo   domain.MatchFairPlayRepository
	logger         logging.Logger
}

func NewMatchResultSyncService(
	matchService MatchServiceInterface,
	fetcher football.FixtureFetcher,
	apiFixtureRepo domain.MatchAPIFixtureRepository,
	fairPlayRepo domain.MatchFairPlayRepository,
	logger logging.Logger,
) *MatchResultSyncService {
	return &MatchResultSyncService{
		matchService:   matchService,
		fetcher:        fetcher,
		apiFixtureRepo: apiFixtureRepo,
		fairPlayRepo:   fairPlayRepo,
		logger:         logger,
	}
}

func (s *MatchResultSyncService) SyncMatch(ctx context.Context, matchID int64) (*domain.MatchSyncResult, error) {
	matches, err := s.matchService.GetMatches(ctx, domain.MatchFilters{MatchIDs: []int64{matchID}})
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, domain.ErrMatchNotFound
	}

	fixtureID, err := s.ResolveFixtureID(ctx, matches[0])
	if err != nil {
		return nil, err
	}

	fixture, err := s.fetcher.GetFixture(ctx, fixtureID)
	if err != nil {
		return nil, fmt.Errorf("fetch fixture %d for match %d: %w", fixtureID, matchID, err)
	}

	if !fixture.IsFinished() {
		return &domain.MatchSyncResult{Finalized: false, Status: fixture.Fixture.Status.Short}, nil
	}

	outcomes, err := s.Finalize(ctx, matchID, fixtureID, fixture)
	if err != nil {
		return nil, err
	}

	return &domain.MatchSyncResult{Finalized: true, Status: fixture.Fixture.Status.Short, Outcomes: outcomes}, nil
}

// Finalize persists the result first (the source of truth) and then the
// fair-play cards. A fair-play failure is logged but does not fail the sync —
// the result has already been recorded and scored.
func (s *MatchResultSyncService) Finalize(ctx context.Context, matchID, fixtureID int64, fixture *football.FixtureResponse) (*domain.SyncGroupStageOutcomes, error) {
	outcomes, err := s.persistResult(ctx, matchID, fixture)
	if err != nil {
		return nil, fmt.Errorf("persist result for match %d: %w", matchID, err)
	}

	if err := s.persistFairPlay(ctx, matchID, fixture); err != nil {
		s.logger.Error("match result sync: persist fair play failed",
			"match_id", matchID,
			"fixture_id", fixtureID,
			"error", err,
		)
	}

	return outcomes, nil
}

func (s *MatchResultSyncService) persistResult(ctx context.Context, matchID int64, fixture *football.FixtureResponse) (*domain.SyncGroupStageOutcomes, error) {
	homeScore := 0
	if fixture.Goals.Home != nil {
		homeScore = *fixture.Goals.Home
	}
	awayScore := 0
	if fixture.Goals.Away != nil {
		awayScore = *fixture.Goals.Away
	}

	update := dtos.UpdateMatchResultDto{
		HomeScore: &homeScore,
		AwayScore: &awayScore,
	}

	// Attach penalty scores when the match was decided by a shootout.
	if fixture.Fixture.Status.Short == "PEN" {
		update.HomePenaltyScore = fixture.Score.Penalty.Home
		update.AwayPenaltyScore = fixture.Score.Penalty.Away
	}

	return s.matchService.UpdateMatchResult(ctx, matchID, update)
}

func (s *MatchResultSyncService) persistFairPlay(ctx context.Context, matchID int64, fixture *football.FixtureResponse) error {
	cardSummaries := football.ParseCardEvents(fixture.Events)
	if len(cardSummaries) == 0 {
		return nil
	}

	var records []domain.MatchFairPlay
	for apiTeamID, summary := range cardSummaries {
		records = append(records, domain.MatchFairPlay{
			MatchID:                     matchID,
			TeamFIFACode:                football.APITeamIDToFIFACode[apiTeamID],
			YellowCards:                 summary.YellowCardCount,
			IndirectRedCards:            summary.IndirectRedCount,
			DirectRedCards:              summary.DirectRedCount,
			YellowCardAndDirectRedCards: summary.YellowCardAndDirectRedCardCount,
		})
	}

	if len(records) == 0 {
		return nil
	}

	return s.fairPlayRepo.Upsert(ctx, records)
}

func (s *MatchResultSyncService) ResolveFixtureID(ctx context.Context, match *domain.Match) (int64, error) {
	fixtureID, err := s.apiFixtureRepo.GetByMatchID(ctx, match.ID)
	if err == nil {
		return fixtureID, nil
	}
	if !errors.Is(err, domain.ErrMatchAPIFixtureNotFound) {
		return 0, fmt.Errorf("lookup fixture ID for match %d: %w", match.ID, err)
	}

	// Not in DB yet — only knockout matches reach this (group-stage IDs are pre-seeded).
	if match.Teams.Home == nil || match.Teams.Away == nil {
		return 0, domain.ErrMatchTeamsNotAssigned
	}

	homeAPITeamID := football.FifaCodeToAPITeamID[match.Teams.Home.FifaCode]
	awayAPITeamID := football.FifaCodeToAPITeamID[match.Teams.Away.FifaCode]

	return s.discoverAndPersistKnockoutFixtureID(ctx, match.ID, homeAPITeamID, awayAPITeamID)
}

func (s *MatchResultSyncService) discoverAndPersistKnockoutFixtureID(ctx context.Context, matchID, homeAPITeamID, awayAPITeamID int64) (int64, error) {
	fixtures, err := s.fetcher.GetFixturesByTeam(ctx, homeAPITeamID)
	if err != nil {
		return 0, fmt.Errorf("get fixtures by team %d: %w", homeAPITeamID, err)
	}

	for _, fixture := range fixtures {
		// We already filtered to the home team's fixtures, so a fixture between
		// the two teams has the away team on either side — the provider's
		// home/away assignment can differ from ours (seeding/venue), so match
		// both orientations.
		if fixture.Teams.Home.ID == awayAPITeamID || fixture.Teams.Away.ID == awayAPITeamID {
			fixtureID := fixture.Fixture.ID
			if err := s.apiFixtureRepo.UpsertFixtureID(ctx, matchID, fixtureID); err != nil {
				s.logger.Warn("match result sync: failed to persist knockout fixture ID",
					"match_id", matchID,
					"fixture_id", fixtureID,
					"error", err,
				)
			}

			return fixtureID, nil
		}
	}

	return 0, domain.ErrMatchFixtureUnresolved
}
