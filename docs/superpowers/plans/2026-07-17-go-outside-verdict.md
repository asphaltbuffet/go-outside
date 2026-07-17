# go-outside Verdict Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the `go-outside` default command: determine the user's Location, fetch current Conditions, compute a discomfort Score, and return an `ok`/`nope` Verdict via output text and process exit status.

**Architecture:** All domain logic lives behind the existing `App` seam (`internal/app`). Providers (geolocation, geocoding, weather, AQI) are small interfaces with resty-backed implementations and hand-rolled fakes. Scoring is a pure function over a `Conditions` value producing a `Score`. The root Cobra command's `RunE` orchestrates: locate → fetch → score → compare to Threshold → emit Verdict. A typed sentinel error carries the exit code (`ok`=0, `nope`=1, error=2), mapped to `os.Exit` in `main`.

**Tech Stack:** Go 1.26.3 (no CGo), Cobra + Viper + afero (existing), resty (`github.com/go-resty/resty/v2`, new) for HTTP, goccy/go-json (existing) for encoding, testify for tests, Open-Meteo + IP-geolocation APIs (keyless).

## Global Constraints

- Go 1.26.3, no CGo.
- Never redeclare a persistent flag (`--quiet`/`-q`, `--json`, `--config`, `--no-config`) as a local flag; they live on the root command and flow via context.
- `testify/assert` for non-fatal assertions, `testify/require` for preconditions.
- Wrap errors: `fmt.Errorf("context: %w", err)`.
- Timestamps: RFC3339 with UTC `Z` suffix.
- No type assertions / `any` casts outside the established context-key pattern — prefer typed interfaces.
- No magic numbers — name constants; scoring constants live in `internal/score` per ADR-0004.
- UI: silence is success; verbose output only with `-v`/`--verbose`; `--json` on all commands.
- Prefer hand-rolled fakes for internal interfaces; reserve mockery for external deps.
- VCS is `jj`, not `git`. Commits below use `jj` commands.
- Domain vocabulary (Location, Conditions, Verdict, Score, Threshold, Factor) per `CONTEXT.md` — use these exact names in code.
- Exit codes per ADR-0002: `ok`=0, `nope`=1, error=2, in every output mode.
- Providers keyless per ADR-0003: no API keys, no secrets in config.
- Penalty curves/Threshold per ADR-0004 (exact values reproduced in Task 4).

---

## File Structure

- `internal/score/score.go` — `Conditions`, `Score`, `Verdict`, `Assess()`; pure scoring, penalty curves, Threshold default. No I/O.
- `internal/score/score_test.go` — table tests for each Factor curve and Verdict boundary.
- `internal/weather/weather.go` — `Report` type + `Provider` interface; Open-Meteo forecast + air-quality client (resty).
- `internal/weather/weather_test.go` — client tests against an `httptest.Server`.
- `internal/weather/fake.go` — `FakeProvider` for use by `app` tests.
- `internal/geo/geo.go` — `Location` type + `Locator` (auto-detect) and `Geocoder` (postal→coords) interfaces; resty impls.
- `internal/geo/geo_test.go` — client tests against `httptest.Server`.
- `internal/geo/fake.go` — `FakeLocator`, `FakeGeocoder`.
- `internal/app/app.go` — MODIFY: `App` gains provider deps + `Assess(ctx, override)` method.
- `internal/app/app_test.go` — CREATE: `App.Assess` orchestration tests using fakes.
- `internal/apptesting/apptesting.go` — MODIFY: `OpenApp` wires fakes.
- `internal/cli/exit.go` — CREATE: `ExitError` typed sentinel + `CodeFor(err)`.
- `cmd/root.go` — MODIFY: add root `RunE`, `--verbose`/`-v` and `--location` flags, exit-code mapping in `Execute`.
- `main.go` — MODIFY: map returned error to `os.Exit(code)`.

---

### Task 1: resty dependency + exit-code sentinel

Adds the HTTP dependency and the typed error that carries the process exit code. Foundation for every later task.

**Files:**
- Modify: `go.mod`, `go.sum` (via `go get`)
- Create: `internal/cli/exit.go`
- Test: `internal/cli/exit_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `cli.ExitError` — `type ExitError struct { Code int; Err error }`; methods `Error() string`, `Unwrap() error`.
  - `cli.NewExitError(code int, err error) *ExitError`
  - `cli.CodeFor(err error) int` — returns the `Code` of the first `*ExitError` in the chain, `0` if `err == nil`, else `2`.

- [ ] **Step 1: Add resty**

Run:
```bash
go get github.com/go-resty/resty/v2@v2.16.5
```
Expected: `go.mod` gains `github.com/go-resty/resty/v2 v2.16.5`.

- [ ] **Step 2: Write the failing test**

Create `internal/cli/exit_test.go`:
```go
package cli_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/asphaltbuffet/go-outside/internal/cli"
)

func TestCodeFor(t *testing.T) {
	assert.Equal(t, 0, cli.CodeFor(nil))

	plain := errors.New("boom")
	assert.Equal(t, 2, cli.CodeFor(plain))

	nope := cli.NewExitError(1, errors.New("nope"))
	assert.Equal(t, 1, cli.CodeFor(nope))

	wrapped := errors.New("wrap: " + nope.Error())
	_ = wrapped
	assert.ErrorIs(t, nope, nope.Unwrap())
}

