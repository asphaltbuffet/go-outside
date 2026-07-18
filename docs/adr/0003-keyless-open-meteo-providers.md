# Keyless Open-Meteo provider stack over API-key services

All external data comes from keyless HTTP providers so the tool runs with zero configuration: **Open-Meteo** supplies weather (temperature, humidity, precipitation/weather codes), air quality (AQI), and postal-code→coordinate geocoding; a keyless **IP-geolocation service** auto-detects the user's Location, returning both coordinates and postal code in one call. The HTTP client is **resty**.

**Auto-detect provider: ip-api.com** (chosen over ipapi.co). ipapi.co's free tier has a low daily cap and throttles bursts aggressively, so the default no-argument invocation frequently returned HTTP 429. ip-api.com's free tier allows ~45 requests/minute, eliminating the throttling for normal use. Its trade-offs, accepted: (1) the free endpoint is **HTTP-only** — HTTPS requires a paid plan — so the geolocation lookup travels in plaintext (acceptable for a low-stakes comfort check with no secrets); (2) it signals errors via a `status: "fail"` body field with HTTP 200, not an HTTP error status, so the client checks the body explicitly. A 429 from any provider is surfaced with a message pointing the user to retry or use `--location`.

We chose keyless providers because a comfort-check CLI's value is `go-outside` working instantly on any machine — requiring users to register for an API key and store a secret would be the single biggest UX tax and would complicate the existing config. The accepted cost is that Open-Meteo takes coordinates rather than a postal code, so the postal-code-canonical Location (see CONTEXT.md, ADR pipeline) must be geocoded to lat/long before Conditions are fetched; we keep that geocode step in-family via Open-Meteo's own geocoding API.

## Considered Options

- **OpenWeatherMap / WeatherAPI.com** — single provider for weather + AQI, accepts postal code directly (no geocode step). Rejected: requires a free API key, i.e. signup, a stored secret, and non-zero-config setup.
- **Open-Meteo + IP geolocation (chosen)** — fully keyless; costs one extra geocode call to bridge postal code → coordinates.
- **ipapi.co for auto-detect** — HTTPS and a clean JSON shape, but its free tier rate-limits the default invocation (HTTP 429). Rejected in favor of ip-api.com's more generous keyless tier.

## Consequences

- Because every provider is coordinate-based, the postal-code → coordinate geocode is a mandatory pipeline stage, not an optional convenience.
- No secret ever enters config; failures are network/availability failures, surfaced as errors (exit `2`), never as `nope`.
