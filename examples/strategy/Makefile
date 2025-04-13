.PHONY: clean build
# Clean generated files
clean:
	rm -f *.wasm

# Build the strategy into WASM
build:
	GOOS=wasip1 GOARCH=wasm go build -o plugin.wasm -buildmode=c-shared example_strategy.go