.PHONY: all clean fmt lint verify test test-cover coverage-check test-coverage-check release-check tidy check bump help

# Go commands
GO := go
GOFMT := gofmt
# Pinned golangci-lint release. Single source of truth: the CI Lint job reads
# this value (see .github/workflows/ci.yml) and `make lint` warns when the
# installed binary differs. Bump it in a dedicated commit.
GOLANGCI_LINT_VERSION := 2.13.1
GOFILES := $(shell find . -type f -name '*.go' -not -path "./vendor/*")

all: fmt lint test

## fmt: Format Go source files
fmt:
	$(GOFMT) -w $(GOFILES)

## lint: Run linter (fails if not installed; warns if the installed version differs from GOLANGCI_LINT_VERSION)
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		installed="$$(golangci-lint version --short 2>/dev/null)"; \
		if [ -n "$$installed" ] && [ "$$installed" != "$(GOLANGCI_LINT_VERSION)" ]; then \
			echo "warning: golangci-lint $$installed installed, CI pins $(GOLANGCI_LINT_VERSION); results may differ"; \
		fi; \
		echo "golangci-lint run"; \
		golangci-lint run; \
	else \
		echo "golangci-lint $(GOLANGCI_LINT_VERSION) is required for make lint (not installed)"; \
		echo "install with: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v$(GOLANGCI_LINT_VERSION)"; \
		exit 1; \
	fi

## test: Run tests (writes coverage.out for the clix package across every test package)
test:
	$(GO) test -v -coverprofile=coverage.out -covermode=atomic -coverpkg=github.com/frostyard/clix ./...

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
	@echo "==> verify: golangci-lint $(GOLANGCI_LINT_VERSION)"
	@installed="$$(golangci-lint version --short 2>/dev/null)" || { echo "golangci-lint $(GOLANGCI_LINT_VERSION) is required for make verify (not installed)"; exit 1; }; \
	if [ "$$installed" != "$(GOLANGCI_LINT_VERSION)" ]; then echo "expected golangci-lint $(GOLANGCI_LINT_VERSION), found $$installed"; exit 1; fi
	golangci-lint run
	@echo "==> verify: tests"
	$(GO) test ./...

## check: Run fmt, lint, test, and the coverage floor
check: fmt lint test test-coverage-check coverage-check

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
