# Force bash as shell (required on Windows where make defaults to cmd.exe)
ifeq ($(OS),Windows_NT)
    SHELL := C:/Program Files/Git/usr/bin/bash.exe
else
    SHELL := /bin/bash
endif

.PHONY: build test unit-test integration-test update-expected clean

# Build the project using GoReleaser
build:
	@echo "========================================="
	@echo "  SimDiag - Local build with GoReleaser"
	@echo "========================================="
	@echo ""
	@# Check Go installation
	@command -v go >/dev/null 2>&1 || { echo "ERROR: Go is not installed or not in PATH"; exit 1; }
	@# Check/install GoReleaser
	@command -v goreleaser >/dev/null 2>&1 || { echo "Installing GoReleaser..."; go install github.com/goreleaser/goreleaser/v2@latest; }
	@# Compute version
	$(eval VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev"))
	@echo "Version: $(VERSION)"
	@echo ""
	@# Build snapshot
	SIMDIAG_VERSION=$(VERSION) goreleaser build --snapshot --clean
	@# Copy binary to project root
	@cp dist/simdiag-windows_windows_amd64_v1/simdiag.exe simdiag.exe 2>/dev/null || \
	 cp dist/simdiag_linux_amd64_v1/simdiag simdiag 2>/dev/null || \
	 cp dist/simdiag_darwin_amd64_v1/simdiag simdiag 2>/dev/null || \
	 { echo "ERROR: Binary not found in dist/"; exit 1; }
	@echo ""
	@echo "Build successful!"
	@echo "Binary copied to project root"

# Run all tests (unit + integration)
test: unit-test integration-test

# Run unit tests
unit-test:
	@echo "Running common package unit tests..."
	go test ./common/ -v
	@echo ""
	@echo "Running CSV package unit tests..."
	go test ./csv/ -v
	@echo ""
	@echo "Running SVG package unit tests..."
	go test ./svg/ -v
	@echo ""
	@echo "Running DCS parser unit tests..."
	go test ./dcs/ -v
	@echo ""
	@echo "Running IL-2 parser unit tests..."
	go test ./il2/ -v
	@echo ""
	@echo "Running OpenKneeboard unit tests..."
	go test ./openkneeboard/ -v
	@echo ""
	@echo "Running TARGET unit tests..."
	go test ./target/ -v
	@echo ""
	@echo "Running SRS unit tests..."
	go test ./srs/ -v
	@echo ""
	@echo "Running Gremlins unit tests..."
	go test ./gremlins/ -v

# Run integration tests
integration-test:
	@echo "Running integration tests..."
	go test ./tests/integration/ -v -count=1

# Update expected CSV after intentional code changes
update-expected:
	cd tests/integration && UPDATE_EXPECTED=1 go test -v -count=1

# Clean build artifacts and output directory
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf dist/
	@rm -rf output/
	@echo "Clean complete!"

release:
	@echo "========================================="
	@echo "  SimDiag - Create Release Tag"
	@echo "========================================="
	@echo ""
	@# Show latest tag
	$(eval LATEST_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "none"))
	@echo "Latest tag: $(LATEST_TAG)"
	@# Compute default next tag (increment patch)
	$(eval DEFAULT_TAG := $(shell \
			if [ "$(LATEST_TAG)" = "none" ]; then \
					echo "v0.1.0"; \
			else \
					echo "$(LATEST_TAG)" | awk -F. '{print $$1"."$$2"."$$3+1}'; \
			fi \
	))
	@read -p "New tag [$(DEFAULT_TAG)]: " TAG && TAG=$${TAG:-$(DEFAULT_TAG)} && \
			echo "" && \
			echo "Creating signed tag: $$TAG" && \
			git tag -s "$$TAG" -m "Release $$TAG" && \
			echo "Pushing tag $$TAG to origin..." && \
			git push origin "$$TAG" && \
			echo "" && \
			echo "Tag $$TAG pushed. GitHub Actions will create the release."