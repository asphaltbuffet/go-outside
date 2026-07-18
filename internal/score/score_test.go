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
	// 96F: 21^2 * 0.25 = 110.25 -> 110 > 100.
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

func TestAssessUnknownWeatherCodeIsPenalized(t *testing.T) {
	r := weather.Report{TempF: 70, HumidityPct: 45, AQI: 30, WeatherCode: 999}
	got := score.Assess(r, score.DefaultThreshold)
	assert.Equal(t, score.Nope, got.Verdict) // 120 > 100
}

func TestAssessCombinedMildTripsNope(t *testing.T) {
	// Each factor mild, summing past 100.
	// temp 55: 5 below 60 -> 0.25*25 = 6.25 -> 6
	// humidity 75: 15 above 60 -> 0.1*225 = 22.5 -> 23
	// aqi 75 -> 20; rain code 61 -> 60. Sum = 109 > 100.
	r := weather.Report{TempF: 55, HumidityPct: 75, AQI: 75, WeatherCode: 61}
	got := score.Assess(r, score.DefaultThreshold)
	assert.Equal(t, 109, got.Score)
	assert.Equal(t, score.Nope, got.Verdict)
}
