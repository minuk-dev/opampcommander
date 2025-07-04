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
