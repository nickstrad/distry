# 10 — Problem 1 (easy): Perfect Point-to-Point Link

**Status:** Done.

**Depends on:** 07 (authorable & unit-testable immediately); 08+09 for platform E2E.
**Independent of** plans 11 and 12.
**Source:** Cachin/Guerraoui/Rodrigues, *Introduction to Reliable and Secure Distributed
Programming*, ch. 2 (fair-loss → stubborn → perfect links). Follow
`docs/problem-authoring.md` from plan 07.

## Statement (for `description.md`)

You are given a **fair-loss link**: `ctx.Send` may drop or duplicate messages but delivers
sent messages with nonzero probability (the simulator's DropRate/DuplicateRate model).
Implement a **perfect (reliable) point-to-point link** on top of it: an API
`SendReliably(to, payload)` such that

- **PL1 Reliable delivery**: if neither sender nor receiver crashes, every message sent is
  eventually delivered to the receiver's application layer.
- **PL2 No duplication**: no message is delivered (to the application) more than once.
- **PL3 No creation**: only messages that were sent are delivered.

Expected technique: retransmit on timer until ACK; sequence-number dedup on receipt.

## Manifest

`slug: perfect-link`, `difficulty: easy`, 2 nodes (sender/receiver), seeds `[1..5]`,
network: `DropRate: 0.3, DuplicateRate: 0.1, MinDelay: 10ms, MaxDelay: 100ms`, no
partitions, no crashes (keep the easy problem easy).

## Template (`template/solution.go`)

```go
package solution
// Node interface the user fills in; harness drives it.
type Deps struct{ Probe simtest.AppProbe } // Probe.Deliver(payload) = "delivered to app"
func New(deps Deps) sim.Node { /* TODO(user) */ }
// Harness calls a well-known message type "app_send" carrying payloads to transmit;
// document this contract in the template comments and description.
```

Keep the template ~30 lines with the handler skeleton laid out and TODOs where logic goes,
plus a comment showing how to unit-test locally with a fake Context.

## Harness (`harness/harness.go`)

- Seed-derived script: sender node receives K=20 `app_send` payloads at staggered virtual
  times; run until quiescent or MaxTime.
- Checkers: `NoDuplicateDelivery`, `NoCreation`, `AllDelivered` (liveness, all 20 by end).

## Reference + adversarial solutions (kept in `harness/testdata/`, used by tests only)

1. `correct/` — retransmit + ack + dedup. Must pass 100 seeds.
2. `bug-no-retransmit/` — fire-and-forget ⇒ `AllDelivered` liveness violation.
3. `bug-no-dedup/` — retransmits but delivers dupes ⇒ `NoDuplicateDelivery` safety
   violation with pinned event.

A Go test in the problem package compiles/runs these through `simtest.Execute` directly
(no platform needed) — this is the problem's own unit test and the pattern all future
problems copy.

## Testable outcome

- `go test ./problems/perfect-link/...` green (correct passes 100 seeds; both bugs caught
  deterministically).
- Through the UI (after 08/09): the shipped template compiles but fails liveness; writing
  the real algorithm turns all seeds green.
