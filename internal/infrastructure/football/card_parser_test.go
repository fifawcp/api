package football_test

import (
	"testing"

	"github.com/fifawcp/api/internal/infrastructure/football"
	"github.com/stretchr/testify/assert"
)

func cardEvent(teamID, playerID int64, elapsed int, extra *int, detail string) football.FixtureEvent {
	return football.FixtureEvent{
		Time:   football.EventTime{Elapsed: elapsed, Extra: extra},
		Team:   football.EventTeam{ID: teamID},
		Player: football.EventPlayer{ID: playerID},
		Type:   "Card",
		Detail: detail,
	}
}

const (
	teamA   = int64(1)
	teamB   = int64(2)
	playerA = int64(10)
	playerB = int64(20)
	playerC = int64(30)
)

func TestParseCardEvents(t *testing.T) {
	t.Parallel()

	t.Run("when a player receives only a yellow card", func(t *testing.T) {
		t.Parallel()

		events := []football.FixtureEvent{
			cardEvent(teamA, playerA, 10, nil, "Yellow Card"),
		}

		result := football.ParseCardEvents(events)

		assert.Equal(t, 1, result[teamA].YellowCardCount)
		assert.Equal(t, 0, result[teamA].IndirectRedCount)
		assert.Equal(t, 0, result[teamA].DirectRedCount)
		assert.Equal(t, 0, result[teamA].YellowCardAndDirectRedCardCount)
	})

	t.Run("when a player receives a second yellow that triggers a red (indirect red)", func(t *testing.T) {
		t.Parallel()

		events := []football.FixtureEvent{
			cardEvent(teamA, playerA, 20, nil, "Yellow Card"),
			cardEvent(teamA, playerA, 60, nil, "Yellow Card"),
			cardEvent(teamA, playerA, 60, nil, "Red Card"),
		}

		result := football.ParseCardEvents(events)

		assert.Equal(t, 0, result[teamA].YellowCardCount)
		assert.Equal(t, 1, result[teamA].IndirectRedCount)
		assert.Equal(t, 0, result[teamA].DirectRedCount)
		assert.Equal(t, 0, result[teamA].YellowCardAndDirectRedCardCount)
	})

	t.Run("when the indirect red occurs during stoppage time (elapsed+extra timestamp)", func(t *testing.T) {
		t.Parallel()

		events := []football.FixtureEvent{
			cardEvent(teamA, playerA, 81, nil, "Yellow Card"),
			cardEvent(teamA, playerA, 90, new(3), "Yellow Card"),
			cardEvent(teamA, playerA, 90, new(3), "Red Card"),
		}

		result := football.ParseCardEvents(events)

		assert.Equal(t, 0, result[teamA].YellowCardCount)
		assert.Equal(t, 1, result[teamA].IndirectRedCount)
		assert.Equal(t, 0, result[teamA].DirectRedCount)
		assert.Equal(t, 0, result[teamA].YellowCardAndDirectRedCardCount)
	})

	t.Run("when a player receives a direct red with no prior yellow", func(t *testing.T) {
		t.Parallel()

		events := []football.FixtureEvent{
			cardEvent(teamA, playerA, 55, nil, "Red Card"),
		}

		result := football.ParseCardEvents(events)

		assert.Equal(t, 0, result[teamA].YellowCardCount)
		assert.Equal(t, 0, result[teamA].IndirectRedCount)
		assert.Equal(t, 1, result[teamA].DirectRedCount)
		assert.Equal(t, 0, result[teamA].YellowCardAndDirectRedCardCount)
	})

	t.Run("when a player receives a yellow then a direct red at a different time", func(t *testing.T) {
		t.Parallel()

		events := []football.FixtureEvent{
			cardEvent(teamA, playerA, 30, nil, "Yellow Card"),
			cardEvent(teamA, playerA, 75, nil, "Red Card"),
		}

		result := football.ParseCardEvents(events)

		assert.Equal(t, 0, result[teamA].YellowCardCount)
		assert.Equal(t, 0, result[teamA].IndirectRedCount)
		assert.Equal(t, 0, result[teamA].DirectRedCount)
		assert.Equal(t, 1, result[teamA].YellowCardAndDirectRedCardCount)
	})

	t.Run("when multiple players on the same team each fall into a different situation", func(t *testing.T) {
		t.Parallel()

		events := []football.FixtureEvent{
			cardEvent(teamA, playerA, 10, nil, "Yellow Card"), // yellow only
			cardEvent(teamA, playerB, 20, nil, "Yellow Card"),
			cardEvent(teamA, playerB, 60, nil, "Yellow Card"),
			cardEvent(teamA, playerB, 60, nil, "Red Card"), // indirect red (second yellow at T=60)
			cardEvent(teamA, playerC, 80, nil, "Red Card"), // direct red
		}

		result := football.ParseCardEvents(events)

		assert.Equal(t, 1, result[teamA].YellowCardCount)
		assert.Equal(t, 1, result[teamA].IndirectRedCount)
		assert.Equal(t, 1, result[teamA].DirectRedCount)
		assert.Equal(t, 0, result[teamA].YellowCardAndDirectRedCardCount)
	})

	t.Run("when two teams have card events, each team's counts are independent", func(t *testing.T) {
		t.Parallel()

		events := []football.FixtureEvent{
			cardEvent(teamA, playerA, 10, nil, "Yellow Card"),
			cardEvent(teamB, playerB, 55, nil, "Red Card"),
		}

		result := football.ParseCardEvents(events)

		assert.Equal(t, 1, result[teamA].YellowCardCount)
		assert.Equal(t, 0, result[teamA].DirectRedCount)

		assert.Equal(t, 0, result[teamB].YellowCardCount)
		assert.Equal(t, 1, result[teamB].DirectRedCount)
	})

	t.Run("when events include goals and substitutions, they are ignored", func(t *testing.T) {
		t.Parallel()

		events := []football.FixtureEvent{
			{Time: football.EventTime{Elapsed: 30}, Team: football.EventTeam{ID: teamA}, Player: football.EventPlayer{ID: playerA}, Type: "Goal", Detail: "Normal Goal"},
			{Time: football.EventTime{Elapsed: 55}, Team: football.EventTeam{ID: teamA}, Player: football.EventPlayer{ID: playerB}, Type: "subst", Detail: "Substitution 1"},
		}

		result := football.ParseCardEvents(events)

		assert.Equal(t, football.TeamCardSummary{}, result[teamA])
	})

	t.Run("when there are no events", func(t *testing.T) {
		t.Parallel()

		result := football.ParseCardEvents(nil)

		assert.Empty(t, result)
	})
}
