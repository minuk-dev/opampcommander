# Copilot Instructions for OpAMP Commander

## Overview

OpAMP Commander is a Go server implementing the [OpAMP](https://opentelemetry.io/docs/specs/opamp/) protocol for managing OpenTelemetry collectors. It consists of two components:
- **apiserver**: The main server handling OpAMP agents via WebSocket and REST APIs
- **opampctl**: CLI tool for interacting with the apiserver

## Build, Test, and Lint Commands

```sh
# Lint
make lint
make lint-fix

# Build (generates swagger docs, then builds)
make build-dev

# Generate mocks and swagger docs
make generate

# Unit tests
make unittest

# All tests including integration
make test

# Run a single test
go test -v ./path/to/package -run TestName

# E2E tests (requires Docker)
make test-e2e
make test-e2e-basic  # Subset without Kafka
```

## Architecture

This project follows **Hexagonal Architecture** (Ports & Adapters) with three layers wired via [Uber FX](https://uber-go.github.io/fx/):

### Layer Structure

```
internal/
├── domain/           # Core business logic (innermost)
│   ├── model/        # Domain entities (Agent, AgentGroup, Connection, etc.)
│   ├── port/         # Port interfaces (in.go = usecases, out.go = persistence)
│   └── service/      # Domain services implementing port.XxxUsecase
│
├── application/      # Application services (orchestration)
│   ├── port/         # Application-level usecase interfaces
│   └── service/      # Application services (agent/, agentgroup/, opamp/)
│
└── adapter/          # Infrastructure adapters (outermost)
    ├── in/           # Inbound adapters
    │   ├── http/v1/  # REST controllers per domain (agent/, agentgroup/, etc.)
    │   └── messaging/# Message consumers (Kafka)
    └── out/          # Outbound adapters
        ├── persistence/mongodb/  # MongoDB repositories
        └── messaging/            # Message producers (Kafka, in-memory)

pkg/apiserver/module/  # FX module wiring
├── domain/           # Wire domain services → domain ports
├── application/      # Wire application services → application ports
└── infrastructure/   # Wire adapters (HTTP, DB, messaging)
```

### Key Principles

- **Domain layer** has no external dependencies (no FX, no framework imports)
- **Ports** define interfaces: `in.go` for usecases (inbound), `out.go` for persistence/messaging (outbound)
- **Adapters** implement ports: controllers use usecases, repositories implement persistence ports
- **FX modules** in `pkg/apiserver/module/` wire implementations to interfaces

### Adding a New Feature

1. Define domain model in `internal/domain/model/`
2. Add usecase interface to `internal/domain/port/in.go`
3. Add persistence interface to `internal/domain/port/out.go`
4. Implement domain service in `internal/domain/service/`
5. (Optional) Add application service if orchestration needed
6. Implement HTTP controller in `internal/adapter/in/http/v1/`
7. Implement MongoDB repository in `internal/adapter/out/persistence/mongodb/`
8. Wire in FX modules under `pkg/apiserver/module/`

## Key Conventions

### Controller Pattern

Controllers implement `RoutesInfo()` returning `gin.RoutesInfo` and are auto-registered via FX:

```go
func (c *Controller) RoutesInfo() gin.RoutesInfo {
    return gin.RoutesInfo{
        {Method: http.MethodGet, Path: "/api/v1/agents", HandlerFunc: c.List},
    }
}
```

### Interface Compliance

Use compile-time interface checks:

```go
var _ port.AgentUsecase = (*AgentService)(nil)
```

### Mocks

Generated via [mockery](https://vektra.github.io/mockery/) into `usecasemock/` directories. Run `make prebuilt-mock` or `make generate` after changing interfaces.

### Testing

- Use `testutil.NewBase(t).ForController()` for controller tests
- Tests use `go.uber.org/goleak` for goroutine leak detection
- E2E tests use [testcontainers-go](https://testcontainers.com/guides/getting-started-with-testcontainers-for-go/) for MongoDB/Kafka

### Linting

Configured in `.golangci.yaml` with:
- `depguard`: Strict import allowlists (different for prod vs test)
- `ireturn`: Specific interface return types are allowlisted
- Import ordering: standard → external → internal (`github.com/minuk-dev/opampcommander`)

### API Versioning

- REST APIs under `/api/v1/`
- API types in `api/v1/`
- Swagger docs generated via `swag init` (run `make prebuilt-doc`)
