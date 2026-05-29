package main

import (
	"context"
	"database/sql"
	"fmt"
	mathrand "math/rand"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/services"
)

// awardEngagement controls how seeded pickem users distribute across the four
// award-engagement levels. Buckets must sum to 1.0.
//
//	AllFour  — submitted all four award picks (Boot, Ball, Glove, Young)
//	Partial  — submitted 1-3 picks (mid-flow draft)
//	None     — never engaged with the awards card
var awardEngagement = struct {
	AllFour float64
	Partial float64
	None    float64
}{
	AllFour: 0.65,
	Partial: 0.25,
	None:    0.10,
}

// playerCandidatePools partitions the player catalog into the buckets each award
// type can draw from. Loaded once per scenario run.
type playerCandidatePools struct {
	Any         []int64 // Golden Boot + Golden Ball (any player)
	Goalkeepers []int64 // Golden Glove
	YoungOnly   []int64 // Young Player (age <= YoungPlayerMaxAge)
}

func (s *Seeder) seedAwardPicks(ctx context.Context, users []*domain.User) error {
	pools, err := s.loadPlayerCandidatePools(ctx)
	if err != nil {
		return fmt.Errorf("load player pools: %w", err)
	}

	if !pools.hasAllAwardCoverage() {
		s.logger.Warn("Skipping award picks seed: player catalog is missing eligible candidates for at least one award type",
			"any", len(pools.Any),
			"goalkeepers", len(pools.Goalkeepers),
			"young", len(pools.YoungOnly),
		)
		return nil
	}

	picksByLevel := map[string]int{}
	for _, user := range users {
		picks, level := generateAwardPicks(pools)
		picksByLevel[level]++

		if len(picks) == 0 {
			continue
		}
		for _, pick := range picks {
			pick.UserID = user.ID
		}
		if err := s.awardPickRepository.UpsertAwardPicks(ctx, user.ID, picks); err != nil {
			s.logger.Error("Error seeding award picks",
				logging.Error, err.Error(),
				"user_id", user.ID,
			)
		}
	}

	s.logger.Info("Seeded award picks",
		"users", len(users),
		"all_four", picksByLevel["all_four"],
		"partial", picksByLevel["partial"],
		"none", picksByLevel["none"],
	)
	return nil
}

func (s *Seeder) loadPlayerCandidatePools(ctx context.Context) (*playerCandidatePools, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, position, age FROM players`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pools := &playerCandidatePools{}
	for rows.Next() {
		var id int64
		var position string
		var age sql.NullInt32
		if err := rows.Scan(&id, &position, &age); err != nil {
			return nil, err
		}

		pools.Any = append(pools.Any, id)
		if position == string(domain.PlayerPositionGoalkeeper) {
			pools.Goalkeepers = append(pools.Goalkeepers, id)
		}
		if age.Valid && int(age.Int32) <= services.YoungPlayerMaxAge {
			pools.YoungOnly = append(pools.YoungOnly, id)
		}
	}

	return pools, rows.Err()
}

func (p *playerCandidatePools) hasAllAwardCoverage() bool {
	return len(p.Any) > 0 && len(p.Goalkeepers) > 0 && len(p.YoungOnly) > 0
}

// generateAwardPicks rolls the engagement level and produces a slice of award
// picks for one user. Returns the slice and a label describing the level so
// the caller can log distribution stats.
func generateAwardPicks(pools *playerCandidatePools) ([]*domain.UserAwardPick, string) {
	roll := mathrand.Float64()
	switch {
	case roll < awardEngagement.AllFour:
		return picksForCount(pools, len(domain.AwardTypes)), "all_four"
	case roll < awardEngagement.AllFour+awardEngagement.Partial:
		return picksForCount(pools, 1+mathrand.Intn(len(domain.AwardTypes)-1)), "partial"
	default:
		return nil, "none"
	}
}

// picksForCount picks `count` award types at random (without replacement) and
// assigns one eligible player to each.
func picksForCount(pools *playerCandidatePools, count int) []*domain.UserAwardPick {
	shuffled := make([]domain.AwardType, len(domain.AwardTypes))
	copy(shuffled, domain.AwardTypes)
	mathrand.Shuffle(len(shuffled), func(a, b int) {
		shuffled[a], shuffled[b] = shuffled[b], shuffled[a]
	})

	picks := make([]*domain.UserAwardPick, 0, count)
	for _, awardType := range shuffled[:count] {
		picks = append(picks, &domain.UserAwardPick{
			AwardType: awardType,
			PlayerID:  randomPlayerID(awardType, pools),
		})
	}
	return picks
}

func randomPlayerID(awardType domain.AwardType, pools *playerCandidatePools) int64 {
	var pool []int64
	switch awardType {
	case domain.AwardGoldenGlove:
		pool = pools.Goalkeepers
	case domain.AwardYoungPlayer:
		pool = pools.YoungOnly
	default: // AwardGoldenBoot, AwardGoldenBall
		pool = pools.Any
	}
	return pool[mathrand.Intn(len(pool))]
}
