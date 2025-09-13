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

generate:
	mockery

dev: prebuilt-doc
	go run ./cmd/apiserver/main.go $(ARGS)

run-dev-server: dev
	./scripts/etcd/etcd && ./dist/apiserver_$(GOOS)_$(GOARCH)/apiserver

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
