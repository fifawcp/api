package services

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/fifawcp/api/internal/test/mocks"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func newTestAwardService(
	awardRepo *mocks.MockAwardPickRepository,
	playerRepo *mocks.MockPlayerRepository,
	scoringSvc *mocks.MockScoringService,
	lockTime time.Time,
) AwardServiceInterface {
	cfg := &config.Config{
		Scoring: config.ScoringConfig{Award: 50},
	}
	logger := &mocks.MockLogger{}
	return NewAwardService(awardRepo, playerRepo, scoringSvc, lockTime, cfg, logger)
}

func ageRef(value int) *int { return &value }

func goalkeeper(id int64) *domain.Player {
	return &domain.Player{
		ID:       id,
		Name:     gofakeit.Name(),
		Team:     &domain.Team{FifaCode: "MEX"},
		Position: domain.PlayerPositionGoalkeeper,
		Age:      ageRef(30),
	}
}

func attacker(id int64) *domain.Player {
	return &domain.Player{
		ID:       id,
		Name:     gofakeit.Name(),
		Team:     &domain.Team{FifaCode: "BRA"},
		Position: domain.PlayerPositionAttacker,
		Age:      ageRef(33),
	}
}

func youngAttacker(id int64) *domain.Player {
	return &domain.Player{
		ID:       id,
		Name:     gofakeit.Name(),
		Team:     &domain.Team{FifaCode: "ESP"},
		Position: domain.PlayerPositionAttacker,
		Age:      ageRef(18),
	}
}

// futureLock returns a lock time well past now — awards are considered unlocked.
func futureLock() time.Time { return time.Now().UTC().Add(24 * time.Hour) }

// pastLock returns a lock time well before now — awards are locked.
func pastLock() time.Time { return time.Now().UTC().Add(-24 * time.Hour) }

// ---------------------------------------------------------------------------
// SaveAwardPicks
// ---------------------------------------------------------------------------

func TestAwardService_SaveAwardPicks_RejectsWhenLocked(t *testing.T) {
	t.Parallel()

	service := newTestAwardService(&mocks.MockAwardPickRepository{}, &mocks.MockPlayerRepository{}, nil, pastLock())

	picks := []*domain.UserAwardPick{{AwardType: domain.AwardGoldenBoot, PlayerID: 1}}
	_, err := service.SaveAwardPicks(context.Background(), gofakeit.UUID(), picks)

	assert.ErrorIs(t, err, domain.ErrAwardsLocked)
}

func TestAwardService_SaveAwardPicks_RejectsGoldenGloveForOutfielder(t *testing.T) {
	t.Parallel()

	playerRepo := &mocks.MockPlayerRepository{
		GetPlayersByIDsFunc: func(ctx context.Context, ids []int64) ([]*domain.Player, error) {
			return []*domain.Player{attacker(7)}, nil
		},
	}
	service := newTestAwardService(&mocks.MockAwardPickRepository{}, playerRepo, nil, futureLock())

	picks := []*domain.UserAwardPick{{AwardType: domain.AwardGoldenGlove, PlayerID: 7}}
	_, err := service.SaveAwardPicks(context.Background(), gofakeit.UUID(), picks)

	assert.ErrorIs(t, err, domain.ErrAwardPlayerIneligible)
}

func TestAwardService_SaveAwardPicks_RejectsYoungPlayerOverCutoff(t *testing.T) {
	t.Parallel()

	playerRepo := &mocks.MockPlayerRepository{
		GetPlayersByIDsFunc: func(ctx context.Context, ids []int64) ([]*domain.Player, error) {
			return []*domain.Player{attacker(11)}, nil
		},
	}
	service := newTestAwardService(&mocks.MockAwardPickRepository{}, playerRepo, nil, futureLock())

	picks := []*domain.UserAwardPick{{AwardType: domain.AwardYoungPlayer, PlayerID: 11}}
	_, err := service.SaveAwardPicks(context.Background(), gofakeit.UUID(), picks)

	assert.ErrorIs(t, err, domain.ErrAwardPlayerIneligible)
}