func TestExitErrorWrap(t *testing.T) {
	inner := errors.New("inner")
	ee := cli.NewExitError(2, inner)
	assert.Equal(t, "inner", ee.Error())
	assert.ErrorIs(t, ee, inner)
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `gotestsum -- -run TestCodeFor ./internal/cli/`
Expected: FAIL — `undefined: cli.NewExitError` / `cli.CodeFor`.

- [ ] **Step 4: Write minimal implementation**

Create `internal/cli/exit.go`:
```go
package cli

import "errors"

// ExitError carries a process exit code alongside an error, so the command
// layer can map a Verdict (nope=1) or a failure (2) to os.Exit. See ADR-0002.
type ExitError struct {
	Code int
	Err  error
}

// NewExitError builds an ExitError with the given exit code and cause.
func NewExitError(code int, err error) *ExitError {
	return &ExitError{Code: code, Err: err}
}

func (e *ExitError) Error() string { return e.Err.Error() }

func (e *ExitError) Unwrap() error { return e.Err }

// CodeFor returns the exit code for err: 0 for nil, the ExitError's Code if
// one is present in the chain, otherwise 2 (an unclassified failure).
func CodeFor(err error) int {
	if err == nil {
		return 0
	}
	var ee *ExitError
	if errors.As(err, &ee) {
		return ee.Code
	}
	return 2
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `gotestsum -- -run 'TestCodeFor|TestExitErrorWrap' ./internal/cli/`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
jj commit -m "feat(cli): add ExitError sentinel and resty dependency"
```

---

### Task 2: Location types and geo providers

Defines `Location` and the two provider interfaces (auto-detect + geocode), with resty implementations and fakes. Keyless per ADR-0003.

**Files:**
- Create: `internal/geo/geo.go`, `internal/geo/fake.go`
- Test: `internal/geo/geo_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `geo.Location` — `type Location struct { PostalCode string; Lat float64; Lon float64 }`.
  - `geo.Locator` interface — `Detect(ctx context.Context) (Location, error)` (IP-based auto-detect).
  - `geo.Geocoder` interface — `Geocode(ctx context.Context, postalCode string) (Location, error)`.
  - `geo.NewIPLocator(client *resty.Client) *IPLocator` implementing `Locator`.
  - `geo.NewOpenMeteoGeocoder(client *resty.Client) *OpenMeteoGeocoder` implementing `Geocoder`.
  - `geo.FakeLocator{ Loc Location; Err error }` implementing `Locator`.
  - `geo.FakeGeocoder{ Loc Location; Err error }` implementing `Geocoder`.
  - Both real constructors accept a nil client and default to `resty.New()`.

- [ ] **Step 1: Write the failing test (types + fakes)**

Create `internal/geo/geo_test.go`:
```go
package geo_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/asphaltbuffet/go-outside/internal/geo"
)

func TestFakeLocator(t *testing.T) {
	want := geo.Location{PostalCode: "22901", Lat: 38.03, Lon: -78.47}
	var l geo.Locator = geo.FakeLocator{Loc: want}
	got, err := l.Detect(context.Background())
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestIPLocatorDetect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"postal":    "22901",
			"latitude":  38.03,
			"longitude": -78.47,
		})
	}))
	defer srv.Close()

	client := resty.New().SetBaseURL(srv.URL)
	loc := geo.NewIPLocator(client)
	got, err := loc.Detect(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "22901", got.PostalCode)
	assert.InDelta(t, 38.03, got.Lat, 0.001)
}

func TestOpenMeteoGeocode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "22901", r.URL.Query().Get("name"))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{
				{"postcodes": []string{"22901"}, "latitude": 38.03, "longitude": -78.47},
			},
		})
	}))
	defer srv.Close()

	client := resty.New().SetBaseURL(srv.URL)
	g := geo.NewOpenMeteoGeocoder(client)
	got, err := g.Geocode(context.Background(), "22901")
	require.NoError(t, err)
	assert.Equal(t, "22901", got.PostalCode)
	assert.InDelta(t, -78.47, got.Lon, 0.001)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `gotestsum -- ./internal/geo/`
Expected: FAIL — `package geo` / undefined symbols.

- [ ] **Step 3: Write the types and fakes**

Create `internal/geo/geo.go`:
```go
// Package geo resolves the user's Location: auto-detection via IP geolocation
// and postal-code-to-coordinate geocoding. Keyless providers only (ADR-0003).
package geo

import (
	"context"
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
		SetResult(&body).
		Get("/json/")
	if err != nil {
		return Location{}, fmt.Errorf("detecting location: %w", err)
	}
	if resp.IsError() {
		return Location{}, fmt.Errorf("detecting location: status %d", resp.StatusCode())
	}
	return Location{PostalCode: body.Postal, Lat: body.Latitude, Lon: body.Longitude}, nil
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
```

