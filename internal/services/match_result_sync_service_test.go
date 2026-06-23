package services

import (
	"context"
	"errors"
	"testing"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
	"github.com/fifawcp/api/internal/infrastructure/football"
	"github.com/fifawcp/api/internal/test/mocks"
	"github.com/stretchr/testify/assert"
)

func syncTestGroupMatch(id int64) *domain.Match {
	return &domain.Match{
		ID:        id,
		StageCode: domain.MatchStageCodeGroupStage,
		Teams: domain.MatchTeams{
			Home: &domain.Team{FifaCode: "BRA"},
			Away: &domain.Team{FifaCode: "MAR"},
		},
	}
}

func newSyncService(
	matchSvc *mocks.MockMatchService,
	fetcher *mocks.MockFixtureFetcher,
	apiFixtureRepo *mocks.MockMatchAPIFixtureRepository,
	fairPlayRepo *mocks.MockMatchFairPlayRepository,
) *MatchResultSyncService {
	return NewMatchResultSyncService(matchSvc, fetcher, apiFixtureRepo, fairPlayRepo, &mocks.MockLogger{})
}

func TestMatchResultSyncService_SyncMatch(t *testing.T) {
	t.Parallel()

	t.Run("returns ErrMatchNotFound when the match does not exist", func(t *testing.T) {
		t.Parallel()
		matchSvc := &mocks.MockMatchService{
			GetMatchesFunc: func(_ context.Context, _ domain.MatchFilters) ([]*domain.Match, error) {
				return nil, nil
			},
		}
		svc := newSyncService(matchSvc, &mocks.MockFixtureFetcher{}, &mocks.MockMatchAPIFixtureRepository{}, &mocks.MockMatchFairPlayRepository{})

		_, err := svc.SyncMatch(context.Background(), 1)

		assert.ErrorIs(t, err, domain.ErrMatchNotFound)
	})

	t.Run("does not finalize when the fixture is still in progress", func(t *testing.T) {
		t.Parallel()
		matchSvc := &mocks.MockMatchService{
			GetMatchesFunc: func(_ context.Context, _ domain.MatchFilters) ([]*domain.Match, error) {
				return []*domain.Match{syncTestGroupMatch(1)}, nil
			},
			// UpdateMatchResultFunc left unset — must NOT be called (would panic).
		}
		fetcher := &mocks.MockFixtureFetcher{
			GetFixtureFunc: func(_ context.Context, fixtureID int64) (*football.FixtureResponse, error) {
				return &football.FixtureResponse{
					Fixture: football.FixtureInfo{ID: fixtureID, Status: football.FixtureStatus{Short: "1H"}},
				}, nil
			},
		}
		apiFixtureRepo := &mocks.MockMatchAPIFixtureRepository{
			GetByMatchIDFunc: func(_ context.Context, _ int64) (int64, error) { return 10, nil },
		}
		svc := newSyncService(matchSvc, fetcher, apiFixtureRepo, &mocks.MockMatchFairPlayRepository{})

		result, err := svc.SyncMatch(context.Background(), 1)

		assert.NoError(t, err)
		assert.False(t, result.Finalized)
		assert.Equal(t, "1H", result.Status)
		assert.Nil(t, result.Outcomes)
	})

	t.Run("finalizes a finished fixture and returns the standings outcomes", func(t *testing.T) {
		t.Parallel()
		home, away := 3, 1
		var gotMatchID int64
		var gotPayload dtos.UpdateMatchResultDto
		outcomes := &domain.SyncGroupStageOutcomes{IsGroupStageFinished: true}
		matchSvc := &mocks.MockMatchService{
			GetMatchesFunc: func(_ context.Context, _ domain.MatchFilters) ([]*domain.Match, error) {
				return []*domain.Match{syncTestGroupMatch(1)}, nil
			},
			UpdateMatchResultFunc: func(_ context.Context, matchID int64, payload dtos.UpdateMatchResultDto) (*domain.SyncGroupStageOutcomes, error) {
				gotMatchID = matchID
				gotPayload = payload
				return outcomes, nil
			},
		}
		fetcher := &mocks.MockFixtureFetcher{
			GetFixtureFunc: func(_ context.Context, fixtureID int64) (*football.FixtureResponse, error) {
				return &football.FixtureResponse{
					Fixture: football.FixtureInfo{ID: fixtureID, Status: football.FixtureStatus{Short: "FT"}},
					Goals:   football.FixtureGoals{Home: &home, Away: &away},
				}, nil
			},
		}
		apiFixtureRepo := &mocks.MockMatchAPIFixtureRepository{
			GetByMatchIDFunc: func(_ context.Context, _ int64) (int64, error) { return 10, nil },
		}
		svc := newSyncService(matchSvc, fetcher, apiFixtureRepo, &mocks.MockMatchFairPlayRepository{})

		result, err := svc.SyncMatch(context.Background(), 1)

		assert.NoError(t, err)
		assert.True(t, result.Finalized)
		assert.Equal(t, "FT", result.Status)
		assert.Same(t, outcomes, result.Outcomes)
		assert.Equal(t, int64(1), gotMatchID)
		assert.Equal(t, 3, *gotPayload.HomeScore)
		assert.Equal(t, 1, *gotPayload.AwayScore)
	})

	t.Run("wraps a provider fetch error", func(t *testing.T) {
		t.Parallel()
		matchSvc := &mocks.MockMatchService{
			GetMatchesFunc: func(_ context.Context, _ domain.MatchFilters) ([]*domain.Match, error) {
				return []*domain.Match{syncTestGroupMatch(1)}, nil
			},
		}
		fetcher := &mocks.MockFixtureFetcher{
			GetFixtureFunc: func(_ context.Context, _ int64) (*football.FixtureResponse, error) {
				return nil, errors.New("provider down")
			},
		}
		apiFixtureRepo := &mocks.MockMatchAPIFixtureRepository{
			GetByMatchIDFunc: func(_ context.Context, _ int64) (int64, error) { return 10, nil },
		}
		svc := newSyncService(matchSvc, fetcher, apiFixtureRepo, &mocks.MockMatchFairPlayRepository{})

		_, err := svc.SyncMatch(context.Background(), 1)

		assert.Error(t, err)
	})
}