func TestAwardService_SaveAwardPicks_AcceptsYoungPlayerWithUnknownAge(t *testing.T) {
	t.Parallel()

	playerWithoutAge := &domain.Player{
		ID:       42,
		Name:     gofakeit.Name(),
		Team:     &domain.Team{FifaCode: "BRA"},
		Position: domain.PlayerPositionAttacker,
		Age:      nil, // article didn't include this player's ESPN profile → DOB unknown
	}

	playerRepo := &mocks.MockPlayerRepository{
		GetPlayersByIDsFunc: func(ctx context.Context, ids []int64) ([]*domain.Player, error) {
			return []*domain.Player{playerWithoutAge}, nil
		},
	}
	awardRepo := &mocks.MockAwardPickRepository{
		UpsertAwardPicksFunc: func(ctx context.Context, userID string, picks []*domain.UserAwardPick) error {
			return nil
		},
	}
	service := newTestAwardService(awardRepo, playerRepo, nil, futureLock())

	picks := []*domain.UserAwardPick{{AwardType: domain.AwardYoungPlayer, PlayerID: 42}}
	_, err := service.SaveAwardPicks(context.Background(), gofakeit.UUID(), picks)

	assert.NoError(t, err)
}

func TestAwardService_SaveAwardPicks_RejectsUnknownPlayer(t *testing.T) {
	t.Parallel()

	playerRepo := &mocks.MockPlayerRepository{
		GetPlayersByIDsFunc: func(ctx context.Context, ids []int64) ([]*domain.Player, error) {
			return []*domain.Player{}, nil
		},
	}
	service := newTestAwardService(&mocks.MockAwardPickRepository{}, playerRepo, nil, futureLock())

	picks := []*domain.UserAwardPick{{AwardType: domain.AwardGoldenBoot, PlayerID: 999}}
	_, err := service.SaveAwardPicks(context.Background(), gofakeit.UUID(), picks)

	assert.ErrorIs(t, err, domain.ErrPlayerNotFound)
}

func TestAwardService_SaveAwardPicks_AcceptsValidMixedPicks(t *testing.T) {
	t.Parallel()

	bootPlayer := attacker(1)
	glovePlayer := goalkeeper(2)
	youngPlayer := youngAttacker(3)

	playerRepo := &mocks.MockPlayerRepository{
		GetPlayersByIDsFunc: func(ctx context.Context, ids []int64) ([]*domain.Player, error) {
			return []*domain.Player{bootPlayer, glovePlayer, youngPlayer}, nil
		},
	}

	var upserted []*domain.UserAwardPick
	awardRepo := &mocks.MockAwardPickRepository{
		UpsertAwardPicksFunc: func(ctx context.Context, userID string, picks []*domain.UserAwardPick) error {
			upserted = picks
			return nil
		},
	}

	service := newTestAwardService(awardRepo, playerRepo, nil, futureLock())

	picks := []*domain.UserAwardPick{
		{AwardType: domain.AwardGoldenBoot, PlayerID: bootPlayer.ID},
		{AwardType: domain.AwardGoldenGlove, PlayerID: glovePlayer.ID},
		{AwardType: domain.AwardYoungPlayer, PlayerID: youngPlayer.ID},
	}
	awards, err := service.SaveAwardPicks(context.Background(), gofakeit.UUID(), picks)

	assert.NoError(t, err)
	assert.Len(t, upserted, 3)
	assert.Equal(t, 3, awards.Progress.Completed)
	assert.Equal(t, 4, awards.Progress.Total)
	assert.False(t, awards.IsLocked)
}

// ---------------------------------------------------------------------------
// GetUserAwards
// ---------------------------------------------------------------------------

func TestAwardService_GetUserAwards_ResolvesInCanonicalOrderWithGaps(t *testing.T) {
	t.Parallel()

	pickedBoot := attacker(10)

	awardRepo := &mocks.MockAwardPickRepository{
		GetAwardPicksFunc: func(ctx context.Context, userID string) ([]*domain.UserAwardPick, error) {
			return []*domain.UserAwardPick{
				{UserID: userID, AwardType: domain.AwardGoldenBoot, PlayerID: pickedBoot.ID},
			}, nil
		},
	}
	playerRepo := &mocks.MockPlayerRepository{
		GetPlayersByIDsFunc: func(ctx context.Context, ids []int64) ([]*domain.Player, error) {
			return []*domain.Player{pickedBoot}, nil
		},
	}

	service := newTestAwardService(awardRepo, playerRepo, nil, futureLock())

	awards, err := service.GetUserAwards(context.Background(), gofakeit.UUID())

	assert.NoError(t, err)
	assert.Len(t, awards.Picks, 4)
	assert.Equal(t, domain.AwardGoldenBoot, awards.Picks[0].AwardType)
	assert.Equal(t, pickedBoot, awards.Picks[0].Player)
	assert.Nil(t, awards.Picks[1].Player) // golden_ball
	assert.Nil(t, awards.Picks[2].Player) // golden_glove
	assert.Nil(t, awards.Picks[3].Player) // young_player
	assert.Equal(t, 1, awards.Progress.Completed)
	assert.Equal(t, 4, awards.Progress.Total)
}

