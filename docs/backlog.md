# Backlog

Distry's MVP is a Go-only distributed-systems practice platform with deterministic
simulation, persisted solutions, seeded submissions, and a browser workspace.

Next roadmap items:

- Replace the in-house email/password auth with a hosted OAuth provider while preserving
  the current auth middleware boundary.
- Add stronger runner sandboxing for untrusted submissions.
- Add more languages by implementing the `submissions.LanguageRunner` contract.
- Grow the problem set beyond Perfect Link, LCR Election, and Uniform Reliable Broadcast.
- Add richer trace replay controls and seed pinning in the UI.
