package services

import (
	"context"
	"slices"
	"sort"
	"time"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"golang.org/x/sync/errgroup"
)

var allKnockoutMatchIDs = []int64{73, 74, 75, 76, 77, 78, 79, 80, 81, 82, 83, 84, 85, 86, 87, 88, 89, 90, 91, 92, 93, 94, 95, 96, 97, 98, 99, 100, 101, 102, 103, 104}

type PickemServiceInterface interface {
	GetUserPickem(ctx context.Context, userID string) (*domain.UserPickem, error)
	GetMemberPickem(ctx context.Context, userID string) (*domain.UserPickem, error)
	GetChampionPick(ctx context.Context, userID string) (*domain.Team, error)
	GetChampionPickCounts(ctx context.Context, limit int) ([]*domain.TitleFavorite, error)
	GetUserPickemProgress(ctx context.Context, userID string) (*domain.PickemProgress, error)
	SaveGroupPicks(ctx context.Context, userID string, picks []*domain.UserGroupPick, lockedCodes []string) error
	SaveBestThirds(ctx context.Context, userID string, teamFifaCodes []string) error
	SaveBracketPicks(ctx context.Context, userID string, picks []*domain.UserBracketPick) error
}

type PickemService struct {
	pickemRepo   domain.PickemRepository
	teams        []*domain.Team
	teamLookup   map[string]*domain.Team
	lockTime     time.Time
	cfg          *config.Config
	logger       logging.Logger
	combinations []domain.ThirdPlaceCombination
}

func NewPickemService(
	pickemRepo domain.PickemRepository,
	teams []*domain.Team,
	lockTime time.Time,
	cfg *config.Config,
	logger logging.Logger,
) PickemServiceInterface {
	teamLookup := make(map[string]*domain.Team, len(teams))
	for _, team := range teams {
		if team.FifaCode != "" {
			teamLookup[team.FifaCode] = team
		}
	}

	return &PickemService{
		pickemRepo:   pickemRepo,
		teams:        teams,
		teamLookup:   teamLookup,
		lockTime:     lockTime,
		cfg:          cfg,
		logger:       logger,
		combinations: loadThirdPlaceCombinations(),
	}
}

func (s *PickemService) GetUserPickem(ctx context.Context, userID string) (*domain.UserPickem, error) {
	var (
		groupPicks   []*domain.UserGroupPick
		bestThirds   []*domain.UserBestThirdPick
		bracketPicks []*domain.UserBracketPick
		lockedCodes  []string
	)

	eg, egCtx := errgroup.WithContext(ctx)

	eg.Go(func() (err error) { groupPicks, err = s.pickemRepo.GetGroupPicks(egCtx, userID); return })
	eg.Go(func() (err error) { bestThirds, err = s.pickemRepo.GetBestThirdPicks(egCtx, userID); return })
	eg.Go(func() (err error) { bracketPicks, err = s.pickemRepo.GetBracketPicks(egCtx, userID); return })
	eg.Go(func() (err error) { lockedCodes, err = s.pickemRepo.GetLockedGroupCodes(egCtx, userID); return })

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	groupPicksByGroup := groupPicksByGroupCode(groupPicks)
	lockedByGroup := make(map[string]bool, len(lockedCodes))
	for _, code := range lockedCodes {
		lockedByGroup[code] = true
	}
	bracket := s.projectBracket(groupPicksByGroup, bestThirds, bracketPicks)

	return &domain.UserPickem{
		GroupPicks: s.buildGroupPicksView(groupPicksByGroup, lockedByGroup),
		BestThirds: s.buildBestThirdsView(bestThirds),
		Bracket:    bracket,
		Progress: domain.PickemProgress{
			Groups:     stepProgress(len(lockedCodes), 12),
			BestThirds: computeBestThirdsProgress(bestThirds),
			Bracket:    computeBracketProgress(bracketPicks),
		},
		IsLocked: s.isPickemLocked(),
	}, nil
}

func (s *PickemService) GetMemberPickem(ctx context.Context, userID string) (*domain.UserPickem, error) {
	if !s.isPickemLocked() {
		return nil, domain.ErrPredictionsHidden
	}

	return s.GetUserPickem(ctx, userID)
}

func (s *PickemService) GetChampionPick(ctx context.Context, userID string) (*domain.Team, error) {
	fifaCode, err := s.pickemRepo.GetChampionPick(ctx, userID)
	if err != nil || fifaCode == nil {
		return nil, err
	}

	return s.teamLookup[*fifaCode], nil
}

