package main

import (
	"context"
	"database/sql"
	"math/rand"
	"strconv"
	"strings"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/logging"
)

type Seeder struct {
	db             *sql.DB
	logger         logging.Logger
	userRepository domain.UserRepository
}

func NewSeeder(
	db *sql.DB,
	logger logging.Logger,
	userRepository domain.UserRepository,
) *Seeder {
	return &Seeder{
		db:             db,
		logger:         logger,
		userRepository: userRepository,
	}
}

func (s *Seeder) Flush() {
	s.logger.Info("Flushing database")
	ctx := context.Background()

	queries := []string{
		"TRUNCATE TABLE users CASCADE;",
	}

	for _, query := range queries {
		_, err := s.db.ExecContext(ctx, query)
		if err != nil {
			s.logger.Error(
				"Error flushing db",
				"error", err,
			)
		}
	}
}

func (s *Seeder) Seed() {
	s.logger.Info("Seeding database")
	ctx := context.Background()

	users := s.generateUsers(usersAmount, s.db)
	for _, user := range users {
		err := s.userRepository.CreateUser(ctx, user)
		if err != nil {
			s.logger.Error(
				"Error seeding users",
				"error", err,
			)
		}
	}
}

func (s *Seeder) Run() {
	s.Flush()
	s.Seed()
	s.logger.Info("Database seeded successfully")
}

func (s *Seeder) generateUsers(amount int, db *sql.DB) []*domain.User {
	users := make([]*domain.User, amount)

	for i := 0; i < amount; i++ {
		firstName := seedFirstNames[rand.Intn(len(seedFirstNames))]
		lastName := seedLastNames[rand.Intn(len(seedLastNames))]
		username := strings.ToLower(string(firstName[0])+strings.TrimSpace(lastName)) + strconv.Itoa(i)
		email := username + "@" + seedEmailDomains[rand.Intn(len(seedEmailDomains))]

		user := &domain.User{
			FirstName: firstName,
			LastName:  lastName,
			Username:  username,
			Email:     email,
		}

		users[i] = user
	}

	return users
}
