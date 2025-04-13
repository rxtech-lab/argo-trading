.PHONY: clean build
# Build the strategy into WASM
build:
	GOOS=wasip1 GOARCH=wasm go build -o sma_plugin.wasm -buildmode=c-shared ./sma/simple_ma_strategy.go

# Clean WASM files
clean:
	rm -f *.wasm