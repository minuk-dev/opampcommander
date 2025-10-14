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

### Database Setup
This project uses MongoDB 4.4 as its database (for Raspberry Pi compatibility). The Makefile provides convenient commands to manage MongoDB:

```sh
# Start MongoDB 4.4 (data persisted in ./default.mongodb)
make start-mongodb

# Stop MongoDB (data is preserved)
make stop-mongodb

# Clean MongoDB data (WARNING: removes all data)
make clean-mongodb-data

# Run dev server (automatically starts MongoDB)
make run-dev-server
```

### Development Commands

```sh
# run opampcommander's apiserver with default settings
make dev

# run opampcommander's apiserver with args
make ARGS="--log.level=warn --log.format=text --metric.enabled" dev

# run opampcommander's apiserver with args and GIN_MODE=release
GIN_MODE=release make ARGS="--log.level=warn --log.format=text --metric.enabled" dev
```

## Release

## Deployment
