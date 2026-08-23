package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/AlexeyKurlevsky/go-diploma/internal/models"
	"github.com/AlexeyKurlevsky/go-diploma/internal/service"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Register обрабатывает POST /api/user/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var creds models.Credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	token, userID, err := h.authService.Register(r.Context(), creds.Login, creds.Password)
	if err != nil {
		switch err {
		case service.ErrInvalidCredentials:
			http.Error(w, "login and password are required", http.StatusBadRequest)
		case service.ErrLoginAlreadyTaken:
			http.Error(w, "login already taken", http.StatusConflict)
		default:
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Успешная регистрация – возвращаем токен
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token":   token,
		"user_id": userID,
	})
}

// Login обрабатывает POST /api/user/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var creds models.Credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	token, userID, err := h.authService.Login(r.Context(), creds.Login, creds.Password)
	if err != nil {
		switch err {
		case service.ErrInvalidCredentials:
			http.Error(w, "invalid login or password", http.StatusUnauthorized)
		default:
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token":   token,
		"user_id": userID,
	})
}
