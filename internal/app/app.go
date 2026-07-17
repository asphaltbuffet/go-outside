package app

import "github.com/asphaltbuffet/go-outside/internal/config"

// App is the business logic layer. It owns all domain operations.
type App struct {
	cfg *config.Config
}

// New creates an App backed by the given config.
func New(cfg *config.Config) (*App, error) {
	return &App{cfg: cfg}, nil
}
