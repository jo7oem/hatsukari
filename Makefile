DEV_BIN:=$(shell pwd)/dev_tools/bin
GOBIN=$(DEV_BIN)
export GOBIN
MAKEFILE_DIR := $(shell cd $(dir $(lastword $(MAKEFILE_LIST)))&&pwd )
ABS_DEV_BIN := $(MAKEFILE_DIR)/$(DEV_BIN)

.PHONY: build
build:
	go build -o hatsukari ./...

.PHONY: fmt
fmt: $(DEV_BIN)/golangci-lint
	$(DEV_BIN)/golangci-lint run --fix --config=.golangci.yml

.PHONY: lint
lint: $(DEV_BIN)/golangci-lint
	$(DEV_BIN)/golangci-lint run --config=.golangci.yml

.PHONY: setup
setup: $(DEV_BIN)/air $(DEV_BIN)/dlv $(DEV_BIN)/golangci-lint

$(DEV_BIN)/air:
	mkdir -p $(@D)
	go install github.com/air-verse/air@v1.62.0


$(DEV_BIN)/golangci-lint:
	mkdir -p $(@D)
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.4.0

$(DEV_BIN)/dlv:
	mkdir -p $(@D)
	go install github.com/go-delve/delve/cmd/dlv@latest

$(DEV_BIN)/sqlc:
	mkdir -p $(@D)
	GOBIN=$(ABS_DEV_BIN) go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

$(DEV_BIN)/migrate:
	mkdir -p $(@D)
	GOBIN=$(ABS_DEV_BIN) go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

.PHONY: migrate-up
migrate-up: $(DEV_BIN)/migrate
	$(DEV_BIN)/migrates -path db/migration -database "postgresql://hatsukari:hatsukari@localhost:5432/hatsukari?sslmode=disable" up

.PHONY: migrate-down
migrate-down: $(DEV_BIN)/migrate
	$(DEV_BIN)/migrates -path db/migration -database "postgresql://hatsukari:hatsukari@localhost:5432/hatsukari?sslmode=disable" down -all

.PHONY: sqlc-gen
sqlc-gen: $(DEV_BIN)/sqlc
	$(DEV_BIN)/sqlc generate

.PHONY: sqlc-diff
sqlc-diff: $(DEV_BIN)/sqlc
	$(DEV_BIN)/sqlc diff

.PHONY: ci
ci: sqlc-diff lint

.PHONY: clean
clean:
	rm -rf dev_tools/bin


