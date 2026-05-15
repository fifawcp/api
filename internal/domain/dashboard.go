package domain

type Dashboard struct {
	PickedChampion *Team                `json:"picked_champion"`
	Stats          DashboardStats       `json:"stats"`
	NextMatch      *Match               `json:"next_match"`
	Progress       DashboardProgress    `json:"progress"`
	Leaderboard    DashboardLeaderboard `json:"leaderboard"`
}

type DashboardStats struct {
	Pickem CompetitionUserStats `json:"pickem"`
	Match  CompetitionUserStats `json:"match"`
}

type DashboardProgress struct {
	MatchPicks StepProgress   `json:"match_picks"`
	Pickem     PickemProgress `json:"pickem"`
}

type DashboardLeaderboard struct {
	Pickem CompetitionTop `json:"pickem"`
	Match  CompetitionTop `json:"match"`
}

type CompetitionTop struct {
	CompetitionName string                 `json:"competition_name" example:"Pick'em"`
	Entries         []DashboardLeaderEntry `json:"entries"`
}

type DashboardLeaderEntry struct {
	CompetitionUserStats
	Member CompetitionLeaderboardMember `json:"member"`
}
