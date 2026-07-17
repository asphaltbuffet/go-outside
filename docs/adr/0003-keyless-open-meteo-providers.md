# Keyless Open-Meteo provider stack over API-key services

All external data comes from keyless HTTP providers so the tool runs with zero configuration: **Open-Meteo** supplies weather (temperature, humidity, precipitation/weather codes), air quality (AQI), and postal-code→coordinate geocoding; a keyless **IP-geolocation service** (ipapi.co / ip-api.com) auto-detects the user's Location, returning both coordinates and postal code in one call. The HTTP client is **resty**.

We chose keyless providers because a comfort-check CLI's value is `go-outside` working instantly on any machine — requiring users to register for an API key and store a secret would be the single biggest UX tax and would complicate the existing config. The accepted cost is that Open-Meteo takes coordinates rather than a postal code, so the postal-code-canonical Location (see CONTEXT.md, ADR pipeline) must be geocoded to lat/long before Conditions are fetched; we keep that geocode step in-family via Open-Meteo's own geocoding API.

## Considered Options

- **OpenWeatherMap / WeatherAPI.com** — single provider for weather + AQI, accepts postal code directly (no geocode step). Rejected: requires a free API key, i.e. signup, a stored secret, and non-zero-config setup.
- **Open-Meteo + IP geolocation (chosen)** — fully keyless; costs one extra geocode call to bridge postal code → coordinates.

## Consequences

- Because every provider is coordinate-based, the postal-code → coordinate geocode is a mandatory pipeline stage, not an optional convenience.
- No secret ever enters config; failures are network/availability failures, surfaced as errors (exit `2`), never as `nope`.
