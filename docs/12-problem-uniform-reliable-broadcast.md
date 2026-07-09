# 12 — Problem 3 (medium): Uniform Reliable Broadcast under crash faults

**Status:** Done.

**Depends on:** 07; 08+09 for platform E2E. Independent of plans 10 and 11.
**Source:** Cachin/Guerraoui/Rodrigues ch. 3 (Majority-Ack Uniform Reliable Broadcast).
Follow `docs/problem-authoring.md`.

Implemented the `uniform-reliable-broadcast` problem package with manifest, statement,
starter template, hidden harness, a reusable `UniformAgreement` checker, and adversarial
testdata covering the majority-ack reference solution, eager delivery, and missing relay.

## Statement

N nodes (N=5), any minority may **crash-stop** mid-run. Links are perfect (the harness
grants reliable delivery — conceptually the layer built in problem 10; delays vary, no
drops). Implement `Broadcast(payload)` / deliver-to-app with **uniform** reliable
broadcast semantics:

- **URB1 Validity**: if a correct node broadcasts m, it eventually delivers m.
- **URB2 No duplication**: no node delivers m more than once.
- **URB3 No creation**: only broadcast messages are delivered.
- **URB4 Uniform agreement**: if **any** node (even one that later crashes) delivers m,
  then every correct node eventually delivers m.

URB4 is the medium-difficulty heart: you may not deliver until a **majority** has
acknowledged/relayed m. A naive eager-relay-then-deliver passes URB1–3 but violates URB4
when the origin delivers and crashes before its relay propagates — the harness's fault
schedules are crafted (seed-derived) to include exactly that scenario.

## Manifest

`slug: uniform-reliable-broadcast`, `difficulty: medium`, seeds `[1..8]`, N=5, network
`MinDelay: 5ms, MaxDelay: 150ms, DropRate: 0`, faults: up to 2 seed-scheduled crashes,
with at least two seeds timed to crash a node immediately after its first deliveries
(targeted adversarial schedules — see harness note).

## Harness notes

- Script: 3 designated nodes each broadcast several payloads at staggered times; crashes
  injected per seed. Some crashes must be *reactive*: the harness watches the probe ledger
  and schedules "crash node X right after its first `deliver`" — add this small hook to
  `FaultConfig` if plan 06 didn't include it (`CrashAfterProbe{Node, Action, Count}`);
  it's the only simulator extension this problem needs, keep it generic.
- Checkers: `NoDuplicateDelivery`, `NoCreation`, `ValidityDelivered`, and new
  `UniformAgreement` (safety-at-end formulation: every message delivered by *anyone* —
  including crashed nodes' pre-crash deliveries — is delivered by every correct node by
  end; report the message and the node that missed it).

## Template

`Deps{ Probe simtest.AppProbe; N int }` (probe has `Deliver`; harness sends `app_broadcast`
messages to trigger broadcasts). Template skeleton suggests state: `pending`, `acks[m]`,
`delivered` sets — mirroring the book's pseudocode variables so book-to-code mapping is
obvious.

## Adversarial solutions (`harness/testdata/`)

1. `correct/` — majority-ack URB; passes 200 seeds (higher count: fault interleavings).
2. `bug-eager-deliver/` — delivers on first receipt ⇒ `UniformAgreement` violation on the
   crafted seeds (this test proves the fault schedules actually discriminate).
3. `bug-no-relay/` — origin-only sends ⇒ validity/agreement violations under crashes.

## Testable outcome

`go test ./problems/uniform-reliable-broadcast/...` green; critically, `bug-eager-deliver`
**must** be caught (if it passes, the fault schedule isn't adversarial enough — iterate on
seeds before shipping). Via UI: correct solution green on all 8 seeds.
