# Solving Problems

The normal loop is:

1. Pick a problem from the problem list.
2. Read the statement and invariants before editing code.
3. Fill in the starter files in the editor.
4. Click **Save**.
5. Click **Run**.
6. Read each seed result in the results panel.
7. Expand a failing seed, inspect the violation, and walk the trace around the pinned
   event.
8. Click **Replay** on a failed seed when you want to rerun that exact historical
   submission snapshot with a full trace.
9. Re-run the same solution until every configured seed passes.

## Reading A Failure

A failure card names the checker, shows the checker message, and pins the simulator event
sequence where the failure became observable. Safety failures have a positive event
number; liveness failures use event `-1` because they are checked after the seeded run
ends.

Use the trace table backwards from the pinned event:

- Filter by node when one process looks suspicious.
- Filter by kind, such as `send`, `deliver`, `drop`, `duplicate`, `timer_set`, or
  `timer_fire`, when you want one class of event.
- Compare `node`, `peer`, `msgType`, and `detail` to understand causality.

The same code, seed, and problem configuration reproduce the same trace. If seed `3`
fails once, seed `3` should fail the same way until your code changes. Replay uses the
files stored with the selected submission, not your current editor draft, so it is safe to
debug old results after making new edits.

Very large traces are capped before storage. When this happens, the UI shows a truncated
trace notice; the seed, violation, and early event context are still preserved.

## Local Unit Tests

You can unit-test solution logic before pressing Run because templates receive their
dependencies explicitly. The compiled example in `examples/perfectlinklocal` constructs a
solution with `simtest.NewProbe()` and drives it with a tiny fake `sim.Context`.

Run it with:

```sh
go test ./examples/perfectlinklocal
```

The important pattern is:

```go
probe := simtest.NewProbe()
node := solution.New(solution.Deps{Probe: probe})
ctx := newFakeContext(0, []sim.NodeID{0, 1})

node.Init(ctx)
node.HandleMessage(ctx, 0, sim.Message{Type: "app_send", Payload: []byte("hello")})
```

From there, assert on calls captured by your fake context and probe records. The platform
harness will still run many seeds; local tests make the basic state machine faster to
debug.

## Seeds And Determinism

A seed controls virtual network delays, drops, duplication, and every node's scoped random
source. The simulator also uses virtual time, so wall-clock speed does not affect event
order.

Use custom or repeated seed runs when a failure is narrow: type one or more seeds in the
input next to **Run**, separated by commas or spaces, such as `3` or `3, 8, 13`. Custom
runs accept 1-20 non-negative seeds. Keep the failing seed fixed, add local tests for the
behavior, then re-run the platform problem after each fix.