Create `internal/geo/fake.go`:
```go
package geo

import "context"

// FakeLocator is a hand-rolled Locator for tests.
type FakeLocator struct {
	Loc Location
	Err error
}

// Detect returns the configured Location or error.
func (f FakeLocator) Detect(_ context.Context) (Location, error) {
	return f.Loc, f.Err
}

// FakeGeocoder is a hand-rolled Geocoder for tests.
type FakeGeocoder struct {
	Loc Location
	Err error
}

// Geocode returns the configured Location or error.
func (f FakeGeocoder) Geocode(_ context.Context, _ string) (Location, error) {
	return f.Loc, f.Err
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `gotestsum -- -race ./internal/geo/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
jj commit -m "feat(geo): add Location, IP locator, and Open-Meteo geocoder"
```

---

### Task 3: Weather + AQI provider

Fetches Conditions inputs from Open-Meteo's forecast and air-quality endpoints. Returns a `weather.Report` the scorer consumes.

**Files:**
- Create: `internal/weather/weather.go`, `internal/weather/fake.go`
- Test: `internal/weather/weather_test.go`

**Interfaces:**
- Consumes: `geo.Location` (for Lat/Lon).
- Produces:
  - `weather.Report` — `type Report struct { TempF float64; HumidityPct float64; AQI int; WeatherCode int }`.
  - `weather.Provider` interface — `Fetch(ctx context.Context, loc geo.Location) (Report, error)`.
  - `weather.NewOpenMeteo(client *resty.Client) *OpenMeteo` implementing `Provider`; nil client defaults to `resty.New()`.
  - `weather.FakeProvider{ Report Report; Err error }` implementing `Provider`.

- [ ] **Step 1: Write the failing test**

Create `internal/weather/weather_test.go`:
```go
package weather_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/asphaltbuffet/go-outside/internal/geo"
	"github.com/asphaltbuffet/go-outside/internal/weather"
)

func TestFakeProvider(t *testing.T) {
	want := weather.Report{TempF: 70, HumidityPct: 45, AQI: 30, WeatherCode: 0}
	var p weather.Provider = weather.FakeProvider{Report: want}
	got, err := p.Fetch(context.Background(), geo.Location{})
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestOpenMeteoFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "air-quality") {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"current": map[string]any{"us_aqi": 42},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"current": map[string]any{
				"temperature_2m":       68.0,
				"relative_humidity_2m": 55.0,
				"weather_code":         61,
			},
		})
	}))
	defer srv.Close()

	p := weather.NewOpenMeteo(resty.New().SetBaseURL(srv.URL))
	got, err := p.Fetch(context.Background(), geo.Location{Lat: 38.03, Lon: -78.47})
	require.NoError(t, err)
	assert.InDelta(t, 68.0, got.TempF, 0.001)
	assert.InDelta(t, 55.0, got.HumidityPct, 0.001)
	assert.Equal(t, 42, got.AQI)
	assert.Equal(t, 61, got.WeatherCode)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `gotestsum -- ./internal/weather/`
Expected: FAIL — undefined `weather.*`.

- [ ] **Step 3: Write the implementation**

Create `internal/weather/weather.go`:
```go
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
	client *resty.Client
}

// NewOpenMeteo builds an OpenMeteo Provider. A nil client defaults to
// resty.New(); callers may pass a client with a test base URL.
func NewOpenMeteo(client *resty.Client) *OpenMeteo {
	if client == nil {
		client = resty.New()
	}
	return &OpenMeteo{client: client}
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
		Get(defaultForecastBaseURL + "/forecast")
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
		Get(defaultAirQualityPath + "/air-quality")
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
```

Note: the test overrides the client base URL, and both requests use absolute `defaultForecastBaseURL`/`defaultAirQualityPath` URLs in production. For the test to route both to `httptest`, the test passes `resty.New().SetBaseURL(srv.URL)` **and** the `Get` calls must use relative paths. Adjust the two `Get` calls to relative paths and move the host into a per-endpoint base — see Step 3a.

- [ ] **Step 3a: Make base URLs injectable so tests route to httptest**

Replace the two `Get(defaultForecastBaseURL + "/forecast")` / `Get(defaultAirQualityPath + "/air-quality")` calls and struct with injectable hosts:
```go
// OpenMeteo is a resty-backed Provider hitting Open-Meteo forecast + air quality.
type OpenMeteo struct {
	client          *resty.Client
	forecastBaseURL string
	airQualityBase  string
}

// NewOpenMeteo builds an OpenMeteo Provider. A nil client defaults to
// resty.New() with production base URLs. A client whose BaseURL is set (tests)
// routes every request to that single host.
func NewOpenMeteo(client *resty.Client) *OpenMeteo {
	if client == nil {
		return &OpenMeteo{
			client:          resty.New(),
			forecastBaseURL: defaultForecastBaseURL + "/forecast",
			airQualityBase:  defaultAirQualityPath + "/air-quality",
		}
	}
	// Test path: client has a BaseURL; use relative endpoints against it.
	return &OpenMeteo{
		client:          client,
		forecastBaseURL: "/forecast",
		airQualityBase:  "/air-quality",
	}
}
```
And change the two `Get(...)` calls to `Get(o.forecastBaseURL)` and `Get(o.airQualityBase)`.

Update the test handler to match both `/forecast` and `/air-quality` — it already branches on `air-quality` in the path, so `/forecast` falls through to the forecast branch. No test change needed.

Create `internal/weather/fake.go`:
```go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `gotestsum -- -race ./internal/weather/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
jj commit -m "feat(weather): add Open-Meteo weather + AQI provider"
```

---

### Task 4: Scoring — Conditions, Factor curves, Verdict

Pure domain logic: map a `weather.Report` to per-Factor penalties, sum to a Score, compare to Threshold, produce a Verdict. Values per ADR-0004. No I/O — this is the most heavily tested file.

**Files:**
- Create: `internal/score/score.go`
- Test: `internal/score/score_test.go`

**Interfaces:**
- Consumes: `weather.Report`.
- Produces:
  - `score.Verdict` — `type Verdict string` with `score.OK Verdict = "ok"` and `score.Nope Verdict = "nope"`.
  - `score.Result` — `type Result struct { Verdict Verdict; Score int }`.
  - `score.DefaultThreshold` — `const DefaultThreshold = 100`.
  - `score.Assess(r weather.Report, threshold int) Result`.
  - Unexported penalty helpers `tempPenalty`, `humidityPenalty`, `aqiPenalty`, `weatherPenalty` (tested via `Assess`).

- [ ] **Step 1: Write the failing tests (Factor curves + boundary)**

Create `internal/score/score_test.go`:
```go
package score_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/asphaltbuffet/go-outside/internal/score"
	"github.com/asphaltbuffet/go-outside/internal/weather"
)

