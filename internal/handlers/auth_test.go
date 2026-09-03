package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AlexeyKurlevsky/go-diploma/internal/service"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mock_service "github.com/AlexeyKurlevsky/go-diploma/internal/service/mocks"
)

func TestAuthHandler_Register(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthService := mock_service.NewMockAuthService(ctrl)
	handler := NewAuthHandler(mockAuthService)

	t.Run("успешная регистрация", func(t *testing.T) {
		userID := uuid.New()
		token := "test-token"
		mockAuthService.EXPECT().
			Register(gomock.Any(), "testuser", "password").
			Return(token, userID, nil)

		body := `{"login":"testuser","password":"password"}`
		req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Register(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Equal(t, token, resp["token"])
		assert.Equal(t, userID.String(), resp["user_id"])

		// Проверяем наличие cookie
		cookies := w.Result().Cookies()
		var cookieFound bool
		for _, c := range cookies {
			if c.Name == "token" && c.Value == token {
				cookieFound = true
				break
			}
		}
		assert.True(t, cookieFound, "cookie 'token' should be set with correct value")
	})

	t.Run("неверный формат запроса (не JSON)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewReader([]byte("invalid")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.Register(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("пустые логин или пароль", func(t *testing.T) {
		mockAuthService.EXPECT().
			Register(gomock.Any(), "", "").
			Return("", uuid.Nil, service.ErrInvalidCredentials)

		body := `{"login":"","password":""}`
		req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.Register(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("логин уже занят", func(t *testing.T) {
		mockAuthService.EXPECT().
			Register(gomock.Any(), "existing", "pass").
			Return("", uuid.Nil, service.ErrLoginAlreadyTaken)

		body := `{"login":"existing","password":"pass"}`
		req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.Register(w, req)
		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("внутренняя ошибка сервера", func(t *testing.T) {
		mockAuthService.EXPECT().
			Register(gomock.Any(), "user", "pass").
			Return("", uuid.Nil, errors.New("db error"))

		body := `{"login":"user","password":"pass"}`
		req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.Register(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestAuthHandler_Login(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthService := mock_service.NewMockAuthService(ctrl)
	handler := NewAuthHandler(mockAuthService)

	t.Run("успешный логин", func(t *testing.T) {
		userID := uuid.New()
		token := "test-token"
		mockAuthService.EXPECT().
			Login(gomock.Any(), "testuser", "password").
			Return(token, userID, nil)

		body := `{"login":"testuser","password":"password"}`
		req := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Login(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Equal(t, token, resp["token"])
		assert.Equal(t, userID.String(), resp["user_id"])

		// Проверяем наличие cookie
		cookies := w.Result().Cookies()
		var cookieFound bool
		for _, c := range cookies {
			if c.Name == "token" && c.Value == token {
				cookieFound = true
				break
			}
		}
		assert.True(t, cookieFound, "cookie 'token' should be set with correct value")
	})

	t.Run("неверный формат запроса", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewReader([]byte("invalid")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.Login(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("неверные учётные данные", func(t *testing.T) {
		mockAuthService.EXPECT().
			Login(gomock.Any(), "wrong", "pass").
			Return("", uuid.Nil, service.ErrInvalidCredentials)

		body := `{"login":"wrong","password":"pass"}`
		req := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.Login(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("внутренняя ошибка сервера", func(t *testing.T) {
		mockAuthService.EXPECT().
			Login(gomock.Any(), "user", "pass").
			Return("", uuid.Nil, errors.New("db error"))

		body := `{"login":"user","password":"pass"}`
		req := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.Login(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
