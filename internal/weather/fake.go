package weather

import (
	"context"

	"github.com/asphaltbuffet/go-outside/internal/geo"
)

// FakeProvider is a hand-rolled Provider for tests.
type FakeProvider struct {
	Report Report
	Err    error
}

// Fetch returns the configured Report or error.
func (f FakeProvider) Fetch(_ context.Context, _ geo.Location) (Report, error) {
	return f.Report, f.Err
}
