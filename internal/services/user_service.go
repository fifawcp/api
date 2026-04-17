package services

import (
	"context"

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
			"failed to get user from storage",
			"error", err,
			"userID", userID,
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
			"failed to cache user",
			"error", err,
			"userID", userID,
		)
	}

	return user, nil
}