func TestAssessComfortableIsOK(t *testing.T) {
	r := weather.Report{TempF: 70, HumidityPct: 45, AQI: 30, WeatherCode: 0}
	got := score.Assess(r, score.DefaultThreshold)
	assert.Equal(t, score.OK, got.Verdict)
	assert.Equal(t, 0, got.Score)
}

func TestAssessTempPenalty(t *testing.T) {
	// 95F is 20 above the 75 band edge: 0.25 * 20^2 = 100.
	r := weather.Report{TempF: 95, HumidityPct: 45, AQI: 30, WeatherCode: 0}
	got := score.Assess(r, score.DefaultThreshold)
	assert.Equal(t, 100, got.Score)
	// Score == threshold is not "exceeds", so still ok.
	assert.Equal(t, score.OK, got.Verdict)
}

func TestAssessTempTripsNope(t *testing.T) {
	// 96F: 21^2 * 0.25 = 110 > 100.
	r := weather.Report{TempF: 96, HumidityPct: 45, AQI: 30, WeatherCode: 0}
	got := score.Assess(r, score.DefaultThreshold)
	assert.Equal(t, score.Nope, got.Verdict)
}

func TestAssessHumidityPenalty(t *testing.T) {
	// 90% is 30 above the 60 band edge: 0.1 * 30^2 = 90.
	r := weather.Report{TempF: 70, HumidityPct: 90, AQI: 30, WeatherCode: 0}
	got := score.Assess(r, score.DefaultThreshold)
	assert.Equal(t, 90, got.Score)
	assert.Equal(t, score.OK, got.Verdict)
}

func TestAssessAQICategories(t *testing.T) {
	cases := []struct {
		aqi  int
		want int
	}{
		{25, 0}, {75, 20}, {125, 60}, {175, 120}, {250, 300}, {350, 600},
	}
	for _, c := range cases {
		r := weather.Report{TempF: 70, HumidityPct: 45, AQI: c.aqi, WeatherCode: 0}
		got := score.Assess(r, 1_000_000) // huge threshold to isolate the score
		assert.Equal(t, c.want, got.Score, "aqi=%d", c.aqi)
	}
}

func TestAssessAQIUnhealthyTripsNope(t *testing.T) {
	r := weather.Report{TempF: 70, HumidityPct: 45, AQI: 175, WeatherCode: 0}
	got := score.Assess(r, score.DefaultThreshold)
	assert.Equal(t, score.Nope, got.Verdict)
}

func TestAssessWeatherCodes(t *testing.T) {
	cases := []struct {
		code int
		want int
	}{
		{0, 0}, {3, 5}, {45, 20}, {51, 30}, {61, 60}, {80, 70}, {71, 80}, {95, 120},
	}
	for _, c := range cases {
		r := weather.Report{TempF: 70, HumidityPct: 45, AQI: 30, WeatherCode: c.code}
		got := score.Assess(r, 1_000_000)
		assert.Equal(t, c.want, got.Score, "code=%d", c.code)
	}
}

func TestAssessThunderstormTripsNope(t *testing.T) {
	r := weather.Report{TempF: 70, HumidityPct: 45, AQI: 30, WeatherCode: 95}
	got := score.Assess(r, score.DefaultThreshold)
	assert.Equal(t, score.Nope, got.Verdict)
}

