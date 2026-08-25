package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/AlexeyKurlevsky/go-diploma/internal/service"

	"github.com/AlexeyKurlevsky/go-diploma/internal/models"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token":   token,
		"user_id": userID.String(), // преобразуем UUID в строку
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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token":   token,
		"user_id": userID.String(),
	})
}
