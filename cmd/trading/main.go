package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	engine "github.com/rxtech-lab/argo-trading/internal/trading/engine"
	enginev1 "github.com/rxtech-lab/argo-trading/internal/trading/engine/engine_v1"
	tradingprovider "github.com/rxtech-lab/argo-trading/internal/trading/provider"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/pkg/marketdata/provider"
)

func main() {
	// Define command-line flags
	strategyWasmFlag := flag.String("strategy-wasm", "", "Path to strategy WASM file (required)")
	strategyConfigFlag := flag.String("strategy-config", "", "Path to strategy configuration file")
	marketDataProviderFlag := flag.String("market-data-provider", "", "Market data provider: binance, polygon (required)")
	polygonApiKeyFlag := flag.String("polygon-api-key", "", "Polygon API key (required if provider=polygon)")
	tradingProviderFlag := flag.String("trading-provider", "", "Trading provider: binance-paper, binance-live (required)")
	tradingConfigFlag := flag.String("trading-config", "", "Path to trading provider config file (required)")
	symbolsFlag := flag.String("symbols", "", "Comma-separated list of symbols (required)")
	intervalFlag := flag.String("interval", "1m", "Candlestick interval")
	cacheSizeFlag := flag.Int("cache-size", 1000, "Market data cache size")
	logOutputFlag := flag.String("log-output", "", "Directory for log output files")

	flag.Parse()

	// Validate required flags
	if *strategyWasmFlag == "" {
		fmt.Println("Error: --strategy-wasm flag is required")
		flag.Usage()
		os.Exit(1)
	}
	if *marketDataProviderFlag == "" {
		fmt.Println("Error: --market-data-provider flag is required")
		flag.Usage()
		os.Exit(1)
	}
	if *tradingProviderFlag == "" {
		fmt.Println("Error: --trading-provider flag is required")
		flag.Usage()
		os.Exit(1)
	}
	if *tradingConfigFlag == "" {
		fmt.Println("Error: --trading-config flag is required")
		flag.Usage()
		os.Exit(1)
	}
	if *symbolsFlag == "" {
		fmt.Println("Error: --symbols flag is required")
		flag.Usage()
		os.Exit(1)
	}

	// Validate polygon API key if polygon provider
	if *marketDataProviderFlag == "polygon" && *polygonApiKeyFlag == "" {
		// Try environment variable
		*polygonApiKeyFlag = os.Getenv("POLYGON_API_KEY")
		if *polygonApiKeyFlag == "" {
			fmt.Println("Error: --polygon-api-key or POLYGON_API_KEY env required for polygon provider")
			os.Exit(1)
		}
	}

	// Parse symbols
	symbols := strings.Split(*symbolsFlag, ",")
	for i := range symbols {
		symbols[i] = strings.TrimSpace(symbols[i])
	}

	// Create engine
	eng, err := enginev1.NewLiveTradingEngineV1()
	if err != nil {
		log.Fatalf("Failed to create engine: %v", err)
	}

	// Initialize engine
	config := engine.LiveTradingEngineConfig{
		Symbols:             symbols,
		Interval:            *intervalFlag,
		MarketDataCacheSize: *cacheSizeFlag,
		EnableLogging:       *logOutputFlag != "",
		LogOutputPath:       *logOutputFlag,
	}
	if err := eng.Initialize(config); err != nil {
		log.Fatalf("Failed to initialize engine: %v", err)
	}

	// Set market data provider
	marketDataProvider, err := provider.NewMarketDataProvider(
		provider.ProviderType(*marketDataProviderFlag), *polygonApiKeyFlag)
	if err != nil {
		log.Fatalf("Failed to create market data provider: %v", err)
	}
	if err := eng.SetMarketDataProvider(marketDataProvider); err != nil {
		log.Fatalf("Failed to set market data provider: %v", err)
	}

	// Set trading provider
	tradingConfigBytes, err := os.ReadFile(*tradingConfigFlag)
	if err != nil {
		log.Fatalf("Failed to read trading config: %v", err)
	}
	var tradingConfig tradingprovider.BinanceProviderConfig
	if err := json.Unmarshal(tradingConfigBytes, &tradingConfig); err != nil {
		log.Fatalf("Failed to parse trading config: %v", err)
	}
	tradingProvider, err := tradingprovider.NewTradingSystemProvider(
		tradingprovider.ProviderType(*tradingProviderFlag), &tradingConfig)
	if err != nil {
		log.Fatalf("Failed to create trading provider: %v", err)
	}
	if err := eng.SetTradingProvider(tradingProvider); err != nil {
		log.Fatalf("Failed to set trading provider: %v", err)
	}

	// Load strategy
	if err := eng.LoadStrategyFromFile(*strategyWasmFlag); err != nil {
		log.Fatalf("Failed to load strategy: %v", err)
	}

	// Set strategy config if provided
	if *strategyConfigFlag != "" {
		strategyConfigBytes, err := os.ReadFile(*strategyConfigFlag)
		if err != nil {
			log.Fatalf("Failed to read strategy config: %v", err)
		}
		if err := eng.SetStrategyConfig(string(strategyConfigBytes)); err != nil {
			log.Fatalf("Failed to set strategy config: %v", err)
		}
	}

	// Setup callbacks
	onStart := engine.OnEngineStartCallback(func(symbols []string, interval string, previousDataPath string) error {
		fmt.Printf("Engine started: symbols=%v, interval=%s\n", symbols, interval)
		if previousDataPath != "" {
			fmt.Printf("Previous data available at: %s\n", previousDataPath)
		}
		return nil
	})
	onStop := engine.OnEngineStopCallback(func(err error) {
		if err != nil {
			fmt.Printf("Engine stopped with error: %v\n", err)
		} else {
			fmt.Println("Engine stopped")
		}
	})
	onMarketData := engine.OnMarketDataCallback(func(data types.MarketData) error {
		fmt.Printf("[%s] %s: O=%.4f H=%.4f L=%.4f C=%.4f V=%.2f\n",
			data.Time.Format("15:04:05"), data.Symbol,
			data.Open, data.High, data.Low, data.Close, data.Volume)
		return nil
	})
	onOrderPlaced := engine.OnOrderPlacedCallback(func(order types.ExecuteOrder) error {
		fmt.Printf("Order placed: %s %s %.4f @ %.4f\n",
			order.Side, order.Symbol, order.Quantity, order.Price)
		return nil
	})
	onError := engine.OnErrorCallback(func(err error) {
		fmt.Printf("Error: %v\n", err)
	})
	onStrategyError := engine.OnStrategyErrorCallback(func(data types.MarketData, err error) {
		fmt.Printf("Strategy error at %s: %v\n", data.Symbol, err)
	})

	callbacks := engine.LiveTradingCallbacks{
		OnEngineStart:   &onStart,
		OnEngineStop:    &onStop,
		OnMarketData:    &onMarketData,
		OnOrderPlaced:   &onOrderPlaced,
		OnError:         &onError,
		OnStrategyError: &onStrategyError,
	}

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal, stopping...")
		cancel()
	}()

	// Run engine
	fmt.Printf("Starting live trading with %d symbols...\n", len(symbols))
	err = eng.Run(ctx, callbacks)
	if err != nil {
		if err == context.Canceled {
			fmt.Println("Trading stopped by user")
			os.Exit(0)
		}
		log.Fatalf("Engine error: %v", err)
	}
}
