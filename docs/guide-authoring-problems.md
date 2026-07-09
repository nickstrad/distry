# Authoring Problems

This checklist turns a textbook algorithm into a Distry problem. The Perfect
Point-to-Point Link problem is the running example.

## Checklist

1. Extract the properties from the source material. Classify each as safety or liveness.
   Perfect Link has safety properties for no duplicate delivery and no creation, plus a
   liveness property that every sent payload eventually reaches the receiver.
2. Map properties to existing `simtest` checkers first. Use `NoDuplicateDelivery`,
   `NoCreation`, `AgreementOnDecision`, `SingleLeader`, `AllDelivered`, and
   `TerminationByEnd` when they match. Write a new checker only for a genuinely new
   property, and add unit tests because it becomes shared library API.
3. Design the environment. Choose topology, `sim.NetworkConfig`, seed-derived variation,
   and fault schedules that discriminate between correct and buggy implementations. The
   URB eager-deliver case in `docs/12-problem-uniform-reliable-broadcast.md` is the
   canonical warning: a schedule that never exposes the bad interleaving is not a real
   test.
4. Write the manifest, description, template, and harness. The template should include
   the solution skeleton, `Deps`, and a comment showing that local tests can drive the node
   with a fake `sim.Context`.
5. Prove the problem. Add `harness/testdata/correct` and one planted bug package per
   property. The correct solution should pass at least 100 seeds; each planted bug should
   fail deterministically with the expected checker.
6. Run the checks: `go test ./problems/<slug>/harness`, `go test ./pkg/simtest`, and
   `go test ./...`.

## Manifest Fields

| Field                  | Meaning                                            |
| ---------------------- | -------------------------------------------------- |
| `slug`                 | Stable URL and storage key, such as `perfect-link` |
| `title`                | Display title                                      |
| `difficulty`           | UI label, usually `easy`, `medium`, or `hard`      |
| `language`             | Runner key; currently `go`                         |
| `tags`                 | Search and display tags                            |
| `order`                | Sort order in the problem list                     |
| `entrypoint`           | Primary template file                              |
| `templates`            | User-editable starter files                        |
| `runs.seeds`           | Seeds executed by the platform                     |
| `runs.timeout_seconds` | Per-seed runner timeout                            |

## Public API Summary

`pkg/sim`:

- `Config`, `NetworkConfig`, `FaultConfig`, `Partition`, and `Crash` describe the seeded
  run.
- `Node` is the user/harness process interface: `Init`, `HandleMessage`, and
  `HandleTimer`.
- `Context` exposes `Self`, `Nodes`, `Send`, `SetTimer`, `Now`, `Rand`, and `Log`.
- `TraceEvent` records `send`, `deliver`, `drop`, `duplicate`, `timer_set`,
  `timer_fire`, `crash`, `partition`, `log`, and checker events.

`pkg/simtest`:

- `Probe` records externally visible actions such as `send`, `deliver`, `decide`, and
  `elected`.
- `Execute` runs a simulator config and returns a report.
- `SafetyChecker` and `LivenessChecker` define reusable invariants.
- Built-in checkers cover duplicate delivery, no creation, all delivered, agreement,
  single leader, and termination by end.

## Report JSON Schema V1

```json
{
  "v": 1,
  "seed": 1,
  "passed": false,
  "violations": [
    { "checker": "NoDuplicateDelivery", "message": "...", "eventSeq": 12 }
  ],
  "stats": {
    "events": 42,
    "virtualDuration": 1000000000,
    "messagesSent": 10,
    "messagesDropped": 2
  },
  "trace": [],
  "truncated": true
}
```

Passing reports omit `trace` unless the harness requests `FullTrace`. Failed reports
include the trace so the UI can show the failure context. The platform caps stored traces
and sets `truncated: true` when the stored event list was shortened.
