# Penalty curve calibration and default Threshold

Each Factor maps its raw Open-Meteo reading to a discomfort penalty, and the Score is their sum (see ADR-0001). We calibrate every curve against a single reference — **default Threshold = 100** — such that a "clearly bad" reading of any one Factor contributes roughly 100 on its own, so it can trip `nope` unaided (honoring the no-hard-caps decision in ADR-0001).

Continuous Factors without an authoritative scale use a flat comfortable band plus a **quadratic** ramp outside it; **AQI** uses the EPA category scale instead of an invented curve, and **weather** uses a lookup on the WMO weather code.

## Starting values (v1)

**Temperature** (°F), band **60–75**: penalty `0.25 · d²` where `d` = °F beyond the band. (~95°F or ~35°F ≈ 100.)

**Humidity** (%), band **30–60**: penalty `0.1 · d²` where `d` = % beyond the band. (~90% ≈ 90.)

**AQI** (EPA category → penalty): 0–50 → 0; 51–100 → 20; 101–150 → 60; 151–200 → 120; 201–300 → 300; 301+ → 600. (Unhealthy, 151+, trips `nope` alone.)

**Weather** (WMO code → penalty): clear 0–1 → 0; cloudy 2–3 → 5; fog 45,48 → 20; drizzle 51–57 → 30; rain 61–67 → 60; rain showers 80–82 → 70; snow 71–77 → 80; thunderstorm 95–99 → 120. (Thunderstorm trips `nope` alone.)

**Threshold**: default 100; user-tunable (the only tunable knob — curves are fixed, per CONTEXT.md).

## Consequences

- These are a deliberate first cut, not empirically validated. They are meant to be revised; the calibration principle ("anchor each curve so a clearly-bad single reading ≈ Threshold") is the durable part, the exact constants are not.
- Changing the default Threshold rescales how much combined mild discomfort is tolerated before `nope`, without touching any curve.
