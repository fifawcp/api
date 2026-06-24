package domain

type Dashboard struct {
	PickedChampion *Team           `json:"picked_champion"`
	Stats          *DashboardStats `json:"stats"`
	// NextMatch is the earliest upcoming match. Deprecated: use NextMatches, which
	// also carries any other matches kicking off at the same time. Kept populated
	// (= NextMatches[0]) for backward compatibility.
	NextMatch      *Match               `json:"next_match"`
	NextMatches    []*Match             `json:"next_matches"`
	Progress       *DashboardProgress   `json:"progress"`
	Leaderboard    DashboardLeaderboard `json:"leaderboard"`
	TitleFavorites []*TitleFavorite     `json:"title_favorites"`
	// NextMatchScorePicks maps each next match's ID to the authenticated user's
	// score pick for it, when one exists.
	NextMatchScorePicks map[int64]*UserMatchScorePick `json:"-"`
}

type TitleFavorite struct {
	Team        *Team `json:"team"`
	PickCount   int   `json:"pick_count"`
	PickPercent int   `json:"pick_percent"`
}

type DashboardStats struct {
	Pickem CompetitionUserStats `json:"pickem"`
	Match  CompetitionUserStats `json:"match"`
}

type DashboardProgress struct {
	MatchPicks StepProgress   `json:"match_picks"`
	Pickem     PickemProgress `json:"pickem"`
	Awards     StepProgress   `json:"awards"`
}

type DashboardLeaderboard struct {
	Pickem CompetitionTop `json:"pickem"`
	Match  CompetitionTop `json:"match"`
}

type CompetitionTop struct {
	CompetitionID   int64                  `json:"competition_id" example:"1"`
	BoardID         int64                  `json:"board_id" example:"1"`
	CompetitionName string                 `json:"competition_name" example:"Pick'em"`
	Entries         []DashboardLeaderEntry `json:"entries"`
}

type DashboardLeaderEntry struct {
	CompetitionUserStats
	Member CompetitionLeaderboardMember `json:"member"`
}
