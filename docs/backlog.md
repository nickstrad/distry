# Backlog

Distry's MVP is a Go-only distributed-systems practice platform with deterministic
simulation, persisted solutions, seeded submissions, and a browser workspace.

Next roadmap items:

- Replace the in-house email/password auth with a hosted OAuth provider while preserving
  the current auth middleware boundary.
- Container sandboxing for the runner so untrusted submissions execute with filesystem,
  CPU, memory, process, and network isolation.
- A second language runtime by implementing the `submissions.LanguageRunner` contract.
- Richer trace visualization, including Lamport timelines or sequence diagrams.
- Problem progression and unlocks for guided curricula.
- Seed fuzzing that searches for violating seeds and stores minimized reproductions.
- Importing more book chapters as authored problem packages.
