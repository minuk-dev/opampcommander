lint:
	golangci-lint run

dev:
	GOOS="darwin" GOARCH="arm64" goreleaser build --snapshot --clean --single-target

build:
	goreleaser build

unittest:
	go test ./...

release:
	goreleaser release --rm-dist

docker:
	goreleaser 

e:
	echo "Running your custom 'e' target"
