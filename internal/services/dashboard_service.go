package services

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
	"golang.org/x/sync/errgroup"
)

type DashboardServiceInterface interface {
	GetDashboard(ctx context.Context, userID string) (*domain.Dashboard, error)
}

type DashboardService struct {
	pickemService            PickemServiceInterface
	awardService             AwardServiceInterface
	matchScorePickRepository domain.MatchScorePickRepository
	matchRepository          domain.MatchRepository
	competitionScoreRepo     domain.CompetitionScoreRepository
	globalPickemCompetition  *domain.Competition
	globalMatchCompetition   *domain.Competition
}

func NewDashboardService(
	pickemService PickemServiceInterface,
	awardService AwardServiceInterface,
	matchScorePickRepository domain.MatchScorePickRepository,
	matchRepository domain.MatchRepository,
	competitionScoreRepo domain.CompetitionScoreRepository,
	globalPickemCompetition *domain.Competition,
	globalMatchCompetition *domain.Competition,
) *DashboardService {
	return &DashboardService{
		pickemService:            pickemService,
		awardService:             awardService,
		matchScorePickRepository: matchScorePickRepository,
		matchRepository:          matchRepository,
		competitionScoreRepo:     competitionScoreRepo,
		globalPickemCompetition:  globalPickemCompetition,
		globalMatchCompetition:   globalMatchCompetition,
	}
}

func (s *DashboardService) GetDashboard(ctx context.Context, userID string) (*domain.Dashboard, error) {
	var (
		// public — always fetched
		nextMatch      *domain.Match
		pickemPage     *domain.CompetitionLeaderboardPage
		matchPage      *domain.CompetitionLeaderboardPage
		titleFavorites []*domain.TitleFavorite

		// per-user — fetched only when authenticated
		champion       *domain.Team
		pickemStats    domain.CompetitionUserStats
		matchStats     domain.CompetitionUserStats
		userMatchPicks []*domain.UserMatchScorePick
		pickemProgress *domain.PickemProgress
		userAwards     *domain.UserAwards
	)

	eg, egCtx := errgroup.WithContext(ctx)

	// public data: always fan out
	eg.Go(func() (err error) {
		nextMatch, err = s.matchRepository.GetNextScheduledMatch(egCtx)
		return
	})
	eg.Go(func() (err error) {
		pickemPage, err = s.competitionScoreRepo.GetLeaderboard(egCtx, s.globalPickemCompetition.ID, 1, 10, "", "", "")
		return
	})
	eg.Go(func() (err error) {
		matchPage, err = s.competitionScoreRepo.GetLeaderboard(egCtx, s.globalMatchCompetition.ID, 1, 10, "", "", "")
		return
	})
	eg.Go(func() error {
		// Decorative data — never fail the whole dashboard over it.
		if favs, err := s.pickemService.GetChampionPickCounts(egCtx, 5); err == nil {
			titleFavorites = favs
		}
		return nil
	})

	// per-user data: only fan out for authenticated callers
	if userID != "" {
		eg.Go(func() (err error) {
			champion, err = s.pickemService.GetChampionPick(egCtx, userID)
			return
		})
		eg.Go(func() (err error) {
			pickemStats, err = s.competitionScoreRepo.GetUserPickemStats(egCtx, s.globalPickemCompetition.ID, userID)
			return
		})
		eg.Go(func() (err error) {
			matchStats, err = s.competitionScoreRepo.GetUserMatchStats(egCtx, s.globalMatchCompetition.ID, userID)
			return
		})
		eg.Go(func() (err error) {
			// Fetches the user's picks once: powers both the match-picks count and
			// the score pick attached to next_match below.
			userMatchPicks, err = s.matchScorePickRepository.GetMatchScorePicksByUser(egCtx, userID)
			return
		})
		eg.Go(func() (err error) {
			pickemProgress, err = s.pickemService.GetUserPickemProgress(egCtx, userID)
			return
		})
		eg.Go(func() (err error) {
			userAwards, err = s.awardService.GetUserAwards(egCtx, userID)
			return
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	dashboard := &domain.Dashboard{
		NextMatch:      nextMatch,
		TitleFavorites: titleFavorites,
		Leaderboard: domain.DashboardLeaderboard{
			Pickem: domain.CompetitionTop{
				CompetitionID:   s.globalPickemCompetition.ID,
				BoardID:         s.globalPickemCompetition.BoardID,
				CompetitionName: s.globalPickemCompetition.Name,
				Entries:         buildLeaderEntries(pickemPage),
			},
			Match: domain.CompetitionTop{
				CompetitionID:   s.globalMatchCompetition.ID,
				BoardID:         s.globalMatchCompetition.BoardID,
				CompetitionName: s.globalMatchCompetition.Name,
				Entries:         buildLeaderEntries(matchPage),
			},
		},
	}

	if userID != "" {
		dashboard.PickedChampion = champion
		dashboard.Stats = &domain.DashboardStats{
			Pickem: pickemStats,
			Match:  matchStats,
		}
		dashboard.Progress = &domain.DashboardProgress{
			MatchPicks: stepProgress(len(userMatchPicks), 104),
			Pickem:     *pickemProgress,
			Awards:     userAwards.Progress,
		}
		// Attach the user's pick for the featured next match (if any) so the
		// dashboard's "up next" card reflects it.
		if nextMatch != nil {
			for _, pick := range userMatchPicks {
				if pick.MatchID == nextMatch.ID {
					dashboard.NextMatchScorePick = pick
					break
				}
			}
		}
	}

	return dashboard, nil
}

func buildLeaderEntries(page *domain.CompetitionLeaderboardPage) []domain.DashboardLeaderEntry {
	if page == nil {
		return []domain.DashboardLeaderEntry{}
	}

	entries := make([]domain.DashboardLeaderEntry, 0, len(page.Members))
	for _, member := range page.Members {
		entries = append(entries, domain.DashboardLeaderEntry{
			CompetitionUserStats: domain.CompetitionUserStats{
				Rank:   member.Rank,
				Points: extractPoints(member.Score),
			},
			Member: member.Member,
		})
	}

	return entries
}

func extractPoints(score any) int {
	switch s := score.(type) {
	case *domain.PickemScore:
		return s.Total
	case *domain.MatchScore:
		return s.Total
	case *domain.AwardsScore:
		return s.Total
	}

	return 0
}
