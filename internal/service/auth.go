package service

import (
	"context"
	"errors"
	"time"

	"github.com/AlexeyKurlevsky/go-diploma/internal/models"
	"github.com/AlexeyKurlevsky/go-diploma/internal/storage"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Register(ctx context.Context, login, password string) (token string, userID uuid.UUID, err error)
	Login(ctx context.Context, login, password string) (token string, userID uuid.UUID, err error)
	ValidateToken(tokenString string) (userID uuid.UUID, err error)
}

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
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

func (s *authService) Register(ctx context.Context, login, password string) (string, uuid.UUID, error) {
	if login == "" || password == "" {
		return "", uuid.Nil, ErrInvalidCredentials
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", uuid.Nil, err
	}
	user := &models.User{
		Login:        login,
		PasswordHash: hash,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			return "", uuid.Nil, ErrLoginAlreadyTaken
		}
		return "", uuid.Nil, err
	}
	token, err := s.generateToken(user.ID)
	if err != nil {
		return "", uuid.Nil, err
	}
	return token, user.ID, nil
}

func (s *authService) Login(ctx context.Context, login, password string) (string, uuid.UUID, error) {
	user, err := s.userRepo.FindByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return "", uuid.Nil, ErrInvalidCredentials
		}
		return "", uuid.Nil, err
	}
	if err := bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(password)); err != nil {
		return "", uuid.Nil, ErrInvalidCredentials
	}
	token, err := s.generateToken(user.ID)
	if err != nil {
		return "", uuid.Nil, err
	}
	return token, user.ID, nil
}

func (s *authService) ValidateToken(tokenString string) (uuid.UUID, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return uuid.Nil, ErrInvalidToken
	}
	return claims.UserID, nil
}

func (s *authService) generateToken(userID uuid.UUID) (string, error) {
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
