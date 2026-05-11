package football

type FixtureResponse struct {
	Fixture FixtureInfo
	Teams   FixtureTeams
	Goals   FixtureGoals
	Score   FixtureScore
	Events  []FixtureEvent
}

type FixtureInfo struct {
	ID     int64
	Status FixtureStatus
}

type FixtureStatus struct {
	Short   string
	Elapsed *int
}

type FixtureTeams struct {
	Home FixtureTeam
	Away FixtureTeam
}

type FixtureTeam struct {
	ID int64
}

type FixtureGoals struct {
	Home *int
	Away *int
}

type FixtureScore struct {
	Fulltime  ScorePair
	Extratime ScorePair
	Penalty   ScorePair
}

type ScorePair struct {
	Home *int
	Away *int
}

// Returns true when the fixture result is final:
// - FT: Finished in the regular time
// - AET: Finished after extra time without going to the penalty shootout
// - PEN: Finished after the penalty shootout
// See: https://www.api-football.com/documentation-v3#tag/Fixtures/operation/get-fixtures
func (f *FixtureResponse) IsFinished() bool {
	s := f.Fixture.Status.Short
	return s == "FT" || s == "AET" || s == "PEN"
}

type FixtureEvent struct {
	Time   EventTime
	Team   EventTeam
	Player EventPlayer
	Type   string // "Goal", "Card", "subst"
	Detail string // "Normal Goal", "Yellow Card", "Red Card"
}

type EventTime struct {
	Elapsed int
	Extra   *int
}

type EventTeam struct {
	ID int64
}

type EventPlayer struct {
	ID int64
}

type TeamCardSummary struct {
	YellowCardCount                 int
	IndirectRedCount                int
	DirectRedCount                  int
	YellowCardAndDirectRedCardCount int
}
