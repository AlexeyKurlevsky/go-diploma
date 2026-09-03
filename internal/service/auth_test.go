package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/AlexeyKurlevsky/go-diploma/internal/models"
	"github.com/AlexeyKurlevsky/go-diploma/internal/storage"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"

	mock_storage "github.com/AlexeyKurlevsky/go-diploma/internal/storage/mocks"
)

func TestAuthService_Register(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mock_storage.NewMockUserRepository(ctrl)
	svc := NewAuthService(mockUserRepo, "testsecret", 24*time.Hour)
	ctx := context.Background()

	t.Run("успешная регистрация", func(t *testing.T) {
		mockUserRepo.EXPECT().
			Create(ctx, gomock.Any()).
			DoAndReturn(func(_ context.Context, user *models.User) error {
				user.ID = uuid.New()
				return nil
			})

		token, userID, err := svc.Register(ctx, "testuser", "password123")
		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.NotEqual(t, uuid.Nil, userID)
	})

	t.Run("логин уже занят", func(t *testing.T) {
		mockUserRepo.EXPECT().
			Create(ctx, gomock.Any()).
			Return(storage.ErrUserExists)

		token, userID, err := svc.Register(ctx, "existing", "pass")
		assert.ErrorIs(t, err, ErrLoginAlreadyTaken)
		assert.Empty(t, token)
		assert.Equal(t, uuid.Nil, userID)
	})

	t.Run("пустой логин или пароль", func(t *testing.T) {
		token, userID, err := svc.Register(ctx, "", "")
		assert.ErrorIs(t, err, ErrInvalidCredentials)
		assert.Empty(t, token)
		assert.Equal(t, uuid.Nil, userID)

		token, userID, err = svc.Register(ctx, "login", "")
		assert.ErrorIs(t, err, ErrInvalidCredentials)
		assert.Empty(t, token)
		assert.Equal(t, uuid.Nil, userID)

		token, userID, err = svc.Register(ctx, "", "pass")
		assert.ErrorIs(t, err, ErrInvalidCredentials)
		assert.Empty(t, token)
		assert.Equal(t, uuid.Nil, userID)
	})

	t.Run("ошибка репозитория (не связанная с уникальностью)", func(t *testing.T) {
		mockUserRepo.EXPECT().
			Create(ctx, gomock.Any()).
			Return(errors.New("db connection lost"))

		token, userID, err := svc.Register(ctx, "user", "pass")
		assert.Error(t, err)
		assert.Empty(t, token)
		assert.Equal(t, uuid.Nil, userID)
	})
}

func TestAuthService_Login(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mock_storage.NewMockUserRepository(ctrl)
	svc := NewAuthService(mockUserRepo, "testsecret", 24*time.Hour)
	ctx := context.Background()

	password := "password123"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	userID := uuid.New()
	existingUser := &models.User{
		ID:           userID,
		Login:        "testuser",
		PasswordHash: hash,
	}

	t.Run("успешный логин", func(t *testing.T) {
		mockUserRepo.EXPECT().
			FindByLogin(ctx, "testuser").
			Return(existingUser, nil)

		token, returnedUserID, err := svc.Login(ctx, "testuser", password)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.Equal(t, userID, returnedUserID)
	})

	t.Run("неверный пароль", func(t *testing.T) {
		mockUserRepo.EXPECT().
			FindByLogin(ctx, "testuser").
			Return(existingUser, nil)

		token, userID, err := svc.Login(ctx, "testuser", "wrongpassword")
		assert.ErrorIs(t, err, ErrInvalidCredentials)
		assert.Empty(t, token)
		assert.Equal(t, uuid.Nil, userID)
	})

	t.Run("пользователь не найден", func(t *testing.T) {
		mockUserRepo.EXPECT().
			FindByLogin(ctx, "unknown").
			Return(nil, storage.ErrUserNotFound)

		token, userID, err := svc.Login(ctx, "unknown", "pass")
		assert.ErrorIs(t, err, ErrInvalidCredentials)
		assert.Empty(t, token)
		assert.Equal(t, uuid.Nil, userID)
	})

	t.Run("ошибка репозитория при поиске", func(t *testing.T) {
		mockUserRepo.EXPECT().
			FindByLogin(ctx, "user").
			Return(nil, errors.New("db down"))

		token, userID, err := svc.Login(ctx, "user", "pass")
		assert.Error(t, err)
		assert.Empty(t, token)
		assert.Equal(t, uuid.Nil, userID)
	})
}

func TestAuthService_ValidateToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mock_storage.NewMockUserRepository(ctrl)
	svc := NewAuthService(mockUserRepo, "testsecret", 24*time.Hour)

	userID := uuid.New()
	// Приводим к конкретному типу для доступа к приватному методу
	tokenString, err := svc.(*authService).generateToken(userID)
	require.NoError(t, err)

	t.Run("валидный токен", func(t *testing.T) {
		returnedUserID, err := svc.ValidateToken(tokenString)
		assert.NoError(t, err)
		assert.Equal(t, userID, returnedUserID)
	})

	t.Run("невалидный токен (поддельный)", func(t *testing.T) {
		fakeToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
		_, err := svc.ValidateToken(fakeToken)
		assert.ErrorIs(t, err, ErrInvalidToken)
	})

	t.Run("просроченный токен", func(t *testing.T) {
		shortLivedService := NewAuthService(mockUserRepo, "testsecret", 1*time.Millisecond)
		shortToken, _ := shortLivedService.(*authService).generateToken(userID)
		time.Sleep(2 * time.Millisecond)
		_, err := shortLivedService.ValidateToken(shortToken)
		assert.ErrorIs(t, err, ErrInvalidToken)
	})
}

func TestAuthService_GenerateToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mock_storage.NewMockUserRepository(ctrl)
	svc := NewAuthService(mockUserRepo, "testsecret", 24*time.Hour)

	t.Run("генерация токена с валидными claims", func(t *testing.T) {
		userID := uuid.New()
		token, err := svc.(*authService).generateToken(userID)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
	})
}
