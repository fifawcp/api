package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"math/big"
	mathrand "math/rand"

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
}

func NewSeeder(
	db *sql.DB,
	logger logging.Logger,
	userRepository domain.UserRepository,
	boardRepository domain.BoardRepository,
	boardMemberRepository domain.BoardMemberRepository,
) *Seeder {
	return &Seeder{
		db:                    db,
		logger:                logger,
		userRepository:        userRepository,
		boardRepository:       boardRepository,
		boardMemberRepository: boardMemberRepository,
	}
}

func (s *Seeder) Flush() {
	s.logger.Info("Flushing database")
	ctx := context.Background()

	queries := []string{
		"TRUNCATE TABLE boards CASCADE;",
		"TRUNCATE TABLE users CASCADE;",
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
	for _, user := range users {
		if err := s.userRepository.CreateUser(ctx, user); err != nil {
			s.logger.Error(
				"Error seeding users",
				logging.Error, err.Error(),
			)
		}
	}

	s.seedBoards(ctx, users)
}

func (s *Seeder) Run() {
	s.Flush()
	s.Seed()
	s.logger.Info("Database seeded successfully")
}

func (s *Seeder) generateUsers(amount int) []*domain.User {
	users := make([]*domain.User, amount)

	for i := 0; i < amount; i++ {
		users[i] = &domain.User{
			FirstName: gofakeit.FirstName(),
			LastName:  gofakeit.LastName(),
			Username:  gofakeit.Username(),
			Email:     gofakeit.Email(),
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
			if err := s.boardMemberRepository.CreateBoardMember(ctx, joinCode, candidates[j].ID); err != nil {
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