func TestAwardService_GetUserAwards_SignalsLockAfterDeadline(t *testing.T) {
	t.Parallel()

	awardRepo := &mocks.MockAwardPickRepository{
		GetAwardPicksFunc: func(ctx context.Context, userID string) ([]*domain.UserAwardPick, error) {
			return []*domain.UserAwardPick{}, nil
		},
	}
	playerRepo := &mocks.MockPlayerRepository{}

	service := newTestAwardService(awardRepo, playerRepo, nil, pastLock())

	awards, err := service.GetUserAwards(context.Background(), gofakeit.UUID())

	assert.NoError(t, err)
	assert.True(t, awards.IsLocked)
}

// ---------------------------------------------------------------------------
// GetPopularPicks
// ---------------------------------------------------------------------------

func TestAwardService_GetPopularPicks_FansOutPerAwardAndPropagatesLimit(t *testing.T) {
	t.Parallel()

	var calls []callRecord
	var mu sync.Mutex
	playerByID := map[int64]*domain.Player{
		1: attacker(1),
		2: goalkeeper(2),
		3: youngAttacker(3),
	}

	awardRepo := &mocks.MockAwardPickRepository{
		GetPopularPicksFunc: func(ctx context.Context, awardType domain.AwardType, limit int, youngMaxAge int) ([]domain.PopularAwardPick, error) {
			mu.Lock()
			calls = append(calls, callRecord{awardType, limit, youngMaxAge})
			mu.Unlock()
			switch awardType {
			case domain.AwardGoldenBoot:
				return []domain.PopularAwardPick{{Player: playerByID[1], PicksCount: 5}}, nil
			case domain.AwardGoldenBall:
				return []domain.PopularAwardPick{{Player: playerByID[1], PicksCount: 3}}, nil
			case domain.AwardGoldenGlove:
				return []domain.PopularAwardPick{{Player: playerByID[2], PicksCount: 7}}, nil
			case domain.AwardYoungPlayer:
				return []domain.PopularAwardPick{{Player: playerByID[3], PicksCount: 4}}, nil
			}
			return nil, nil
		},
	}
	service := newTestAwardService(awardRepo, &mocks.MockPlayerRepository{}, nil, futureLock())

	result, err := service.GetPopularPicks(context.Background(), 10)

	assert.NoError(t, err)
	assert.Len(t, calls, len(domain.AwardTypes))

	awardsCalled := make(map[domain.AwardType]bool, len(calls))
	for _, c := range calls {
		awardsCalled[c.awardType] = true
		assert.Equal(t, 10, c.limit)
		assert.Equal(t, YoungPlayerMaxAge, c.youngMaxAge)
	}
	for _, awardType := range domain.AwardTypes {
		assert.True(t, awardsCalled[awardType], "expected fan-out call for %q", awardType)
		assert.Len(t, result[awardType], 1, "expected one pick row for %q", awardType)
	}
	assert.Equal(t, 7, result[domain.AwardGoldenGlove][0].PicksCount)
	assert.Equal(t, int64(2), result[domain.AwardGoldenGlove][0].Player.ID)
}

type callRecord struct {
	awardType   domain.AwardType
	limit       int
	youngMaxAge int
}

// ---------------------------------------------------------------------------
// RecordWinners
// ---------------------------------------------------------------------------

func TestAwardService_RecordWinners_RejectsIncompleteSet(t *testing.T) {
	t.Parallel()

	service := newTestAwardService(&mocks.MockAwardPickRepository{}, &mocks.MockPlayerRepository{}, nil, futureLock())

	winners := []*domain.AwardWinner{
		{AwardType: domain.AwardGoldenBoot, PlayerID: 1},
	}
	err := service.RecordWinners(context.Background(), winners)

	assert.ErrorIs(t, err, domain.ErrAwardWinnersIncomplete)
}

func TestAwardService_RecordWinners_RejectsDuplicateAwardType(t *testing.T) {
	t.Parallel()

	service := newTestAwardService(&mocks.MockAwardPickRepository{}, &mocks.MockPlayerRepository{}, nil, futureLock())

	winners := []*domain.AwardWinner{
		{AwardType: domain.AwardGoldenBoot, PlayerID: 1},
		{AwardType: domain.AwardGoldenBoot, PlayerID: 2},
		{AwardType: domain.AwardGoldenBall, PlayerID: 3},
		{AwardType: domain.AwardGoldenGlove, PlayerID: 4},
	}
	err := service.RecordWinners(context.Background(), winners)

	assert.ErrorIs(t, err, domain.ErrAwardWinnersIncomplete)
}

