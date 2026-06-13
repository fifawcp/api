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

// DashboardResponseDto mirrors domain.Dashboard but serializes next_match through
// MatchResponseDto so it carries the authenticated user's score pick.
type DashboardResponseDto struct {
	PickedChampion *domain.Team                `json:"picked_champion"`
	Stats          *domain.DashboardStats      `json:"stats"`
	NextMatch      *MatchResponseDto           `json:"next_match"`
	Progress       *domain.DashboardProgress   `json:"progress"`
	Leaderboard    domain.DashboardLeaderboard `json:"leaderboard"`
	TitleFavorites []*domain.TitleFavorite     `json:"title_favorites"`
}

func NewDashboardResponse(d *domain.Dashboard) *DashboardResponseDto {
	if d == nil {
		return nil
	}

	var nextMatch *MatchResponseDto
	if d.NextMatch != nil {
		nextMatch = &MatchResponseDto{Match: d.NextMatch}
		if pick := d.NextMatchScorePick; pick != nil {
			nextMatch.UserScorePick = &UserScorePickDto{HomeScore: pick.HomeScore, AwayScore: pick.AwayScore}
		}
	}

	return &DashboardResponseDto{
		PickedChampion: d.PickedChampion,
		Stats:          d.Stats,
		NextMatch:      nextMatch,
		Progress:       d.Progress,
		Leaderboard:    d.Leaderboard,
		TitleFavorites: d.TitleFavorites,
	}
}
