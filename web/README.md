# OpAMP Commander Web

The web dashboard for [OpAMP Commander](../README.md). It is a
[Next.js 16](https://nextjs.org) (App Router) application using
[MUI](https://mui.com) for UI and [SWR](https://swr.vercel.app) for data fetching,
organized with [Feature-Sliced Design](https://feature-sliced.design/).

For the layer rules and where things live, read [`ARCHITECTURE.md`](ARCHITECTURE.md).

## How it talks to the apiserver

The browser never calls the apiserver directly. Requests go to a Next.js route
handler (`app/api/proxy/[...path]/route.ts`) that forwards them to the apiserver,
attaching the session credential. The backend URL is read from the `OPAMP_API_URL`
environment variable (default `http://localhost:8080`).

```
Browser ──▶ Next.js (/api/proxy/*) ──▶ apiserver (OPAMP_API_URL)
```

Authentication uses a server-side session set via `app/api/session/route.ts`, with
GitHub OAuth handled through `app/login/github/callback`.

## Getting started

Requires Node.js 20.9+.

```bash
npm install

# point at a running apiserver and start the dev server
OPAMP_API_URL=http://localhost:8080 npm run dev
```

Open <http://localhost:3000>.

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `OPAMP_API_URL` | `http://localhost:8080` | apiserver base URL the proxy forwards to (server-side only). |
| `NEXT_PUBLIC_WEB_VERSION` | `package.json` version | Version shown in the UI footer. |
| `NEXT_PUBLIC_WEB_GIT_COMMIT` | — | Build commit shown in the footer. |
| `NEXT_PUBLIC_WEB_GIT_TREE_STATE` | — | `clean` / `dirty` build state. |
| `NEXT_PUBLIC_WEB_BUILD_DATE` | — | Build timestamp. |

## Scripts

```bash
npm run dev          # development server
npm run build        # production build (standalone output)
npm run start        # serve the production build
npm run lint         # eslint
npm test             # vitest (run once)
npm run test:watch   # vitest watch mode
npm run format       # prettier --write
npm run format:check # prettier --check
```

Run the full check suite before pushing:

```bash
npx tsc --noEmit && npm run lint && npm test && npm run build
```

## Routes

`app/` contains routing only; each `page.tsx` re-exports a view from `src/views/`.
Pages include the dashboard (`/`), `agents`, `agentgroups`, `agentpackages`,
`agentremoteconfigs`, `certificates`, `connections`, `namespaces`, `roles`,
`rolebindings`, `users`, `servers`, `platform`, `version`, `preferences`, `profile`,
and `login`.

## Production build

`next.config.ts` uses `output: 'standalone'`, producing a self-contained server
bundle in `.next/standalone/`. The included [`Dockerfile`](Dockerfile) builds and
serves that bundle with Node.js — no `npm install` at runtime.

```bash
npm run build
node .next/standalone/server.js
```
