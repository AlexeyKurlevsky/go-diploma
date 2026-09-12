package models

import (
	"context"
	"net/http"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

func (h *Handler) PingHandler(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
		return
	}
	ctx := r.Context()
	if err := h.db.Ping(ctx); err != nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
