package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"messageservice/internal/service"
)

type ChatHandler struct {
	svc *service.ChatService
}

func NewChatHandler(svc *service.ChatService) *ChatHandler {
	return &ChatHandler{svc: svc}
}

func (h *ChatHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /chats/", h.listChats)
	mux.HandleFunc("POST /chats/", h.createChat)
	mux.HandleFunc("GET /chats/{id}/", h.getChat)
	mux.HandleFunc("DELETE /chats/{id}/", h.deleteChat)
	mux.HandleFunc("GET /chats/{id}/members/", h.listMembers)
	mux.HandleFunc("POST /chats/{id}/members/", h.addMember)
	mux.HandleFunc("DELETE /chats/{id}/members/{user_id}/", h.removeMember)
}

// GET /chats/
// Header: X-User-Id
func (h *ChatHandler) listChats(w http.ResponseWriter, r *http.Request) {
	userID, ok := userID(w, r)
	if !ok {
		return
	}

	chats, err := h.svc.ListUserChats(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, detail("failed to list chats"))
		return
	}
	writeJSON(w, http.StatusOK, chats)
}

// POST /chats/
// Header: X-User-Id
// Body: {"name": "..."}
func (h *ChatHandler) createChat(w http.ResponseWriter, r *http.Request) {
	userID, ok := userID(w, r)
	if !ok {
		return
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		writeJSON(w, http.StatusBadRequest, detail("name is required"))
		return
	}

	chat, err := h.svc.CreateChat(r.Context(), body.Name, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, detail("failed to create chat"))
		return
	}
	writeJSON(w, http.StatusCreated, chat)
}

// GET /chats/{id}/
// Header: X-User-Id
func (h *ChatHandler) getChat(w http.ResponseWriter, r *http.Request) {
	userID, ok := userID(w, r)
	if !ok {
		return
	}

	id, err := pathInt64(r, "id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, detail("invalid chat id"))
		return
	}

	chat, err := h.svc.GetChat(r.Context(), id, userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotFound):
			writeJSON(w, http.StatusNotFound, detail(err.Error()))
		case errors.Is(err, service.ErrNotMember):
			writeJSON(w, http.StatusForbidden, detail(err.Error()))
		default:
			writeJSON(w, http.StatusInternalServerError, detail("failed to get chat"))
		}
		return
	}
	writeJSON(w, http.StatusOK, chat)
}

// DELETE /chats/{id}/
// Header: X-User-Id
func (h *ChatHandler) deleteChat(w http.ResponseWriter, r *http.Request) {
	userID, ok := userID(w, r)
	if !ok {
		return
	}

	id, err := pathInt64(r, "id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, detail("invalid chat id"))
		return
	}

	if err := h.svc.DeleteChat(r.Context(), id, userID); err != nil {
		switch {
		case errors.Is(err, service.ErrNotFound):
			writeJSON(w, http.StatusNotFound, detail(err.Error()))
		case errors.Is(err, service.ErrNotMember):
			writeJSON(w, http.StatusForbidden, detail(err.Error()))
		default:
			writeJSON(w, http.StatusInternalServerError, detail("failed to delete chat"))
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /chats/{id}/members/
func (h *ChatHandler) listMembers(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt64(r, "id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, detail("invalid chat id"))
		return
	}

	members, err := h.svc.ListMembers(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, detail("failed to list members"))
		return
	}
	writeJSON(w, http.StatusOK, members)
}

// POST /chats/{id}/members/
// Body: {"user_id": "..."}
func (h *ChatHandler) addMember(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt64(r, "id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, detail("invalid chat id"))
		return
	}

	var body struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.UserID == "" {
		writeJSON(w, http.StatusBadRequest, detail("user_id is required"))
		return
	}

	member, err := h.svc.AddMember(r.Context(), id, body.UserID)
	if err != nil {
		if errors.Is(err, service.ErrAlreadyMember) {
			writeJSON(w, http.StatusBadRequest, detail(err.Error()))
			return
		}
		writeJSON(w, http.StatusInternalServerError, detail("failed to add member"))
		return
	}
	writeJSON(w, http.StatusCreated, member)
}

// DELETE /chats/{id}/members/{user_id}/
// Header: X-User-Id
func (h *ChatHandler) removeMember(w http.ResponseWriter, r *http.Request) {
	requesterID, ok := userID(w, r)
	if !ok {
		return
	}

	chatID, err := pathInt64(r, "id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, detail("invalid chat id"))
		return
	}

	targetUserID := r.PathValue("user_id")

	if err := h.svc.RemoveMember(r.Context(), chatID, targetUserID, requesterID); err != nil {
		switch {
		case errors.Is(err, service.ErrNotFound):
			writeJSON(w, http.StatusNotFound, detail(err.Error()))
		case errors.Is(err, service.ErrNotMember):
			writeJSON(w, http.StatusForbidden, detail(err.Error()))
		default:
			writeJSON(w, http.StatusInternalServerError, detail("failed to remove member"))
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- helpers ---

func userID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := r.Header.Get("X-User-Id")
	if id == "" {
		writeJSON(w, http.StatusUnauthorized, detail("X-User-Id header is required"))
		return "", false
	}
	return id, true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func detail(msg string) map[string]string {
	return map[string]string{"detail": msg}
}

func pathInt64(r *http.Request, key string) (int64, error) {
	return strconv.ParseInt(r.PathValue(key), 10, 64)
}
