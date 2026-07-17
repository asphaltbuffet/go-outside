package apptesting

import (
	"testing"

	"github.com/asphaltbuffet/go-outside/internal/app"
	"github.com/asphaltbuffet/go-outside/internal/config"
)

// OpenApp creates an App instance suitable for use in tests.
// It uses default config; extend as needed when the App gains real dependencies.
func OpenApp(t *testing.T) *app.App {
	t.Helper()

	cfg := config.GetDefaults()

	a, err := app.New(cfg)
	if err != nil {
		t.Fatalf("apptesting.OpenApp: %v", err)
	}

	return a
}