func TestAssessCombinedMildTripsNope(t *testing.T) {
	// Cold-ish + humid + moderate AQI + rain, each mild, summing past 100.
	// temp 55: 5 below 60 -> 0.25*25 = 6; humidity 75: 15 above -> 0.1*225=22;
	// aqi 75 -> 20; rain code 61 -> 60. Sum = 108 > 100.
	r := weather.Report{TempF: 55, HumidityPct: 75, AQI: 75, WeatherCode: 61}
	got := score.Assess(r, score.DefaultThreshold)
	assert.Equal(t, 108, got.Score)
	assert.Equal(t, score.Nope, got.Verdict)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `gotestsum -- ./internal/score/`
Expected: FAIL — undefined `score.*`.

- [ ] **Step 3: Write the implementation**

Create `internal/score/score.go`:
```go
// Package score turns a weather.Report into a Verdict: it computes a per-Factor
// discomfort penalty, sums them into a Score, and compares to a Threshold.
// Penalty curves and defaults are calibrated per ADR-0004. Pure, no I/O.
package score

import (
	"math"

	"github.com/asphaltbuffet/go-outside/internal/weather"
)

// Verdict is the go/no-go result (CONTEXT.md).
type Verdict string

const (
	// OK means Conditions are comfortable for going outside.
	OK Verdict = "ok"
	// Nope means Conditions are outside the comfortable range.
	Nope Verdict = "nope"
)

// DefaultThreshold is the default Score boundary separating ok from nope.
const DefaultThreshold = 100

// Comfortable band edges (ADR-0004).
const (
	tempBandLowF  = 60.0
	tempBandHighF = 75.0
	tempCoeff     = 0.25

	humidityBandLowPct  = 30.0
	humidityBandHighPct = 60.0
	humidityCoeff       = 0.1
)

// Result is the outcome of assessing a Report.
type Result struct {
	Verdict Verdict
	Score   int
}

// Assess computes the Score for r and returns a Verdict: Nope when the Score
// exceeds threshold, otherwise OK.
func Assess(r weather.Report, threshold int) Result {
	total := tempPenalty(r.TempF) +
		humidityPenalty(r.HumidityPct) +
		aqiPenalty(r.AQI) +
		weatherPenalty(r.WeatherCode)

	verdict := OK
	if total > threshold {
		verdict = Nope
	}
	return Result{Verdict: verdict, Score: total}
}

// quadratic returns coeff * distance^2 rounded to an int, where distance is how
// far v falls outside [low, high]; 0 while inside the band.
func quadratic(v, low, high, coeff float64) int {
	var d float64
	switch {
	case v < low:
		d = low - v
	case v > high:
		d = v - high
	default:
		return 0
	}
	return int(math.Round(coeff * d * d))
}

func tempPenalty(f float64) int {
	return quadratic(f, tempBandLowF, tempBandHighF, tempCoeff)
}

func humidityPenalty(pct float64) int {
	return quadratic(pct, humidityBandLowPct, humidityBandHighPct, humidityCoeff)
}

// aqiPenalty maps the US AQI to a penalty by EPA category (ADR-0004).
func aqiPenalty(aqi int) int {
	switch {
	case aqi <= 50:
		return 0
	case aqi <= 100:
		return 20
	case aqi <= 150:
		return 60
	case aqi <= 200:
		return 120
	case aqi <= 300:
		return 300
	default:
		return 600
	}
}

// weatherPenalty maps a WMO weather code to a penalty (ADR-0004).
func weatherPenalty(code int) int {
	switch {
	case code <= 1: // clear / mainly clear
		return 0
	case code <= 3: // partly cloudy / overcast
		return 5
	case code == 45 || code == 48: // fog
		return 20
	case code >= 51 && code <= 57: // drizzle
		return 30
	case code >= 61 && code <= 67: // rain
		return 60
	case code >= 71 && code <= 77: // snow
		return 80
	case code >= 80 && code <= 82: // rain showers
		return 70
	case code >= 95 && code <= 99: // thunderstorm
		return 120
	default:
		return 5
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `gotestsum -- -race ./internal/score/`
Expected: PASS (all cases).

- [ ] **Step 5: Commit**

```bash
jj commit -m "feat(score): add Factor penalties, Score, and Verdict"
```

---

### Task 5: App orchestration — App.Assess

Wires the providers and scorer into the `App`. `App.Assess` resolves the Location (override or auto-detect), fetches the Report, scores it, and returns a Verdict with the Location used.

**Files:**
- Modify: `internal/app/app.go`
- Modify: `internal/apptesting/apptesting.go`
- Test: `internal/app/app_test.go` (create)

**Interfaces:**
- Consumes: `geo.Locator`, `geo.Geocoder`, `weather.Provider`, `score.Assess`, `config.Config`.
- Produces:
  - `app.Assessment` — `type Assessment struct { Verdict score.Verdict; Score int; Location geo.Location }`.
  - `app.App.Assess(ctx context.Context, override string) (Assessment, error)` — when `override` is non-empty, geocode it; else auto-detect. Threshold from config (Task 6 adds the field; until then use `score.DefaultThreshold`).
  - `app.New(cfg, locator, geocoder, provider)` — updated signature.

- [ ] **Step 1: Write the failing test**

Create `internal/app/app_test.go`:
```go
package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/asphaltbuffet/go-outside/internal/app"
	"github.com/asphaltbuffet/go-outside/internal/config"
	"github.com/asphaltbuffet/go-outside/internal/geo"
	"github.com/asphaltbuffet/go-outside/internal/score"
	"github.com/asphaltbuffet/go-outside/internal/weather"
)

func newApp(t *testing.T, loc geo.Locator, gc geo.Geocoder, wp weather.Provider) *app.App {
	t.Helper()
	a, err := app.New(config.GetDefaults(), loc, gc, wp)
	require.NoError(t, err)
	return a
}

func TestAssessAutoDetectOK(t *testing.T) {
	loc := geo.FakeLocator{Loc: geo.Location{PostalCode: "22901", Lat: 38, Lon: -78}}
	wp := weather.FakeProvider{Report: weather.Report{TempF: 70, HumidityPct: 45, AQI: 30, WeatherCode: 0}}
	a := newApp(t, loc, geo.FakeGeocoder{}, wp)

	got, err := a.Assess(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, score.OK, got.Verdict)
	assert.Equal(t, "22901", got.Location.PostalCode)
}

func TestAssessOverrideGeocodes(t *testing.T) {
	gc := geo.FakeGeocoder{Loc: geo.Location{PostalCode: "90210", Lat: 34, Lon: -118}}
	wp := weather.FakeProvider{Report: weather.Report{TempF: 96, HumidityPct: 45, AQI: 30, WeatherCode: 0}}
	// Locator must NOT be consulted when override is present.
	loc := geo.FakeLocator{Err: errors.New("should not be called")}
	a := newApp(t, loc, gc, wp)

	got, err := a.Assess(context.Background(), "90210")
	require.NoError(t, err)
	assert.Equal(t, score.Nope, got.Verdict)
	assert.Equal(t, "90210", got.Location.PostalCode)
}

func TestAssessDetectErrorPropagates(t *testing.T) {
	loc := geo.FakeLocator{Err: errors.New("network down")}
	a := newApp(t, loc, geo.FakeGeocoder{}, weather.FakeProvider{})

	_, err := a.Assess(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "network down")
}

