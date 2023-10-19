.PHONY: generate test depend

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

# Run Go tests
test:
	@go test ./...

# Install dependencies
depend:
	# Install Go dependencies
	@go mod download

	# Install helm-docs
	@go install github.com/norwoodj/helm-docs/cmd/helm-docs@latest
