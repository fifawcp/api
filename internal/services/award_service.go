package services

import (
	"context"
	"sync"
	"time"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"golang.org/x/sync/errgroup"
)

const YoungPlayerMaxAge = 21

type AwardServiceInterface interface {
	GetUserAwards(ctx context.Context, userID string) (*domain.UserAwards, error)
	GetMemberAwards(ctx context.Context, userID string) (*domain.UserAwards, error)
	SaveAwardPicks(ctx context.Context, userID string, picks []*domain.UserAwardPick) (*domain.UserAwards, error)
	GetPopularPicks(ctx context.Context, limit int) (domain.PopularPicksByAward, error)
	RecordWinners(ctx context.Context, winners []*domain.AwardWinner) error
}

type AwardService struct {
	awardRepository  domain.AwardPickRepository
	playerRepository domain.PlayerRepository
	scoringService   ScoringServiceInterface
	lockTime         time.Time
	cfg              *config.Config
	logger           logging.Logger
}

func NewAwardService(
	awardRepository domain.AwardPickRepository,
	playerRepository domain.PlayerRepository,
	scoringService ScoringServiceInterface,
	lockTime time.Time,
	cfg *config.Config,
	logger logging.Logger,
) AwardServiceInterface {
	return &AwardService{
		awardRepository:  awardRepository,
		playerRepository: playerRepository,
		scoringService:   scoringService,
		lockTime:         lockTime,
		cfg:              cfg,
		logger:           logger,
	}
}

func (s *AwardService) GetUserAwards(ctx context.Context, userID string) (*domain.UserAwards, error) {
	picks, err := s.awardRepository.GetAwardPicks(ctx, userID)
	if err != nil {
		return nil, err
	}

	return s.buildUserAwards(ctx, picks)
}

func (s *AwardService) GetMemberAwards(ctx context.Context, userID string) (*domain.UserAwards, error) {
	if !s.isLocked() {
		return nil, domain.ErrPredictionsHidden
	}

	return s.GetUserAwards(ctx, userID)
}

func (s *AwardService) SaveAwardPicks(
	ctx context.Context,
	userID string,
	picks []*domain.UserAwardPick,
) (*domain.UserAwards, error) {
	if s.isLocked() {
		return nil, domain.ErrAwardsLocked
	}

	if err := s.validatePicksEligibility(ctx, picks); err != nil {
		return nil, err
	}

	if err := s.awardRepository.UpsertAwardPicks(ctx, userID, picks); err != nil {
		return nil, err
	}

	return s.buildUserAwards(ctx, picks)
}

