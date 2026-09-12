package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/AlexeyKurlevsky/go-diploma/internal/models"
	"github.com/AlexeyKurlevsky/go-diploma/internal/service"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func setAuthCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		HttpOnly: true,
		Secure:   false, // для локальной разработки (если не HTTPS)
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		MaxAge:   int(24 * time.Hour / time.Second), // срок жизни как в JWT
	})
}

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
			// логируем ошибку
			log.Printf("registration error: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}
	// Устанавливаем cookie
	setAuthCookie(w, token)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token":   token,
		"user_id": userID.String(),
	})
}

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
			log.Printf("login error: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}
	setAuthCookie(w, token)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token":   token,
		"user_id": userID.String(),
	})
}
