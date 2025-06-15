GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

lint:
	golangci-lint run

lint-fix:
	golangci-lint run --fix

prebuilt-doc:
	swag init --pd -o ./pkg/apiserver/docs -g ./cmd/apiserver/main.go

build-dev: prebuilt-doc
	GOOS="darwin" GOARCH="arm64" goreleaser build --snapshot --clean --single-target

dev: prebuilt-doc
	go run ./cmd/apiserver/main.go

run-dev-server: dev
	./scripts/etcd/etcd && ./dist/apiserver_$(GOOS)_$(GOARCH)/apiserver

build: prebuilt-doc
	goreleaser build

unittest:
	go test -short ./... 

test:
	go test ./... 

release:
	goreleaser release --rm-dist

docker:
	goreleaser 
