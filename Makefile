.PHONY: generate clean

# Generate Go code from proto files and run go generate
generate:
	cd pkg/strategy && protoc --go-plugin_out=. --go-plugin_opt=paths=source_relative strategy.proto
	go generate ./...

build-swift-argo:
	gomobile init
	gomobile bind -target=macos -o pkg/swift-argo/ArgoTrading.xcframework github.com/rxtech-lab/argo-trading/pkg/swift-argo
	cp pkg/swift-argo/duckdb.h pkg/swift-argo/ArgoTrading.xcframework/macos-arm64_x86_64/ArgoTrading.framework/Headers
# Clean generated files
clean:
	cd pkg/strategy && rm -f *.pb.go 

# run golangci-lint
lint:
	golangci-lint run ./...

test:
	go test ./...