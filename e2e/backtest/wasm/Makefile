.PHONY: clean build

# Build the strategy into WASM
build:
	GOOS=wasip1 GOARCH=wasm go build -o ./sma/sma_plugin.wasm -buildmode=c-shared ./sma/simple_ma_strategy.go
	GOOS=wasip1 GOARCH=wasm go build -o ./place_order/place_order_plugin.wasm -buildmode=c-shared ./place_order/place_order.go
	GOOS=wasip1 GOARCH=wasm go build -o ./place_fail_order/place_fail_order_plugin.wasm -buildmode=c-shared ./place_fail_order/place_fail_order.go
	GOOS=wasip1 GOARCH=wasm go build -o ./place_decimal_order/place_decimal_order_plugin.wasm -buildmode=c-shared ./place_decimal_order/place_decimal_order.go
	GOOS=wasip1 GOARCH=wasm go build -o ./complex_strategy/complex_strategy_plugin.wasm -buildmode=c-shared ./complex_strategy/complex_strategy.go
	GOOS=wasip1 GOARCH=wasm go build -o ./log_error/log_error_plugin.wasm -buildmode=c-shared ./log_error/log_error_strategy.go
	GOOS=wasip1 GOARCH=wasm go build -o ./portfolio_comparison/portfolio_comparison_plugin.wasm -buildmode=c-shared ./portfolio_comparison/portfolio_comparison.go
	GOOS=wasip1 GOARCH=wasm go build -o ./rsi_comparison/rsi_comparison_plugin.wasm -buildmode=c-shared ./rsi_comparison/rsi_comparison.go
	GOOS=wasip1 GOARCH=wasm go build -o ./multi_confirm/multi_confirm_plugin.wasm -buildmode=c-shared ./multi_confirm/multi_confirm_strategy.go
# Clean WASM files
clean:
	rm -f *.wasm