func TestMatchResultSyncService_ResolveFixtureID(t *testing.T) {
	t.Parallel()

	knockoutMatch := &domain.Match{
		ID:        50,
		StageCode: domain.MatchStageCodeRoundOf16,
		Teams: domain.MatchTeams{
			Home: &domain.Team{FifaCode: "BRA"}, // API team 6
			Away: &domain.Team{FifaCode: "MAR"}, // API team 31
		},
	}

	t.Run("discovers a knockout fixture even when the provider flips home/away", func(t *testing.T) {
		t.Parallel()
		var upserted int64
		apiFixtureRepo := &mocks.MockMatchAPIFixtureRepository{
			GetByMatchIDFunc: func(_ context.Context, _ int64) (int64, error) {
				return 0, domain.ErrMatchAPIFixtureNotFound
			},
			UpsertFixtureIDFunc: func(_ context.Context, _, apiFixtureID int64) error {
				upserted = apiFixtureID
				return nil
			},
		}
		fetcher := &mocks.MockFixtureFetcher{
			GetFixturesByTeamFunc: func(_ context.Context, _ int64) ([]football.FixtureResponse, error) {
				// Provider lists our away team (MAR=31) as the home side.
				return []football.FixtureResponse{{
					Fixture: football.FixtureInfo{ID: 555},
					Teams:   football.FixtureTeams{Home: football.FixtureTeam{ID: 31}, Away: football.FixtureTeam{ID: 6}},
				}}, nil
			},
		}
		svc := newSyncService(&mocks.MockMatchService{}, fetcher, apiFixtureRepo, &mocks.MockMatchFairPlayRepository{})

		fixtureID, err := svc.ResolveFixtureID(context.Background(), knockoutMatch)

		assert.NoError(t, err)
		assert.Equal(t, int64(555), fixtureID)
		assert.Equal(t, int64(555), upserted)
	})

	t.Run("returns ErrMatchFixtureUnresolved when no fixture matches", func(t *testing.T) {
		t.Parallel()
		apiFixtureRepo := &mocks.MockMatchAPIFixtureRepository{
			GetByMatchIDFunc: func(_ context.Context, _ int64) (int64, error) {
				return 0, domain.ErrMatchAPIFixtureNotFound
			},
		}
		fetcher := &mocks.MockFixtureFetcher{
			GetFixturesByTeamFunc: func(_ context.Context, _ int64) ([]football.FixtureResponse, error) {
				return []football.FixtureResponse{{
					Fixture: football.FixtureInfo{ID: 999},
					Teams:   football.FixtureTeams{Home: football.FixtureTeam{ID: 111}, Away: football.FixtureTeam{ID: 222}},
				}}, nil
			},
		}
		svc := newSyncService(&mocks.MockMatchService{}, fetcher, apiFixtureRepo, &mocks.MockMatchFairPlayRepository{})

		_, err := svc.ResolveFixtureID(context.Background(), knockoutMatch)

		assert.ErrorIs(t, err, domain.ErrMatchFixtureUnresolved)
	})

	t.Run("returns ErrMatchTeamsNotAssigned for an undetermined knockout match", func(t *testing.T) {
		t.Parallel()
		apiFixtureRepo := &mocks.MockMatchAPIFixtureRepository{
			GetByMatchIDFunc: func(_ context.Context, _ int64) (int64, error) {
				return 0, domain.ErrMatchAPIFixtureNotFound
			},
		}
		svc := newSyncService(&mocks.MockMatchService{}, &mocks.MockFixtureFetcher{}, apiFixtureRepo, &mocks.MockMatchFairPlayRepository{})

		_, err := svc.ResolveFixtureID(context.Background(), &domain.Match{ID: 51, StageCode: domain.MatchStageCodeRoundOf16})

		assert.ErrorIs(t, err, domain.ErrMatchTeamsNotAssigned)
	})
}
