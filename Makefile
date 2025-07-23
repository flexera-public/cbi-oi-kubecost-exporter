.PHONY: generate test depend lint fmt vet check-fmt security clean all

# Default target
all: depend fmt vet lint test

generate: depend test
	# Extract appVersion from Chart.yaml and update it in values.yaml
	@VERSION=$$(grep 'appVersion:' ./helm-chart/Chart.yaml | awk '{print $$2}' | tr -d '"') && \
	sed -i '' -E "s/^(  tag: ).*/\1\"$$VERSION\"/" ./helm-chart/values.yaml

	# Run go generate
	@go generate ./...

	# Update README
	@cd ./helm-chart && helm-docs

	# Package the Helm chart
	@cd ./helm-chart && helm package .

	# Update Helm repo index
	@helm repo index .

# Run all checks before tests
check: depend fmt vet lint

# Run Go tests
test:
	@echo "Running tests..."
	@go test -v -race -cover ./...

# Format Go code
fmt:
	@echo "Formatting Go code..."
	@go fmt ./...

# Check if code is formatted
check-fmt:
	@echo "Checking code formatting..."
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "The following files are not formatted:"; \
		gofmt -l .; \
		echo "Please run 'make fmt' to format them."; \
		exit 1; \
	fi

# Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...

# Run linters
lint: depend-lint
	@echo "Running golangci-lint..."
	@golangci-lint run --timeout=5m

# Security checks
security: depend-security
	@echo "Running security checks..."
	@gosec ./...

# Install dependencies
depend:
	@echo "Installing Go dependencies..."
	@go mod download

	@echo "Installing helm-docs..."
	@go install github.com/norwoodj/helm-docs/cmd/helm-docs@latest

# Install linting dependencies
depend-lint:
	@echo "Installing golangci-lint..."
	@which golangci-lint > /dev/null || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin

# Install security dependencies
depend-security:
	@echo "Installing gosec..."
	@which gosec > /dev/null || go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest

# Clean build artifacts
clean:
	@echo "Cleaning up..."
	@rm -f coverage.out coverage.html
	@go clean ./...

# Run CI pipeline (what GitHub Actions should run)
ci: depend check-fmt vet lint test

# Development workflow
dev: fmt vet test