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

start-nats:
	@docker run -d --name nats-dev \
		-p 4222:4222 \
		-p 8222:8222 \
		nats:latest || docker start nats-dev
	@echo "NATS server started on port 4222 (client) and 8222 (monitoring)"

stop-nats:
	@docker stop nats-dev || true
	@echo "NATS server stopped"

clean-nats:
	@docker stop nats-dev || true
	@docker rm nats-dev || true
	@echo "NATS container removed"

start-dev-services: start-mongodb start-nats
	@echo "Development services (MongoDB and NATS) started"

stop-dev-services: stop-mongodb stop-nats
	@echo "Development services stopped"

clean-dev-services: clean-mongodb-data clean-nats
	@echo "Development services cleaned"

run-dev-server: build-dev start-dev-services
	@sleep 2
	go run ./cmd/apiserver/main.go --config ./configs/apiserver/dev.yaml

debug-server: start-dev-services
	@echo "Starting debug server with delve..."
	@echo "MongoDB and NATS should be running. Connect your debugger to localhost:2345"
	@sleep 2
	dlv debug ./cmd/apiserver/main.go --headless --listen=:2345 --api-version=2 --accept-multiclient -- --config ./configs/apiserver/dev.yaml

debug-server-console: start-dev-services
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
