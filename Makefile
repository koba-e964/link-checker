# Release: https://github.com/itchyny/gojq/blob/v0.12.17/Makefile
BIN := link-checker
GOBIN ?= $(shell go env GOPATH)/bin
SHELL := /bin/bash
VERSION := $$(make -s version)
CURRENT_REVISION = $(shell git rev-parse --short HEAD)
BUILD_LDFLAGS = "-s -w -X github.com/koba-e964/$(BIN)/cli.revision=$(CURRENT_REVISION)"

.PHONY: all
all: dependency_graph.png 

%.png: %.dot
	dot -Tpng $< -o $@

.PHONY: cross
cross: $(GOBIN)/goxz
	$(GOBIN)/goxz -n $(BIN) -pv=$(VERSION) \
		-build-ldflags=$(BUILD_LDFLAGS) ./

$(GOBIN)/goxz:
	@go install github.com/Songmu/goxz/cmd/goxz@v0.9.1

.PHONY: version
version:
	@git describe --tags

.PHONY: clean
clean:
	rm -rf $(BIN) goxz
	go clean
