package football

func ParseCardEvents(events []FixtureEvent) map[int64]TeamCardSummary {
	type playerState struct {
		yellowCount   int
		yellowTimes   []EventTime // timestamps of each yellow card
		redTime       *EventTime  // set when a red card is received
		isIndirectRed bool        // true if a yellow was issued at the same moment as the red
	}

	// playerStates[teamID][playerID]
	playerStates := map[int64]map[int64]*playerState{}

	getState := func(teamID, playerID int64) *playerState {
		// If the team does not exist in the map, create it
		if playerStates[teamID] == nil {
			playerStates[teamID] = map[int64]*playerState{}
		}

		// If the player does not exist in the team map, create it
		if playerStates[teamID][playerID] == nil {
			playerStates[teamID][playerID] = &playerState{}
		}

		return playerStates[teamID][playerID]
	}

	// First pass: record all yellow and red card events per player
	for index, event := range events {
		if event.Type != "Card" {
			continue
		}

		state := getState(event.Team.ID, event.Player.ID)

		switch event.Detail {
		case "Yellow Card":
			state.yellowCount++
			state.yellowTimes = append(state.yellowTimes, event.Time)

		case "Red Card":
			state.redTime = &events[index].Time
			// Check if there is a yellow card for the same player at the same moment,
			// which indicates this red is the result of a second yellow (indirect red)
			state.isIndirectRed = hasConcurrentYellow(events, event.Team.ID, event.Player.ID, event.Time)
		}
	}

	// Second pass: classify each player into a FIFA disciplinary situation
	result := map[int64]TeamCardSummary{}

	for teamID, players := range playerStates {
		summary := result[teamID]

		for _, state := range players {
			switch {
			case state.redTime == nil:
				// No red card — yellow cards only
				summary.YellowCardCount += state.yellowCount

			case state.isIndirectRed:
				summary.IndirectRedCount++

			case state.yellowCount > 0:
				// Red card after a prior yellow, but the red is at a different time → direct red
				// following a yellow (the yellow is counted separately per FIFA rules)
				summary.YellowCardAndDirectRedCardCount++

			default:
				// Red card with no prior yellow in the match → direct red
				summary.DirectRedCount++
			}
		}

		result[teamID] = summary
	}

	return result
}

func hasConcurrentYellow(events []FixtureEvent, teamID, playerID int64, at EventTime) bool {
	for _, event := range events {
		if event.Type != "Card" || event.Detail != "Yellow Card" {
			continue
		}

		// Check if the event is for the same team and player
		if event.Team.ID != teamID || event.Player.ID != playerID {
			continue
		}

		// Check if the event is at the same time as the provided timestamp
		if sameTime(event.Time, at) {
			return true
		}
	}

	return false
}

func sameTime(a, b EventTime) bool {
	if a.Elapsed != b.Elapsed {
		return false
	}

	if a.Extra == nil && b.Extra == nil {
		return true
	}

	if a.Extra == nil || b.Extra == nil {
		return false
	}

	return *a.Extra == *b.Extra
}
