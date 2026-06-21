# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Project Does

OpAMP Commander is a Go server implementing the [OpAMP protocol](https://opentelemetry.io/docs/specs/opamp/) for managing OpenTelemetry collectors/agents. It provides:
- A WebSocket server that agents connect to for bidirectional management
- A REST API for configuring agents, namespaces, RBAC, and certificates
- A CLI tool (`opampctl`) for interacting with the server
- Multi-server coordination via Kafka (or in-memory for single-node deployments)

## Commands

### Build & Generate
```sh
make generate        # regenerate Swagger docs + mocks (required before build)
make build-dev       # build single-target binary via goreleaser
make build           # full goreleaser build
```

### Run
```sh
make start-dev-services    # start MongoDB + Kafka in Docker
make run-dev-server        # build + start services + run apiserver
make run-standalone        # single-node mode (MongoDB only, no Kafka)
go run ./cmd/apiserver/main.go --config ./configs/apiserver/dev.yaml
```

### Test
```sh
make unittest        # go test -short ./...
make test            # go test -race ./...
make test-e2e        # E2E tests (requires Docker) — runs all suites
make test-e2e-basic  # E2E tests without Kafka
go test -v ./path/to/package -run TestName   # single test
```

### Lint
```sh
make lint            # golangci-lint
make lint-fix        # golangci-lint --fix
```

### Debug
```sh
make debug-server    # delve headless on :2345
```

## Architecture

The project uses **Hexagonal Architecture** wired with **Uber FX** dependency injection. There are three explicit layers — domain, application, and adapter — plus shared `pkg/` and `api/v1/` packages.

### Layer Summary

| Layer | Path | Role |
|---|---|---|
| Domain | `internal/domain/` | Pure business logic; no FX imports allowed (enforced by depguard) |
| Application | `internal/application/` | Orchestration between domain and HTTP layer; operates on `api/v1` types |
| Adapter (in) | `internal/adapter/in/` | Gin HTTP controllers, Kafka/in-memory event consumers |
| Adapter (out) | `internal/adapter/out/` | MongoDB repositories, Kafka producer, Casbin RBAC, GitHub identity |

### FX Wiring
The root `fx.New(...)` is in `pkg/apiserver/apiserver.go`. Modules:
- `pkg/apiserver/module/domain/` — domain services → domain port interfaces
- `pkg/apiserver/module/application/` — application services → application port interfaces
- `pkg/apiserver/module/infrastructure/` — controllers, repositories, messaging, Casbin

### Key Packages
- `internal/domain/agent/port/in.go` — inbound usecase interfaces consumed by the application layer
- `internal/domain/agent/port/out.go` — outbound persistence/messaging interfaces implemented by adapters
- `internal/application/service/opamp/` — central OpAMP protocol handler (`OnConnected`, `OnMessage`, `OnConnectionClose`)
- `api/v1/` — HTTP request/response DTOs (kept separate from domain models)
- `pkg/client/` — Go client library used by `opampctl`
- `pkg/testutil/` — testcontainers helpers for MongoDB/Kafka/OTel Collector

### Multi-Server Coordination
When an agent is connected to server B but server A receives a management request, server A publishes a CloudEvent to Kafka and server B's consumer delivers it over the agent's WebSocket. In single-node mode, `internal/adapter/in/messaging/inmemory/` replaces Kafka.

### Agent Namespace
Derived from the OpAMP agent's `service.namespace` identifying attribute; defaults to `"default"`.

## Key Conventions

### Controller Registration
Every controller implements `RoutesInfo() gin.RoutesInfo` and is auto-registered into Gin by the FX infrastructure module — no manual route wiring needed.

### Adapters Depend Only on the Application Layer
Primary (driving) adapters — HTTP controllers, auth flows — must reach the core
only through the application layer (`application/port` + `api/v1`), never by
importing the domain directly. This is enforced by the `adapter-primary-no-domain`
depguard rule (the `adapter/primary/messaging` adapters are exempt because they
implement domain driven/out ports).

Each HTTP controller depends on **at most one** application use case (a single
`...Usecase` field; trivial controllers such as `ping`/`version` may need none).
If a handler needs a different use case, it belongs in a different controller, or
the use cases should be composed behind one application service — not injected
side-by-side into the controller.

### Compile-Time Interface Checks
All service implementations include:
```go
var _ port.SomeInterface = (*SomeService)(nil)
```

### Mock Generation
Mocks live in `usecasemock/` subdirectories next to the packages defining the interfaces. Regenerate with `make prebuilt-mock` (uses `.mockery.yml`).

### Error Responses
All HTTP errors use RFC 9457 Problem Details format via `ginutil.HandleDomainError`. `port.ErrResourceNotExist` → 404; others → 500.

### Import Ordering (enforced by `gci`)
1. Standard library
2. External packages
3. Internal (`github.com/minuk-dev/opampcommander`)

### Import Restrictions (enforced by `depguard`)
- `internal/**` packages must not import `go.uber.org/fx`
- Production code must not import `pkg/testutil`

### Swagger Docs
Generated from godoc annotations in controllers via `swag init` (`make prebuilt-doc`). Never edit `pkg/apiserver/docs/docs.go` manually.

## Web (Frontend)

The `web/` directory is a Next.js 16 (App Router) + MUI client, organized with
**Feature-Sliced Design**. FSD layers live under `web/src/`
(`shared → entities → features → widgets → views → app`, importing downward
only); `web/app/` holds Next.js routing only (route handlers + thin `page.tsx`
wrappers re-exporting a view). Path aliases: `@shared/* @entities/* @features/*
@widgets/* @views/* @app/*`. Import from a slice's `index.ts` barrel, not its
internal files. See `web/ARCHITECTURE.md` for layer rules, where types/contexts
live, and how to add an entity/feature/route. Checks: `cd web && npx tsc
--noEmit && npm run lint && npm test && npm run build`.

### Mobile / Responsive
The UI must stay usable on phone-width viewports (≈375px). When building or
changing any web feature, verify it at a narrow width — don't assume desktop.
Conventions to follow:
- **Breakpoint:** treat `< md` (`theme.breakpoints.down('md')`) as mobile. The
  app shell (`widgets/app-layout`) already switches the sidebar to a `temporary`
  overlay drawer below `md` and a `persistent` drawer above it; don't reintroduce
  a fixed-width sidebar that pushes content.
- **No fixed widths that overflow:** prefer responsive `sx` objects
  (`minWidth: { xs: 120, sm: 200 }`, `flexDirection: { xs: 'column', sm: 'row' }`,
  `display: { xs: 'none', sm: 'block' }`) over hardcoded `px` widths. Hide or
  shrink secondary chrome (titles, dividers, labels) on `xs`.
- **Wide content scrolls, not overflows:** wrap tables in MUI `TableContainer`
  (as `widgets/resource-list-page` does) so they scroll horizontally instead of
  breaking the layout.
- **Dialogs:** use `fullWidth` with a `maxWidth`; consider `fullScreen` on `xs`
  for large forms.
- Reuse `shared/ui` and the `widgets/*` shells rather than hand-rolling layout —
  they already carry the responsive behavior.