func TestAssessWeatherErrorPropagates(t *testing.T) {
	loc := geo.FakeLocator{Loc: geo.Location{PostalCode: "22901"}}
	wp := weather.FakeProvider{Err: errors.New("api timeout")}
	a := newApp(t, loc, geo.FakeGeocoder{}, wp)

	_, err := a.Assess(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "api timeout")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `gotestsum -- ./internal/app/`
Expected: FAIL — `app.New` signature mismatch, `app.Assess` undefined.

- [ ] **Step 3: Rewrite app.go**

Replace `internal/app/app.go` with:
```go
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
	return Assessment{Verdict: res.Verdict, Score: res.Score, Location: loc}, nil
}

// threshold returns the configured Threshold, defaulting when unset.
func (a *App) threshold() int {
	if a.cfg.Threshold > 0 {
		return a.cfg.Threshold
	}
	return score.DefaultThreshold
}
```

Note: `a.cfg.Threshold` is added to `config.Config` in Task 6. Until then this won't compile. **Task 5 and Task 6 must be committed together** — implement Task 6's config field before running Step 4. Proceed to Task 6 Step 3 now, then return here.

- [ ] **Step 4: Update apptesting.OpenApp**

Replace `internal/apptesting/apptesting.go`:
```go
package apptesting

import (
	"testing"

	"github.com/asphaltbuffet/go-outside/internal/app"
	"github.com/asphaltbuffet/go-outside/internal/config"
	"github.com/asphaltbuffet/go-outside/internal/geo"
	"github.com/asphaltbuffet/go-outside/internal/weather"
)

// OpenApp creates an App wired with fake providers for use in tests.
func OpenApp(t *testing.T) *app.App {
	t.Helper()

	cfg := config.GetDefaults()
	a, err := app.New(
		cfg,
		geo.FakeLocator{Loc: geo.Location{PostalCode: "22901"}},
		geo.FakeGeocoder{Loc: geo.Location{PostalCode: "22901"}},
		weather.FakeProvider{Report: weather.Report{TempF: 70, HumidityPct: 45, AQI: 30, WeatherCode: 0}},
	)
	if err != nil {
		t.Fatalf("apptesting.OpenApp: %v", err)
	}
	return a
}
```

- [ ] **Step 5: Run tests to verify they pass** (after Task 6 Step 3 is done)

Run: `gotestsum -- -race ./internal/app/ ./internal/apptesting/`
Expected: PASS.

- [ ] **Step 6: Commit** (jointly with Task 6)

```bash
jj commit -m "feat(app): add Assess orchestration and Threshold config"
```

---

### Task 6: Threshold config field

Adds the single user-tunable knob (ADR-0004): a `Threshold` on `Config`. Small; committed with Task 5.

**Files:**
- Modify: `internal/config/config.go`
- Test: `internal/config/config_test.go` (add case)

**Interfaces:**
- Consumes: nothing.
- Produces: `config.Config.Threshold int` field (mapstructure/toml key `threshold`), defaulting to `score.DefaultThreshold` is **not** applied here (App owns the default to avoid an import cycle); config default stays `0` meaning "unset".

- [ ] **Step 1: Write the failing test**

Create or append to `internal/config/config_test.go`:
```go
package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/asphaltbuffet/go-outside/internal/config"
)

func TestThresholdDefaultsToZeroMeaningUnset(t *testing.T) {
	cfg := config.GetDefaults()
	assert.Equal(t, 0, cfg.Threshold)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `gotestsum -- -run TestThresholdDefaults ./internal/config/`
Expected: FAIL — `cfg.Threshold` undefined.

- [ ] **Step 3: Add the field**

In `internal/config/config.go`, add to the `Config` struct (after the `Output` field):
```go
	// Threshold is the Score boundary separating ok from nope. 0 means unset;
	// the App substitutes score.DefaultThreshold. User-tunable (ADR-0004).
	Threshold int `mapstructure:"threshold" toml:"threshold"`
```

- [ ] **Step 4: Run test to verify it passes**

Run: `gotestsum -- -run TestThresholdDefaults ./internal/config/`
Expected: PASS.

- [ ] **Step 5: Return to Task 5 Step 4-5**, then commit both together (see Task 5 Step 6).

---

### Task 7: Root command wiring, flags, and output formatting

Gives the root command a `RunE` that runs an assessment and formats output for all three modes. Adds `--verbose`/`-v` and `--location` flags. Maps the Verdict to an `ExitError`.

**Files:**
- Modify: `cmd/root.go`
- Test: `cmd/root_test.go` (create)

**Interfaces:**
- Consumes: `app.App.Assess`, `app.Assessment`, `cli.NewOutputWriterFromConfig`, `cli.NewExitError`, `score.Nope`.
- Produces: root command with `RunE`; production `newApp` wires real providers.

- [ ] **Step 1: Write the failing test**

Create `cmd/root_test.go`:
```go
package cmd_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/asphaltbuffet/go-outside/cmd"
)

// runRoot executes the root command with args and returns stdout, the error.
func runRoot(t *testing.T, args ...string) (string, error) {
	t.Helper()
	c := cmd.NewRootCmd()
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&out)
	c.SetArgs(args)
	err := c.ExecuteContext(context.Background())
	return out.String(), err
}

