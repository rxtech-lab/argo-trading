---
title: Schema APIs for Swift
description: JSON schemas for market data providers and live trading engine configuration, exposed to Swift via gomobile
---

# Schema APIs for Swift

The schema APIs expose JSON schemas for configuring streaming market data providers and the live trading engine. This enables SwiftUI to dynamically generate configuration forms based on the provider's and engine's requirements.

## Swift API

### Available Functions

```swift
// Get list of supported providers
let providers: StringCollection = SwiftargoGetSupportedMarketDataProviders()
// Returns: ["binance", "polygon"]

// Get JSON schema for a provider's streaming config
let schema: String = SwiftargoGetMarketDataProviderSchema("binance")

// Get keychain-protected field names (e.g., API keys)
let keychainFields: StringCollection? = SwiftargoGetMarketDataProviderKeychainFields("polygon")
// Returns: ["apiKey"]
```

### Setting Up a Market Data Provider

```swift
// Configure with JSON matching the schema
let config = """
{
    "symbols": ["BTCUSDT", "ETHUSDT"],
    "interval": "1m"
}
"""
try engine.setMarketDataProvider("binance", config)

// Polygon requires an API key
let polygonConfig = """
{
    "symbols": ["SPY", "AAPL"],
    "interval": "1m",
    "apiKey": "your-polygon-api-key"
}
"""
try engine.setMarketDataProvider("polygon", polygonConfig)
```

## Provider Schemas

### Binance

```json
{
  "type": "object",
  "properties": {
    "symbols": {
      "type": "array",
      "items": { "type": "string" },
      "title": "Symbols",
      "description": "List of symbols to stream (e.g. BTCUSDT)"
    },
    "interval": {
      "type": "string",
      "title": "Interval",
      "description": "Candlestick interval for streaming data",
      "enum": ["1s","1m","3m","5m","15m","30m","1h","2h","4h","6h","8h","12h","1d","3d","1w","1M"]
    }
  },
  "required": ["symbols", "interval"]
}
```

### Polygon

```json
{
  "type": "object",
  "properties": {
    "symbols": {
      "type": "array",
      "items": { "type": "string" },
      "title": "Symbols",
      "description": "List of symbols to stream (e.g. SPY)"
    },
    "interval": {
      "type": "string",
      "title": "Interval",
      "description": "Candlestick interval for streaming data",
      "enum": ["1s","1m","3m","5m","15m","30m","1h","2h","4h","6h","8h","12h","1d","3d","1w","1M"]
    },
    "apiKey": {
      "type": "string",
      "title": "API Key",
      "description": "Polygon.io API key for authentication"
    }
  },
  "required": ["symbols", "interval", "apiKey"]
}
```

## SwiftUI Dynamic Form Example

Use the schema to dynamically render a configuration form:

```swift
import SwiftUI
import Swiftargo

struct MarketDataProviderConfigView: View {
    @State private var selectedProvider: String = "binance"
    @State private var symbols: String = ""
    @State private var interval: String = "1m"
    @State private var apiKey: String = ""

    var body: some View {
        Form {
            // Provider selection
            Picker("Provider", selection: $selectedProvider) {
                let providers = SwiftargoGetSupportedMarketDataProviders()
                ForEach(0..<providers.size(), id: \.self) { i in
                    Text(providers.get(i)).tag(providers.get(i))
                }
            }

            // Common fields
            TextField("Symbols (comma-separated)", text: $symbols)
            Picker("Interval", selection: $interval) {
                Text("1m").tag("1m")
                Text("5m").tag("5m")
                Text("15m").tag("15m")
                Text("1h").tag("1h")
            }

            // Provider-specific fields (check keychain fields)
            let keychainFields = SwiftargoGetMarketDataProviderKeychainFields(selectedProvider)
            if keychainFields != nil {
                SecureField("API Key", text: $apiKey)
            }

            Button("Connect") {
                connect()
            }
        }
    }

    func connect() {
        let symbolArray = symbols.split(separator: ",").map { String($0).trimmingCharacters(in: .whitespaces) }
        var config: [String: Any] = [
            "symbols": symbolArray,
            "interval": interval
        ]
        if !apiKey.isEmpty {
            config["apiKey"] = apiKey
        }

        let jsonData = try! JSONSerialization.data(withJSONObject: config)
        let configJSON = String(data: jsonData, encoding: .utf8)!

        do {
            try engine.setMarketDataProvider(selectedProvider, configJSON)
        } catch {
            print("Failed to set provider: \(error)")
        }
    }
}
```

