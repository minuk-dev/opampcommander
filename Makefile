GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

.PHONY: lint lint-fix prebuilt-doc build-dev prebuilt-mock generate start-mongodb stop-mongodb clean-mongodb-data start-kafka stop-kafka clean-kafka start-dev-services stop-dev-services clean-dev-services run-dev-server run-standalone debug-server debug-server-console build unittest test test-e2e test-e2e-kafka test-e2e-basic release docker docker-image

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

start-kafka:
	@docker run -d --name kafka-dev \
		-p 9092:9092 \
		-e KAFKA_BROKER_ID=1 \
		-e KAFKA_LISTENERS=PLAINTEXT://0.0.0.0:9092 \
		-e KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9092 \
		-e KAFKA_ZOOKEEPER_CONNECT=zookeeper:2181 \
		-e KAFKA_AUTO_CREATE_TOPICS_ENABLE=true \
		confluentinc/cp-kafka:7.5.0 || docker start kafka-dev
	@echo "Kafka started on port 9092"

stop-kafka:
	@docker stop kafka-dev || true
	@echo "Kafka stopped"

clean-kafka:
	@docker stop kafka-dev || true
	@docker rm kafka-dev || true
	@echo "Kafka container removed"

start-dev-services: start-mongodb start-kafka
	@echo "Development services (MongoDB and Kafka) started"

stop-dev-services: stop-mongodb stop-kafka
	@echo "Development services stopped"

clean-dev-services: clean-mongodb-data clean-kafka
	@echo "Development services cleaned"

run-dev-server: build-dev start-dev-services
	@sleep 2
	go run ./cmd/apiserver/main.go --config ./configs/apiserver/dev.yaml

run-standalone: build-dev start-mongodb
	@sleep 2
	@echo "Starting server in standalone mode (no NATS)..."
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
	go test -short ./... -coverprofile=coverage.txt

test:
	go test -race ./... -coverprofile=coverage.txt

test-e2e:
	@echo "Running E2E tests (requires Docker)..."
	@echo "This may take 10-15 minutes..."
	go test ./test/e2e/... -v -timeout=20m

test-e2e-kafka:
	@echo "Running Kafka E2E tests only..."
	go test ./test/e2e/apiserver -run TestE2E_APIServer_Kafka -v -timeout=15m

test-e2e-basic:
	@echo "Running basic E2E tests only..."
	go test ./test/e2e/apiserver -run "TestE2E_APIServer_WithOTelCollector|TestE2E_APIServer_MultipleCollectors" -v -timeout=10m

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
