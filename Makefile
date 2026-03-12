SHELL := /bin/bash

# Variables
GOPATH ?= $(strip $(shell go env GOPATH)/bin) ## Location of dev dependencies

# Default goal
.DEFAULT_GOAL := help

# Add GOPATH to PATH
export PATH := $(GOPATH):${PATH}

# All targets are phony
.PHONY: all help devdeps clean tidy generate format build spell lint test update

# Set the 'all' target
all: tidy generate format devdeps lint spell test ## Execute all targets

help: ## Show this help menu
	@sed -ne "s/^##\(.*\)/\1/p" $(MAKEFILE_LIST)
	@printf "────────────────────────`tput bold``tput setaf 2` Make Commands `tput sgr0`────────────────────────────────\n"
	@sed -ne "/@sed/!s/\(^[^#?=]*:\).*##\(.*\)/`tput setaf 2``tput bold`\1`tput sgr0`\2/p" $(MAKEFILE_LIST)
	@printf "────────────────────────`tput bold``tput setaf 4` Make Variables `tput sgr0`───────────────────────────────\n"
	@sed -ne "/@sed/!s/\(.*\)?=\(.*\)##\(.*\)/`tput setaf 4``tput bold`\1:`tput setaf 5`\2`tput sgr0`\3/p" $(MAKEFILE_LIST)
	@printf "───────────────────────────────────────────────────────────────────────\n"

devdeps: ## Install development dependencies
	@printf "Executing target: [$@] 🎯\n"
	@which -a golangci-lint > /dev/null || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH) v2.11.3
	@which -a goreleaser > /dev/null || go install github.com/goreleaser/goreleaser/v2@v2.14.3
	@which -a gocover-cobertura > /dev/null || go install github.com/boumenot/gocover-cobertura@latest
	@which -a misspell > /dev/null || go install github.com/client9/misspell/cmd/misspell@latest
	@which -a gotestdox > /dev/null || go install github.com/bitfield/gotestdox/cmd/gotestdox@latest
	@which -a go-junit-report > /dev/null || go install github.com/jstemmer/go-junit-report/v2@latest

clean: ## Remove build and transient test artifacts
	@printf "Executing target: [$@] 🎯\n"
	@rm -rf dist coverage.* test-results.txt junit-report.xml '"$(shell go env GOCACHE)/../golangci-lint"'
	@go clean -i -cache -testcache -modcache -fuzzcache -x 2>&1 > /dev/null

tidy: generate ## Tidy up modules
	@printf "Executing target: [$@] 🎯\n"
	@go mod tidy

generate: ## Run //go:generate commands, if any
	@printf "Executing target: [$@] 🎯\n"
	@go generate ./...

build: tidy clean devdeps ## Build binaries
	@printf "Executing target: [$@] 🎯\n"
	@goreleaser build --snapshot

spell: format ## Determine spelling errors in code and docs
	@printf "Executing target: [$@] 🎯\n"
	@misspell -error -locale=US -w **/*

format: ## Format source code
	@printf "Executing target: [$@] 🎯\n"
	@gofmt -s -w -e .

lint: devdeps spell ## Lint source code
	@printf "Executing target: [$@] 🎯\n"
	@golangci-lint run -c .golangci.yml

test: clean tidy devdeps spell ## Run unit tests and generate reports
	@printf "Executing target: [$@] 🎯\n"
	@touch coverage.out
	@mkdir -p /tmp/scribe/installed_packages
	@go test -json -race -covermode=atomic -coverprofile=coverage.out -coverpkg=./... ./... 2>&1 > test-results.txt || cat test-results.txt
	@cat test-results.txt | gotestdox
	@cat test-results.txt | go-junit-report -parser gojson > junit-report.xml
	@go tool cover -html=coverage.out -o coverage.html
	@gocover-cobertura < coverage.out > coverage.xml

types: ## Examine Go types and their transitive dependencies
	@printf "Executing target: [$@] 🎯\n"
	@typex -t -u .

update: ## Update all dependencies
	@printf "Executing target: [$@] 🎯\n"
	@go get -u all
