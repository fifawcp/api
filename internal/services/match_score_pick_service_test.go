package services

import (
	"context"
	"testing"
	"time"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestMatchScorePickService(
	pickRepo *mocks.MockMatchScorePickRepository,
	matchRepo *mocks.MockMatchRepository,
	competitionRepo *mocks.MockCompetitionRepository,
) MatchScorePickServiceInterface {
	return NewMatchScorePickService(pickRepo, matchRepo, competitionRepo)
}

func int64Ptr(value int64) *int64 { return &value }
func intPtr(value int) *int       { return &value }

func lockedMatch(id int64) *domain.Match {
	return &domain.Match{
		ID:        id,
		Status:    domain.MatchStatusScheduled,
		KickoffAt: time.Now().UTC().Add(-1 * time.Hour),
	}
}

func unlockedMatch(id int64) *domain.Match {
	return &domain.Match{
		ID:        id,
		Status:    domain.MatchStatusScheduled,
		KickoffAt: time.Now().UTC().Add(1 * time.Hour),
	}
}

// ---------------------------------------------------------------------------
// GetMemberCompetitionPicks
// ---------------------------------------------------------------------------

func TestMatchScorePickService_GetMemberCompetitionPicks_PickTypeReturnsSingleLockedMatch(t *testing.T) {
	t.Parallel()

	match := lockedMatch(42)
	pick := &domain.UserMatchScorePick{UserID: "user-1", MatchID: 42, HomeScore: 2, AwayScore: 1}

	competitionRepo := &mocks.MockCompetitionRepository{
		GetCompetitionByIDFunc: func(ctx context.Context, boardID, competitionID int64) (*domain.Competition, error) {
			return &domain.Competition{
				ID:          competitionID,
				BoardID:     boardID,
				Type:        domain.CompetitionTypePick,
				PickMatchID: int64Ptr(42),
			}, nil
		},
	}
	matchRepo := &mocks.MockMatchRepository{
		GetMatchesFunc: func(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
			return []*domain.Match{match}, nil
		},
	}
	pickRepo := &mocks.MockMatchScorePickRepository{
		GetMatchScorePicksByUserAndMatchesFunc: func(ctx context.Context, userID string, matchIDs []int64) ([]*domain.UserMatchScorePick, error) {
			return []*domain.UserMatchScorePick{pick}, nil
		},
	}

	service := newTestMatchScorePickService(pickRepo, matchRepo, competitionRepo)

	matches, picks, err := service.GetMemberCompetitionPicks(context.Background(), 1, 10, "user-1")

	require.NoError(t, err)
	assert.Len(t, matches, 1)
	assert.Equal(t, int64(42), matches[0].ID)
	assert.Len(t, picks, 1)
	assert.Equal(t, int64(42), picks[0].MatchID)
}

func TestMatchScorePickService_GetMemberCompetitionPicks_MatchTypeReturnsOnlyLockedMatches(t *testing.T) {
	t.Parallel()

	locked := lockedMatch(10)
	unlocked := unlockedMatch(11)
	pick := &domain.UserMatchScorePick{UserID: "user-1", MatchID: 10, HomeScore: 1, AwayScore: 0}

	competitionRepo := &mocks.MockCompetitionRepository{
		GetCompetitionByIDFunc: func(ctx context.Context, boardID, competitionID int64) (*domain.Competition, error) {
			return &domain.Competition{
				ID:      competitionID,
				BoardID: boardID,
				Type:    domain.CompetitionTypeMatch,
			}, nil
		},
		GetScopeMatchIDsFunc: func(ctx context.Context, competitionID int64) ([]int64, error) {
			return []int64{10, 11}, nil
		},
	}
	matchRepo := &mocks.MockMatchRepository{
		GetMatchesFunc: func(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
			return []*domain.Match{locked, unlocked}, nil
		},
	}
	pickRepo := &mocks.MockMatchScorePickRepository{
		GetMatchScorePicksByUserAndMatchesFunc: func(ctx context.Context, userID string, matchIDs []int64) ([]*domain.UserMatchScorePick, error) {
			return []*domain.UserMatchScorePick{pick}, nil
		},
	}

	service := newTestMatchScorePickService(pickRepo, matchRepo, competitionRepo)

	matches, picks, err := service.GetMemberCompetitionPicks(context.Background(), 1, 10, "user-1")

	require.NoError(t, err)
	assert.Len(t, matches, 1)
	assert.Equal(t, int64(10), matches[0].ID)
	assert.Len(t, picks, 1)
}

func TestMatchScorePickService_GetMemberCompetitionPicks_UnlockedScopeMatchExcluded(t *testing.T) {
	t.Parallel()

	competitionRepo := &mocks.MockCompetitionRepository{
		GetCompetitionByIDFunc: func(ctx context.Context, boardID, competitionID int64) (*domain.Competition, error) {
			return &domain.Competition{
				ID:      competitionID,
				BoardID: boardID,
				Type:    domain.CompetitionTypeMatch,
			}, nil
		},
		GetScopeMatchIDsFunc: func(ctx context.Context, competitionID int64) ([]int64, error) {
			return []int64{20}, nil
		},
	}
	matchRepo := &mocks.MockMatchRepository{
		GetMatchesFunc: func(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
			return []*domain.Match{unlockedMatch(20)}, nil
		},
	}

	service := newTestMatchScorePickService(&mocks.MockMatchScorePickRepository{}, matchRepo, competitionRepo)

	matches, picks, err := service.GetMemberCompetitionPicks(context.Background(), 1, 10, "user-1")

	require.NoError(t, err)
	assert.Empty(t, matches)
	assert.Empty(t, picks)
}

func TestMatchScorePickService_GetMemberCompetitionPicks_LockedMatchWithNoMemberPick(t *testing.T) {
	t.Parallel()

	competitionRepo := &mocks.MockCompetitionRepository{
		GetCompetitionByIDFunc: func(ctx context.Context, boardID, competitionID int64) (*domain.Competition, error) {
			return &domain.Competition{
				ID:          competitionID,
				BoardID:     boardID,
				Type:        domain.CompetitionTypePick,
				PickMatchID: int64Ptr(30),
			}, nil
		},
	}
	matchRepo := &mocks.MockMatchRepository{
		GetMatchesFunc: func(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
			return []*domain.Match{lockedMatch(30)}, nil
		},
	}
	pickRepo := &mocks.MockMatchScorePickRepository{
		GetMatchScorePicksByUserAndMatchesFunc: func(ctx context.Context, userID string, matchIDs []int64) ([]*domain.UserMatchScorePick, error) {
			return []*domain.UserMatchScorePick{}, nil
		},
	}

	service := newTestMatchScorePickService(pickRepo, matchRepo, competitionRepo)

	matches, picks, err := service.GetMemberCompetitionPicks(context.Background(), 1, 10, "user-1")

	require.NoError(t, err)
	assert.Len(t, matches, 1)
	assert.Equal(t, int64(30), matches[0].ID)
	assert.Empty(t, picks)
}

func TestMatchScorePickService_GetMemberCompetitionPicks_PickemType_ReturnsErrCompetitionNotMatchBased(t *testing.T) {
	t.Parallel()

	competitionRepo := &mocks.MockCompetitionRepository{
		GetCompetitionByIDFunc: func(ctx context.Context, boardID, competitionID int64) (*domain.Competition, error) {
			return &domain.Competition{
				ID:      competitionID,
				BoardID: boardID,
				Type:    domain.CompetitionTypePickem,
			}, nil
		},
	}

	service := newTestMatchScorePickService(&mocks.MockMatchScorePickRepository{}, &mocks.MockMatchRepository{}, competitionRepo)

	_, _, err := service.GetMemberCompetitionPicks(context.Background(), 1, 10, "user-1")

	require.ErrorIs(t, err, domain.ErrCompetitionNotMatchBased)
}

func TestMatchScorePickService_GetMemberCompetitionPicks_AwardsType_ReturnsErrCompetitionNotMatchBased(t *testing.T) {
	t.Parallel()

	competitionRepo := &mocks.MockCompetitionRepository{
		GetCompetitionByIDFunc: func(ctx context.Context, boardID, competitionID int64) (*domain.Competition, error) {
			return &domain.Competition{
				ID:      competitionID,
				BoardID: boardID,
				Type:    domain.CompetitionTypeAwards,
			}, nil
		},
	}

	service := newTestMatchScorePickService(&mocks.MockMatchScorePickRepository{}, &mocks.MockMatchRepository{}, competitionRepo)

	_, _, err := service.GetMemberCompetitionPicks(context.Background(), 1, 10, "user-1")

	require.ErrorIs(t, err, domain.ErrCompetitionNotMatchBased)
}

func TestMatchScorePickService_GetMemberCompetitionPicks_UnknownCompetition_PropagatesErrCompetitionNotFound(t *testing.T) {
	t.Parallel()

	competitionRepo := &mocks.MockCompetitionRepository{
		GetCompetitionByIDFunc: func(ctx context.Context, boardID, competitionID int64) (*domain.Competition, error) {
			return nil, domain.ErrCompetitionNotFound
		},
	}

	service := newTestMatchScorePickService(&mocks.MockMatchScorePickRepository{}, &mocks.MockMatchRepository{}, competitionRepo)

	_, _, err := service.GetMemberCompetitionPicks(context.Background(), 1, 99, "user-1")

	require.ErrorIs(t, err, domain.ErrCompetitionNotFound)
}

func TestMatchScorePickService_GetMemberCompetitionPicks_PickTypeWithNilPickMatchID_ReturnsErrCompetitionNotMatchBased(t *testing.T) {
	t.Parallel()

	competitionRepo := &mocks.MockCompetitionRepository{
		GetCompetitionByIDFunc: func(ctx context.Context, boardID, competitionID int64) (*domain.Competition, error) {
			return &domain.Competition{
				ID:          competitionID,
				BoardID:     boardID,
				Type:        domain.CompetitionTypePick,
				PickMatchID: nil,
			}, nil
		},
	}

	service := newTestMatchScorePickService(&mocks.MockMatchScorePickRepository{}, &mocks.MockMatchRepository{}, competitionRepo)

	_, _, err := service.GetMemberCompetitionPicks(context.Background(), 1, 10, "user-1")

	require.ErrorIs(t, err, domain.ErrCompetitionNotMatchBased)
}

// ---------------------------------------------------------------------------
// GetBoardMatchPicks
// ---------------------------------------------------------------------------

func TestMatchScorePickService_GetBoardMatchPicks_UnknownMatch_ReturnsErrMatchNotFound(t *testing.T) {
	t.Parallel()

	matchRepo := &mocks.MockMatchRepository{
		GetMatchesFunc: func(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
			return []*domain.Match{}, nil
		},
	}

	service := newTestMatchScorePickService(&mocks.MockMatchScorePickRepository{}, matchRepo, &mocks.MockCompetitionRepository{})

	_, _, err := service.GetBoardMatchPicks(context.Background(), 1, 999)

	require.ErrorIs(t, err, domain.ErrMatchNotFound)
}

func TestMatchScorePickService_GetBoardMatchPicks_FutureScheduledMatch_ReturnsErrMatchPicksHidden(t *testing.T) {
	t.Parallel()

	matchRepo := &mocks.MockMatchRepository{
		GetMatchesFunc: func(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
			return []*domain.Match{unlockedMatch(5)}, nil
		},
	}

	service := newTestMatchScorePickService(&mocks.MockMatchScorePickRepository{}, matchRepo, &mocks.MockCompetitionRepository{})

	_, _, err := service.GetBoardMatchPicks(context.Background(), 1, 5)

	require.ErrorIs(t, err, domain.ErrMatchPicksHidden)
}

func TestMatchScorePickService_GetBoardMatchPicks_LockedMatch_ReturnsAllMembersWithNilScoreForNonPickers(t *testing.T) {
	t.Parallel()

	match := lockedMatch(7)
	memberWithPick := domain.CompetitionLeaderboardMember{UserID: "user-1", UserName: "alice"}
	memberWithoutPick := domain.CompetitionLeaderboardMember{UserID: "user-2", UserName: "bob"}

	matchRepo := &mocks.MockMatchRepository{
		GetMatchesFunc: func(ctx context.Context, filters domain.MatchFilters) ([]*domain.Match, error) {
			return []*domain.Match{match}, nil
		},
	}
	pickRepo := &mocks.MockMatchScorePickRepository{
		GetBoardMembersMatchPicksFunc: func(ctx context.Context, boardID, matchID int64) ([]*domain.BoardMemberMatchPick, error) {
			return []*domain.BoardMemberMatchPick{
				{Member: memberWithPick, HomeScore: intPtr(2), AwayScore: intPtr(1)},
				{Member: memberWithoutPick, HomeScore: nil, AwayScore: nil},
			}, nil
		},
	}

	service := newTestMatchScorePickService(pickRepo, matchRepo, &mocks.MockCompetitionRepository{})

	returnedMatch, memberPicks, err := service.GetBoardMatchPicks(context.Background(), 1, 7)

	require.NoError(t, err)
	assert.Equal(t, match, returnedMatch)
	assert.Len(t, memberPicks, 2)

	assert.Equal(t, "user-1", memberPicks[0].Member.UserID)
	assert.NotNil(t, memberPicks[0].HomeScore)
	assert.NotNil(t, memberPicks[0].AwayScore)

	assert.Equal(t, "user-2", memberPicks[1].Member.UserID)
	assert.Nil(t, memberPicks[1].HomeScore)
	assert.Nil(t, memberPicks[1].AwayScore)
}
