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

// AQI category upper bounds (US EPA) and their discomfort penalties (ADR-0004).
const (
	aqiGoodMax          = 50
	aqiModerateMax      = 100
	aqiUSGMax           = 150
	aqiUnhealthyMax     = 200
	aqiVeryUnhealthyMax = 300

	aqiPenaltyGood          = 0
	aqiPenaltyModerate      = 20
	aqiPenaltyUSG           = 60
	aqiPenaltyUnhealthy     = 120
	aqiPenaltyVeryUnhealthy = 300
	aqiPenaltyHazardous     = 600
)

// WMO weather-code group penalties (ADR-0004).
const (
	weatherPenaltyClear       = 0
	weatherPenaltyCloudy      = 5
	weatherPenaltyFog         = 20
	weatherPenaltyDrizzle     = 30
	weatherPenaltyRain        = 60
	weatherPenaltyRainShowers = 70
	weatherPenaltySnow        = 80
	weatherPenaltyThunder     = 120

	// weatherPenaltyUnknown is applied to WMO codes outside the known 0-99
	// groups, so unrecognized/garbage data never reads as a benign day.
	weatherPenaltyUnknown = 120
)

// WMO weather-code boundaries (see https://open-meteo.com weather codes).
const (
	wmoClearMax       = 1
	wmoCloudyMax      = 3
	wmoFogA           = 45
	wmoFogB           = 48
	wmoDrizzleMin     = 51
	wmoDrizzleMax     = 57
	wmoRainMin        = 61
	wmoRainMax        = 67
	wmoSnowMin        = 71
	wmoSnowMax        = 77
	wmoRainShowersMin = 80
	wmoRainShowersMax = 82
	wmoThunderMin     = 95
	wmoThunderMax     = 99
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
	case aqi <= aqiGoodMax:
		return aqiPenaltyGood
	case aqi <= aqiModerateMax:
		return aqiPenaltyModerate
	case aqi <= aqiUSGMax:
		return aqiPenaltyUSG
	case aqi <= aqiUnhealthyMax:
		return aqiPenaltyUnhealthy
	case aqi <= aqiVeryUnhealthyMax:
		return aqiPenaltyVeryUnhealthy
	default:
		return aqiPenaltyHazardous
	}
}

// weatherPenalty maps a WMO weather code to a penalty (ADR-0004).
func weatherPenalty(code int) int {
	switch {
	case code <= wmoClearMax:
		return weatherPenaltyClear
	case code <= wmoCloudyMax:
		return weatherPenaltyCloudy
	case code == wmoFogA || code == wmoFogB:
		return weatherPenaltyFog
	case code >= wmoDrizzleMin && code <= wmoDrizzleMax:
		return weatherPenaltyDrizzle
	case code >= wmoRainMin && code <= wmoRainMax:
		return weatherPenaltyRain
	case code >= wmoSnowMin && code <= wmoSnowMax:
		return weatherPenaltySnow
	case code >= wmoRainShowersMin && code <= wmoRainShowersMax:
		return weatherPenaltyRainShowers
	case code >= wmoThunderMin && code <= wmoThunderMax:
		return weatherPenaltyThunder
	default:
		return weatherPenaltyUnknown
	}
}
