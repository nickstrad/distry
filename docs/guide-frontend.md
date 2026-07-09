# Frontend Guide

The frontend uses Vite, React 19, Tailwind v4, and local shadcn/ui component source under
`frontend/src/components/ui/`.

## shadcn/ui

The project is initialized with `components.json`, the `@/*` alias, Lucide icons, and
TypeScript component output. Add or update components from `frontend/`:

```sh
npx shadcn@latest add button
npx shadcn@latest add --all
```

Keep generated component files close to registry output. Prefer composing or styling them
from app code before editing generated source.

The current `--all` registry output includes the shipped components under
`src/components/ui/`. Docs entries such as data table, date picker, and typography are
patterns rather than single generated files in the current CLI output; build those from the
generated primitives when needed.

## Tailwind v4

Tailwind is wired through `@tailwindcss/vite` in `frontend/vite.config.ts`. Theme tokens
live in `frontend/src/styles.css`, which starts with `@import "tailwindcss";` and uses
CSS-first `@theme inline` variables instead of a Tailwind v3 config file.

## TypeScript

The app can consume `.ts` and `.tsx` files through `tsconfig.json`,
`tsconfig.app.json`, and `tsconfig.node.json`. `allowJs` is enabled and `checkJs` is
disabled so existing `.jsx` files can migrate incrementally. New shared components should
be TypeScript when practical, especially if they wrap shadcn components.

Use `npm run typecheck` from `frontend/` to check TypeScript separately from the Vite
production build.
