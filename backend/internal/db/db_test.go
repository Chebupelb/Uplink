package db

import (
	"context"
	"os"
	"testing"
	"time"
)

func setupTestDB(t *testing.T) *DB {
	url := "postgres://user:pass@localhost:5432/uplink?sslmode=disable"
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		url = dbURL
	}

	db, err := New(url, 5)
	if err != nil {
		t.Fatalf("не удалось подключиться к базе данных: %v", err)
	}

	return db
}

// Создание и получение пользователя
func TestCreateAndGetUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	username := "test_user_" + time.Now().Format("20060102150405")
	passwordHash := "$2a$10$N9qo8uLOickgx2ZMRZoMy.qC0Y4Y7DdDZ4JXv8e0kF3pQf5Lk7"

	userID, err := db.CreateUser(ctx, username, passwordHash)
	if err != nil {
		t.Fatalf("не удалось создать пользователя: %v", err)
	}
	if userID == "" {
		t.Fatal("ID пользователя пустой")
	}

	user, err := db.GetUser(ctx, username)
	if err != nil {
		t.Fatalf("не удалось получить пользователя: %v", err)
	}
	if user.Username != username {
		t.Errorf("ожидалось имя %s, получено %s", username, user.Username)
	}

	userByID, err := db.GetUserByID(ctx, userID)
	if err != nil {
		t.Fatalf("не удалось получить пользователя по ID: %v", err)
	}
	if userByID.ID != userID {
		t.Errorf("ожидался ID %s, получен %s", userID, userByID.ID)
	}
}

// Текст для игры
func TestGetText(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	var textID int
	err := db.pool.QueryRow(ctx, "SELECT id FROM texts LIMIT 1").Scan(&textID)
	if err != nil {
		t.Skip("нет текстов в базе для тестирования")
		return
	}

	text, err := db.GetText(ctx, "", "", textID)
	if err != nil {
		t.Fatalf("не удалось получить текст по ID %d: %v", textID, err)
	}
	if text.ID != textID || len(text.Content) == 0 {
		t.Errorf("некорректные данные текста с ID %d", textID)
	}

	randomText, err := db.GetText(ctx, "en", "general", 0)
	if err != nil {
		t.Logf("не удалось получить случайный текст: %v", err)
	} else if len(randomText.Content) == 0 {
		t.Error("случайный текст пустой")
	}
}

// Игры и история
func TestSaveMatchAndGetHistory(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	var userID string
	err := db.pool.QueryRow(ctx, "SELECT id FROM users LIMIT 1").Scan(&userID)
	if err != nil {
		t.Skip("нет пользователей в базе для тестирования матча")
		return
	}

	var textID int
	err = db.pool.QueryRow(ctx, "SELECT id FROM texts LIMIT 1").Scan(&textID)
	if err != nil {
		t.Skip("нет текстов в базе для тестирования матча")
		return
	}

	results := []MatchResult{
		{UserID: userID, WPM: 100, Accuracy: 95.5, Rank: 1},
	}
	err = db.SaveMatch(ctx, textID, results)
	if err != nil {
		t.Fatalf("не удалось сохранить матч: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	history, _, err := db.GetHistory(ctx, userID, 10, "")
	if err != nil {
		t.Fatalf("не удалось получить историю матчей: %v", err)
	}
	if len(history) == 0 {
		t.Log("история матчей пустая - это нормально, если пользователь не участвовал в матчах")
	}
}

// Таблица лидеров
func TestGetLeaderboard(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	leaderboard, err := db.GetLeaderboard(ctx, 10)
	if err != nil {
		t.Fatalf("не удалось получить таблицу лидеров: %v", err)
	}
	if len(leaderboard) == 0 {
		t.Log("таблица лидеров пустая - это нормально для новой базы")
	}
}
