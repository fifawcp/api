package dtos

import "github.com/fifawcp/api/internal/domain"

type MatchResponseDto struct {
	*domain.Match
	UserScorePick *UserScorePickDto `json:"user_score_pick,omitempty"`
}

type UserScorePickDto struct {
	HomeScore int `json:"home_score"`
	AwayScore int `json:"away_score"`
}

type SaveMatchScorePickDto struct {
	HomeScore *int `json:"home_score" validate:"required,gte=0,lte=20" example:"2"`
	AwayScore *int `json:"away_score" validate:"required,gte=0,lte=20" example:"1"`
}

// MemberPickDto represents one board member alongside their (possibly nil) score pick.
type MemberPickDto struct {
	Member domain.CompetitionLeaderboardMember `json:"member"`
	Pick   *UserScorePickDto                   `json:"pick"`
}

// MatchMemberPicksDto is the response for viewing all board members' picks for a single match.
type MatchMemberPicksDto struct {
	Match *domain.Match   `json:"match"`
	Picks []MemberPickDto `json:"picks"`
}

// DashboardResponseDto mirrors domain.Dashboard but serializes the next matches
// through MatchResponseDto so each carries the authenticated user's score pick.
type DashboardResponseDto struct {
	PickedChampion *domain.Team           `json:"picked_champion"`
	Stats          *domain.DashboardStats `json:"stats"`
	// NextMatch is deprecated: use NextMatches. Kept (= NextMatches[0]) for
	// backward compatibility while clients migrate.
	NextMatch      *MatchResponseDto           `json:"next_match"`
	NextMatches    []*MatchResponseDto         `json:"next_matches"`
	Progress       *domain.DashboardProgress   `json:"progress"`
	Leaderboard    domain.DashboardLeaderboard `json:"leaderboard"`
	TitleFavorites []*domain.TitleFavorite     `json:"title_favorites"`
}

func NewDashboardResponse(d *domain.Dashboard) *DashboardResponseDto {
	if d == nil {
		return nil
	}

	nextMatches := make([]*MatchResponseDto, 0, len(d.NextMatches))
	for _, match := range d.NextMatches {
		dto := &MatchResponseDto{Match: match}
		if pick := d.NextMatchScorePicks[match.ID]; pick != nil {
			dto.UserScorePick = &UserScorePickDto{HomeScore: pick.HomeScore, AwayScore: pick.AwayScore}
		}
		nextMatches = append(nextMatches, dto)
	}

	var nextMatch *MatchResponseDto
	if len(nextMatches) > 0 {
		nextMatch = nextMatches[0]
	}

	return &DashboardResponseDto{
		PickedChampion: d.PickedChampion,
		Stats:          d.Stats,
		NextMatch:      nextMatch,
		NextMatches:    nextMatches,
		Progress:       d.Progress,
		Leaderboard:    d.Leaderboard,
		TitleFavorites: d.TitleFavorites,
	}
}
