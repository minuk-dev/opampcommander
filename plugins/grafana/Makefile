.PHONY: build frontend backend check serve stop

IMAGE ?= opampcommander-grafana:dev
GRAFANA_VERSION ?= 13.2.3
.PHONY: image
image: build
	docker build --build-arg GRAFANA_VERSION=$(GRAFANA_VERSION) -t $(IMAGE) .

build: frontend backend

frontend:
	npm run build

# The SDK discovers src/datasource/plugin.json and writes its binaries to
# dist/datasource/, alongside the nested plugin's frontend entry point.
backend:
	go run github.com/magefile/mage -v build:linux build:linuxARM64

check:
	npm run typecheck
	npm run lint
	npm run format:check
	npm run test:ci
	go test -race ./pkg/...
	go vet ./pkg/...

serve: build
	docker compose up -d --wait --force-recreate
	@echo "Grafana preview: http://127.0.0.1:$${GRAFANA_PORT:-3300}/a/minukdev-opampcommander-app"

stop:
	docker compose down
