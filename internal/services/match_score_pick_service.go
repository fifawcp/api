package services

import (
	"context"
	"time"

	"github.com/fifawcp/api/internal/domain"
)

type MatchScorePickServiceInterface interface {
	GetMatchScorePicksByUser(ctx context.Context, userID string) ([]*domain.UserMatchScorePick, error)
	SaveMatchScorePick(ctx context.Context, userID string, matchID int64, homeScore, awayScore int) error
	GetMemberCompetitionPicks(ctx context.Context, boardID, competitionID int64, userID string) ([]*domain.Match, []*domain.UserMatchScorePick, error)
	GetBoardMatchPicks(ctx context.Context, boardID, matchID int64) (*domain.Match, []*domain.BoardMemberMatchPick, error)
}

type MatchScorePickService struct {
	matchScorePickRepository domain.MatchScorePickRepository
	matchRepository          domain.MatchRepository
	competitionRepository    domain.CompetitionRepository
}

func NewMatchScorePickService(
	matchScorePickRepository domain.MatchScorePickRepository,
	matchRepository domain.MatchRepository,
	competitionRepository domain.CompetitionRepository,
) *MatchScorePickService {
	return &MatchScorePickService{
		matchScorePickRepository: matchScorePickRepository,
		matchRepository:          matchRepository,
		competitionRepository:    competitionRepository,
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

func (s *MatchScorePickService) GetMemberCompetitionPicks(
	ctx context.Context,
	boardID, competitionID int64,
	userID string,
) ([]*domain.Match, []*domain.UserMatchScorePick, error) {
	competition, err := s.competitionRepository.GetCompetitionByID(ctx, boardID, competitionID)
	if err != nil {
		return nil, nil, err
	}

	var matchIDs []int64
	switch competition.Type {
	case domain.CompetitionTypePick:
		if competition.PickMatchID == nil {
			return nil, nil, domain.ErrCompetitionNotMatchBased
		}
		matchIDs = []int64{*competition.PickMatchID}
	case domain.CompetitionTypeMatch:
		matchIDs, err = s.competitionRepository.GetScopeMatchIDs(ctx, competitionID)
		if err != nil {
			return nil, nil, err
		}
	default:
		return nil, nil, domain.ErrCompetitionNotMatchBased
	}

	if len(matchIDs) == 0 {
		return []*domain.Match{}, []*domain.UserMatchScorePick{}, nil
	}

	matches, err := s.matchRepository.GetMatches(ctx, domain.MatchFilters{MatchIDs: matchIDs})
	if err != nil {
		return nil, nil, err
	}

	var lockedMatches []*domain.Match
	for _, match := range matches {
		if isMatchPickLocked(match) {
			lockedMatches = append(lockedMatches, match)
		}
	}

	if len(lockedMatches) == 0 {
		return []*domain.Match{}, []*domain.UserMatchScorePick{}, nil
	}

	lockedIDs := make([]int64, len(lockedMatches))
	for index, match := range lockedMatches {
		lockedIDs[index] = match.ID
	}

	picks, err := s.matchScorePickRepository.GetMatchScorePicksByUserAndMatches(ctx, userID, lockedIDs)
	if err != nil {
		return nil, nil, err
	}

	return lockedMatches, picks, nil
}

func (s *MatchScorePickService) GetBoardMatchPicks(
	ctx context.Context,
	boardID, matchID int64,
) (*domain.Match, []*domain.BoardMemberMatchPick, error) {
	matches, err := s.matchRepository.GetMatches(ctx, domain.MatchFilters{MatchIDs: []int64{matchID}})
	if err != nil {
		return nil, nil, err
	}
	if len(matches) == 0 {
		return nil, nil, domain.ErrMatchNotFound
	}

	match := matches[0]
	if !isMatchPickLocked(match) {
		return nil, nil, domain.ErrMatchPicksHidden
	}

	memberPicks, err := s.matchScorePickRepository.GetBoardMembersMatchPicks(ctx, boardID, matchID)
	if err != nil {
		return nil, nil, err
	}

	return match, memberPicks, nil
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
