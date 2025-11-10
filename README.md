# OpAMP Commander

## Components
### opampctl
- opampctl is a command line tool to control opampcommander.

#### Install
```sh
# opampctl
go install github.com/minuk-dev/opampcommander/cmd/opampctl@latest

# create config file
opampctl config init
```

### apiserver
- apiserver is a server component to support [OpAMP](https://opentelemetry.io/docs/specs/opamp/)

#### How to run
- TBD

## Development

### Infrastructure Setup
This project uses MongoDB 4.4 as its database (for Raspberry Pi compatibility) and Kafka for distributed messaging. The Makefile provides convenient commands:

```sh
# Start MongoDB 4.4 (data persisted in ./default.mongodb)
make start-mongodb

# Start Kafka (for distributed mode)
make start-kafka

# Start all development services (MongoDB + Kafka)
make start-dev-services

# Stop services (data is preserved)
make stop-dev-services

# Clean all data (WARNING: removes all data)
make clean-dev-services
```

### Development Commands

```sh
# Build and run apiserver with default settings
make run-dev-server

# Run standalone server (MongoDB only, no Kafka)
make run-standalone

# Run with custom arguments
make ARGS="--log.level=warn --log.format=text --metric.enabled" dev

# Run with GIN_MODE=release
GIN_MODE=release make ARGS="--log.level=warn --log.format=text --metric.enabled" dev
```

### Testing

```sh
# Run unit tests
make unittest

# Run all tests including integration tests
make test

# Run E2E tests (requires Docker)
make test-e2e

# Run only Kafka E2E tests
make test-e2e-kafka

# Run only basic E2E tests
make test-e2e-basic
```

## Release

## Deployment
