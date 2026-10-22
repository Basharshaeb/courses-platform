// Package httpx holds HTTP helpers shared by all modules.
package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/teatchable/backend/internal/platform/apperr"
)

// Problem is an RFC 9457 problem details body.
type Problem struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Status    int    `json:"status"`
	Code      string `json:"code"`
	Detail    string `json:"detail,omitempty"`
	RequestID string `json:"requestId,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func WriteProblem(w http.ResponseWriter, r *http.Request, status int, code, detail string) {
	p := Problem{
		Type:      "about:blank",
		Title:     http.StatusText(status),
		Status:    status,
		Code:      code,
		Detail:    detail,
		RequestID: middleware.GetReqID(r.Context()),
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(p)
}

// WriteError renders err as a problem. Unexpected errors are logged and hidden from the client.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	if e, ok := apperr.As(err); ok {
		WriteProblem(w, r, e.Status, e.Code, e.Detail)
		return
	}
	slog.ErrorContext(r.Context(), "unhandled error", "err", err, "path", r.URL.Path, "request_id", middleware.GetReqID(r.Context()))
	WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "")
}
