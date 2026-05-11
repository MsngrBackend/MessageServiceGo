package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"yourmodule/db"
	"yourmodule/nats_bus"
	"yourmodule/schemas"
	"yourmodule/service"
)

// MessageHandler регистрирует маршруты для работы с сообщениями.
type MessageHandler struct {
	mux *http.ServeMux
}

// NewMessageHandler создаёт MessageHandler и регистрирует все маршруты на переданном mux.
// Если mux == nil, создаётся новый.
func NewMessageHandler(mux *http.ServeMux) *MessageHandler {
	if mux == nil {
		mux = http.NewServeMux()
	}
	h := &MessageHandler{mux: mux}
	h.setupRoutes()
	return h
}

// Mux возвращает http.ServeMux с зарегистрированными маршрутами.
func (h *MessageHandler) Mux() *http.ServeMux {
	return h.mux
}

func (h *MessageHandler) setupRoutes() {
	// POST /messages/
	h.mux.HandleFunc("POST /messages/", h.createMessage)

	// GET /messages/chats/{chat_id}/messages/
	h.mux.HandleFunc("GET /messages/chats/", h.readMessages)

	// PATCH /messages/{message_id}/
	h.mux.HandleFunc("PATCH /messages/", h.updateMessage)

	// DELETE /messages/{message_id}/
	h.mux.HandleFunc("DELETE /messages/", h.deleteMessage)
}

// POST /messages/?chat_id=1&content=hello&sender_id=abc
func (h *MessageHandler) createMessage(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	chatID, err := strconv.Atoi(q.Get("chat_id"))
	if err != nil {
		http.Error(w, `{"detail":"invalid chat_id"}`, http.StatusBadRequest)
		return
	}
	content := q.Get("content")
	senderID := q.Get("sender_id")

	database, err := db.GetDB(r.Context())
	if err != nil {
		http.Error(w, `{"detail":"db error"}`, http.StatusInternalServerError)
		return
	}

	message, err := service.AddMessage(r.Context(), database, chatID, content, senderID)
	if err != nil {
		http.Error(w, `{"detail":"failed to add message"}`, http.StatusInternalServerError)
		return
	}

	if err := nats_bus.NotifyChatEvent(r.Context(), chatID, map[string]any{
		"type":      schemas.EventTypeMessage,
		"text":      content,
		"sender_id": senderID,
	}); err != nil {
		// логируем, но не прерываем ответ
		_ = err
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":      message.ID,
		"content": message.Content,
	})
}

// GET /messages/chats/{chat_id}/messages/?limit=100&offset=0
func (h *MessageHandler) readMessages(w http.ResponseWriter, r *http.Request) {
	// Путь: /messages/chats/{chat_id}/messages/
	// Вырезаем chat_id из URL вручную
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/messages/chats/"), "/")
	if len(parts) < 2 || parts[1] != "messages" {
		http.NotFound(w, r)
		return
	}

	chatID, err := strconv.Atoi(parts[0])
	if err != nil {
		http.Error(w, `{"detail":"invalid chat_id"}`, http.StatusBadRequest)
		return
	}

	q := r.URL.Query()
	limit := queryInt(q.Get("limit"), 100)
	offset := queryInt(q.Get("offset"), 0)

	database, err := db.GetDB(r.Context())
	if err != nil {
		http.Error(w, `{"detail":"db error"}`, http.StatusInternalServerError)
		return
	}

	messages, err := service.GetMessagesByChat(r.Context(), database, chatID, limit, offset)
	if err != nil {
		http.Error(w, `{"detail":"failed to get messages"}`, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, messages)
}

// PATCH /messages/{message_id}/?content=new_text
func (h *MessageHandler) updateMessage(w http.ResponseWriter, r *http.Request) {
	messageID, err := parseIDFromPath(r.URL.Path, "/messages/")
	if err != nil {
		http.Error(w, `{"detail":"invalid message_id"}`, http.StatusBadRequest)
		return
	}

	content := r.URL.Query().Get("content")

	database, err := db.GetDB(r.Context())
	if err != nil {
		http.Error(w, `{"detail":"db error"}`, http.StatusInternalServerError)
		return
	}

	message, err := service.UpdateMessage(r.Context(), database, messageID, content)
	if err != nil {
		http.Error(w, `{"detail":"failed to update message"}`, http.StatusInternalServerError)
		return
	}
	if message == nil {
		http.Error(w, `{"detail":"Message not found"}`, http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":      message.ID,
		"content": message.Content,
	})
}

// DELETE /messages/{message_id}/
func (h *MessageHandler) deleteMessage(w http.ResponseWriter, r *http.Request) {
	messageID, err := parseIDFromPath(r.URL.Path, "/messages/")
	if err != nil {
		http.Error(w, `{"detail":"invalid message_id"}`, http.StatusBadRequest)
		return
	}

	database, err := db.GetDB(r.Context())
	if err != nil {
		http.Error(w, `{"detail":"db error"}`, http.StatusInternalServerError)
		return
	}

	ok, err := service.DeleteMessage(r.Context(), database, messageID)
	if err != nil {
		http.Error(w, `{"detail":"failed to delete message"}`, http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, `{"detail":"Message not found"}`, http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"detail": "Message deleted"})
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// parseIDFromPath извлекает числовой ID из пути вида /prefix/{id}/
func parseIDFromPath(path, prefix string) (int, error) {
	trimmed := strings.TrimPrefix(path, prefix)
	trimmed = strings.TrimSuffix(trimmed, "/")
	return strconv.Atoi(trimmed)
}

func queryInt(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}
