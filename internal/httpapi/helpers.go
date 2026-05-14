package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
)

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

func decodeJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func userID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := r.Header.Get("X-User-Id")
	if id == "" {
		writeJSON(w, http.StatusUnauthorized, detail("X-User-Id header is required"))
		return "", false
	}
	return id, true
}
