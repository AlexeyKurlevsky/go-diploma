package service

import (
	"context"
	"errors"
	"time"

	"github.com/AlexeyKurlevsky/go-diploma/internal/models"
	"github.com/AlexeyKurlevsky/go-diploma/internal/storage"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthService — интерфейс для работы с аутентификацией
type AuthService interface {
	Register(ctx context.Context, login, password string) (token string, userID int64, err error)
	Login(ctx context.Context, login, password string) (token string, userID int64, err error)
	ValidateToken(tokenString string) (userID int64, err error)
}

// Кастомные claims с полем user_id
type Claims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

type authService struct {
	userRepo    storage.UserRepository
	jwtSecret   []byte
	tokenExpire time.Duration
}

func NewAuthService(userRepo storage.UserRepository, jwtSecret string, tokenExpire time.Duration) AuthService {
	return &authService{
		userRepo:    userRepo,
		jwtSecret:   []byte(jwtSecret),
		tokenExpire: tokenExpire,
	}
}

// Register — регистрация нового пользователя
func (s *authService) Register(ctx context.Context, login, password string) (string, int64, error) {
	if login == "" || password == "" {
		return "", 0, ErrInvalidCredentials
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", 0, err
	}

	user := &models.User{
		Login:        login,
		PasswordHash: hash,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			return "", 0, ErrLoginAlreadyTaken
		}
		return "", 0, err
	}

	token, err := s.generateToken(user.ID)
	if err != nil {
		return "", 0, err
	}

	return token, user.ID, nil
}

// Login — аутентификация пользователя
func (s *authService) Login(ctx context.Context, login, password string) (string, int64, error) {
	user, err := s.userRepo.FindByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return "", 0, ErrInvalidCredentials
		}
		return "", 0, err
	}

	if err := bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(password)); err != nil {
		return "", 0, ErrInvalidCredentials
	}

	token, err := s.generateToken(user.ID)
	if err != nil {
		return "", 0, err
	}

	return token, user.ID, nil
}

// ValidateToken — проверка JWT и извлечение userID
func (s *authService) ValidateToken(tokenString string) (int64, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})
	if err != nil {
		return 0, ErrInvalidToken
	}
	if !token.Valid {
		return 0, ErrInvalidToken
	}
	return claims.UserID, nil
}

// generateToken — создание нового JWT
func (s *authService) generateToken(userID int64) (string, error) {
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.tokenExpire)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}
