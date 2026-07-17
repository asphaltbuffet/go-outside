# Verdict encoded as process exit status

The tool's exit code carries the Verdict so it composes as a shell predicate (`go-outside && go_for_run`): **`ok` exits `0`, `nope` exits `1`, and an error (Location or Conditions could not be determined) exits `2`.** This holds in every output mode, including `--json`.

This deliberately breaks the usual "exit 0 = the program ran successfully" reading — a successful assessment that returns `nope` still exits non-zero. We accept that surprise because it makes the Verdict directly usable in shell control flow without parsing stdout, and the three-way space (`0` good / `1` bad / `2` couldn't-assess) keeps "conditions are bad" cleanly distinct from "the tool failed."

## Consequences

- `nope` must never be conflated with an error; errors are reserved for exit `2` and written to stderr.
- Scripts and CI that treat any non-zero exit as failure will see `nope` as a failure — this is intended, but callers who only want to distinguish success/error must check for exit `2` specifically.
