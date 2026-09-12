package models

import (
	"github.com/AlexeyKurlevsky/go-diploma/internal/config"
)

type Handler struct {
	cfg *config.Config
	db  Pinger
}

func NewHandler(cfg *config.Config, db Pinger) *Handler {
	h := &Handler{
		cfg: cfg,
		db:  db,
	}
	return h
}
