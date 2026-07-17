// Package geo resolves the user's Location: auto-detection via IP geolocation
// and postal-code-to-coordinate geocoding. Keyless providers only (ADR-0003).
package geo

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-resty/resty/v2"
)

// Location is the place whose Conditions are assessed. The PostalCode is its
// canonical identity (CONTEXT.md); Lat/Lon are the detection artifact used to
// query coordinate-based providers.
type Location struct {
	PostalCode string
	Lat        float64
	Lon        float64
}

// Locator auto-detects the current Location.
type Locator interface {
	Detect(ctx context.Context) (Location, error)
}

// Geocoder resolves a postal code to a Location with coordinates.
type Geocoder interface {
	Geocode(ctx context.Context, postalCode string) (Location, error)
}

const (
	defaultIPBaseURL      = "https://ipapi.co"
	defaultGeocodeBaseURL = "https://geocoding-api.open-meteo.com/v1"
)

// IPLocator detects Location from the caller's IP via a keyless service.
type IPLocator struct {
	client *resty.Client
}

// NewIPLocator builds an IPLocator. A nil client defaults to resty.New() with
// the production base URL.
func NewIPLocator(client *resty.Client) *IPLocator {
	if client == nil {
		client = resty.New().SetBaseURL(defaultIPBaseURL)
	}
	return &IPLocator{client: client}
}

type ipResponse struct {
	Postal    string  `json:"postal"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// Detect returns the Location inferred from the caller's IP address.
func (l *IPLocator) Detect(ctx context.Context) (Location, error) {
	var body ipResponse
	resp, err := l.client.R().
		SetContext(ctx).
		ForceContentType("application/json").
		SetResult(&body).
		Get("/json/")
	if err != nil {
		return Location{}, fmt.Errorf("detecting location: %w", err)
	}
	if resp.IsError() {
		return Location{}, fmt.Errorf("detecting location: status %d", resp.StatusCode())
	}
	loc := Location{PostalCode: body.Postal, Lat: body.Latitude, Lon: body.Longitude}
	if loc.PostalCode == "" {
		return Location{}, errors.New("detecting location: empty response (no postal code)")
	}
	return loc, nil
}

// OpenMeteoGeocoder resolves postal codes via Open-Meteo's keyless geocoder.
type OpenMeteoGeocoder struct {
	client *resty.Client
}

// NewOpenMeteoGeocoder builds an OpenMeteoGeocoder. A nil client defaults to
// resty.New() with the production base URL.
func NewOpenMeteoGeocoder(client *resty.Client) *OpenMeteoGeocoder {
	if client == nil {
		client = resty.New().SetBaseURL(defaultGeocodeBaseURL)
	}
	return &OpenMeteoGeocoder{client: client}
}

type geocodeResponse struct {
	Results []struct {
		Postcodes []string `json:"postcodes"`
		Latitude  float64  `json:"latitude"`
		Longitude float64  `json:"longitude"`
	} `json:"results"`
}

// Geocode resolves a postal code to a Location with coordinates.
func (g *OpenMeteoGeocoder) Geocode(ctx context.Context, postalCode string) (Location, error) {
	var body geocodeResponse
	resp, err := g.client.R().
		SetContext(ctx).
		SetQueryParam("name", postalCode).
		SetQueryParam("count", "1").
		ForceContentType("application/json").
		SetResult(&body).
		Get("/search")
	if err != nil {
		return Location{}, fmt.Errorf("geocoding %q: %w", postalCode, err)
	}
	if resp.IsError() {
		return Location{}, fmt.Errorf("geocoding %q: status %d", postalCode, resp.StatusCode())
	}
	if len(body.Results) == 0 {
		return Location{}, fmt.Errorf("geocoding %q: no results", postalCode)
	}
	r := body.Results[0]
	return Location{PostalCode: postalCode, Lat: r.Latitude, Lon: r.Longitude}, nil
}

// Compile-time interface checks.
var (
	_ Locator  = (*IPLocator)(nil)
	_ Geocoder = (*OpenMeteoGeocoder)(nil)
)