func (s *PickemService) GetChampionPickCounts(ctx context.Context, limit int) ([]*domain.TitleFavorite, error) {
	return s.pickemRepo.GetChampionPickCounts(ctx, limit)
}

func (s *PickemService) GetUserPickemProgress(ctx context.Context, userID string) (*domain.PickemProgress, error) {
	counts, err := s.pickemRepo.GetUserProgressCounts(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &domain.PickemProgress{
		Groups:     stepProgress(counts.Groups, 12),
		BestThirds: stepProgress(counts.BestThirds, 8),
		Bracket:    stepProgress(counts.Bracket, 32),
	}, nil
}

func stepProgress(completed, total int) domain.StepProgress {
	return domain.StepProgress{Completed: completed, Total: total}
}

// SaveGroupPicks persists the team order for all 12 groups and syncs each group's lock state.
// lockedCodes is the declarative set of locked groups from the client. The pick upsert (which
// cascade-clears best-thirds + bracket) is skipped when the order is unchanged, so toggling
// only a lock never wipes downstream picks; lock state is synced unconditionally.
func (s *PickemService) SaveGroupPicks(
	ctx context.Context,
	userID string,
	picks []*domain.UserGroupPick,
	lockedCodes []string,
) error {
	if s.isPickemLocked() {
		return domain.ErrPickemLocked
	}

	existing, err := s.pickemRepo.GetGroupPicks(ctx, userID)
	if err != nil {
		return err
	}

	if !sameGroupPicks(existing, picks) {
		if err := s.pickemRepo.UpsertGroupPicks(ctx, userID, picks); err != nil {
			return err
		}
	}

	return s.pickemRepo.SetGroupLocks(ctx, userID, lockedCodes)
}

func (s *PickemService) SaveBestThirds(ctx context.Context, userID string, teamFifaCodes []string) error {
	if s.isPickemLocked() {
		return domain.ErrPickemLocked
	}

	// All 12 groups must be fully saved before best thirds are accepted
	groupPicks, err := s.pickemRepo.GetGroupPicks(ctx, userID)
	if err != nil {
		return err
	}

	groupPicksByGroup := groupPicksByGroupCode(groupPicks)

	if !groupOrderingComplete(groupPicksByGroup) {
		return domain.ErrGroupPicksRequired
	}

	// Build the set of valid position-3 teams across all 12 groups
	validThirds := make(map[string]bool, 12)
	for _, picks := range groupPicksByGroup {
		for _, p := range picks {
			if p.PredictedPosition == 3 {
				validThirds[p.TeamFifaCode] = true
			}
		}
	}

	// Check if the FIFA codes are for valid third-place teams
	for _, fifa := range teamFifaCodes {
		if !validThirds[fifa] {
			return domain.ErrInvalidBestThirdTeam
		}
	}

	bestThirds := make([]*domain.UserBestThirdPick, len(teamFifaCodes))
	for i, fifa := range teamFifaCodes {
		bestThirds[i] = &domain.UserBestThirdPick{TeamFifaCode: fifa}
	}

	// No-op detection: if the incoming picks are identical to what's stored, skip the upsert
	// to prevent the cascade reset of bracket picks from firing
	existing, err := s.pickemRepo.GetBestThirdPicks(ctx, userID)
	if err != nil {
		return err
	}
	if sameBestThirds(existing, bestThirds) {
		return nil
	}

	return s.pickemRepo.UpsertBestThirds(ctx, userID, bestThirds)
}

func (s *PickemService) SaveBracketPicks(ctx context.Context, userID string, bracketPicks []*domain.UserBracketPick) error {
	if s.isPickemLocked() {
		return domain.ErrPickemLocked
	}

	var (
		groupPicks []*domain.UserGroupPick
		bestThirds []*domain.UserBestThirdPick
	)

	eg, egCtx := errgroup.WithContext(ctx)

	eg.Go(func() (err error) { groupPicks, err = s.pickemRepo.GetGroupPicks(egCtx, userID); return })

	eg.Go(func() (err error) { bestThirds, err = s.pickemRepo.GetBestThirdPicks(egCtx, userID); return })

	if err := eg.Wait(); err != nil {
		return err
	}

	groupPicksByGroup := groupPicksByGroupCode(groupPicks)

	// Check if all 12 groups and 8 best-thirds are complete before bracket picks are accepted
	groupsAndThirdsComplete := groupOrderingComplete(groupPicksByGroup) && len(bestThirds) == 8
	if !groupsAndThirdsComplete {
		return domain.ErrGroupPicksRequired
	}

	bracket := s.projectBracket(groupPicksByGroup, bestThirds, bracketPicks)
	bracketByMatchID := make(map[int64]*domain.BracketMatchSlot, len(bracket))
	for _, slot := range bracket {
		bracketByMatchID[slot.MatchID] = slot
	}

	for _, pick := range bracketPicks {
		slot := bracketByMatchID[pick.MatchID]
		// Check that the picked team is the home or away team for the match
		if pick.TeamFifaCode != slot.HomeTeam.FifaCode &&
			pick.TeamFifaCode != slot.AwayTeam.FifaCode {
			return domain.ErrInvalidBracketPickTeam
		}
	}

	return s.pickemRepo.UpsertBracketPicks(ctx, userID, bracketPicks)
}

func (s *PickemService) isPickemLocked() bool {
	return time.Now().UTC().After(s.lockTime)
}

// groupOrderingComplete reports whether every group has exactly 4 picks in positions 1–4.
// Group ordering completeness no longer drives step-1 progress (locks do), but it remains a
// precondition for accepting best-thirds and bracket picks.
func groupOrderingComplete(groupPicks map[string][]*domain.UserGroupPick) bool {
	completed := 0
	for _, picks := range groupPicks {
		if len(picks) == 4 {
			completed++
		}
	}

	return completed == 12
}

func computeBestThirdsProgress(bestThirds []*domain.UserBestThirdPick) domain.StepProgress {
	return stepProgress(len(bestThirds), 8)
}

func computeBracketProgress(bracketPicks []*domain.UserBracketPick) domain.StepProgress {
	return stepProgress(len(bracketPicks), 32)
}

func sameGroupPicks(existing, incoming []*domain.UserGroupPick) bool {
	if len(existing) != len(incoming) {
		return false
	}

	type pickKey struct {
		groupCode string
		position  int
	}

	// Build a map of the stored picks by group code and position
	storedByKey := make(map[pickKey]string, len(existing))
	for _, pick := range existing {
		storedByKey[pickKey{pick.TeamGroupCode, pick.PredictedPosition}] = pick.TeamFifaCode
	}

	// Check if the incoming picks are identical to the stored picks
	for _, pick := range incoming {
		if storedByKey[pickKey{pick.TeamGroupCode, pick.PredictedPosition}] != pick.TeamFifaCode {
			return false
		}
	}

	return true
}

func sameBestThirds(existing, incoming []*domain.UserBestThirdPick) bool {
	if len(existing) != len(incoming) {
		return false
	}

	// Build a map of the stored picks by FIFA code
	storedByKey := make(map[string]bool, len(existing))
	for _, pick := range existing {
		storedByKey[pick.TeamFifaCode] = true
	}

	// Check if the incoming picks are identical to the stored picks
	for _, pick := range incoming {
		if !storedByKey[pick.TeamFifaCode] {
			return false
		}
	}

	return true
}

// Assembles the 32 knockout slots in match-id order using the user's group picks, best-third picks, and any bracket picks already submitted
func (s *PickemService) projectBracket(
	groupPicks map[string][]*domain.UserGroupPick,
	bestThirds []*domain.UserBestThirdPick,
	bracketPicks []*domain.UserBracketPick,
) []*domain.BracketMatchSlot {
	// Predicted 1st / 2nd / 3rd / 4th from group picks, keyed for `SourceGroupPosition`
	teamByGroupPosition := make(map[string]map[int]*domain.Team)
	for groupCode, picks := range groupPicks {
		teamByGroupPosition[groupCode] = make(map[int]*domain.Team, 4)
		for _, pick := range picks {
			teamByGroupPosition[groupCode][pick.PredictedPosition] = s.teamLookup[pick.TeamFifaCode]
		}
	}

	// Third placed team that should sit in the away slot for a given match
	bestThirdByMatch := s.bestThirdSlotAssignment(groupPicks, bestThirds)

	// User's picked FIFA code of the team for a given match
	pickedTeamByMatchID := make(map[int64]string, len(bracketPicks))
	for _, pick := range bracketPicks {
		pickedTeamByMatchID[pick.MatchID] = pick.TeamFifaCode
	}

	slots := make([]*domain.BracketMatchSlot, 0, 32)
	slotByMatchID := make(map[int64]*domain.BracketMatchSlot, 32)

	for _, matchID := range allKnockoutMatchIDs {
		slotRule := domain.MatchSlotRules[matchID]
		slot := &domain.BracketMatchSlot{
			MatchID:   matchID,
			StageCode: getStageCodeFromMatchID(matchID),
		}

		// Resolve the home team
		slot.HomeTeam = resolveSlotSource(slotRule.Home, matchID, teamByGroupPosition, bestThirdByMatch, pickedTeamByMatchID, s.teamLookup, slotByMatchID)

		// Resolve the away team
		slot.AwayTeam = resolveSlotSource(slotRule.Away, matchID, teamByGroupPosition, bestThirdByMatch, pickedTeamByMatchID, s.teamLookup, slotByMatchID)

		if pickedTeamFifaCode, ok := pickedTeamByMatchID[matchID]; ok {
			// Resolve the picked team
			slot.PickedTeam = s.teamLookup[pickedTeamFifaCode]
		}

		slots = append(slots, slot)
		slotByMatchID[matchID] = slot
	}

	return slots
}

func resolveSlotSource(
	src domain.Source,
	currentMatchID int64,
	teamByGroupPosition map[string]map[int]*domain.Team,
	bestThirdByMatch map[int64]*domain.Team,
	pickedTeamByMatchID map[int64]string,
	teamLookup map[string]*domain.Team,
	slotByMatchID map[int64]*domain.BracketMatchSlot,
) *domain.Team {
	switch src.Kind {
	case domain.SourceGroupPosition:
		if byPos, ok := teamByGroupPosition[src.GroupCode]; ok {
			return byPos[src.Position]
		}
		return nil

	case domain.SourceBestThird:
		return bestThirdByMatch[currentMatchID]

	case domain.SourceWinner:
		pickedTeamFifaCode, ok := pickedTeamByMatchID[src.MatchID]
		if !ok {
			return nil
		}
		return teamLookup[pickedTeamFifaCode]

	case domain.SourceLoser:
		// Loser of source match = the team in source slot that the user did NOT pick
		sourceSlot, ok := slotByMatchID[src.MatchID]
		if !ok {
			return nil
		}

		pickedTeamFifaCode, ok := pickedTeamByMatchID[src.MatchID]
		if !ok || sourceSlot.HomeTeam == nil || sourceSlot.AwayTeam == nil {
			return nil
		}

		if pickedTeamFifaCode == sourceSlot.HomeTeam.FifaCode {
			return sourceSlot.AwayTeam
		}

		return sourceSlot.HomeTeam
	}

	return nil
}

func groupPicksByGroupCode(picks []*domain.UserGroupPick) map[string][]*domain.UserGroupPick {
	byGroup := make(map[string][]*domain.UserGroupPick, 12)

	for _, pick := range picks {
		byGroup[pick.TeamGroupCode] = append(byGroup[pick.TeamGroupCode], pick)
	}

	for groupCode := range byGroup {
		group := byGroup[groupCode]
		sort.SliceStable(group, func(i, j int) bool {
			return group[i].PredictedPosition < group[j].PredictedPosition
		})
	}

	return byGroup
}

func (s *PickemService) buildGroupPicksView(picksByGroup map[string][]*domain.UserGroupPick, lockedByGroup map[string]bool) []domain.ResolvedGroupPick {
	defaults := s.defaultGroupPicksView()

	for i, group := range defaults {
		saved, ok := picksByGroup[group.GroupCode]
		if ok && len(saved) > 0 {
			teams := make([]domain.RankedTeam, 0, len(saved))

			for _, pick := range saved {
				teams = append(teams, domain.RankedTeam{Position: pick.PredictedPosition, Team: *s.teamLookup[pick.TeamFifaCode]})
			}

			defaults[i].Teams = teams
		}
		defaults[i].Locked = lockedByGroup[group.GroupCode]
	}

	return defaults
}

func (s *PickemService) defaultGroupPicksView() []domain.ResolvedGroupPick {
	groupPicks := make([]domain.ResolvedGroupPick, 0, 12)
	var currentGroup *domain.ResolvedGroupPick

	for _, team := range s.teams {
		if currentGroup == nil || currentGroup.GroupCode != team.GroupCode {
			// Append the group to the view and set the current group
			groupPicks = append(groupPicks, domain.ResolvedGroupPick{GroupCode: team.GroupCode})
			currentGroup = &groupPicks[len(groupPicks)-1]
		}

		// Append the team to the current group
		currentGroup.Teams = append(currentGroup.Teams, domain.RankedTeam{Position: len(currentGroup.Teams) + 1, Team: *team})
	}

	return groupPicks
}

func (s *PickemService) buildBestThirdsView(bestThirdPicks []*domain.UserBestThirdPick) []*domain.Team {
	bestThirdTeams := make([]*domain.Team, 0, len(bestThirdPicks))
	for _, pick := range bestThirdPicks {
		bestThirdTeams = append(bestThirdTeams, s.teamLookup[pick.TeamFifaCode])
	}

	return bestThirdTeams
}

// Maps each R32 best-third slot (match ID) to the team that should fill it
func (s *PickemService) bestThirdSlotAssignment(
	groupPicks map[string][]*domain.UserGroupPick,
	bestThirdsPicks []*domain.UserBestThirdPick,
) map[int64]*domain.Team {
	if len(bestThirdsPicks) != 8 {
		return make(map[int64]*domain.Team)
	}

	// Group code → Team FIFA code for each group's 3rd-place team
	thirdPlaceByGroup := make(map[string]string, 12)
	for groupCode, picks := range groupPicks {
		for _, pick := range picks {
			if pick.PredictedPosition == 3 {
				thirdPlaceByGroup[groupCode] = pick.TeamFifaCode
			}
		}
	}

	// Reverse: Team FIFA code → Group code, to resolve which group each chosen best-third belongs to
	groupByTeamFifaCode := make(map[string]string, len(thirdPlaceByGroup))
	for groupCode, teamFifaCode := range thirdPlaceByGroup {
		groupByTeamFifaCode[teamFifaCode] = groupCode
	}

	// Get the combinations key from the chosen best-thirds picks
	qualifyingGroupCodes := make([]string, 0, 8)
	for _, pick := range bestThirdsPicks {
		// Append the group code for the chosen best-third
		qualifyingGroupCodes = append(qualifyingGroupCodes, groupByTeamFifaCode[pick.TeamFifaCode])
	}

	// Sort the group codes to get a canonical key in combinations table
	sort.Strings(qualifyingGroupCodes)

	var combination domain.ThirdPlaceCombination
	for _, c := range s.combinations {
		if slices.Equal(c.QualifyingGroups, qualifyingGroupCodes) {
			combination = c
			break
		}
	}

	// Get the R32 match ID where the group's winner plays at home
	matchIDByHomeGroupCode := make(map[string]int64, 8)
	for matchID, rule := range domain.MatchSlotRules {
		if rule.Home.Kind == domain.SourceGroupPosition && rule.Home.Position == 1 {
			matchIDByHomeGroupCode[rule.Home.GroupCode] = matchID
		}
	}

	// Assign each best-third team to its slot using the combination's pairing map
	slotAssignment := make(map[int64]*domain.Team, 8)
	for firstPlaceSlot, thirdPlaceSlot := range combination.Assignments {
		homeGroupCode := string(firstPlaceSlot[1])
		awayGroupCode := string(thirdPlaceSlot[1])

		matchID, ok := matchIDByHomeGroupCode[homeGroupCode]
		if !ok {
			continue
		}

		fifaCode, ok := thirdPlaceByGroup[awayGroupCode]
		if !ok {
			continue
		}

		slotAssignment[matchID] = s.teamLookup[fifaCode]
	}

	return slotAssignment
}

func getStageCodeFromMatchID(matchID int64) domain.MatchStageCode {
	switch {
	case matchID >= 73 && matchID <= 88:
		return domain.MatchStageCodeRoundOf32
	case matchID >= 89 && matchID <= 96:
		return domain.MatchStageCodeRoundOf16
	case matchID >= 97 && matchID <= 100:
		return domain.MatchStageCodeQuarterFinals
	case matchID >= 101 && matchID <= 102:
		return domain.MatchStageCodeSemiFinals
	case matchID == 103:
		return domain.MatchStageCodeThirdPlace
	case matchID == 104:
		return domain.MatchStageCodeFinal
	}

	return domain.MatchStageCodeGroupStage
}
