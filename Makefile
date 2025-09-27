GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

lint:
	golangci-lint run

lint-fix:
	golangci-lint run --fix

prebuilt-doc:
	swag init -o ./pkg/apiserver/docs -g ./cmd/apiserver/main.go

build-dev: prebuilt-doc
	goreleaser build --snapshot --clean --single-target

prebuilt-mock:
	mockery

generate: prebuilt-doc prebuilt-mock

run-dev-server: build-dev
	./scripts/etcd/etcd & go run ./cmd/apiserver/main.go --config ./configs/apiserver/dev.yaml

build: prebuilt-doc
	goreleaser build

unittest:
	go test -short ./... 

test:
	go test -race ./... 

release:
	goreleaser release --rm-dist

docker:
	goreleaser 
