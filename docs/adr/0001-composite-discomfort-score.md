# Composite discomfort Score over per-factor gating

The Verdict is derived from a single open-ended **Score** — the sum of each Factor's discomfort contribution (temperature, humidity, AQI, weather) — compared against one **Threshold**, rather than from per-factor gating checks where any one out-of-range Factor alone forces `nope`.

We chose the composite because it lets several mildly-uncomfortable Factors combine to tip the Verdict to `nope` even when no single Factor is bad enough to fail on its own, which matches how overall comfort actually degrades. The known cost is that a single catastrophic Factor (e.g. hazardous AQI) must be forced to dominate through a steep penalty curve rather than a hard gate; we accept this and deliberately use **no hard caps**, keeping exactly one mental model (Score vs. Threshold).

## Considered Options

- **Per-factor gating** — Verdict is `nope` if any Factor leaves its ok-range; verbose output shows the offending Factor. Rejected: no clean way to let two mediocre Factors combine, and weighting is moot.
- **Composite score (chosen)** — heterogeneous inputs mapped onto one discomfort axis and summed.
