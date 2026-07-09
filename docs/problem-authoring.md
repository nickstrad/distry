# Problem Authoring

Distry problems are a manifest, a user-facing description, starter templates, and a
hidden harness. The harness is where textbook properties become executable checks.

## Harness shape

Each harness exposes:

```go
func Run(seed int64) *simtest.Report
```

Inside `Run`, build a `sim.Config`, construct a `simtest.Probe`, wire user nodes with the
probe as a dependency, then call `simtest.Execute` with safety and liveness checkers.

## Choosing observations

Have user code report externally visible actions through the probe, such as:

- `send`: an application message entered the algorithm.
- `deliver`: a node delivered a message to the application.
- `decide`: a node decided a consensus value.
- `elected`: a node announced a leader.

Checkers should validate these observations, not private node fields. This matches the
distributed-systems property statements users learn from: no duplicate delivery, no
creation, agreement, single leader, and eventual termination.

## Checker selection

Use safety checkers for properties that can be violated during a run:

- `simtest.NoDuplicateDelivery`
- `simtest.NoCreation`
- `simtest.AgreementOnDecision`
- `simtest.SingleLeader`

Use liveness checkers for bounded "eventually by the end of this seeded run" properties:

- `simtest.AllDelivered`
- `simtest.TerminationByEnd`

Reports are JSON with schema version `v: 1`, the seed, pass/fail, violations, summary
stats, and a trace on failures. The runner stores this report and the UI renders it.
