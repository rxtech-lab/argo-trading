.PHONY: generate clean test lint build-swift-argo test-swift-e2e test-swift-e2e-only clean-swift

# Generate Go code from proto files and run go generate
generate:
	cd pkg/strategy && protoc --go-plugin_out=. --go-plugin_opt=paths=source_relative strategy.proto
	go generate ./...

# Version to inject (default: main for development builds)
VERSION ?= main

build-swift-argo:
	gomobile init
	gomobile bind -ldflags "-X github.com/rxtech-lab/argo-trading/internal/version.Version=$(VERSION)" \
		-target=macos -o pkg/swift-argo/ArgoTrading.xcframework \
		github.com/rxtech-lab/argo-trading/pkg/swift-argo
	cp pkg/swift-argo/duckdb.h pkg/swift-argo/ArgoTrading.xcframework/macos-arm64_x86_64/ArgoTrading.framework/Headers

# Clean generated files
clean:
	cd pkg/strategy && rm -f *.pb.go

# run golangci-lint
lint:
	./scripts/lint.sh

test:
	go test ./...

# Build xcframework and run Swift e2e tests
test-swift-e2e: build-swift-argo
	swift test --package-path e2e/swift-pkg

# Just run Swift tests (assumes xcframework already built)
test-swift-e2e-only:
	swift test --package-path e2e/swift-pkg

# Clean Swift build artifacts
clean-swift:
	swift package clean --package-path e2e/swift-pkg

fmt:
	go fmt ./...