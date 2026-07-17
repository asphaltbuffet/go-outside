# go-outside

A command-line tool that assesses current outdoor conditions at the user's location and returns a go/no-go verdict on whether it's comfortable to go outside.

## Language

**Location**:
The place whose outdoor conditions are being assessed. Canonically identified by a postal code (e.g. `22901`) — that is what the tool displays and what conditions are fetched for. By default the tool detects it via IP geolocation; any coordinates produced during detection are an intermediate artifact, not the Location's identity.
_Avoid_: place, area, position, coordinates

**Conditions**:
The set of current outdoor measurements at a Location that feed the assessment — temperature, humidity, air quality (AQI), and weather (rain/snow/etc).
_Avoid_: weather, environment, readings, data

**Verdict**:
The go/no-go result of assessing Conditions: `ok` (comfortable to go outside) or `nope` (outside the comfortable range). Derived by comparing the Score against the Threshold. A Verdict exists only when both Location and Conditions were successfully obtained — an inability to determine either is an error, never `nope`. The Verdict is also the tool's exit status: `ok` exits `0`, `nope` exits `1`, and an error (couldn't assess) exits `2`.
_Avoid_: result, status, outcome, answer, unknown

**Score**:
A single unitless discomfort composite: the sum of every Factor's contribution. `0` is ideal; higher is worse; open-ended (not normalized to a fixed maximum). Shown in parentheses in verbose output, e.g. `nope (999)`.
_Avoid_: rating, index, value, discomfort index, comfort score

**Factor**:
One Condition input (temperature, humidity, AQI, or weather) together with the discomfort contribution it adds to the Score. A Factor contributes `0` while within its comfortable band and a rising penalty as it leaves that band; categorical weather contributes via a lookup rather than a band.
_Avoid_: input, metric, dimension, component

**Threshold**:
The Score boundary that separates `ok` from `nope`. The Verdict is `nope` when Score exceeds the Threshold. Defaults to a value comfortable for most people, and is the single user-tunable knob — the per-Factor penalty curves are fixed defaults, not configurable.
_Avoid_: limit, cutoff, boundary, range
