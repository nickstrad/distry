# 06 — Simulator core (`pkg/sim`): deterministic seeded network simulator

**Depends on:** nothing (pure library; only assumes plan 01's repo layout).
**Enables:** 07, 08, 10–12. Can be built fully in parallel with plans 02–05.

**Status:** Done.

Implemented `pkg/sim` as a pure deterministic discrete-event simulator with virtual time,
seeded network delivery, drop/duplicate/partition behavior, timers, crash-stop faults,
cutoffs, panic capture, structured traces, per-node state access, package docs, and high
coverage tests.

## Goal

A deterministic discrete-event simulator for message-passing distributed algorithms.
Given (node implementations, topology, fault config, seed), a run is **bit-for-bit
reproducible**. This library is what user harnesses import, so its public API is the
platform's most important contract — design it for the problems in plans 10–12 and for
converting textbook algorithms (Lynch; Cachin/Guerraoui/Rodrigues) generally.

## Core model (discrete-event, single-threaded)

No goroutines in the simulated world, no wall-clock time. A priority queue of events
ordered by (virtualTime, sequence). The scheduler pops an event, delivers it to a node
handler, which may send messages / set timers — each becoming future events. Ties broken
by monotonically increasing sequence number ⇒ total determinism.

## Public API (target shape — refine while implementing, keep it this small)

```go
package sim

type NodeID int

// What a node implementation receives; all side effects go through Context.
type Context interface {
    Self() NodeID
    Nodes() []NodeID                      // all node IDs in the cluster
    Send(to NodeID, msg Message)          // async, unreliable per NetworkConfig
    SetTimer(d time.Duration, name string)// fires HandleTimer after virtual d
    Now() time.Time                       // virtual time
    Rand() *rand.Rand                     // node-scoped seeded RNG (determinism!)
    Log(format string, args ...any)       // recorded into the trace
}

type Message struct { Type string; Payload []byte } // payload = encoding/json or gob

type Node interface {
    Init(ctx Context)
    HandleMessage(ctx Context, from NodeID, msg Message)
    HandleTimer(ctx Context, name string)
}

type NetworkConfig struct {
    MinDelay, MaxDelay time.Duration
    DropRate, DuplicateRate float64
    Partitions []Partition // scheduled partitions: {At, Heal time; Groups [][]NodeID}
}

type FaultConfig struct {
    Crashes []Crash // {Node NodeID, At time.Duration} — crash-stop for MVP
}

type Runner struct{ ... }
func NewRunner(cfg Config) *Runner // Config: Seed int64, NumNodes, Network, Faults, MaxTime, MaxEvents
func (r *Runner) Run(newNode func(id NodeID) Node) *Result
```

`Result` carries: final status (completed / max-events / max-time / panic with node+stack),
the full **event trace**, and per-node state access hook for checkers (plan 07).

## Event trace (the replay/debug artifact)

Every occurrence appends a structured `TraceEvent`:
`{Seq, Time, Kind: send|deliver|drop|duplicate|timer_set|timer_fire|crash|partition|log|checker, Node, Peer, MsgType, Detail}`.
JSON-serializable — plan 09 renders it, plan 08 stores it. The trace of a run is part of
determinism: same seed ⇒ identical trace.

## Steps

1. Event queue (container/heap) + scheduler loop + virtual clock. Test: events fire in
   time order; ties in insertion order.
2. Message delivery through `NetworkConfig`: delay uniform in [min,max] from the seeded
   RNG; drop/duplicate decisions from the same RNG; partitions block delivery between
   groups (messages sent across a partition are dropped, traced as `drop`).
3. Timers, crash-stop faults (crashed nodes receive nothing, their timers are discarded),
   `MaxEvents`/`MaxTime` cutoffs (prevent user infinite loops at the simulation level).
4. Panic capture: a panicking node handler fails the run gracefully with node ID + stack
   in the Result (user code quality varies!).
5. Trace recording. Golden test: fixed seed + toy ping-pong nodes ⇒ trace matches golden
   file exactly; two runs with the same seed produce identical traces; different seeds
   produce different delivery orders.
6. `doc.go` with a worked example (two nodes, retransmit until ack) — this doubles as the
   documentation problem-authors read.

## Testing / DI notes

This package must have **zero** dependencies on db/http/auth. All randomness from
`rand.New(rand.NewSource(seed))` threaded through; all time virtual. Coverage target:
this is the highest-value test surface in the repo — aim ≥90% on the scheduler/network.

## Testable outcome

`go test ./pkg/sim/...` green, including the determinism golden test (same seed ⇒
identical trace; N=100 seeds smoke test with a ping-pong algorithm never deadlocks or
panics).
