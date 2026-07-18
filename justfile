# List available commands
default:
    @just --list

# Build the binary
build:
    go build -o bin/velocirepo ./cmd/velocirepo

# Run all tests
test:
    go test ./...

# Run tests with verbose output
test-v:
    go test -v ./...

# Run linter
lint:
    golangci-lint run

# Install to ~/.local/bin
install: build
    mkdir -p ~/.local/bin
    cp bin/velocirepo ~/.local/bin/

# Clean build artifacts
clean:
    rm -rf bin/

great-docs-preview:
    uvx --from git+https://github.com/posit-dev/great-docs great-docs preview

great-docs-build:
    uvx --from git+https://github.com/posit-dev/great-docs great-docs build
