# Figgo build automation
# Run 'just' to see all available targets

# Set shell for Windows compatibility
set windows-shell := ["pwsh.exe", "-NoLogo", "-Command"]
set shell := ["bash", "-uc"]

# Default recipe - show available targets
default:
    @just --list

# Build the figgo binary
build:
    go build -v -o figgo ./cmd/figgo

# Run all tests
test:
    go test -v -race ./...

# Run linting with golangci-lint
lint:
    @if command -v golangci-lint >/dev/null 2>&1; then \
        golangci-lint run ./...; \
    else \
        echo "golangci-lint not installed, running go vet instead"; \
        go vet ./...; \
    fi

# Format all Go code
fmt:
    go fmt ./...
    gofmt -s -w .

# Run benchmarks
bench:
    go test -bench=. -benchmem ./...

# Generate test coverage report
coverage:
    go test -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    @echo "Coverage report generated: coverage.html"

# Generate golden test files
generate-goldens:
    @if [ -f ./tools/generate-goldens.sh ]; then \
        ./tools/generate-goldens.sh; \
    else \
        echo "Golden test generator not yet implemented"; \
    fi

# Run CI checks locally (lint, test, build)
ci: lint test build
    @echo "All CI checks passed!"

# Manage Go module dependencies
mod:
    go mod tidy
    go mod verify
    go mod download

# Clean build artifacts
clean:
    rm -f figgo
    rm -f coverage.out coverage.html
    go clean -cache -testcache

# Install the figgo binary to GOPATH/bin
install:
    go install ./cmd/figgo

# Run the figgo binary with example text
run text="Hello, World!":
    go run ./cmd/figgo "{{text}}"

# Show Go environment information
env:
    go env
    go version

# Run tests with verbose output and coverage
test-verbose:
    go test -v -cover -race ./...

# Update all dependencies to latest versions
update-deps:
    go get -u ./...
    go mod tidy

# Verify the project compiles for multiple platforms
verify-build:
    GOOS=linux GOARCH=amd64 go build -o /dev/null ./cmd/figgo
    GOOS=linux GOARCH=arm64 go build -o /dev/null ./cmd/figgo
    GOOS=darwin GOARCH=amd64 go build -o /dev/null ./cmd/figgo
    GOOS=darwin GOARCH=arm64 go build -o /dev/null ./cmd/figgo
    GOOS=windows GOARCH=amd64 go build -o /dev/null ./cmd/figgo
    @echo "Cross-compilation verification successful!"

# Run security vulnerability check
vuln:
    @if command -v govulncheck >/dev/null 2>&1; then \
        govulncheck ./...; \
    else \
        echo "govulncheck not installed, install with: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
    fi