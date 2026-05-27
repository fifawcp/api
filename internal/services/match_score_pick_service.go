package services

import (
	"context"
	"time"

	"github.com/fifawcp/api/internal/domain"
)

type MatchScorePickServiceInterface interface {
	GetMatchScorePicksByUser(ctx context.Context, userID string) ([]*domain.UserMatchScorePick, error)
	SaveMatchScorePick(ctx context.Context, userID string, matchID int64, homeScore, awayScore int) error
}

type MatchScorePickService struct {
	matchScorePickRepository domain.MatchScorePickRepository
	matchRepository          domain.MatchRepository
}

func NewMatchScorePickService(
	matchScorePickRepository domain.MatchScorePickRepository,
	matchRepository domain.MatchRepository,
) *MatchScorePickService {
	return &MatchScorePickService{
		matchScorePickRepository: matchScorePickRepository,
		matchRepository:          matchRepository,
	}
}

func (s *MatchScorePickService) GetMatchScorePicksByUser(ctx context.Context, userID string) ([]*domain.UserMatchScorePick, error) {
	return s.matchScorePickRepository.GetMatchScorePicksByUser(ctx, userID)
}

func (s *MatchScorePickService) SaveMatchScorePick(ctx context.Context, userID string, matchID int64, homeScore, awayScore int) error {
	matches, err := s.matchRepository.GetMatches(ctx, domain.MatchFilters{MatchIDs: []int64{matchID}})
	if err != nil {
		return err
	}
	if len(matches) == 0 {
		return domain.ErrMatchNotFound
	}

	if matches[0].Teams.Home == nil || matches[0].Teams.Away == nil {
		return domain.ErrMatchTeamsNotAssigned
	}

	if isMatchPickLocked(matches[0]) {
		return domain.ErrMatchPickLocked
	}

	return s.matchScorePickRepository.UpsertMatchScorePick(ctx, &domain.UserMatchScorePick{
		UserID:    userID,
		MatchID:   matchID,
		HomeScore: homeScore,
		AwayScore: awayScore,
	})
}

func isMatchPickLocked(match *domain.Match) bool {
	if match == nil {
		return true
	}

	if match.Status != domain.MatchStatusScheduled {
		return true
	}

	return time.Now().UTC().After(match.KickoffAt)
}
