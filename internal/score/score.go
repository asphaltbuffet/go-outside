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
