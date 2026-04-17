package services

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/logging"
)

type GroupStandingServiceInterface interface {
	GetGroupStandings(ctx context.Context, groupCodes []string, position *int64) ([]*domain.GroupStanding, error)
	RecalculateStandings(ctx context.Context) error
}

type GroupStandingService struct {
	groupStandingRepository domain.GroupStandingRepository
	matchRepository         domain.MatchRepository
	logger                  logging.Logger
}

func NewGroupStandingService(
	groupStandingRepository domain.GroupStandingRepository,
	matchRepository domain.MatchRepository,
	logger logging.Logger,
) GroupStandingServiceInterface {
	return &GroupStandingService{
		groupStandingRepository: groupStandingRepository,
		matchRepository:         matchRepository,
		logger:                  logger,
	}
}

func (s *GroupStandingService) GetGroupStandings(ctx context.Context, groupCodes []string, position *int64) ([]*domain.GroupStanding, error) {
	return s.groupStandingRepository.GetGroupStandings(ctx, groupCodes, position)
}

func (s *GroupStandingService) RecalculateStandings(ctx context.Context) error {
	// Get all finished matches for the group stage
	matches, err := s.matchRepository.GetMatches(ctx, domain.MatchFilters{
		Status:     domain.MatchStatusFinished,
		StageCodes: []domain.MatchStageCode{domain.MatchStageCodeGroupStage},
	})
	if err != nil {
		return err
	}

	matchesGroupedByGroup := make(map[string][]*domain.Match)
	for _, match := range matches {
		matchesGroupedByGroup[*match.GroupCode] = append(matchesGroupedByGroup[*match.GroupCode], match)
	}

	groupCodes := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L"}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	for _, groupCode := range groupCodes {
		wg.Add(1)
		go func(groupCode string) {
			defer wg.Done()
			if err := s.recalculateStandingsByGroup(ctx, matchesGroupedByGroup[groupCode]); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}(groupCode)
	}
	wg.Wait()

	if len(errs) > 0 {
		return fmt.Errorf("failed to recalculate group standings: %v", errs)
	}

	return nil
}

func (s *GroupStandingService) recalculateStandingsByGroup(
	ctx context.Context,
	groupMatches []*domain.Match,
) error {
	teamStats := calculateOverallStats(groupMatches)

	// Convert map to slice for sorting
	var standings []*domain.GroupStanding
	for _, stat := range teamStats {
		standings = append(standings, stat)
	}

	standings = sortByOverallStats(standings)
	tiedGroups := identifyTiedGroups(standings)

	// Build FIFA code map once (only if there are tied groups)
	var fifaCodeToIndex map[string]int

	if len(tiedGroups) > 0 {
		fifaCodeToIndex = make(map[string]int)
		for i, standing := range standings {
			fifaCodeToIndex[*standing.Team.FifaCode] = i
		}
	}

	// Process tied groups only if any exist
	for _, tiedGroup := range tiedGroups {
		h2hMatches := filterHeadToHeadMatches(tiedGroup, groupMatches)
		h2hStats := calculateHeadToHeadStats(tiedGroup, h2hMatches)
		sortedGroup := sortTiedGroupByHeadToHead(tiedGroup, h2hStats)

		if isStillTied(sortedGroup, h2hStats) {
			// Fallback to overall stats (FIFA rules d, e)
			sortedGroup = sortByOverallStats(sortedGroup)
		}

		// Update the original standings with the sorted group
		// Replace in original standings array
		for _, team := range sortedGroup {
			originalIndex := fifaCodeToIndex[*team.Team.FifaCode]
			standings[originalIndex] = team
		}
	}

	for i, standing := range standings {
		standing.Position = i + 1
	}

	return s.groupStandingRepository.UpdateGroupStandings(ctx, standings)
}

func identifyTiedGroups(standings []*domain.GroupStanding) [][]*domain.GroupStanding {
	if len(standings) == 0 {
		return nil
	}

	var tiedGroups [][]*domain.GroupStanding
	currentGroup := []*domain.GroupStanding{standings[0]}

	for i := 1; i < len(standings); i++ {
		prev := standings[i-1]
		curr := standings[i]

		// Check if current team has identical stats to previous
		if prev.Points == curr.Points &&
			prev.GoalDifference == curr.GoalDifference &&
			prev.GoalsFor == curr.GoalsFor {
			// Still tied, add to current group
			currentGroup = append(currentGroup, curr)
		} else {
			// No longer tied, save current group and start new one
			if len(currentGroup) > 1 {
				tiedGroups = append(tiedGroups, currentGroup)
			}
			currentGroup = []*domain.GroupStanding{curr}
		}
	}

	// Last group
	if len(currentGroup) > 1 {
		tiedGroups = append(tiedGroups, currentGroup)
	}

	return tiedGroups
}

func filterHeadToHeadMatches(tiedTeams []*domain.GroupStanding, allMatches []*domain.Match) []*domain.Match {
	// Get FIFA codes of tied teams
	tiedCodes := make(map[string]bool)
	for _, team := range tiedTeams {
		tiedCodes[*team.Team.FifaCode] = true
	}

	// Filter matches where both teams are in the tied group
	var h2hMatches []*domain.Match
	for _, match := range allMatches {
		homeCode := *match.HomeTeam.FifaCode
		awayCode := *match.AwayTeam.FifaCode
		if tiedCodes[homeCode] && tiedCodes[awayCode] {
			h2hMatches = append(h2hMatches, match)
		}
	}

	return h2hMatches
}

