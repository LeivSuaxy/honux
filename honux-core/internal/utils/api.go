package utils

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
)

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("WriteJSON encode error", "error", err)
	}
}

func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]string{"error": message})
}

func ExtractPathID(r *http.Request, key string) (int64, error) {
	s := r.PathValue(key)
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %q", key, s)
	}
	return id, nil
}
