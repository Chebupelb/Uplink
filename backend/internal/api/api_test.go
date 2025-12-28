package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"uplink/backend/internal/db"
	"uplink/backend/internal/game"

	"log/slog"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestAPI(t *testing.T) (*httptest.Server, *db.DB) {
	url := "postgres://user:pass@localhost:5432/uplink?sslmode=disable"
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		url = dbURL
	}

	dbConn, err := db.New(url, 5)
	require.NoError(t, err)

	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	gameManager := game.New(dbConn, log)

	origins := []string{"*", "http://localhost:3000"}
	api := New(dbConn, gameManager, "test_secret", origins, log)

	server := httptest.NewServer(api)
	return server, dbConn
}

// Регистрация, вход
func TestRegisterAndLogin(t *testing.T) {
	server, db := setupTestAPI(t)
	defer server.Close()
	defer db.Close()

	username := "test_user_" + time.Now().Format("20060102150405")

	// Рег
	registerReq := map[string]string{
		"Username": username,
		"Password": "password123",
	}
	regBody, _ := json.Marshal(registerReq)
	regResp, err := http.Post(server.URL+"/api/v1/auth/register", "application/json", bytes.NewBuffer(regBody))
	require.NoError(t, err)
	defer regResp.Body.Close()

	assert.Equal(t, http.StatusOK, regResp.StatusCode)
	var regResult map[string]string
	require.NoError(t, json.NewDecoder(regResp.Body).Decode(&regResult))
	assert.Contains(t, regResult, "token")

	// Вход
	loginReq := map[string]string{
		"Username": username,
		"Password": "password123",
	}
	loginBody, _ := json.Marshal(loginReq)
	loginResp, err := http.Post(server.URL+"/api/v1/auth/login", "application/json", bytes.NewBuffer(loginBody))
	require.NoError(t, err)
	defer loginResp.Body.Close()

	assert.Equal(t, http.StatusOK, loginResp.StatusCode)
	var loginResult map[string]string
	require.NoError(t, json.NewDecoder(loginResp.Body).Decode(&loginResult))
	assert.Contains(t, loginResult, "token")
}

// Авторизация
func TestAuthMiddleware(t *testing.T) {
	server, db := setupTestAPI(t)
	defer server.Close()
	defer db.Close()

	ctx := context.Background()
	username := "auth_test_user_" + time.Now().Format("20060102150405")
	userID, err := db.CreateUser(ctx, username, "$2a$10$N9qo8uLOickgx2ZMRZoMy.qC0Y4Y7DdDZ4JXv8e0kF3pQf5Lk7")
	require.NoError(t, err)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":      userID,
		"username": username,
		"exp":      time.Now().Add(time.Hour).Unix(),
	})
	validToken, _ := token.SignedString([]byte("test_secret"))

	req, _ := http.NewRequest("GET", server.URL+"/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// Категории
func TestGetCategories(t *testing.T) {
	server, db := setupTestAPI(t)
	defer server.Close()
	defer db.Close()

	ctx := context.Background()
	_, err := db.CreateUser(ctx, "temp_user_"+time.Now().Format("20060102150405"), "$2a$10$N9qo8uLOickgx2ZMRZoMy.qC0Y4Y7DdDZ4JXv8e0kF3pQf5Lk7")
	require.NoError(t, err)

	req, _ := http.NewRequest("GET", server.URL+"/api/v1/categories", nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var result map[string][]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	categories := result["categories"]

	hasGeneral := false
	for _, cat := range categories {
		if cat == "general" {
			hasGeneral = true
			break
		}
	}
	assert.True(t, hasGeneral, "должна быть категория 'general'")
}

// CORS
func TestCorsHeaders(t *testing.T) {
	server, db := setupTestAPI(t)
	defer server.Close()
	defer db.Close()

	req, _ := http.NewRequest("GET", server.URL+"/api/v1/leaderboard", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "http://localhost:3000", resp.Header.Get("Access-Control-Allow-Origin"))
}
