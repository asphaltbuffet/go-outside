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
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success",
			"zip":    "22901",
			"lat":    38.03,
			"lon":    -78.47,
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

func TestIPLocatorDetectRateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	loc := geo.NewIPLocator(resty.New().SetBaseURL(srv.URL))
	_, err := loc.Detect(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rate limited")
	assert.Contains(t, err.Error(), "--location")
}

func TestIPLocatorDetectAPIFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// ip-api.com returns HTTP 200 with status:fail on error.
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  "fail",
			"message": "reserved range",
		})
	}))
	defer srv.Close()

	loc := geo.NewIPLocator(resty.New().SetBaseURL(srv.URL))
	_, err := loc.Detect(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reserved range")
}

func TestOpenMeteoGeocode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "22901", r.URL.Query().Get("name"))
		w.Header().Set("Content-Type", "application/json")
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

func TestIPLocatorDetectNonJSONContentType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success", "zip": "22901", "lat": 38.03, "lon": -78.47,
		})
	}))
	defer srv.Close()

	loc := geo.NewIPLocator(resty.New().SetBaseURL(srv.URL))
	got, err := loc.Detect(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "22901", got.PostalCode)
}

func TestIPLocatorDetectEmptyResponseErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{}) // no postal
	}))
	defer srv.Close()

	loc := geo.NewIPLocator(resty.New().SetBaseURL(srv.URL))
	_, err := loc.Detect(context.Background())
	require.Error(t, err)
}
