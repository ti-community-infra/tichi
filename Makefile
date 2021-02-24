PROJECT=tichi
GOPATH ?= $(shell go env GOPATH)

# Ensure GOPATH is set before running build process.
ifeq "$(GOPATH)" ""
  $(error Please set the environment variable GOPATH before running `make`)
endif
FAIL_ON_STDOUT := awk '{ print } END { if (NR > 0) { exit 1 } }'

CURDIR := $(shell pwd)
path_to_add := $(addsuffix /bin,$(subst :,/bin:,$(GOPATH))):$(PWD)/tools/bin
export PATH := $(path_to_add):$(PATH)

GO              := GO111MODULE=on go
GOBUILD         := $(GO) build
GOTEST          := $(GO) test

PACKAGE_LIST  := go list ./...
PACKAGES  := $$($(PACKAGE_LIST))
PACKAGE_DIRECTORIES := $(PACKAGE_LIST) | sed 's|github.com/ti-community-infra/$(PROJECT)/||'
FILES     := $$(find $$($(PACKAGE_DIRECTORIES)) -name "*.go")


.PHONY: clean test dev check tidy

clean:
	$(GO) clean -i ./...
	rm -rf *.out

test:
	$(GOTEST) $(PACKAGES)
	@>&2 echo "Great, all tests passed."

cover:
	$(GOTEST) $(PACKAGES) -race -coverprofile=coverage.txt -covermode=atomic
	echo "Uploading coverage results..."
	@curl -s https://codecov.io/bash | bash

dev: check test staticcheck

check: fmt tidy

fmt:
	@echo "gofmt (simplify)"
	@gofmt -s -l -w $(FILES) 2>&1 | $(FAIL_ON_STDOUT)

tidy:
	@echo "go mod tidy"
	./tools/check/check-tidy.sh

staticcheck: tools/bin/golangci-lint
	tools/bin/golangci-lint run  $$($(PACKAGE_DIRECTORIES)) --timeout 500s

tools/bin/golangci-lint:
	curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh| sh -s -- -b ./tools/bin v1.31.0

