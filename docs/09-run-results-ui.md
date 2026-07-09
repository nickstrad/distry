# 09 — Run & results UI: Run button, polling, per-seed results, trace viewer

**Depends on:** 04, 05, 08.
**Enables:** the MVP loop; plans 10–12 are validated through this UI.
**Status:** Done.

## Goal

From the problem workspace: press **Run** (auto-saves first), optionally pin custom seeds,
watch status progress, then see per-seed pass/fail, invariant violations, compile errors,
and an event-trace viewer for failing seeds.

## Steps

1. ✅ **Run flow**: enable the Run button (plan 04 placeholder). Click ⇒ save current files
   (reuse plan 05 PUT) ⇒ `POST /api/problems/:slug/run` ⇒ store submission id ⇒ poll
   `GET /api/submissions/:id` every ~1.5s until terminal. Disable Run while in flight
   (server 409 also handled gracefully). If the seed input is filled, send
   `POST .../run { seeds: [...] }`.
2. ✅ **Results panel** (bottom pane of the workspace, resizable):
   - Status line: queued → compiling → running → PASSED/FAILED/ERROR with timing.
   - `error`: compile output in a monospace block (this is the tight feedback loop —
     make it good).
   - Per-seed row: seed number, ✓/✗, stats (events, messages sent/dropped, virtual
     duration). Failed seeds expandable.
   - Violation card: checker name, message, "at event #Seq".
   - Replay button on failed seed rows: `POST /api/submissions/:id/replay` reruns that
     seed from the historical submission snapshot with full trace.
3. ✅ **Trace viewer** (failing seeds): virtualized table of TraceEvents
   (Seq | time | kind | node | peer | msg type | detail), color-coded by kind
   (send/deliver/drop/crash/log/…), filter by node and kind, and auto-scroll-to +
   highlight the violating event seq. Keep it a table — no graph rendering in MVP. If a
   stored report has `truncated: true`, show an explicit trace-truncated notice.
4. ✅ **Submission history**: small list under the results panel from
   `GET /api/problems/:slug/submissions`; clicking an old one loads its results
   (read-only). Show which seed set was used.
5. ✅ **Empty/edge states**: no run yet; run while unsaved; session expired mid-poll (401 →
   login redirect preserving the return path).

## Testing notes

- Vitest: results panel rendering from fixture `Report` JSON (passed, failed-with-trace,
  compile-error variants); poll hook with mocked timers reaches terminal state and stops
  polling.
- Manual E2E (the money path): open problem → break the template intentionally → Run →
  see compile error; write a wrong-but-compiling solution → see violation + trace; correct
  solution → all seeds green.

## Testable outcome

The full MVP loop works in the browser end-to-end: edit → Run → per-seed results, with a
readable trace pinpointing the violating event for failures and replay for exact failed
seed reproduction.
