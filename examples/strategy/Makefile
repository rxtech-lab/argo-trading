.PHONY: generate clean build-wasm clean-wasm

# Generate Go code from proto files
generate:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-plugin_out=. --go-plugin_opt=paths=source_relative \
		strategy.proto

# Clean generated files
clean:
	rm -f *.pb.go
	rm -f plugin.wasm

# Build the strategy into WASM
build-wasm: generate
	GOOS=wasip1 GOARCH=wasm go build -o plugin.wasm -buildmode=c-shared example_strategy.go

# Clean WASM files
clean-wasm:
	rm -f plugin.wasm 