package ws

import (
	"encoding/json"
	"log"
	"strconv"
	"sync"

	"golang.org/x/net/websocket"
)

// --- Типы событий и схемы ---

type EventType string

const (
	EventTypeMessage EventType = "message"
	EventTypeTyping  EventType = "typing"
)

type IncomingEvent struct {
	Type    EventType `json:"type"`
	ChatID  int       `json:"chat_id"`
	UserID  string    `json:"user_id"`
	Message string    `json:"message,omitempty"`
}

type OutgoingMessage struct {
	Text     string `json:"text,omitempty"`
	SenderID string `json:"sender_id"`
}

// --- ConnectionManager ---

// ConnectionManager хранит активные WebSocket-соединения в виде:
// map[chat_id]map[user_id]*websocket.Conn
type ConnectionManager struct {
	mu                sync.RWMutex
	activeConnections map[int]map[string]*websocket.Conn
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		activeConnections: make(map[int]map[string]*websocket.Conn),
	}
}

// Connect регистрирует новое соединение для пользователя в указанном чате.
func (cm *ConnectionManager) Connect(ws *websocket.Conn, chatID int, userID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, ok := cm.activeConnections[chatID]; !ok {
		cm.activeConnections[chatID] = make(map[string]*websocket.Conn)
	}
	cm.activeConnections[chatID][userID] = ws

	log.Printf("User %s connected to chat %d", userID, chatID)
}

// Disconnect удаляет соединение пользователя из чата.
// Если в чате больше нет участников — удаляет и чат.
func (cm *ConnectionManager) Disconnect(chatID int, userID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if users, ok := cm.activeConnections[chatID]; ok {
		if _, ok := users[userID]; ok {
			delete(users, userID)
			log.Printf("User %s disconnected from chat %d", userID, chatID)
		}
		if len(cm.activeConnections[chatID]) == 0 {
			delete(cm.activeConnections, chatID)
			log.Printf("Chat %d has no active connections, removed", chatID)
		}
	}
}

// BroadcastTyping рассылает всем участникам чата уведомление о наборе текста.
func (cm *ConnectionManager) BroadcastTyping(chatID int, senderID string) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	users, ok := cm.activeConnections[chatID]
	if !ok {
		return
	}

	payload := OutgoingMessage{SenderID: senderID}
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("BroadcastTyping marshal error: %v", err)
		return
	}

	for userID, conn := range users {
		if err := websocket.Message.Send(conn, string(data)); err != nil {
			log.Printf("BroadcastTyping send error to user %s: %v", userID, err)
		}
	}
}

// BroadcastMessage рассылает текстовое сообщение всем участникам чата.
func (cm *ConnectionManager) BroadcastMessage(message string, chatID int, senderID string) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	users, ok := cm.activeConnections[chatID]
	if !ok {
		return
	}

	payload := OutgoingMessage{
		Text:     message,
		SenderID: senderID,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("BroadcastMessage marshal error: %v", err)
		return
	}

	for userID, conn := range users {
		if err := websocket.Message.Send(conn, string(data)); err != nil {
			log.Printf("BroadcastMessage send error to user %s: %v", userID, err)
		}
	}
}

// --- HTTP Handler ---

var manager = NewConnectionManager()

// wsHandler обрабатывает входящие WebSocket-соединения.
// Ожидает query-параметры: chat_id и user_id.
func WsHandler(ws *websocket.Conn) {
	r := ws.Request()

	chatIDStr := r.URL.Query().Get("chat_id")
	userID := r.URL.Query().Get("user_id")

	chatID, err := strconv.Atoi(chatIDStr)
	if err != nil || userID == "" {
		log.Printf("Invalid chat_id or user_id: chat_id=%s, user_id=%s", chatIDStr, userID)
		return
	}

	manager.Connect(ws, chatID, userID)
	defer manager.Disconnect(chatID, userID)

	// Читаем входящие сообщения в цикле
	for {
		var raw string
		if err := websocket.Message.Receive(ws, &raw); err != nil {
			log.Printf("User %s disconnected from chat %d: %v", userID, chatID, err)
			break
		}

		var event IncomingEvent
		if err := json.Unmarshal([]byte(raw), &event); err != nil {
			log.Printf("Invalid event from user %s: %v", userID, err)
			continue
		}

		switch event.Type {
		case EventTypeMessage:
			manager.BroadcastMessage(event.Message, chatID, userID)
		case EventTypeTyping:
			manager.BroadcastTyping(chatID, userID)
		default:
			log.Printf("Unknown event type from user %s: %s", userID, event.Type)
		}
	}
}
