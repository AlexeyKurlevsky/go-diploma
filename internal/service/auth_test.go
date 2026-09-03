package service

import (
	"context"
	"testing"
	"time"

	"github.com/AlexeyKurlevsky/go-diploma/internal/models"
	"github.com/AlexeyKurlevsky/go-diploma/internal/storage"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mock_storage "github.com/AlexeyKurlevsky/go-diploma/internal/storage/mocks"
)

func TestAuthService_Register(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mock_storage.NewMockUserRepository(ctrl)
	authService := NewAuthService(mockUserRepo, "testsecret", 24*time.Hour)

	ctx := context.Background()

	t.Run("successful registration", func(t *testing.T) {
		mockUserRepo.EXPECT().
			Create(ctx, gomock.Any()).
			DoAndReturn(func(_ context.Context, user *models.User) error {
				user.ID = uuid.New()
				return nil
			})

		token, userID, err := authService.Register(ctx, "testuser", "password123")
		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.NotEqual(t, uuid.Nil, userID)
	})

	t.Run("duplicate login", func(t *testing.T) {
		mockUserRepo.EXPECT().
			Create(ctx, gomock.Any()).
			Return(storage.ErrUserExists)

		token, userID, err := authService.Register(ctx, "testuser", "password123")
		assert.ErrorIs(t, err, ErrLoginAlreadyTaken)
		assert.Empty(t, token)
		assert.Equal(t, uuid.Nil, userID)
	})

	t.Run("empty login/password", func(t *testing.T) {
		token, userID, err := authService.Register(ctx, "", "")
		assert.ErrorIs(t, err, ErrInvalidCredentials)
		assert.Empty(t, token)
		assert.Equal(t, uuid.Nil, userID)
	})
}

func TestAuthService_Login(t *testing.T) {
	// аналогично с моками FindByLogin и проверкой пароля
}
