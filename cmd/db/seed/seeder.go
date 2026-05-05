package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	mathrand "math/rand"
	"strconv"
	"strings"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/logging"
)

type Seeder struct {
	db                    *sql.DB
	logger                logging.Logger
	userRepository        domain.UserRepository
	boardRepository       domain.BoardRepository
	boardMemberRepository domain.BoardMemberRepository
	pickemRepository      domain.PickemRepository
}

func NewSeeder(
	db *sql.DB,
	logger logging.Logger,
	userRepository domain.UserRepository,
	boardRepository domain.BoardRepository,
	boardMemberRepository domain.BoardMemberRepository,
	pickemRepository domain.PickemRepository,
) *Seeder {
	return &Seeder{
		db:                    db,
		logger:                logger,
		userRepository:        userRepository,
		boardRepository:       boardRepository,
		boardMemberRepository: boardMemberRepository,
		pickemRepository:      pickemRepository,
	}
}

func (s *Seeder) Flush() {
	s.logger.Info("Flushing database")
	ctx := context.Background()

	queries := []string{
		"DELETE FROM boards WHERE name != 'Global';",
		"DELETE FROM users;",
	}

	for _, query := range queries {
		_, err := s.db.ExecContext(ctx, query)
		if err != nil {
			s.logger.Error(
				"Error flushing db",
				logging.Error, err.Error(),
			)
		}
	}
}

func (s *Seeder) Seed() {
	s.logger.Info("Seeding database")
	ctx := context.Background()

	users := s.generateUsers(usersAmount)
	createdUsers := make([]*domain.User, 0, len(users))

	for _, user := range users {
		if err := s.userRepository.CreateUser(ctx, user); err != nil {
			s.logger.Error(
				"Error seeding users",
				logging.Error, err.Error(),
			)
			continue
		}

		createdUsers = append(createdUsers, user)
	}

	s.seedBoards(ctx, createdUsers)

	// Shuffle the users to get a random subset for pickem data
	mathrand.Shuffle(len(createdUsers), func(a, b int) {
		createdUsers[a], createdUsers[b] = createdUsers[b], createdUsers[a]
	})

	// Get a random subset of users for pickem data
	pickemCount := int(float64(len(createdUsers)) * pickemUserPercentage)
	pickemUsers := createdUsers[:pickemCount]

	s.seedPickemData(ctx, pickemUsers)
	s.seedMatchScores(ctx, pickemUsers)
}

func (s *Seeder) Run() {
	s.Flush()
	s.Seed()
	s.logger.Info("Database seeded successfully")
}

func (s *Seeder) generateUsers(amount int) []*domain.User {
	users := make([]*domain.User, amount)

	for i := range amount {
		firstName := gofakeit.FirstName()
		lastName := gofakeit.LastName()
		username := strings.ToLower(firstName) + "_" + strings.ToLower(lastName) + strconv.Itoa(i)
		email := username + "@email.com"

		users[i] = &domain.User{
			FirstName: firstName,
			LastName:  lastName,
			Username:  username,
			Email:     email,
		}
	}

	return users
}

func (s *Seeder) seedBoards(ctx context.Context, users []*domain.User) {
	for i := range boardsAmount {
		owner := users[i]
		joinCode := generateJoinCode()
		ownerID := owner.ID

		board := &domain.Board{
			Name:        gofakeit.Company(),
			OwnerUserID: &ownerID,
			JoinCode:    &joinCode,
		}

		if err := s.boardRepository.CreateBoardWithOwner(ctx, board); err != nil {
			s.logger.Error(
				"Error seeding board",
				logging.Error, err.Error(),
			)
			continue
		}

		memberCount := boardMembersMin + mathrand.Intn(boardMembersMax-boardMembersMin+1)
		candidates := make([]*domain.User, 0, len(users)-1)

		// Get all users except the owner
		for _, user := range users {
			if user.ID != owner.ID {
				candidates = append(candidates, user)
			}
		}

		// Get a random subset of candidates
		mathrand.Shuffle(len(candidates), func(a, b int) {
			candidates[a], candidates[b] = candidates[b], candidates[a]
		})

		// Insert the members into the board
		for j := 0; j < memberCount && j < len(candidates); j++ {
			if _, err := s.boardMemberRepository.CreateBoardMember(ctx, joinCode, candidates[j].ID); err != nil {
				s.logger.Error(
					"Error seeding board member",
					logging.Error, err.Error(),
				)
			}
		}
	}
}

func generateJoinCode() string {
	const (
		charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		length  = 8
	)

	result := make([]byte, length)
	for i := range result {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result[i] = charset[num.Int64()]
	}

	return string(result)
}

func (s *Seeder) seedPickemData(ctx context.Context, users []*domain.User) {
	groupCodes := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L"}

	for _, user := range users {
		picks := make([]*domain.UserGroupPick, 0, 48)
		thirdPlaceTeams := make([]string, 0, 12)

		for _, groupCode := range groupCodes {
			teams := make([]string, len(teamsByGroup[groupCode]))
			copy(teams, teamsByGroup[groupCode])
			mathrand.Shuffle(len(teams), func(a, b int) { teams[a], teams[b] = teams[b], teams[a] })

			for pos, fifaCode := range teams {
				picks = append(picks, &domain.UserGroupPick{
					UserID:            user.ID,
					TeamFifaCode:      fifaCode,
					TeamGroupCode:     groupCode,
					PredictedPosition: pos + 1,
				})

				if pos == 2 {
					thirdPlaceTeams = append(thirdPlaceTeams, fifaCode)
				}
			}
		}

		if err := s.pickemRepository.UpsertGroupPicks(ctx, user.ID, picks); err != nil {
			s.logger.Error("Error seeding group picks", logging.Error, err.Error())
			continue
		}

		mathrand.Shuffle(len(thirdPlaceTeams), func(a, b int) {
			thirdPlaceTeams[a], thirdPlaceTeams[b] = thirdPlaceTeams[b], thirdPlaceTeams[a]
		})

		bestThirds := make([]*domain.UserBestThirdPick, 8)
		for i, code := range thirdPlaceTeams[:8] {
			bestThirds[i] = &domain.UserBestThirdPick{
				UserID:       user.ID,
				TeamFifaCode: code,
			}
		}

		if err := s.pickemRepository.UpsertBestThirds(ctx, user.ID, bestThirds); err != nil {
			s.logger.Error("Error seeding best thirds", logging.Error, err.Error())
		}
	}
}

func (s *Seeder) seedMatchScores(ctx context.Context, users []*domain.User) {
	placeholders := make([]string, len(groupStageMatchIDs))
	for i := range groupStageMatchIDs {
		base := i * 4
		placeholders[i] = fmt.Sprintf("($%d, $%d, $%d, $%d)", base+1, base+2, base+3, base+4)
	}

	// Idempotent insert of match scores for the users
	query := "INSERT INTO user_match_score_picks (user_id, match_id, home_score, away_score) VALUES " +
		strings.Join(placeholders, ", ") +
		" ON CONFLICT (user_id, match_id) DO UPDATE SET home_score = EXCLUDED.home_score, away_score = EXCLUDED.away_score"

	for _, user := range users {
		args := make([]any, 0, len(groupStageMatchIDs)*4)

		for _, matchID := range groupStageMatchIDs {
			args = append(args, user.ID, matchID, mathrand.Intn(6), mathrand.Intn(6))
		}

		if _, err := s.db.ExecContext(ctx, query, args...); err != nil {
			s.logger.Error("Error seeding match scores", logging.Error, err.Error())
		}
	}
}
