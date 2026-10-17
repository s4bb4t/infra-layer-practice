GOLANGCI_LINT_VERSION := v2.15.0
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

# Integration tests start containers: give them time.
TEST_TIMEOUT ?= 15m

.PHONY: help
help: ## Показать цели
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

# ---------------------------------------------------------------------------
# ОБЯЗАТЕЛЬНО перед каждым push: make check
# ---------------------------------------------------------------------------

.PHONY: check
check: fmt-check vet lint test integration ## Всё, что обязано быть зелёным перед push

.PHONY: fmt
fmt: ## Отформатировать код
	gofmt -w .

.PHONY: fmt-check
fmt-check: ## Упасть, если есть неотформатированные файлы
	@out="$$(gofmt -l .)"; if [ -n "$$out" ]; then echo "gofmt: файлы не отформатированы (make fmt):"; echo "$$out"; exit 1; fi

.PHONY: vet
vet: ## go vet со всеми тегами
	go vet ./...
	go vet -tags=integration,e2e,chaos ./...

.PHONY: lint
lint: ## golangci-lint (ставится сам через go run, версия закреплена)
	$(GOLANGCI_LINT) run --build-tags=integration,e2e,chaos ./...

.PHONY: test
test: ## Unit-тесты (без Docker)
	go test -race -count=1 ./...

.PHONY: integration
integration: docker-check ## Интеграционные тесты адаптеров (testcontainers, нужен Docker)
	go test -race -count=1 -tags=integration -timeout=$(TEST_TIMEOUT) ./internal/infrastructure/...

.PHONY: e2e
e2e: docker-check ## Матрица конфигураций оркестратора (долго)
	go test -count=1 -tags=e2e -timeout=30m ./test/e2e/...

.PHONY: chaos
chaos: docker-check ## Тесты отказов: Toxiproxy, убийство контейнеров (долго)
	go test -count=1 -tags=chaos -timeout=30m ./test/chaos/...

.PHONY: docker-check
docker-check:
	@docker info >/dev/null 2>&1 || (echo "Docker не запущен: интеграционные тесты без него не работают"; exit 1)

# ---------------------------------------------------------------------------

.PHONY: hooks
hooks: ## Поставить git pre-push хук, который запускает make check
	git config core.hooksPath .githooks
	@echo "pre-push хук включён: .githooks/pre-push"

.PHONY: run
run: ## Запустить сервис
	go run ./cmd/orders

.PHONY: infra-up
infra-up: ## Поднять инфраструктуру для ручного запуска (PROFILES="db broker obs")
	docker compose -f deploy/compose.yaml $(foreach p,$(or $(PROFILES),db),--profile $(p)) up -d

.PHONY: infra-down
infra-down: ## Остановить всю инфраструктуру
	docker compose -f deploy/compose.yaml --profile '*' down -v
