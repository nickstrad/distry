# 11 — Problem 2 (easy): LCR Leader Election on a Ring

**Status:** Done.

**Depends on:** 07; 08+09 for platform E2E. Independent of plans 10 and 12.
**Source:** Lynch, *Distributed Algorithms*, ch. 3.3 (LeLann–Chang–Roberts). Follow
`docs/problem-authoring.md`.

## Statement

N nodes are arranged in a unidirectional ring; each node knows only `Self()`, its
successor (provided by the harness as a dep), and can send messages. Node IDs are unique
and comparable. Elect a leader:

- **EL1 Safety (single leader)**: at most one node ever announces itself leader, and every
  node that announces a leader announces the same one (the maximum ID).
- **EL2 Liveness**: eventually every non-crashed node announces the leader.

Announcement via `deps.Probe.Elected(leader NodeID)`. Classic LCR: forward your ID; on
receiving an ID, forward if greater than yours, swallow if smaller, declare victory if it
equals yours; winner circulates a "leader" message.

## Twist vs. problem 10 (what makes it a distinct learning slice)

The link here is **reliable but with variable delays** (`DropRate: 0`), so the challenge
is the algorithm itself, not retransmission — teaches that problems compose: later
problems can layer LCR *over* lossy links. Ring size varies by seed (3–8 nodes, derived
from seed in the harness) so hardcoded topologies fail.

## Manifest

`slug: lcr-election`, `difficulty: easy`, seeds `[1..5]`, network
`MinDelay: 5ms, MaxDelay: 200ms, DropRate: 0`, no crashes.

## Template

`solution.go` with `Deps{ Probe simtest.AppProbe; Successor sim.NodeID }` and a skeleton
`sim.Node`. Comments state the message-forwarding contract and that IDs are `int`-ordered.

## Harness & checkers

- Build ring of seed-derived size; wire each node's `Successor` dep.
- Checkers: `SingleLeader` (safety: all `elected` records agree and equal max ID — reuse
  from plan 07, parameterized with expected winner), `AllAnnounced` (liveness).

## Adversarial solutions (`harness/testdata/`)

1. `correct/` — LCR; passes 100 seeds across ring sizes.
2. `bug-everyone-leader/` — each node announces itself ⇒ `SingleLeader` violation.
3. `bug-swallow-all/` — never forwards ⇒ liveness violation.

## Testable outcome

`go test ./problems/lcr-election/...` green with the same structure as plan 10; via the
UI, the correct LCR implementation passes all seeds including different ring sizes.
