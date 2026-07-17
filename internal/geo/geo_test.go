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
