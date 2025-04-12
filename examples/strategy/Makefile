.PHONY: generate clean build-wasm clean-wasm
# Clean generated files
clean:
	rm -f *.pb.go
	rm -f plugin.wasm

# Build the strategy into WASM
build: generate
	GOOS=wasip1 GOARCH=wasm go build -o plugin.wasm -buildmode=c-shared example_strategy.go

# Clean WASM files
clean-wasm:
	rm -f plugin.wasm 