func TestRootVerboseFormat(t *testing.T) {
	// Uses --location so the run is deterministic (no network) only if the
	// command supports a test seam; see Step 3 note. This test asserts the
	// verbose format shape against a nope from an override that geocodes to a
	// hot location via the injected fake in Step 3.
	out, _ := runRoot(t, "--no-config", "--location", "00000", "-v")
	// Format: "<verdict> (<score>) [<postal>]"
	assert.Regexp(t, `^(ok|nope) \(\d+\) \[\S+\]\n?$`, strings.TrimSpace(out)+"\n")
}
```

Note: this command hits the network in production. To keep `cmd` tests hermetic, Step 3 introduces a package-level provider-factory seam that tests override. See Step 3.

- [ ] **Step 2: Run test to verify it fails**

Run: `gotestsum -- ./cmd/`
Expected: FAIL — root has no `RunE`, `--location`/`-v` undefined.

- [ ] **Step 3: Wire the root command**

In `cmd/root.go`:

3a. Add flags after the existing persistent flags block:
```go
	rootCmd.Flags().BoolP("verbose", "v", false, "show score and location: nope (999) [22901]")
	rootCmd.Flags().String("location", "", "postal code to assess instead of auto-detecting")
```

3b. Add a provider-factory seam so tests can inject fakes. Near the top of the file:
```go
// appFactory builds the App from a config; overridable in tests to inject fakes.
var appFactory = func(cfg *config.Config) (*app.App, error) {
	return app.New(
		cfg,
		geo.NewIPLocator(nil),
		geo.NewOpenMeteoGeocoder(nil),
		weather.NewOpenMeteo(nil),
	)
}
```
Add imports for `geo` and `weather`. Replace `newApp` body to use it:
```go
func newApp(ctx context.Context) (*app.App, error) {
	cfg, ok := ctx.Value(config.ConfigKey).(*config.Config)
	if !ok {
		return nil, errors.New("config not found in context")
	}
	return appFactory(cfg)
}
```

3c. Add `RunE` to `rootCmd`:
```go
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := newApp(cmd.Context())
			if err != nil {
				return err
			}

			override, _ := cmd.Flags().GetString("location")
			assessment, err := a.Assess(cmd.Context(), override)
			if err != nil {
				return err // unclassified -> exit 2 via cli.CodeFor
			}

			cfg := cli.MustGetConfig(cmd.Context())
			w := cli.NewOutputWriterFromConfig(cmd.OutOrStdout(), cmd.ErrOrStderr(), cfg)
			verbose, _ := cmd.Flags().GetBool("verbose")

			switch {
			case w.IsJSON():
				if jErr := w.JSON(map[string]any{
					"verdict":  string(assessment.Verdict),
					"score":    assessment.Score,
					"location": assessment.Location.PostalCode,
				}); jErr != nil {
					return jErr
				}
			case verbose:
				w.Success(fmt.Sprintf("%s (%d) [%s]",
					assessment.Verdict, assessment.Score, assessment.Location.PostalCode))
			default:
				w.Success(string(assessment.Verdict))
			}

			if assessment.Verdict == score.Nope {
				return cli.NewExitError(1, errNope)
			}
			return nil
		},
```
Add `errNope` sentinel and imports (`fmt`, `cli`, `score`):
```go
var errNope = errors.New("nope")
```

3d. Update the failing test to inject a fake via `appFactory`. Replace `cmd/root_test.go` `TestRootVerboseFormat` with a version that sets the seam:
```go
func TestRootVerboseFormat(t *testing.T) {
	restore := cmd.SetAppFactoryForTest(func(cfg *config.Config) (*app.App, error) {
		return app.New(cfg,
			geo.FakeLocator{Loc: geo.Location{PostalCode: "22901"}},
			geo.FakeGeocoder{Loc: geo.Location{PostalCode: "22901"}},
			weather.FakeProvider{Report: weather.Report{TempF: 70, HumidityPct: 45, AQI: 30, WeatherCode: 0}},
		)
	})
	defer restore()

	out, err := runRoot(t, "--no-config", "-v")
	require.NoError(t, err)
	assert.Equal(t, "ok (0) [22901]", strings.TrimSpace(out))
}

func TestRootNopeExitCode(t *testing.T) {
	restore := cmd.SetAppFactoryForTest(func(cfg *config.Config) (*app.App, error) {
		return app.New(cfg,
			geo.FakeLocator{Loc: geo.Location{PostalCode: "22901"}},
			geo.FakeGeocoder{},
			weather.FakeProvider{Report: weather.Report{TempF: 96, HumidityPct: 45, AQI: 30, WeatherCode: 0}},
		)
	})
	defer restore()

	out, err := runRoot(t, "--no-config")
	require.Error(t, err)
	assert.Equal(t, 1, cli.CodeFor(err))
	assert.Equal(t, "nope", strings.TrimSpace(out))
}

func TestRootJSON(t *testing.T) {
	restore := cmd.SetAppFactoryForTest(func(cfg *config.Config) (*app.App, error) {
		return app.New(cfg,
			geo.FakeLocator{Loc: geo.Location{PostalCode: "22901"}},
			geo.FakeGeocoder{},
			weather.FakeProvider{Report: weather.Report{TempF: 70, HumidityPct: 45, AQI: 30, WeatherCode: 0}},
		)
	})
	defer restore()

	out, err := runRoot(t, "--no-config", "--json")
	require.NoError(t, err)
	assert.Contains(t, out, `"verdict": "ok"`)
	assert.Contains(t, out, `"location": "22901"`)
}
```
Add the required imports to the test file: `cli`, `app`, `config`, `geo`, `weather`.

3e. Add the test seam export in a new file `cmd/export_test.go`:
```go
package cmd

