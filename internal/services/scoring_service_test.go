package services

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/fifawcp/api/internal/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func bracketScoringConfig() *config.Config {
	return &config.Config{Scoring: config.ScoringConfig{
		RoundOf32:     4,
		RoundOf16:     6,
		Quarterfinals: 8,
		Semifinals:    12,
		ThirdPlace:    16,
		Final:         20,
	}}
}

func knockoutMatch(id int64, stage domain.MatchStageCode, winner string) *domain.Match {
	return &domain.Match{
		ID:        id,
		StageCode: stage,
		Status:    domain.MatchStatusFinished,
		Result:    &domain.MatchResult{WinnerTeamFifaCode: &winner},
	}
}

// scoreBracketPicks awards points to every user who backed the winning team to
// advance at the match's stage — looked up by team + stage, not by match slot.
func TestScoringService_scoreBracketPicks(t *testing.T) {
	t.Parallel()

	t.Run("awards stage points to every user who picked the winner at that stage", func(t *testing.T) {
		t.Parallel()

		var gotTeam string
		var gotStage domain.MatchStageCode
		pickemRepo := &mocks.MockPickemRepository{
			GetBracketPickUserIDsByTeamAndStageFunc: func(ctx context.Context, team string, stage domain.MatchStageCode) ([]string, error) {
				gotTeam, gotStage = team, stage
				return []string{"user-1", "user-2"}, nil
			},
		}
		service := &ScoringService{pickemRepository: pickemRepo, cfg: bracketScoringConfig()}

		events, affected, err := service.scoreBracketPicks(context.Background(), knockoutMatch(73, domain.MatchStageCodeRoundOf32, "KOR"))

		require.NoError(t, err)
		// Looked up by the winning team + stage, never by match slot.
		assert.Equal(t, "KOR", gotTeam)
		assert.Equal(t, domain.MatchStageCodeRoundOf32, gotStage)

		require.Len(t, events, 2)
		for _, event := range events {
			assert.Equal(t, domain.ScoreSourceBracketPick, event.SourceType)
			assert.Equal(t, "73", event.SourceRef) // stays the match ID
			assert.Equal(t, 4, event.Points)       // round_of_32
		}
		assert.Equal(t, map[string]struct{}{"user-1": {}, "user-2": {}}, affected)
	})

	t.Run("uses the stage's point value", func(t *testing.T) {
		t.Parallel()

		pickemRepo := &mocks.MockPickemRepository{
			GetBracketPickUserIDsByTeamAndStageFunc: func(ctx context.Context, team string, stage domain.MatchStageCode) ([]string, error) {
				return []string{"user-1"}, nil
			},
		}
		service := &ScoringService{pickemRepository: pickemRepo, cfg: bracketScoringConfig()}

		events, _, err := service.scoreBracketPicks(context.Background(), knockoutMatch(104, domain.MatchStageCodeFinal, "ARG"))

		require.NoError(t, err)
		require.Len(t, events, 1)
		assert.Equal(t, 20, events[0].Points) // final
		assert.Equal(t, "104", events[0].SourceRef)
	})

	t.Run("no winning team picked yields no events", func(t *testing.T) {
		t.Parallel()

		pickemRepo := &mocks.MockPickemRepository{
			GetBracketPickUserIDsByTeamAndStageFunc: func(ctx context.Context, team string, stage domain.MatchStageCode) ([]string, error) {
				return nil, nil
			},
		}
		service := &ScoringService{pickemRepository: pickemRepo, cfg: bracketScoringConfig()}

		events, affected, err := service.scoreBracketPicks(context.Background(), knockoutMatch(73, domain.MatchStageCodeRoundOf32, "KOR"))

		require.NoError(t, err)
		assert.Empty(t, events)
		assert.Empty(t, affected)
	})

	t.Run("nil winner is skipped without querying picks", func(t *testing.T) {
		t.Parallel()

		// GetBracketPickUserIDsByTeamAndStageFunc unset → panics if called.
		service := &ScoringService{pickemRepository: &mocks.MockPickemRepository{}, cfg: bracketScoringConfig()}

		match := &domain.Match{ID: 73, StageCode: domain.MatchStageCodeRoundOf32, Result: &domain.MatchResult{}}
		events, affected, err := service.scoreBracketPicks(context.Background(), match)

		require.NoError(t, err)
		assert.Empty(t, events)
		assert.Empty(t, affected)
	})

	t.Run("nil result is skipped without querying picks", func(t *testing.T) {
		t.Parallel()

		service := &ScoringService{pickemRepository: &mocks.MockPickemRepository{}, cfg: bracketScoringConfig()}

		match := &domain.Match{ID: 73, StageCode: domain.MatchStageCodeRoundOf32}
		events, affected, err := service.scoreBracketPicks(context.Background(), match)

		require.NoError(t, err)
		assert.Empty(t, events)
		assert.Empty(t, affected)
	})

	t.Run("non-bracket stage scores nothing without querying picks", func(t *testing.T) {
		t.Parallel()

		service := &ScoringService{pickemRepository: &mocks.MockPickemRepository{}, cfg: bracketScoringConfig()}

		match := knockoutMatch(1, domain.MatchStageCodeGroupStage, "MEX")
		events, affected, err := service.scoreBracketPicks(context.Background(), match)

		require.NoError(t, err)
		assert.Empty(t, events)
		assert.Empty(t, affected)
	})

	t.Run("propagates repository errors", func(t *testing.T) {
		t.Parallel()

		repoErr := errors.New("db down")
		pickemRepo := &mocks.MockPickemRepository{
			GetBracketPickUserIDsByTeamAndStageFunc: func(ctx context.Context, team string, stage domain.MatchStageCode) ([]string, error) {
				return nil, repoErr
			},
		}
		service := &ScoringService{pickemRepository: pickemRepo, cfg: bracketScoringConfig()}

		events, affected, err := service.scoreBracketPicks(context.Background(), knockoutMatch(73, domain.MatchStageCodeRoundOf32, "KOR"))

		assert.ErrorIs(t, err, repoErr)
		assert.Nil(t, events)
		assert.Nil(t, affected)
	})

	// Each match is scored independently with source_ref = its own ID, so scoring a
	// whole stage produces one distinct, idempotent key per match (no batch collisions).
	t.Run("each match in a stage yields a distinct source_ref", func(t *testing.T) {
		t.Parallel()

		pickemRepo := &mocks.MockPickemRepository{
			GetBracketPickUserIDsByTeamAndStageFunc: func(ctx context.Context, team string, stage domain.MatchStageCode) ([]string, error) {
				return []string{"user-1"}, nil
			},
		}
		service := &ScoringService{pickemRepository: pickemRepo, cfg: bracketScoringConfig()}

		var refs []string
		for _, match := range []*domain.Match{
			knockoutMatch(73, domain.MatchStageCodeRoundOf32, "KOR"),
			knockoutMatch(86, domain.MatchStageCodeRoundOf32, "URU"),
		} {
			events, _, err := service.scoreBracketPicks(context.Background(), match)
			require.NoError(t, err)
			require.Len(t, events, 1)
			refs = append(refs, events[0].SourceRef)
		}

		sort.Strings(refs)
		assert.Equal(t, []string{"73", "86"}, refs)
	})
}
