package services

import (
	"context"
	"fmt"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/logging"
)

type UserServiceInterface interface {
	GetUser(ctx context.Context, userID string) (*domain.User, error)
}

type UserService struct {
	userRepository domain.UserRepository
	userStorage    domain.UserStorage
	logger         logging.Logger
}

func NewUserService(
	userRepository domain.UserRepository,
	userStorage domain.UserStorage,
	logger logging.Logger,
) UserServiceInterface {
	return &UserService{
		userRepository: userRepository,
		userStorage:    userStorage,
		logger:         logger,
	}
}

func (s *UserService) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	// Try storage first
	user, err := s.userStorage.GetUser(ctx, userID)
	if err != nil {
		s.logger.Error(
			fmt.Sprintf("failed to get user with ID: %s from storage", userID),
			logging.Error, err.Error(),
		)
	}

	if user != nil {
		return user, nil
	}

	// If cache miss get user from database
	user, err = s.userRepository.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Cache the user
	err = s.userStorage.SetUser(ctx, user)
	if err != nil {
		s.logger.Error(
			fmt.Sprintf("failed to set user with ID: %s in cache", userID),
			logging.Error, err.Error(),
		)
	}

	return user, nil
}
