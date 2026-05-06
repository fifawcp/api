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
	matches, err := s.matchRepository.GetMatches(ctx, domain.MatchFilters{
		Status:     domain.MatchStatusFinished,
		StageCodes: []domain.MatchStageCode{domain.MatchStageCodeGroupStage},
	})
	if err != nil {
		return err
	}

	matchesByGroup := make(map[string][]*domain.Match)
	for _, match := range matches {
		matchesByGroup[*match.GroupCode] = append(matchesByGroup[*match.GroupCode], match)
	}

	groupCodes := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L"}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	for _, groupCode := range groupCodes {
		wg.Add(1)
		go func(groupCode string) {
			defer wg.Done()
			if err := s.recalculateStandingsByGroup(ctx, matchesByGroup[groupCode]); err != nil {
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

func (s *GroupStandingService) recalculateStandingsByGroup(ctx context.Context, groupMatches []*domain.Match) error {
	standings := rankGroup(groupMatches)
	return s.groupStandingRepository.UpdateGroupStandings(ctx, standings)
}

// rankGroup is the full FIFA Article 13 ranking pipeline
func rankGroup(matches []*domain.Match) []*domain.GroupStanding {
	standings := computeStandings(matches)
	sortBy(standings, overallSortChain)
	breakPointsTies(standings, matches)
	assignPositions(standings)
	return standings
}

// Builds a GroupStanding per team from a slice of matches
// Reused for both the overall table and h2h mini-tables
// The returned slice has no defined order
func computeStandings(matches []*domain.Match) []*domain.GroupStanding {
	// Map team FIFA code to group standing stats
	statsByTeam := make(map[string]*domain.GroupStanding)

	// For each match, update the group standing stats for the home and away teams
	for _, match := range matches {
		homeTeam := match.Teams.Home.FifaCode
		awayTeam := match.Teams.Away.FifaCode

		if _, ok := statsByTeam[homeTeam]; !ok {
			statsByTeam[homeTeam] = &domain.GroupStanding{Team: *match.Teams.Home}
		}

		if _, ok := statsByTeam[awayTeam]; !ok {
			statsByTeam[awayTeam] = &domain.GroupStanding{Team: *match.Teams.Away}
		}

		applyMatchToStandings(
			statsByTeam[homeTeam],
			statsByTeam[awayTeam],
			match.Result.HomeScore,
			match.Result.AwayScore,
		)
	}

	standings := make([]*domain.GroupStanding, 0, len(statsByTeam))
	for _, s := range statsByTeam {
		standings = append(standings, s)
	}

	return standings
}

func applyMatchToStandings(home, away *domain.GroupStanding, homeScore, awayScore int) {
	// Update the matches played count
	home.MatchesPlayed++
	away.MatchesPlayed++

	// Update the goals for, against counts and goal difference
	home.GoalsFor += homeScore
	home.GoalsAgainst += awayScore
	home.GoalDifference = home.GoalsFor - home.GoalsAgainst

	away.GoalsFor += awayScore
	away.GoalsAgainst += homeScore
	away.GoalDifference = away.GoalsFor - away.GoalsAgainst

	// Update the wins, draws, losses and points counts
	switch {
	case homeScore > awayScore:
		home.Wins++
		home.Points += 3
		away.Losses++
	case awayScore > homeScore:
		away.Wins++
		away.Points += 3
		home.Losses++
	default:
		home.Draws++
		home.Points++
		away.Draws++
		away.Points++
	}
}

// breakPointsTies finds consecutive runs of teams equal on Points and re-orders
// each run via the FIFA Article 13 tiebreaker chain (Step 1 + Step 2 with recursion)
// Modifies `standings` in place
func breakPointsTies(standings []*domain.GroupStanding, allMatches []*domain.Match) {
	i := 0 // Index of the current team

	// Iterate over the standings
	for i < len(standings) {
		j := i + 1 // Index of the next team

		// Iterate over the standings until the points are not equal
		for j < len(standings) && standings[j].Points == standings[i].Points {
			j++ // If the next team has the same points, move to the next team
		}

		// If there are at least two teams with the same points, rank them
		if j-i > 1 {
			ranked := rankTiedSubgroup(standings[i:j], allMatches)
			// Replace the tied teams with the ranked teams
			for k, team := range ranked {
				standings[i+k] = team
			}
		}

		// Move to the next group of teams with the same points
		i = j
	}
}

// rankTiedSubgroup applies FIFA Article 13 a/b/c (head-to-head) to a subgroup of
// teams equal on points, then recurses on any sub-run that remains tied on h2h
// criteria (Step 2, first sentence). If the entire input remains h2h-tied (no
// separation possible), falls back to the overall sort chain (rules d, e, [f])
func rankTiedSubgroup(tiedTeams []*domain.GroupStanding, allMatches []*domain.Match) []*domain.GroupStanding {
	// Get the matches between the tied teams
	h2hMatches := matchesBetween(tiedTeams, allMatches)
	// Compute the standings for the head-to-head matches
	h2hStandings := computeStandings(h2hMatches)
	// Map the head-to-head standings by team FIFA code
	h2hByTeam := make(map[string]*domain.GroupStanding, len(h2hStandings))

	for _, s := range h2hStandings {
		h2hByTeam[s.Team.FifaCode] = s
	}

	// Sort the head-to-head standings by the head-to-head sort chain (points, GD, GF)
	sortBy(h2hStandings, headToHeadSortChain)

	tiedByCode := make(map[string]*domain.GroupStanding, len(tiedTeams))
	for _, t := range tiedTeams {
		tiedByCode[t.Team.FifaCode] = t
	}

	ordered := make([]*domain.GroupStanding, 0, len(tiedTeams))
	for _, s := range h2hStandings {
		if t, ok := tiedByCode[s.Team.FifaCode]; ok {
			ordered = append(ordered, t)
		}
	}

	subgroups := partitionByH2HCriteria(ordered, h2hByTeam)

	result := make([]*domain.GroupStanding, 0, len(tiedTeams))
	for _, subgroup := range subgroups {
		switch {
		case len(subgroup) == 1:
			result = append(result, subgroup...)
		case len(subgroup) < len(tiedTeams):
			// If there are fewer teams in the subgroup than the tied teams,
			// recursively rank the subgroup
			result = append(result, rankTiedSubgroup(subgroup, allMatches)...)
		default:
			sortBy(subgroup, overallSortChain)
			result = append(result, subgroup...)
		}
	}

	return result
}

// Returns the matches in allMatches where both teams are in `teams`
func matchesBetween(teams []*domain.GroupStanding, allMatches []*domain.Match) []*domain.Match {
	codes := make(map[string]bool, len(teams))
	for _, t := range teams {
		codes[t.Team.FifaCode] = true
	}

	var filtered []*domain.Match
	for _, match := range allMatches {
		if codes[match.Teams.Home.FifaCode] && codes[match.Teams.Away.FifaCode] {
			filtered = append(filtered, match)
		}
	}
	return filtered
}

// partitionByH2HCriteria splits a slice (already sorted by the head-to-head
// chain) into consecutive runs of teams identical on h2h Points + GoalDifference + GoalsFor
func partitionByH2HCriteria(sortedTeams []*domain.GroupStanding, h2hByCode map[string]*domain.GroupStanding) [][]*domain.GroupStanding {
	if len(sortedTeams) == 0 {
		return nil
	}

	var groups [][]*domain.GroupStanding
	current := []*domain.GroupStanding{sortedTeams[0]}

	for i := 1; i < len(sortedTeams); i++ {
		prev := h2hByCode[sortedTeams[i-1].Team.FifaCode]
		curr := h2hByCode[sortedTeams[i].Team.FifaCode]
		if prev.Points == curr.Points &&
			prev.GoalDifference == curr.GoalDifference &&
			prev.GoalsFor == curr.GoalsFor {
			current = append(current, sortedTeams[i])
		} else {
			groups = append(groups, current)
			current = []*domain.GroupStanding{sortedTeams[i]}
		}
	}
	groups = append(groups, current)

	return groups
}

func assignPositions(standings []*domain.GroupStanding) {
	for i, s := range standings {
		s.Position = i + 1
	}
}

// tiebreaker returns a negative value if a ranks above b, positive if b above a,
// zero if equal on this criterion
type tiebreaker func(a, b *domain.GroupStanding) int

var (
	byPoints tiebreaker = func(a, b *domain.GroupStanding) int {
		return b.Points - a.Points
	}

	byGoalDifference tiebreaker = func(a, b *domain.GroupStanding) int {
		return b.GoalDifference - a.GoalDifference
	}

	byGoalsFor tiebreaker = func(a, b *domain.GroupStanding) int {
		return b.GoalsFor - a.GoalsFor
	}

	// FIFA rule f (fair play / cards) is not implemented yet. To enable:
	//   1. Add a FairPlayScore int field on domain.GroupStanding
	//   2. Populate it inside applyMatchToStandings from card data on domain.Match
	//   3. Define byFairPlay below and append it to overallSortChain (before byFifaWorldRanking)
	// No structural changes elsewhere are required
	//
	// byFairPlay tiebreaker = func(a, b *domain.GroupStanding) int {
	//     return b.FairPlayScore - a.FairPlayScore
	// }

	// FIFA rule g — most recent published FIFA/Coca-Cola Men's World Ranking
	byFifaWorldRanking tiebreaker = func(a, b *domain.GroupStanding) int {
		return fifaWorldRankingPosition[a.Team.FifaCode] - fifaWorldRankingPosition[b.Team.FifaCode]
	}
)

var overallSortChain = []tiebreaker{
	byPoints,
	byGoalDifference, // FIFA rule d
	byGoalsFor,       // FIFA rule e
	// byFairPlay,      // FIFA rule f - enable when card data exists
	byFifaWorldRanking, // FIFA rule g
}

var headToHeadSortChain = []tiebreaker{
	byPoints,
	byGoalDifference,
	byGoalsFor,
}

func sortBy(standings []*domain.GroupStanding, chain []tiebreaker) {
	sort.SliceStable(standings, func(i, j int) bool {
		// For each criterion in the chain, compare the standings at the current indices
		for _, criterion := range chain {
			if comparison := criterion(standings[i], standings[j]); comparison != 0 {
				return comparison < 0
			}
		}

		return false
	})
}
