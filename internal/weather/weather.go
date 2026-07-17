// Package weather fetches the Conditions inputs (temperature, humidity, AQI,
// weather code) for a Location from Open-Meteo's keyless APIs (ADR-0003).
package weather

import (
	"context"
	"fmt"

	"github.com/go-resty/resty/v2"

	"github.com/asphaltbuffet/go-outside/internal/geo"
)

// Report holds the raw Conditions inputs for one Location at one time.
// WeatherCode is a WMO weather-interpretation code (0-99).
type Report struct {
	TempF       float64
	HumidityPct float64
	AQI         int
	WeatherCode int
}

// Provider fetches a Report for a Location.
type Provider interface {
	Fetch(ctx context.Context, loc geo.Location) (Report, error)
}

const (
	defaultForecastBaseURL = "https://api.open-meteo.com/v1"
	defaultAirQualityPath  = "https://air-quality-api.open-meteo.com/v1"
)

// OpenMeteo is a resty-backed Provider hitting Open-Meteo forecast + air quality.
type OpenMeteo struct {
	client          *resty.Client
	forecastBaseURL string
	airQualityBase  string
}

// NewOpenMeteo builds an OpenMeteo Provider. A nil client defaults to
// resty.New() with production base URLs. A client whose BaseURL is set (tests)
// routes every request to that single host via relative endpoints.
func NewOpenMeteo(client *resty.Client) *OpenMeteo {
	if client == nil {
		return &OpenMeteo{
			client:          resty.New(),
			forecastBaseURL: defaultForecastBaseURL + "/forecast",
			airQualityBase:  defaultAirQualityPath + "/air-quality",
		}
	}
	return &OpenMeteo{
		client:          client,
		forecastBaseURL: "/forecast",
		airQualityBase:  "/air-quality",
	}
}

type forecastResponse struct {
	Current struct {
		Temperature2m      float64 `json:"temperature_2m"`
		RelativeHumidity2m float64 `json:"relative_humidity_2m"`
		WeatherCode        int     `json:"weather_code"`
	} `json:"current"`
}

type airQualityResponse struct {
	Current struct {
		USAQI int `json:"us_aqi"`
	} `json:"current"`
}

// Fetch retrieves temperature/humidity/weather and AQI for loc.
func (o *OpenMeteo) Fetch(ctx context.Context, loc geo.Location) (Report, error) {
	var fc forecastResponse
	fResp, err := o.client.R().
		SetContext(ctx).
		SetQueryParams(map[string]string{
			"latitude":         fmt.Sprintf("%f", loc.Lat),
			"longitude":        fmt.Sprintf("%f", loc.Lon),
			"current":          "temperature_2m,relative_humidity_2m,weather_code",
			"temperature_unit": "fahrenheit",
		}).
		SetResult(&fc).
		Get(o.forecastBaseURL)
	if err != nil {
		return Report{}, fmt.Errorf("fetching forecast: %w", err)
	}
	if fResp.IsError() {
		return Report{}, fmt.Errorf("fetching forecast: status %d", fResp.StatusCode())
	}

	var aq airQualityResponse
	aResp, err := o.client.R().
		SetContext(ctx).
		SetQueryParams(map[string]string{
			"latitude":  fmt.Sprintf("%f", loc.Lat),
			"longitude": fmt.Sprintf("%f", loc.Lon),
			"current":   "us_aqi",
		}).
		SetResult(&aq).
		Get(o.airQualityBase)
	if err != nil {
		return Report{}, fmt.Errorf("fetching air quality: %w", err)
	}
	if aResp.IsError() {
		return Report{}, fmt.Errorf("fetching air quality: status %d", aResp.StatusCode())
	}

	return Report{
		TempF:       fc.Current.Temperature2m,
		HumidityPct: fc.Current.RelativeHumidity2m,
		AQI:         aq.Current.USAQI,
		WeatherCode: fc.Current.WeatherCode,
	}, nil
}

// Compile-time interface check.
var _ Provider = (*OpenMeteo)(nil)
