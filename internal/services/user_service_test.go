package services

import (
	"context"
	"errors"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/ncondes/fifawcp/internal/domain"
	"github.com/ncondes/fifawcp/internal/packages/mocks"
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
				if msg == "failed to get user from storage" {
					storageErrorLogged = true
				}
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
				if msg == "failed to cache user" {
					cacheErrorLogged = true
				}
			},
		}

		service := newTestUserService(ur, us, logger)

		result, err := service.GetUser(context.Background(), userID)

		assert.NoError(t, err)
		assert.Equal(t, expectedUser, result)
		assert.True(t, cacheErrorLogged, "cache error should be logged")
	})
}
