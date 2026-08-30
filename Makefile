.PHONY: all clean fmt lint lint-version-check verify test test-race test-cover coverage-check test-coverage-check release-check tidy check ci bump help

# Go commands
GO := go
GOFMT := gofmt
# Pinned golangci-lint release, read from mise.toml — the single source of
# every tool pin (core ADR-0043): `mise install` provisions it locally, in CI
# (jdx/mise-action), and on Snowcat workers, verified against mise.lock.
# Bump it there in a dedicated commit; never edit this line.
GOLANGCI_LINT_VERSION := $(strip $(shell sed -n 's/^golangci-lint = "\(.*\)"/\1/p' mise.toml))
# The Go release this module is built with, from go.mod's toolchain line —
# the only Go pin (mise reads the same line). golangci-lint must be built
# with a Go at least this new, or its embedded gofmt and typechecker disagree
# with the toolchain.
GO_TOOLCHAIN := $(strip $(shell sed -n 's/^toolchain go\(.*\)/\1/p' go.mod))
GOFILES := $(shell find . -type f -name '*.go' -not -path "./vendor/*")

all: fmt lint test

## fmt: Format Go source files
fmt:
	$(GOFMT) -w $(GOFILES)

## lint: Run linter (fails if mise.toml pins no golangci-lint, if not installed, or warns if the installed version differs from the pin)
lint:
	@test -n "$(GOLANGCI_LINT_VERSION)" || { echo "mise.toml pins no golangci-lint"; exit 1; }
	@if command -v golangci-lint >/dev/null 2>&1; then \
		installed="$$(golangci-lint version --short 2>/dev/null)"; \
		if [ -n "$$installed" ] && [ "$$installed" != "$(GOLANGCI_LINT_VERSION)" ]; then \
			echo "warning: golangci-lint $$installed installed, mise.toml pins $(GOLANGCI_LINT_VERSION); results may differ"; \
		fi; \
		echo "golangci-lint run"; \
		golangci-lint run; \
	else \
		echo "golangci-lint $(GOLANGCI_LINT_VERSION) not installed; provision every pinned tool with:"; \
		echo "mise install"; \
		exit 1; \
	fi

## lint-version-check: Fail unless the installed golangci-lint is the mise.toml pin and was built with a Go no older than go.mod's toolchain
lint-version-check:
	@test -n "$(GOLANGCI_LINT_VERSION)" || { echo "mise.toml pins no golangci-lint"; exit 1; }
	@installed="$$(golangci-lint version --short 2>/dev/null)" || { echo "golangci-lint $(GOLANGCI_LINT_VERSION) is required (not installed; run: mise install)"; exit 1; }; \
	if [ "$$installed" != "$(GOLANGCI_LINT_VERSION)" ]; then echo "expected golangci-lint $(GOLANGCI_LINT_VERSION), found $$installed (run: mise install)"; exit 1; fi; \
	built="$$(golangci-lint version 2>/dev/null | sed -n 's/.*built with go\([0-9.]*\).*/\1/p')"; \
	if [ -n "$$built" ] && [ "$$(printf '%s\n%s\n' "$(GO_TOOLCHAIN)" "$$built" | sort -V | head -1)" != "$(GO_TOOLCHAIN)" ]; then \
		echo "golangci-lint $(GOLANGCI_LINT_VERSION) was built with go$$built, older than go.mod's toolchain go$(GO_TOOLCHAIN): bump golangci-lint first (core ADR-0043)"; exit 1; fi

## test: Run tests (writes coverage.out for the clix package across every test package)
test:
	$(GO) test -v -coverprofile=coverage.out -covermode=atomic -coverpkg=github.com/frostyard/clix ./...

## test-race: Run tests under the race detector (mirrors the CI Race Detection job)
test-race:
	$(GO) test -race ./...

## coverage-check: Enforce the 95.0% total statement-coverage floor on coverage.out (scripts/check-coverage.sh)
coverage-check:
	./scripts/check-coverage.sh coverage.out 95.0

## test-coverage-check: Self-test scripts/check-coverage.sh against fixture profiles
test-coverage-check:
	./scripts/test-coverage-check.sh

## test-cover: Run tests with coverage
test-cover:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

## release-check: Validate .goreleaser.yaml (skips if goreleaser is not installed)
release-check:
	@if command -v goreleaser >/dev/null 2>&1; then \
		echo "goreleaser check -f .goreleaser.yaml"; \
		goreleaser check -f .goreleaser.yaml; \
	else \
		echo "goreleaser not installed, skipping"; \
	fi

## tidy: Tidy go modules
tidy:
	$(GO) mod tidy

## clean: Remove generated artifacts
clean:
	rm -f coverage.out coverage.html
	$(GO) clean

## verify: Credential-free, non-mutating gate (what a read-only reviewer runs): tidy diff, gofmt -l, vet, lint at the exact pin, tests
verify:
	@echo "==> verify: go.mod is tidy"
	$(GO) mod tidy -diff
	@echo "==> verify: gofmt"
	@unformatted="$$($(GOFMT) -l $(GOFILES))"; \
	if [ -n "$$unformatted" ]; then echo "$$unformatted"; echo "gofmt: files need formatting (run make fmt)"; exit 1; fi
	@echo "==> verify: go vet"
	$(GO) vet ./...
	@echo "==> verify: golangci-lint $(GOLANGCI_LINT_VERSION) (built with go >= $(GO_TOOLCHAIN))"
	@$(MAKE) --no-print-directory lint-version-check
	golangci-lint run
	@echo "==> verify: tests"
	$(GO) test ./...

## check: Run fmt, lint, test, and the coverage floor
check: fmt lint test test-coverage-check coverage-check

## ci: Credential-free gate for CI (core ADR-0022/ADR-0043): verify, the race detector, then this repository's coverage floor
ci:
	@echo "==> ci: verify"
	@$(MAKE) --no-print-directory verify
	@echo "==> ci: race detector"
	@$(MAKE) --no-print-directory test-race
	@echo "==> ci: tests with coverage"
	@$(MAKE) --no-print-directory test
	@echo "==> ci: self-test the coverage floor script"
	@$(MAKE) --no-print-directory test-coverage-check
	@echo "==> ci: coverage floor"
	@$(MAKE) --no-print-directory coverage-check

## bump: Tag and push next version (requires clean tree and svu)
bump:
	@$(MAKE) check
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "Working directory not clean. Commit or stash first."; \
		exit 1; \
	fi
	@version=$$(svu next); \
		git tag -a $$version -m "Version $$version"; \
		echo "Tagged $$version"; \
		git push origin $$version

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':'
