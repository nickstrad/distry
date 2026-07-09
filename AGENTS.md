# Agent Notes

- When creating a new feature or other substantial change, always create a new worktree first.
- The normal workflow is to do the work inside that feature worktree, verify it there, and then merge the finished branch back into `main`.
- When creating a new worktree as part of doing new work, install both Go and Node dependencies in that worktree before implementing or testing changes. For this repo, that means running the appropriate Go dependency setup and `npm ci` in the root and `frontend/` package as needed.
