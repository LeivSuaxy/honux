package utils

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
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

func ExtractPathUUID(r *http.Request, key string) (*uuid.UUID, error) {
	idStr := strings.TrimPrefix(r.URL.Path, key) // TODO Missing check key
	id, err := uuid.Parse(idStr)

	if err != nil {
		return nil, fmt.Errorf("UUID not valid")
	}

	return &id, nil
}
