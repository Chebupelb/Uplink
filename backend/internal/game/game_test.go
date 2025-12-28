package game

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"uplink/backend/internal/db"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/assert"
)

func setupTestGame(t *testing.T) (*Manager, *db.DB) {
	url := "postgres://user:pass@localhost:5432/uplink?sslmode=disable"
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		url = dbURL
	}

	dbConn, err := db.New(url, 5)
	if err != nil {
		t.Fatalf("не удалось подключиться к базе данных: %v", err)
	}

	manager := New(dbConn, nil)
	return manager, dbConn
}

// Подсчет рейтинга
func TestCalculateElo(t *testing.T) {
	tests := []struct {
		name     string
		ratingA  int
		ratingB  int
		score    float64
		expected int
	}{
		{
			name:     "equal_rating_win",
			ratingA:  1000,
			ratingB:  1000,
			score:    1.0,
			expected: 12,
		},
		{
			name:     "equal_rating_loss",
			ratingA:  1000,
			ratingB:  1000,
			score:    0.0,
			expected: -12,
		},
		{
			name:     "higher_rating_win",
			ratingA:  1200,
			ratingB:  1000,
			score:    1.0,
			expected: 6,
		},
		{
			name:     "lower_rating_win",
			ratingA:  1000,
			ratingB:  1200,
			score:    1.0,
			expected: 18,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateElo(tt.ratingA, tt.ratingB, tt.score)
			assert.Equal(t, tt.expected, result,
				"calculateElo(%d, %d, %.1f) = %d, expected %d",
				tt.ratingA, tt.ratingB, tt.score, result, tt.expected)
		})
	}
}

// Создание лобби
func TestCreateRoom(t *testing.T) {
	manager, _ := setupTestGame(t)
	defer manager.Shutdown()

	settings := Settings{
		Language:   "en",
		TextMode:   "standard",
		Category:   "general",
		MaxPlayers: 2,
	}
	roomID := manager.CreateRoom("test_owner", "matchmaking", settings)

	val, ok := manager.rooms.Load(roomID)
	assert.True(t, ok, "комната не найдена в менеджере")

	room := val.(*Room)
	assert.Equal(t, roomID, room.ID, "некорректный ID комнаты")
	assert.Equal(t, "test_owner", room.Owner, "некорректный владелец комнаты")
	assert.Equal(t, StateLobby, room.State, "некорректное состояние комнаты")
	assert.Equal(t, 2, room.Settings.MaxPlayers, "некорректное максимальное количество игроков")
}

// Процесс игры
func TestGameFlow(t *testing.T) {
	manager, db := setupTestGame(t)
	defer manager.Shutdown()

	ctx := context.Background()

	username := "test_game_user_" + time.Now().Format("20060102150405")
	userID, err := db.CreateUser(ctx, username, "$2a$10$N9qo8uLOickgx2ZMRZoMy.qC0Y4Y7DdDZ4JXv8e0kF3pQf5Lk7")
	assert.NoError(t, err)

	settings := Settings{
		Language:   "en",
		TextMode:   "standard",
		Category:   "general",
		MaxPlayers: 1,
	}
	roomID := manager.CreateRoom(userID, "solo", settings)

	val, ok := manager.rooms.Load(roomID)
	assert.True(t, ok, "комната не найдена")
	room := val.(*Room)

	client := &Client{
		ID:        userID,
		Username:  username,
		send:      make(chan any, 64),
		room:      room,
		lastInput: time.Now(),
		Progress:  0,
		Accuracy:  100,
		Ready:     false,
	}

	room.join(client)

	time.Sleep(100 * time.Millisecond)

	room.mu.RLock()
	assert.Equal(t, StateLobby, room.State, "комната должна быть в состоянии Lobby")
	room.mu.RUnlock()

	client.mu.Lock()
	client.Ready = true
	client.mu.Unlock()
	room.sendPlayers()

	// Запуск
	go room.startGame()

	time.Sleep(3500 * time.Millisecond)

	room.mu.RLock()
	assert.Equal(t, StateGame, room.State, "комната должна быть в состоянии Game")
	assert.NotNil(t, room.Text, "текст для игры не загружен")
	textLength := room.Text.Length
	room.mu.RUnlock()

	go func() {
		time.Sleep(500 * time.Millisecond)
		for i := 1; i <= textLength; i++ {
			room.input <- &inputMsg{c: client, idx: i}
			time.Sleep(5 * time.Millisecond)
		}
	}()

	finished := false
	for i := 0; i < 60; i++ {
		time.Sleep(100 * time.Millisecond)

		room.mu.RLock()
		currentState := room.State
		room.mu.RUnlock()

		client.mu.Lock()
		isFinished := client.Finished
		client.mu.Unlock()

		if currentState == StateFinished || isFinished {
			finished = true
			break
		}
	}

	assert.True(t, finished, "игра должна завершиться")

	room.mu.RLock()
	assert.Equal(t, StateFinished, room.State, "комната должна быть в состоянии Finished")
	room.mu.RUnlock()
}

// Веб сокет
func TestWebSocketIntegration(t *testing.T) {
	manager, _ := setupTestGame(t)
	defer manager.Shutdown()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ws" {
			manager.HandleWS(w, r, "test_user", "test_username")
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[4:] + "/ws"

	opts := &websocket.DialOptions{
		Subprotocols: []string{"echo"},
	}
	conn, _, err := websocket.Dial(context.Background(), wsURL, opts)
	assert.NoError(t, err, "ошибка подключения к WebSocket")
	defer conn.Close(websocket.StatusNormalClosure, "")

	initMsg := map[string]interface{}{
		"type": "join",
		"payload": map[string]interface{}{
			"mode":     "solo",
			"language": "en",
			"textMode": "standard",
			"category": "general",
		},
	}
	err = wsjson.Write(context.Background(), conn, initMsg)
	assert.NoError(t, err, "ошибка отправки инициализации")

	var response map[string]interface{}
	err = wsjson.Read(context.Background(), conn, &response)
	assert.NoError(t, err, "ошибка чтения ответа")
	assert.NotNil(t, response, "не получен ответ от сервера")

	assert.Contains(t, response, "type", "ответ должен содержать поле type")
}