func calculateOverallStats(matches []*domain.Match) map[string]*domain.GroupStanding {
	stats := make(map[string]*domain.GroupStanding)

	for _, match := range matches {
		homeCode := *match.HomeTeam.FifaCode
		awayCode := *match.AwayTeam.FifaCode

		// Initialize stats with team data
		if _, ok := stats[homeCode]; !ok {
			stats[homeCode] = &domain.GroupStanding{
				Team: match.HomeTeam,
			}
		}

		if _, ok := stats[awayCode]; !ok {
			stats[awayCode] = &domain.GroupStanding{
				Team: match.AwayTeam,
			}
		}

		homeScore := *match.HomeScore
		awayScore := *match.AwayScore

		// Update matches played
		stats[homeCode].MatchesPlayed++
		stats[awayCode].MatchesPlayed++

		// Update goals
		stats[homeCode].GoalsFor += homeScore
		stats[homeCode].GoalsAgainst += awayScore
		stats[homeCode].GoalDifference = stats[homeCode].GoalsFor - stats[homeCode].GoalsAgainst

		stats[awayCode].GoalsFor += awayScore
		stats[awayCode].GoalsAgainst += homeScore
		stats[awayCode].GoalDifference = stats[awayCode].GoalsFor - stats[awayCode].GoalsAgainst

		// Update points and wins / draws / losses
		if homeScore > awayScore {
			stats[homeCode].Wins++
			stats[homeCode].Points += 3
			stats[awayCode].Losses++
		} else if awayScore > homeScore {
			stats[awayCode].Wins++
			stats[awayCode].Points += 3
			stats[homeCode].Losses++
		} else {
			stats[homeCode].Draws++
			stats[homeCode].Points += 1
			stats[awayCode].Draws++
			stats[awayCode].Points += 1
		}
	}

	return stats
}

func calculateHeadToHeadStats(tiedTeams []*domain.GroupStanding, h2hMatches []*domain.Match) map[string]*domain.GroupStanding {
	teamMap := make(map[string]*domain.GroupStanding)
	for _, team := range tiedTeams {
		teamMap[*team.Team.FifaCode] = team
	}

	stats := make(map[string]*domain.GroupStanding)

	for _, match := range h2hMatches {
		homeCode := *match.HomeTeam.FifaCode
		awayCode := *match.AwayTeam.FifaCode

		// Initialize stats with team data from teamMap
		if _, ok := stats[homeCode]; !ok {
			stats[homeCode] = &domain.GroupStanding{
				Team: teamMap[homeCode].Team,
			}
		}

		if _, ok := stats[awayCode]; !ok {
			stats[awayCode] = &domain.GroupStanding{
				Team: teamMap[awayCode].Team,
			}
		}

		homeScore := *match.HomeScore
		awayScore := *match.AwayScore

		// Update matches played
		stats[homeCode].MatchesPlayed++
		stats[awayCode].MatchesPlayed++

		// Update goals
		stats[homeCode].GoalsFor += homeScore
		stats[homeCode].GoalsAgainst += awayScore
		stats[homeCode].GoalDifference = stats[homeCode].GoalsFor - stats[homeCode].GoalsAgainst

		stats[awayCode].GoalsFor += awayScore
		stats[awayCode].GoalsAgainst += homeScore
		stats[awayCode].GoalDifference = stats[awayCode].GoalsFor - stats[awayCode].GoalsAgainst

		// Update points and wins / draws / losses
		if homeScore > awayScore {
			stats[homeCode].Wins++
			stats[homeCode].Points += 3
			stats[awayCode].Losses++
		} else if awayScore > homeScore {
			stats[awayCode].Wins++
			stats[awayCode].Points += 3
			stats[homeCode].Losses++
		} else {
			stats[homeCode].Draws++
			stats[homeCode].Points += 1
			stats[awayCode].Draws++
			stats[awayCode].Points += 1
		}
	}

	return stats
}

func sortByOverallStats(standings []*domain.GroupStanding) []*domain.GroupStanding {
	sort.Slice(standings, func(i, j int) bool {
		teamA := standings[i]
		teamB := standings[j]

		if teamA.Points != teamB.Points {
			return teamA.Points > teamB.Points
		}

		if teamA.GoalDifference != teamB.GoalDifference {
			return teamA.GoalDifference > teamB.GoalDifference
		}

		return teamA.GoalsFor > teamB.GoalsFor
	})

	return standings
}

func sortTiedGroupByHeadToHead(tiedTeams []*domain.GroupStanding, h2hStats map[string]*domain.GroupStanding) []*domain.GroupStanding {
	sort.Slice(tiedTeams, func(i, j int) bool {
		teamA := tiedTeams[i]
		teamB := tiedTeams[j]
		statsA := h2hStats[*teamA.Team.FifaCode]
		statsB := h2hStats[*teamB.Team.FifaCode]

		// Rule a: Head-to-head points
		if statsA.Points != statsB.Points {
			return statsA.Points > statsB.Points
		}

		// Rule b: Head-to-head goal difference
		if statsA.GoalDifference != statsB.GoalDifference {
			return statsA.GoalDifference > statsB.GoalDifference
		}

		// Rule c: Head-to-head goals for
		return statsA.GoalsFor > statsB.GoalsFor
	})

	return tiedTeams
}

func isStillTied(sortedGroup []*domain.GroupStanding, h2hStats map[string]*domain.GroupStanding) bool {
	if len(sortedGroup) <= 1 {
		return false
	}

	firstCode := *sortedGroup[0].Team.FifaCode
	firstStats := h2hStats[firstCode]

	for _, team := range sortedGroup[1:] {
		teamStats := h2hStats[*team.Team.FifaCode]
		if teamStats.Points != firstStats.Points ||
			teamStats.GoalDifference != firstStats.GoalDifference ||
			teamStats.GoalsFor != firstStats.GoalsFor {
			return false
		}
	}

	return true
}
