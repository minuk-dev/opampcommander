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

start-mongodb:
	@mkdir -p ./default.mongodb
	@docker run -d --name mongodb-dev \
		-p 27017:27017 \
		-v $(PWD)/default.mongodb:/data/db \
		mongo:latest || docker start mongodb-dev
	@echo "MongoDB started with data persisted in ./default.mongodb"

stop-mongodb:
	@docker stop mongodb-dev || true
	@echo "MongoDB stopped (data preserved in ./default.mongodb)"

clean-mongodb-data:
	@docker stop mongodb-dev || true
	@docker rm mongodb-dev || true
	@rm -rf ./default.mongodb
	@echo "MongoDB container and data removed"

run-dev-server: build-dev start-mongodb
	@sleep 2
	go run ./cmd/apiserver/main.go --config ./configs/apiserver/dev.yaml

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
