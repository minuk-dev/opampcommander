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
		mongo:4.4 || docker start mongodb-dev
	@echo "MongoDB 4.4 started with data persisted in ./default.mongodb"

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

debug-server: start-mongodb
	@echo "Starting debug server with delve..."
	@echo "MongoDB should be running. Connect your debugger to localhost:2345"
	@sleep 2
	dlv debug ./cmd/apiserver/main.go --headless --listen=:2345 --api-version=2 --accept-multiclient -- --config ./configs/apiserver/dev.yaml

debug-server-console: start-mongodb
	@echo "Starting debug server in console mode..."
	@sleep 2
	dlv debug ./cmd/apiserver/main.go -- --config ./configs/apiserver/dev.yaml

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

docker-image:
	@if [ -z "$(TAG)" ]; then \
		echo "Error: TAG is required"; \
		echo "Usage: make docker-test TAG=<tag>"; \
		echo "Example: make docker-test TAG=test-v1.0.0"; \
		exit 1; \
	fi
	@echo "Building Docker images with tag: $(TAG)"
	@echo "==========================================="
	@GORELEASER_CURRENT_TAG=$(TAG) goreleaser release --snapshot --clean --skip=publish
	@echo ""
	@echo "==========================================="
	@echo "Docker images built successfully!"
	@echo ""
	@echo "To push the images, run the following commands:"
	@echo ""
	@echo "docker push minukdev/opampcommander:$(TAG)-amd64"
	@echo "docker push minukdev/opampcommander:$(TAG)-arm64"
	@echo ""
	@echo "To create and push the manifest:"
	@echo ""
	@echo "docker manifest create minukdev/opampcommander:$(TAG) \\"
	@echo "  minukdev/opampcommander:$(TAG)-amd64 \\"
	@echo "  minukdev/opampcommander:$(TAG)-arm64"
	@echo ""
	@echo "docker manifest push minukdev/opampcommander:$(TAG)" 
