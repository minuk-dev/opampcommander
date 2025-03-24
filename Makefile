GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

lint:
	golangci-lint run

dev:
	GOOS="darwin" GOARCH="arm64" goreleaser build --snapshot --clean --single-target

run-dev-server: dev
	./scripts/etcd/etcd && ./dist/apiserver_$(GOOS)_$(GOARCH)/apiserver

build:
	goreleaser build

unittest:
	go test ./...

release:
	goreleaser release --rm-dist

docker:
	goreleaser 
