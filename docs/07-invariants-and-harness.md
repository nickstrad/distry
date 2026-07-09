# 07 — Invariant checkers & harness contract (`pkg/simtest`)

**Depends on:** 06.
**Enables:** 08 (the runner executes harnesses), 10–12 (each problem authors one).

## Goal

A small framework on top of `pkg/sim` that (a) expresses safety and liveness properties as
checkers, (b) defines the **harness contract** every problem implements, and (c) emits a
machine-readable verdict the runner (plan 08) stores and the UI (plan 09) renders.

## Checkers

```go
package simtest

// Safety: evaluated on every event (or on state change); a violation fails immediately
// and pins the violating event's Seq in the report.
type SafetyChecker interface {
    Name() string
    OnEvent(ev sim.TraceEvent, cluster ClusterView) *Violation // nil = ok
}

// Liveness (bounded): evaluated when the run ends; "eventually" means "by end of run".
type LivenessChecker interface {
    Name() string
    AtEnd(cluster ClusterView, res *sim.Result) *Violation
}

type Violation struct { Checker, Message string; EventSeq int64 /* -1 for liveness */ }

// ClusterView lets checkers observe node-reported facts without touching node internals.
```

**Observation mechanism:** algorithms report externally visible actions (deliver(m),
decide(v), elected(l)) via a `simtest.Probe` the harness injects into user constructors —
e.g. `probe.Record(node, "deliver", payload)`. Checkers read the probe's ledger. This keeps
user code honest: checkers validate what the algorithm *claims to its application layer*,
exactly how the textbooks specify properties ("no message delivered twice", "every correct
process eventually decides").

Ship reusable checkers with tests: `NoDuplicateDelivery`, `NoCreation` (only sent payloads
delivered), `AllDelivered(from correct nodes)`, `AgreementOnDecision`, `SingleLeader`,
`TerminationByEnd`.

## Harness contract (what a problem's hidden `harness/harness.go` implements)

```go
package harness // compiled together with user files by plan 08

// The runner invokes this via a fixed main() shim (see plan 08).
func Run(seed int64) *simtest.Report

// Inside: build sim.Config (topology/network/faults may themselves vary by seed),
// construct user nodes via the problem's constructor — user code implements an
// interface declared in the problem's template, e.g.:
//   func New(deps solution.Deps) solution.Node
// inject Probe, register checkers, run, return report.
```

`simtest.Report` (JSON): `{Seed, Passed, Violations []Violation, Stats {Events, VirtualDuration, MessagesSent/Dropped}, Trace []sim.TraceEvent}` —
trace included only on failure or when requested (size control).

**DI is load-bearing here:** user templates depend only on interfaces (`sim.Context`
passed to handlers; constructor receives declared deps), never on concrete
simulator/harness types — that's what lets us swap fault schedules per seed without users
special-casing, and what makes user code unit-testable *by the user* against fakes.

## Steps

1. Implement Probe + ClusterView + checker interfaces + the reusable checkers above.
2. `simtest.Execute(cfg sim.Config, newNode ..., checkers ...) *Report` — runs the sim,
   pumps events through safety checkers (fail fast), runs liveness checkers at end.
3. Report JSON schema (versioned: `"v":1`) — this is the runner↔platform wire contract.
4. Write an **example problem end-to-end in tests**: broadcast-with-drops toy, one correct
   solution, two deliberately buggy solutions (one violating safety, one liveness); assert
   each is caught with the right checker name and a pinned event seq.
5. `docs/problem-authoring.md` (short): how to go from a textbook property list to
   manifest + template + harness + checkers. Plans 10–12 follow it.

## Testable outcome

`go test ./pkg/simtest/...` green: correct toy solution passes all checkers across 50
seeds; each planted bug is detected deterministically (same seed ⇒ same violation and
event seq).
