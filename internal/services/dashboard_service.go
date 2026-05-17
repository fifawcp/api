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
	matchScorePickRepository domain.MatchScorePickRepository
	matchRepository          domain.MatchRepository
	competitionScoreRepo     domain.CompetitionScoreRepository
	globalPickemCompetition  *domain.Competition
	globalMatchCompetition   *domain.Competition
}

func NewDashboardService(
	pickemService PickemServiceInterface,
	matchScorePickRepository domain.MatchScorePickRepository,
	matchRepository domain.MatchRepository,
	competitionScoreRepo domain.CompetitionScoreRepository,
	globalPickemCompetition *domain.Competition,
	globalMatchCompetition *domain.Competition,
) *DashboardService {
	return &DashboardService{
		pickemService:            pickemService,
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
		nextMatch  *domain.Match
		pickemPage *domain.CompetitionLeaderboardPage
		matchPage  *domain.CompetitionLeaderboardPage

		// per-user — fetched only when authenticated
		champion       *domain.Team
		pickemStats    domain.CompetitionUserStats
		matchStats     domain.CompetitionUserStats
		matchPicksMade int
		pickemProgress *domain.PickemProgress
	)

	eg, egCtx := errgroup.WithContext(ctx)

	// public data: always fan out
	eg.Go(func() (err error) {
		nextMatch, err = s.matchRepository.GetNextScheduledMatch(egCtx)
		return
	})
	eg.Go(func() (err error) {
		pickemPage, err = s.competitionScoreRepo.GetLeaderboard(egCtx, s.globalPickemCompetition.ID, 1, 5)
		return
	})
	eg.Go(func() (err error) {
		matchPage, err = s.competitionScoreRepo.GetLeaderboard(egCtx, s.globalMatchCompetition.ID, 1, 5)
		return
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
			matchPicksMade, err = s.matchScorePickRepository.CountMatchScorePicksByUser(egCtx, userID)
			return
		})
		eg.Go(func() (err error) {
			pickemProgress, err = s.pickemService.GetUserPickemProgress(egCtx, userID)
			return
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	dashboard := &domain.Dashboard{
		NextMatch: nextMatch,
		Leaderboard: domain.DashboardLeaderboard{
			Pickem: domain.CompetitionTop{
				CompetitionName: s.globalPickemCompetition.Name,
				Entries:         buildLeaderEntries(pickemPage),
			},
			Match: domain.CompetitionTop{
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
			MatchPicks: stepProgress(matchPicksMade, 104),
			Pickem:     *pickemProgress,
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
	}

	return 0
}