## Go API

### Getting the Schema

```go
import "github.com/rxtech-lab/argo-trading/pkg/marketdata/provider"

// Get JSON schema
schema, err := provider.GetStreamConfigSchema("polygon")

// Get keychain fields
fields, err := provider.GetStreamKeychainFields("polygon")
// Returns: ["apiKey"]

// Parse and validate config
config, err := provider.ParseStreamConfig("binance", `{"symbols": ["BTCUSDT"], "interval": "1m"}`)
```

### Config Structs

```go
// Base config shared by all providers
type BaseStreamConfig struct {
    Symbols  []string `json:"symbols"`
    Interval string   `json:"interval"`
}

// Polygon adds apiKey
type PolygonStreamConfig struct {
    BaseStreamConfig
    ApiKey string `json:"apiKey" keychain:"true"`
}

// Binance has no extra fields
type BinanceStreamConfig struct {
    BaseStreamConfig
}
```

## Live Trading Engine Config Schema

The live trading engine configuration schema is also exposed to Swift, enabling dynamic form generation for engine settings.

### Swift API

```swift
// Get JSON schema for engine configuration
let schema: String = SwiftargoGetLiveTradingEngineConfigSchema()
```

### Engine Config Schema

Note: `symbols` and `interval` are configured via the market data provider (see above), not the engine config.

```json
{
  "type": "object",
  "properties": {
    "market_data_cache_size": {
      "type": "integer",
      "description": "Number of market data points to cache per symbol",
      "default": 1000
    },
    "enable_logging": {
      "type": "boolean",
      "description": "Enable strategy log storage",
      "default": true
    },
    "prefetch": {
      "type": "object",
      "description": "Historical data prefetch configuration",
      "properties": {
        "enabled": { "type": "boolean", "description": "Enable historical data prefetch" },
        "start_time_type": { "type": "string", "enum": ["date", "days"], "description": "How to specify start time" },
        "start_time": { "type": "string", "format": "date-time", "description": "Absolute start time (when type is date)" },
        "days": { "type": "integer", "description": "Number of days to prefetch (when type is days)" }
      }
    }
  },
  "required": []
}
```

### Initializing the Engine with JSON

```swift
let engine = SwiftargoNewTradingEngine(helper, &error)

let configJSON = """
{
    "market_data_cache_size": 1000,
    "enable_logging": true,
    "prefetch": {
        "enabled": true,
        "start_time_type": "days",
        "days": 30
    }
}
"""
try engine.initialize(configJSON)

// Set data output path separately
try engine.setDataOutputPath("/path/to/data")

// Symbols and interval are configured via the market data provider
let providerConfig = """
{
    "symbols": ["BTCUSDT"],
    "interval": "1m"
}
"""
try engine.setMarketDataProvider("binance", providerConfig)
```

## Related Files

- [stream_config.go](../pkg/marketdata/provider/stream_config.go) - Market data provider config structs
- [stream_registry.go](../pkg/marketdata/provider/stream_registry.go) - Schema and keychain registry
- [engine.go](../internal/trading/engine/engine.go) - Engine config struct and schema generation
- [trading.go](../pkg/swift-argo/trading.go) - Swift bindings (`GetMarketDataProviderSchema`, `GetMarketDataProviderKeychainFields`, `GetLiveTradingEngineConfigSchema`)
