DEV_BIN:=dev_tools/bin
MAKEFILE_DIR := $(shell cd $(dir $(lastword $(MAKEFILE_LIST)))&&pwd )

.PHONY: nop
nop:
	# nop

.PHONY: setup
setup: $(DEV_BIN)/air $(DEV_BIN)/golangci-lint $(DEV_BIN)/dlv

$(DEV_BIN)/air:
	mkdir -p $(@D)
	curl -sSfL https://raw.githubusercontent.com/cosmtrek/air/master/install.sh | sh -s -- -b $(@D)

$(DEV_BIN)/golangci-lint:
	mkdir -p $(@D)
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(@D) v1.54.2

$(DEV_BIN)/dlv:
	mkdir -p $(@D)
	$(eval TMP_DIR=$(shell mktemp -d))
	cd $(TMP_DIR) &&\
		git clone https://github.com/go-delve/delve.git -b v1.21.1 --depth 1 &&\
		cd delve&&\
		make build&&\
		chmod 755 dlv &&\
		mv dlv $(MAKEFILE_DIR)/$(@D)/dlv &&\
		rm -rf $(TMP_DIR)


.PHONY: clean
clean:
	rm -rf dev_tools/bin