import "github.com/asphaltbuffet/go-outside/internal/config"
import "github.com/asphaltbuffet/go-outside/internal/app"

// SetAppFactoryForTest overrides the App factory and returns a restore func.
func SetAppFactoryForTest(f func(cfg *config.Config) (*app.App, error)) func() {
	prev := appFactory
	appFactory = f
	return func() { appFactory = prev }
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `gotestsum -- -race ./cmd/`
Expected: PASS (verbose format, nope exit code, JSON).

- [ ] **Step 5: Commit**

```bash
jj commit -m "feat(cmd): add root assessment command with verbose/location/json"
```

---

### Task 8: main.go exit-code mapping

Maps the error returned by `Execute` to the correct process exit code (ADR-0002). Final wiring.

**Files:**
- Modify: `main.go`
- Test: exercised end-to-end via `cmd` tests (Task 7) + a manual run in Step 3.

**Interfaces:**
- Consumes: `cmd.Execute`, `cli.CodeFor`.
- Produces: process exit codes 0/1/2.

- [ ] **Step 1: Read current main.go**

Run: `cat main.go` — note how it currently calls `cmd.Execute` and exits.

- [ ] **Step 2: Rewrite main.go to map exit codes**

Replace `main.go` with:
```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/asphaltbuffet/go-outside/cmd"
	"github.com/asphaltbuffet/go-outside/internal/cli"
)

func main() {
	err := cmd.Execute(context.Background())
	if err != nil {
		// nope prints its own stdout line; only surface real failures on stderr.
		if code := cli.CodeFor(err); code >= 2 {
			fmt.Fprintln(os.Stderr, "go-outside:", err)
		}
		os.Exit(cli.CodeFor(err))
	}
}
```

- [ ] **Step 3: Verify exit codes end-to-end**

Run and check exit codes (the override avoids relying on your real location for the ok case; a real network call still happens for weather):
```bash
mise run build
./dist/go-outside --no-config --location 22901 ; echo "exit=$?"
```
Expected: prints `ok` or `nope`; `exit=0` for ok, `exit=1` for nope. On network failure: an error on stderr and `exit=2`.

- [ ] **Step 4: Commit**

```bash
jj commit -m "feat: map Verdict to process exit code in main"
```

---

### Task 9: Full verification and docs audit

Runs the repo's full gate and updates docs.

**Files:**
- Modify: `CLAUDE.md` (fill Architecture / Repository Layout / Key Patterns if `/audit-docs` flags them)

- [ ] **Step 1: Regenerate the Nix lockfile for the new dependency**

Run: `mise run mod-tidy`
Expected: `go.mod`/`go.sum` tidy; gomod2nix lockfile updated to include resty.

- [ ] **Step 2: Lint**

Run: `mise run lint`
Expected: no errors (golangci-lint --fix applied).

- [ ] **Step 3: Full test with race + coverage**

Run: `mise run test`
Expected: all packages PASS under `-race`.

- [ ] **Step 4: Run the docs audit**

Run `/audit-docs` (per CLAUDE.md Task Completion Checklist) and apply any fixes to `CLAUDE.md`'s Architecture / Repository Layout / Key Patterns sections describing the new `geo`, `weather`, `score` packages and the App orchestration.

- [ ] **Step 5: Commit**

```bash
jj commit -m "chore: tidy modules, docs audit for verdict feature"
```

---

## Self-Review

**1. Spec coverage** (against CONTEXT.md + ADRs 0001-0004):
- Location canonical postal code, IP auto-detect, override → Task 2 (`geo`), Task 5 (`Assess` override branch), Task 7 (`--location`). ✓
- Conditions from keyless Open-Meteo (ADR-0003) → Task 3. ✓
- Score = summed Factor penalties, quadratic temp/humidity, EPA AQI, WMO weather lookup, no hard caps (ADR-0001/0004) → Task 4. ✓
- Verdict ok/nope, `nope` when Score > Threshold → Task 4. ✓
- Threshold default 100, user-tunable, only knob (ADR-0004) → Task 4 (`DefaultThreshold`), Task 6 (config field). ✓
- Undetermined inputs = error, never nope (ADR-0002) → Task 5 (error propagation), Task 8 (exit 2). ✓
- Output: default `ok`/`nope`; verbose `nope (999) [22901]`; JSON always-full → Task 7. ✓
- Exit codes ok=0/nope=1/error=2 in all modes (ADR-0002) → Task 1 (`ExitError`), Task 7 (`NewExitError(1,…)`), Task 8 (`os.Exit`). ✓
- resty HTTP client → Task 1. ✓

**2. Placeholder scan:** No TBD/TODO/"handle edge cases"/"similar to Task N" — every code step contains full code. ✓

**3. Type consistency:** `geo.Location{PostalCode,Lat,Lon}`, `weather.Report{TempF,HumidityPct,AQI,WeatherCode}`, `score.Result{Verdict,Score}`, `score.Assess(Report,int)`, `app.Assessment{Verdict,Score,Location}`, `app.New(cfg,locator,geocoder,provider)`, `app.Assess(ctx,override)`, `cli.NewExitError(code,err)`/`cli.CodeFor(err)` — used identically across Tasks 2-8. ✓

**Cross-task ordering note:** Tasks 5 and 6 are interdependent (App reads `cfg.Threshold`); implement Task 6 Step 3 before running Task 5 Step 5, and commit them together. This is called out in both tasks.
