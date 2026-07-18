package cmd

import (
	"github.com/asphaltbuffet/go-outside/internal/app"
	"github.com/asphaltbuffet/go-outside/internal/config"
)

// SetAppFactoryForTest overrides the App factory and returns a restore func.
func SetAppFactoryForTest(f func(cfg *config.Config) (*app.App, error)) func() {
	prev := appFactory
	appFactory = f
	return func() { appFactory = prev }
}
