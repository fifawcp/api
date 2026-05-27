package services

import (
	"context"
	"fmt"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
	"github.com/fifawcp/api/internal/infrastructure/logging"
)

type UserServiceInterface interface {
	GetUser(ctx context.Context, userID string) (*domain.User, error)
	UpdateUser(ctx context.Context, userID string, payload *dtos.UpdateUserDto) (*domain.User, error)
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

	// TODO: where should we remove the chached user?
	// TODO: e.g what happens if a useer becomes admin? and its still cached as role: user
	// TODO: perhaps we don't need to worry about this, as we can manually remove the cached key for the only admin users
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

func (s *UserService) UpdateUser(
	ctx context.Context,
	userID string,
	payload *dtos.UpdateUserDto,
) (*domain.User, error) {
	updatedUser, err := s.userRepository.UpdateUser(ctx, userID, domain.UserUpdate{
		FirstName: payload.FirstName,
		LastName:  payload.LastName,
		Username:  payload.Username,
	})
	if err != nil {
		return nil, err
	}

	if err := s.userStorage.SetUser(ctx, updatedUser); err != nil {
		s.logger.Error(
			fmt.Sprintf("failed to refresh cache for user with ID: %s", userID),
			logging.Error, err.Error(),
		)
	}

	return updatedUser, nil
}
