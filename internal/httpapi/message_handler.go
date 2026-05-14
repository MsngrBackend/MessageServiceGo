package handler

import (
	"net/http"
	"strconv"
	"strings"

	"messageservice/internal/nats"
	"messageservice/internal/service"
)

type MessageHandler struct {
	svc  *service.MessageService
	nats *nats.Bus
}

func NewMessageHandler(svc *service.MessageService, bus *nats.Bus) *MessageHandler {
	return &MessageHandler{svc: svc, nats: bus}
}

func (h *MessageHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /messages/", h.createMessage)
	mux.HandleFunc("GET /messages/chats/", h.readMessages)
	mux.HandleFunc("PATCH /messages/{id}/", h.updateMessage)
	mux.HandleFunc("DELETE /messages/{id}/", h.deleteMessage)
}

// POST /messages/
// Body: {"chat_id": 1, "content": "hello", "sender_id": "abc"}
func (h *MessageHandler) createMessage(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ChatID   int64  `json:"chat_id"`
		Content  string `json:"content"`
		SenderID string `json:"sender_id"`
	}
	if err := decodeJSON(r, &body); err != nil || body.ChatID == 0 || body.Content == "" || body.SenderID == "" {
		writeJSON(w, http.StatusBadRequest, detail("chat_id, content, sender_id are required"))
		return
	}

	msg, err := h.svc.AddMessage(r.Context(), body.ChatID, body.Content, body.SenderID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, detail("failed to add message"))
		return
	}

	if h.nats != nil {
		_ = h.nats.NotifyChatEvent(r.Context(), body.ChatID, map[string]any{
			"type":      "message",
			"text":      body.Content,
			"sender_id": body.SenderID,
		})
	}

	writeJSON(w, http.StatusCreated, msg)
}

// GET /messages/chats/{chat_id}/messages/?limit=100&offset=0
func (h *MessageHandler) readMessages(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/messages/chats/"), "/")
	if len(parts) < 2 || parts[1] != "messages" {
		http.NotFound(w, r)
		return
	}

	chatID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, detail("invalid chat_id"))
		return
	}

	q := r.URL.Query()
	limit := queryInt(q.Get("limit"), 100)
	offset := queryInt(q.Get("offset"), 0)

	messages, err := h.svc.GetMessagesByChat(r.Context(), chatID, limit, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, detail("failed to get messages"))
		return
	}

	writeJSON(w, http.StatusOK, messages)
}

// PATCH /messages/{id}/
// Body: {"content": "new text"}
func (h *MessageHandler) updateMessage(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt64(r, "id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, detail("invalid message id"))
		return
	}

	var body struct {
		Content string `json:"content"`
	}
	if err := decodeJSON(r, &body); err != nil || body.Content == "" {
		writeJSON(w, http.StatusBadRequest, detail("content is required"))
		return
	}

	msg, err := h.svc.UpdateMessage(r.Context(), id, body.Content)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, detail("failed to update message"))
		return
	}
	if msg == nil {
		writeJSON(w, http.StatusNotFound, detail("message not found"))
		return
	}

	writeJSON(w, http.StatusOK, msg)
}

// DELETE /messages/{id}/
func (h *MessageHandler) deleteMessage(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt64(r, "id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, detail("invalid message id"))
		return
	}

	ok, err := h.svc.DeleteMessage(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, detail("failed to delete message"))
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, detail("message not found"))
		return
	}

	writeJSON(w, http.StatusOK, detail("message deleted"))
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
