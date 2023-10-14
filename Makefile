DEV_BIN:=dev_tools/bin
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
setup: $(DEV_BIN)/air $(DEV_BIN)/dlv $(DEV_BIN)/golangci-lint $(DEV_BIN)/sqlc $(DEV_BIN)/migrate

$(DEV_BIN)/air:
	mkdir -p $(@D)
	curl -sSfL https://raw.githubusercontent.com/cosmtrek/air/master/install.sh | sh -s -- -b $(@D)

$(DEV_BIN)/golangci-lint:
	mkdir -p $(@D)
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(@D) v1.54.2

$(DEV_BIN)/dlv:
	mkdir -p $(@D)
	GOBIN=$(ABS_DEV_BIN) go install github.com/go-delve/delve/cmd/dlv@latest

$(DEV_BIN)/sqlc:
	mkdir -p $(@D)
	GOBIN=$(ABS_DEV_BIN) go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

$(DEV_BIN)/migrate:
	mkdir -p $(@D)
	GOBIN=$(ABS_DEV_BIN) go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest


.PHONY: clean
clean:
	rm -rf dev_tools/bin


