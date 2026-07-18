package app

import (
	"context"
	"fmt"

	"github.com/asphaltbuffet/go-outside/internal/config"
	"github.com/asphaltbuffet/go-outside/internal/geo"
	"github.com/asphaltbuffet/go-outside/internal/score"
	"github.com/asphaltbuffet/go-outside/internal/weather"
)

// App is the business logic layer. It owns all domain operations.
type App struct {
	cfg      *config.Config
	locator  geo.Locator
	geocoder geo.Geocoder
	provider weather.Provider
}

// Assessment is the result of assessing a Location: the Verdict, the Score
// behind it, and the Location that was used.
type Assessment struct {
	Verdict  score.Verdict
	Score    int
	Location geo.Location
	Report   weather.Report
}

// New creates an App backed by the given config and providers.
func New(cfg *config.Config, locator geo.Locator, geocoder geo.Geocoder, provider weather.Provider) (*App, error) {
	return &App{cfg: cfg, locator: locator, geocoder: geocoder, provider: provider}, nil
}

// Assess resolves the Location (auto-detect when override is empty, otherwise
// geocode the override), fetches Conditions, and returns the Verdict. Any
// failure to determine Location or Conditions is returned as an error and must
// never be collapsed into a Nope Verdict (ADR-0002).
func (a *App) Assess(ctx context.Context, override string) (Assessment, error) {
	var loc geo.Location
	var err error

	if override != "" {
		loc, err = a.geocoder.Geocode(ctx, override)
	} else {
		loc, err = a.locator.Detect(ctx)
	}
	if err != nil {
		return Assessment{}, fmt.Errorf("resolving location: %w", err)
	}

	report, err := a.provider.Fetch(ctx, loc)
	if err != nil {
		return Assessment{}, fmt.Errorf("fetching conditions: %w", err)
	}

	res := score.Assess(report, a.threshold())
	return Assessment{Verdict: res.Verdict, Score: res.Score, Location: loc, Report: report}, nil
}

// threshold returns the configured Threshold, defaulting when unset.
func (a *App) threshold() int {
	if a.cfg.Threshold > 0 {
		return a.cfg.Threshold
	}
	return score.DefaultThreshold
}