func TestAwardService_RecordWinners_RejectsIneligibleWinner(t *testing.T) {
	t.Parallel()

	playerRepo := &mocks.MockPlayerRepository{
		GetPlayersByIDsFunc: func(ctx context.Context, ids []int64) ([]*domain.Player, error) {
			return []*domain.Player{
				attacker(1), attacker(2), attacker(3), attacker(4),
			}, nil
		},
	}
	service := newTestAwardService(&mocks.MockAwardPickRepository{}, playerRepo, nil, futureLock())

	winners := []*domain.AwardWinner{
		{AwardType: domain.AwardGoldenBoot, PlayerID: 1},
		{AwardType: domain.AwardGoldenBall, PlayerID: 2},
		{AwardType: domain.AwardGoldenGlove, PlayerID: 3},
		{AwardType: domain.AwardYoungPlayer, PlayerID: 4},
	}
	err := service.RecordWinners(context.Background(), winners)

	assert.ErrorIs(t, err, domain.ErrAwardPlayerIneligible)
}

func TestAwardService_RecordWinners_PersistsAndTriggersScoring(t *testing.T) {
	t.Parallel()

	bootPlayer := attacker(1)
	ballPlayer := attacker(2)
	glovePlayer := goalkeeper(3)
	youngPlayer := youngAttacker(4)

	playerRepo := &mocks.MockPlayerRepository{
		GetPlayersByIDsFunc: func(ctx context.Context, ids []int64) ([]*domain.Player, error) {
			return []*domain.Player{bootPlayer, ballPlayer, glovePlayer, youngPlayer}, nil
		},
	}

	var persisted []*domain.AwardWinner
	awardRepo := &mocks.MockAwardPickRepository{
		UpsertAwardWinnersFunc: func(ctx context.Context, winners []*domain.AwardWinner) error {
			persisted = winners
			return nil
		},
	}

	scoreCalled := make(chan struct{}, 1)
	scoringSvc := &mocks.MockScoringService{
		ScoreAwardsFunc: func(ctx context.Context) ([]string, error) {
			scoreCalled <- struct{}{}
			return []string{"user-1", "user-2"}, nil
		},
	}

	service := newTestAwardService(awardRepo, playerRepo, scoringSvc, futureLock())

	winners := []*domain.AwardWinner{
		{AwardType: domain.AwardGoldenBoot, PlayerID: bootPlayer.ID},
		{AwardType: domain.AwardGoldenBall, PlayerID: ballPlayer.ID},
		{AwardType: domain.AwardGoldenGlove, PlayerID: glovePlayer.ID},
		{AwardType: domain.AwardYoungPlayer, PlayerID: youngPlayer.ID},
	}
	err := service.RecordWinners(context.Background(), winners)

	assert.NoError(t, err)
	assert.Len(t, persisted, 4)

	assertSignaledWithin(t, scoreCalled, time.Second, "ScoreAwards not invoked")
}

func TestAwardService_RecordWinners_SurfacesPersistenceFailureSynchronously(t *testing.T) {
	t.Parallel()

	bootPlayer := attacker(1)
	ballPlayer := attacker(2)
	glovePlayer := goalkeeper(3)
	youngPlayer := youngAttacker(4)

	playerRepo := &mocks.MockPlayerRepository{
		GetPlayersByIDsFunc: func(ctx context.Context, ids []int64) ([]*domain.Player, error) {
			return []*domain.Player{bootPlayer, ballPlayer, glovePlayer, youngPlayer}, nil
		},
	}

	persistenceErr := errors.New("db down")
	awardRepo := &mocks.MockAwardPickRepository{
		UpsertAwardWinnersFunc: func(ctx context.Context, winners []*domain.AwardWinner) error {
			return persistenceErr
		},
	}

	service := newTestAwardService(awardRepo, playerRepo, nil, futureLock())

	winners := []*domain.AwardWinner{
		{AwardType: domain.AwardGoldenBoot, PlayerID: bootPlayer.ID},
		{AwardType: domain.AwardGoldenBall, PlayerID: ballPlayer.ID},
		{AwardType: domain.AwardGoldenGlove, PlayerID: glovePlayer.ID},
		{AwardType: domain.AwardYoungPlayer, PlayerID: youngPlayer.ID},
	}
	err := service.RecordWinners(context.Background(), winners)

	assert.ErrorIs(t, err, persistenceErr)
}

func assertSignaledWithin(t *testing.T, ch <-chan struct{}, timeout time.Duration, msg string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(timeout):
		t.Fatal(msg)
	}
}
