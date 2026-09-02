# Web Architecture — Feature-Sliced Design (FSD)

This frontend (Next.js App Router + MUI) is organized with
[Feature-Sliced Design](https://feature-sliced.design/). The goal is a clear,
one-directional dependency graph so every file has an obvious home and imports
only "downward".

## Layers

Dependencies flow **top → bottom only**. A layer may import from layers below
it, never above or sideways at the same level (except `shared`, which anything
may use).

| Layer    | Path            | Responsibility                                                                                                                         |
| -------- | --------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| app      | `src/app/`      | Composition root: providers, theme, app shell, global styles                                                                           |
| views    | `src/views/`    | Route page bodies (the "pages" layer; named `views` to avoid clashing with Next's `app/`)                                              |
| widgets  | `src/widgets/`  | Self-contained composite blocks (app layout, dashboard, resource list scaffold, version footer)                                        |
| features | `src/features/` | User interactions (edit agent / group / remote config / package, apply remote config, select namespace)                                |
| entities | `src/entities/` | Domain models: types, data helpers, and domain React contexts                                                                          |
| shared   | `src/shared/`   | Reusable infra with no domain knowledge: `api`, `lib` (pure utils), `ui` (the design-system kit), `preferences` (cross-cutting config) |

`web/app/` is reserved for **Next.js routing only** — route handlers
(`api/proxy`, `api/session`), `layout.tsx`, the dashboard RSC loader
(`page.tsx`), and thin `page.tsx` wrappers that re-export a view:

```ts
// web/app/agents/page.tsx
export { default } from '@views/agents';
```

## Path aliases

Defined in both `tsconfig.json` and `vitest.config.ts`:

```
@app/*      → src/app/*
@views/*    → src/views/*
@widgets/*  → src/widgets/*
@features/* → src/features/*
@entities/* → src/entities/*
@shared/*   → src/shared/*
@/*         → ./*          (web root; used by app/ route files)
```

## Slice structure & public API

Each slice exposes a public API via `index.ts` (the barrel). **Import from the
slice barrel, not its internal files** — e.g. `import { PageHeader } from
'@shared/ui'`, not `@shared/ui/PageHeader`. Internal segments follow the FSD
convention `model/` (types, stores, contexts), `api/` (data access), `ui/`
(components), `lib/` (slice-local helpers).

```
src/entities/agent/
  model/types.ts          # Agent, AgentSpec, AgentStatus, ...
  api/delete-agent.ts     # deleteAgent(), agentDeleteConfirmMessage()
  lib/capabilities.ts     # capabilityNames()
  index.ts                # public API
```

## Where things live

- **Domain types** are split per entity under `entities/<name>/model/types.ts`.
  Cross-cutting structural types shared by several entities (`ListResponse`,
  `Condition`, `Attributes`, `ConnectionSettings`, `AgentRemoteConfigSpec`) live
  in `shared/api/model/types.ts`.
- **Data access**: the browser client (`api`), SWR helpers, client session
  cookie sync, and localStorage auth are in `shared/api` and exported from the
  `@shared/api` barrel. The **server-only** RSC client is in
  `shared/api/server.ts` and is imported directly via `@shared/api/server` — it
  is intentionally **not** re-exported from the `@shared/api` barrel so that
  importing the barrel from a Client Component never pulls `server-only` into the
  browser bundle.

## Intentional exceptions

- **Preferences live in `shared`**, not as a feature. Timezone / time-format are
  app-wide cross-cutting settings, and the generic `shared/ui` `TimeDisplay`
  consumes the preferences context. Putting them in a feature would create a
  `shared → features` back-edge. So the model, provider, `TimeDisplay`, and the
  selector controls all sit in `shared/preferences`.
- **Domain contexts live in `entities`**, not `app`. `AuthProvider`/`useAuth`,
  `PermissionsProvider`/`usePermissions` (in `entities/session`) and
  `NamespaceProvider`/`useNamespace` (in `entities/namespace`) are composed by
  `src/app/app-shell.tsx`, but defined in entities because features and widgets
  consume their hooks — they must sit below those layers.
- A few **entities reference sibling entities** directly (e.g. `entities/user`
  uses `Role`/`RoleBinding`; `entities/namespace` uses `useAuth`). Strict FSD
  discourages entity↔entity imports; we allow these narrow, well-understood
  cases instead of introducing the `@x` cross-import indirection.

## Adding things

- **New entity**: `src/entities/<name>/{model,api,...}/` + `index.ts`. Put types
  in `model/types.ts`; reuse shared envelope types from `@shared/api`.
- **New feature** (an interaction): `src/features/<verb-noun>/ui/` + `index.ts`.
  May import entities and shared.
- **New route**: add `src/views/<name>/ui/<Name>Page.tsx` + `index.ts`, then a
  thin `web/app/<route>/page.tsx` that re-exports the view's default.

## Design system

The UI is **Tailwind CSS v4 + Radix primitives**, in the shadcn/ui style: the
component source lives in this repo (`src/shared/ui/`) rather than in a
component library, so every element is ours to restyle.

- **Tokens** are CSS variables in `src/app/styles/globals.css` — `--background`,
  `--foreground`, `--primary`, `--muted`, `--border`, … each mapped into
  Tailwind through `@theme inline`. Components reference the token utilities
  (`bg-card`, `text-muted-foreground`), never raw colours, so a palette change
  is a one-file edit.
- **Dark mode** is a `dark` class on `<html>`. `shared/preferences` owns the
  `system | light | dark` setting; `ThemeScript` (inlined in `<head>`) applies
  the stored choice before first paint so there is no white flash.
- **Primitives** (`Button`, `Input`, `Dialog`, `Select`, `Table`, `Tabs`,
  `DropdownMenu`, …) come from `@shared/ui`. Radix supplies the behaviour —
  focus traps, keyboard nav, ARIA — and the styling is Tailwind classes with
  `cva` variants. Compose with `cn()` and pass `className` to override.
- **Forms** use `Field`, which owns the label / control / message layout and
  wires `htmlFor` + `aria-describedby`.
- **Icons** are `lucide-react`.
- **Dialogs** are full-bleed below `sm` and centred cards above it — that comes
  from `DialogContent`, so no dialog needs its own responsive handling.

## Verify

```sh
cd web
npx tsc --noEmit     # types + path aliases
npm run lint         # eslint (type-aware)
npm test             # vitest
npm run build        # next build (validates routes/RSC)
```
