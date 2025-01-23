
dev:
	goreleaser build --snapshot --clean

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