// GetPopularPicks returns the top-`limit` eligible players per award, ranked
// by current pick count. The four awards are fetched in parallel.
func (s *AwardService) GetPopularPicks(ctx context.Context, limit int) (domain.PopularPicksByAward, error) {
	result := make(domain.PopularPicksByAward, len(domain.AwardTypes))
	var mu sync.Mutex

	group, groupCtx := errgroup.WithContext(ctx)
	for _, awardType := range domain.AwardTypes {
		awardType := awardType
		group.Go(func() error {
			picks, err := s.awardRepository.GetPopularPicks(groupCtx, awardType, limit, YoungPlayerMaxAge)
			if err != nil {
				return err
			}
			mu.Lock()
			result[awardType] = picks
			mu.Unlock()
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, err
	}
	return result, nil
}

// RecordWinners persists the actual award winners and kicks off async scoring:
// per-user award_pick score events for everyone who picked correctly, then a
// pickem competition recompute. Mirrors MatchService.asyncScoreMatches so the
// admin request returns immediately.
func (s *AwardService) RecordWinners(ctx context.Context, winners []*domain.AwardWinner) error {
	if err := s.validateWinners(ctx, winners); err != nil {
		return err
	}

	if err := s.awardRepository.UpsertAwardWinners(ctx, winners); err != nil {
		return err
	}

	s.asyncScoreAwards()
	return nil
}

func (s *AwardService) isLocked() bool {
	return time.Now().UTC().After(s.lockTime)
}

func (s *AwardService) buildUserAwards(ctx context.Context, picks []*domain.UserAwardPick) (*domain.UserAwards, error) {
	playerByID, err := s.lookupPlayersForPicks(ctx, picks)
	if err != nil {
		return nil, err
	}

	playerByAward := make(map[domain.AwardType]*domain.Player, len(picks))
	for _, pick := range picks {
		playerByAward[pick.AwardType] = playerByID[pick.PlayerID]
	}

	resolved := make([]domain.ResolvedAwardPick, 0, len(domain.AwardTypes))
	for _, awardType := range domain.AwardTypes {
		resolved = append(resolved, domain.ResolvedAwardPick{
			AwardType: awardType,
			Player:    playerByAward[awardType],
		})
	}

	return &domain.UserAwards{
		Picks: resolved,
		Progress: domain.StepProgress{
			Completed: len(picks),
			Total:     len(domain.AwardTypes),
		},
		IsLocked: s.isLocked(),
	}, nil
}

func (s *AwardService) lookupPlayersForPicks(ctx context.Context, picks []*domain.UserAwardPick) (map[int64]*domain.Player, error) {
	if len(picks) == 0 {
		return map[int64]*domain.Player{}, nil
	}

	ids := make([]int64, 0, len(picks))
	for _, pick := range picks {
		ids = append(ids, pick.PlayerID)
	}

	players, err := s.playerRepository.GetPlayersByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	playerByID := make(map[int64]*domain.Player, len(players))
	for _, player := range players {
		playerByID[player.ID] = player
	}

	return playerByID, nil
}

func (s *AwardService) validatePicksEligibility(ctx context.Context, picks []*domain.UserAwardPick) error {
	if len(picks) == 0 {
		return nil
	}

	playerByID, err := s.lookupPlayersForPicks(ctx, picks)
	if err != nil {
		return err
	}

	for _, pick := range picks {
		if !pick.AwardType.IsValid() {
			return domain.ErrInvalidAwardType
		}

		player, ok := playerByID[pick.PlayerID]
		if !ok {
			return domain.ErrPlayerNotFound
		}

		if !s.isPlayerEligible(pick.AwardType, player) {
			return domain.ErrAwardPlayerIneligible
		}
	}

	return nil
}

func (s *AwardService) validateWinners(ctx context.Context, winners []*domain.AwardWinner) error {
	if len(winners) != len(domain.AwardTypes) {
		return domain.ErrAwardWinnersIncomplete
	}

	seen := make(map[domain.AwardType]bool, len(winners))
	for _, winner := range winners {
		if !winner.AwardType.IsValid() {
			return domain.ErrInvalidAwardType
		}
		if seen[winner.AwardType] {
			return domain.ErrAwardWinnersIncomplete
		}
		seen[winner.AwardType] = true
	}

	for _, awardType := range domain.AwardTypes {
		if !seen[awardType] {
			return domain.ErrAwardWinnersIncomplete
		}
	}

	asPicks := make([]*domain.UserAwardPick, len(winners))
	for i, winner := range winners {
		asPicks[i] = &domain.UserAwardPick{
			AwardType: winner.AwardType,
			PlayerID:  winner.PlayerID,
		}
	}

	return s.validatePicksEligibility(ctx, asPicks)
}

// isPlayerEligible enforces the per-award business rules: Golden Glove needs a
// goalkeeper; Young Player must be at or under the configured age ceiling
// (players with an unknown age get the benefit of the doubt and are accepted);
// Boot and Ball are open to any player.
func (s *AwardService) isPlayerEligible(awardType domain.AwardType, player *domain.Player) bool {
	switch awardType {
	case domain.AwardGoldenGlove:
		return player.Position == domain.PlayerPositionGoalkeeper
	case domain.AwardYoungPlayer:
		if player.Age == nil {
			return true
		}
		return *player.Age <= YoungPlayerMaxAge
	}
	return true
}

func (s *AwardService) asyncScoreAwards() {
	go func() {
		if _, err := s.scoringService.ScoreAwards(context.Background()); err != nil {
			s.logger.Error("award scoring failed",
				logging.Error, err.Error(),
			)
		}
	}()
}
