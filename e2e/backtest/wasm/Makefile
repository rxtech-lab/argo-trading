.PHONY: clean build
# Build the strategy into WASM
build:
	GOOS=wasip1 GOARCH=wasm go build -o ./sma/sma_plugin.wasm -buildmode=c-shared ./sma/simple_ma_strategy.go
	GOOS=wasip1 GOARCH=wasm go build -o ./place_order/place_order_plugin.wasm -buildmode=c-shared ./place_order/place_order.go

# Clean WASM files
clean:
	rm -f *.wasm