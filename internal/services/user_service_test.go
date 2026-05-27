package services

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
	"github.com/fifawcp/api/internal/test/mocks"
	"github.com/stretchr/testify/assert"
)

func newTestUserService(
	ur *mocks.MockUserRepository,
	us *mocks.MockUserStorage,
	logger *mocks.MockLogger,
) UserServiceInterface {
	return NewUserService(ur, us, logger)
}

// ---------------------------------------------------------------------------
// TestUserService_GetUser
// ---------------------------------------------------------------------------
func TestUserService_GetUser(t *testing.T) {
	t.Parallel()

	t.Run("returns user from storage on cache hit", func(t *testing.T) {
		t.Parallel()

		userID := gofakeit.UUID()
		expectedUser := &domain.User{
			ID:        userID,
			FirstName: gofakeit.FirstName(),
			LastName:  gofakeit.LastName(),
			Email:     gofakeit.Email(),
		}

		us := &mocks.MockUserStorage{
			GetUserFunc: func(ctx context.Context, uid string) (*domain.User, error) {
				assert.Equal(t, userID, uid)
				return expectedUser, nil
			},
		}

		logger := &mocks.MockLogger{}

		service := newTestUserService(nil, us, logger)

		result, err := service.GetUser(context.Background(), userID)

		assert.NoError(t, err)
		assert.Equal(t, expectedUser, result)
	})

	t.Run("returns user from repository on cache miss", func(t *testing.T) {
		t.Parallel()

		userID := gofakeit.UUID()
		expectedUser := &domain.User{
			ID:        userID,
			FirstName: gofakeit.FirstName(),
			LastName:  gofakeit.LastName(),
			Email:     gofakeit.Email(),
		}

		us := &mocks.MockUserStorage{
			GetUserFunc: func(ctx context.Context, uid string) (*domain.User, error) {
				return nil, nil
			},
			SetUserFunc: func(ctx context.Context, user *domain.User) error {
				assert.Equal(t, expectedUser, user)
				return nil
			},
		}

		ur := &mocks.MockUserRepository{
			GetUserByIDFunc: func(ctx context.Context, uid string) (*domain.User, error) {
				assert.Equal(t, userID, uid)
				return expectedUser, nil
			},
		}

		logger := &mocks.MockLogger{}

		service := newTestUserService(ur, us, logger)

		result, err := service.GetUser(context.Background(), userID)

		assert.NoError(t, err)
		assert.Equal(t, expectedUser, result)
	})

	t.Run("propagates user repository error", func(t *testing.T) {
		t.Parallel()

		userID := gofakeit.UUID()

		us := &mocks.MockUserStorage{
			GetUserFunc: func(ctx context.Context, uid string) (*domain.User, error) {
				return nil, nil
			},
		}

		ur := &mocks.MockUserRepository{
			GetUserByIDFunc: func(ctx context.Context, uid string) (*domain.User, error) {
				return nil, errors.New("database error")
			},
		}

		logger := &mocks.MockLogger{}

		service := newTestUserService(ur, us, logger)

		result, err := service.GetUser(context.Background(), userID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.Nil(t, result)
	})

	t.Run("logs storage error when cache miss but continues to repository", func(t *testing.T) {
		t.Parallel()

		userID := gofakeit.UUID()
		expectedUser := &domain.User{
			ID:        userID,
			FirstName: gofakeit.FirstName(),
			LastName:  gofakeit.LastName(),
			Email:     gofakeit.Email(),
		}

		var storageErrorLogged bool

		us := &mocks.MockUserStorage{
			GetUserFunc: func(ctx context.Context, uid string) (*domain.User, error) {
				return nil, errors.New("storage error")
			},
			SetUserFunc: func(ctx context.Context, user *domain.User) error {
				return nil
			},
		}

		ur := &mocks.MockUserRepository{
			GetUserByIDFunc: func(ctx context.Context, uid string) (*domain.User, error) {
				return expectedUser, nil
			},
		}

		logger := &mocks.MockLogger{
			ErrorFunc: func(msg string, keysAndValues ...any) {
				expectedMsg := fmt.Sprintf("failed to get user with ID: %s from storage", userID)
				assert.Equal(t, expectedMsg, msg)
				storageErrorLogged = true
			},
		}

		service := newTestUserService(ur, us, logger)

		result, err := service.GetUser(context.Background(), userID)

		assert.NoError(t, err)
		assert.Equal(t, expectedUser, result)
		assert.True(t, storageErrorLogged, "storage error should be logged")
	})

	t.Run("logs cache set error but returns user", func(t *testing.T) {
		t.Parallel()

		userID := gofakeit.UUID()
		expectedUser := &domain.User{
			ID:        userID,
			FirstName: gofakeit.FirstName(),
			LastName:  gofakeit.LastName(),
			Email:     gofakeit.Email(),
		}

		var cacheErrorLogged bool

		us := &mocks.MockUserStorage{
			GetUserFunc: func(ctx context.Context, uid string) (*domain.User, error) {
				return nil, nil
			},
			SetUserFunc: func(ctx context.Context, user *domain.User) error {
				return errors.New("cache error")
			},
		}

		ur := &mocks.MockUserRepository{
			GetUserByIDFunc: func(ctx context.Context, uid string) (*domain.User, error) {
				return expectedUser, nil
			},
		}

		logger := &mocks.MockLogger{
			ErrorFunc: func(msg string, keysAndValues ...any) {
				expectedMsg := fmt.Sprintf("failed to set user with ID: %s in cache", userID)
				assert.Equal(t, expectedMsg, msg)
				cacheErrorLogged = true
			},
		}

		service := newTestUserService(ur, us, logger)

		result, err := service.GetUser(context.Background(), userID)

		assert.NoError(t, err)
		assert.Equal(t, expectedUser, result)
		assert.True(t, cacheErrorLogged, "cache error should be logged")
	})
}

// ---------------------------------------------------------------------------
// TestUserService_UpdateUser
// ---------------------------------------------------------------------------
func TestUserService_UpdateUser(t *testing.T) {
	t.Parallel()

	t.Run("updates only provided fields and refreshes cache", func(t *testing.T) {
		t.Parallel()

		userID := gofakeit.UUID()
		newUsername := gofakeit.Username()
		updatedUser := &domain.User{ID: userID, Username: newUsername}

		var cacheRefreshed bool

		ur := &mocks.MockUserRepository{
			UpdateUserFunc: func(ctx context.Context, uid string, updates domain.UserUpdate) (*domain.User, error) {
				assert.Equal(t, userID, uid)
				assert.Nil(t, updates.FirstName)
				assert.Nil(t, updates.LastName)
				assert.NotNil(t, updates.Username)
				assert.Equal(t, newUsername, *updates.Username)
				return updatedUser, nil
			},
		}

		us := &mocks.MockUserStorage{
			SetUserFunc: func(ctx context.Context, user *domain.User) error {
				assert.Equal(t, updatedUser, user)
				cacheRefreshed = true
				return nil
			},
		}

		service := newTestUserService(ur, us, &mocks.MockLogger{})

		result, err := service.UpdateUser(context.Background(), userID, &dtos.UpdateUserDto{Username: &newUsername})

		assert.NoError(t, err)
		assert.Equal(t, updatedUser, result)
		assert.True(t, cacheRefreshed, "cache should be refreshed after update")
	})

	t.Run("propagates username conflict from repository", func(t *testing.T) {
		t.Parallel()

		newUsername := gofakeit.Username()

		ur := &mocks.MockUserRepository{
			UpdateUserFunc: func(ctx context.Context, uid string, updates domain.UserUpdate) (*domain.User, error) {
				return nil, domain.ErrUsernameAlreadyExists
			},
		}

		service := newTestUserService(ur, &mocks.MockUserStorage{}, &mocks.MockLogger{})

		result, err := service.UpdateUser(context.Background(), gofakeit.UUID(), &dtos.UpdateUserDto{Username: &newUsername})

		assert.ErrorIs(t, err, domain.ErrUsernameAlreadyExists)
		assert.Nil(t, result)
	})

	t.Run("returns user even when cache refresh fails", func(t *testing.T) {
		t.Parallel()

		userID := gofakeit.UUID()
		firstName := gofakeit.FirstName()
		updatedUser := &domain.User{ID: userID, FirstName: firstName}

		var cacheErrorLogged bool

		ur := &mocks.MockUserRepository{
			UpdateUserFunc: func(ctx context.Context, uid string, updates domain.UserUpdate) (*domain.User, error) {
				return updatedUser, nil
			},
		}

		us := &mocks.MockUserStorage{
			SetUserFunc: func(ctx context.Context, user *domain.User) error {
				return errors.New("cache error")
			},
		}

		logger := &mocks.MockLogger{
			ErrorFunc: func(msg string, keysAndValues ...any) {
				assert.Equal(t, fmt.Sprintf("failed to refresh cache for user with ID: %s", userID), msg)
				cacheErrorLogged = true
			},
		}

		service := newTestUserService(ur, us, logger)

		result, err := service.UpdateUser(context.Background(), userID, &dtos.UpdateUserDto{FirstName: &firstName})

		assert.NoError(t, err)
		assert.Equal(t, updatedUser, result)
		assert.True(t, cacheErrorLogged, "cache error should be logged")
	})
}
