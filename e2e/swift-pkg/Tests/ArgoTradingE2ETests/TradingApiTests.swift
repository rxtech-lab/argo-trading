import ArgoTrading
import XCTest

/// Tests for trading-related APIs (provider discovery, engine creation, configuration)
final class TradingApiTests: XCTestCase {

    // MARK: - GetSupportedTradingProviders Tests

    func testGetSupportedTradingProviders_ReturnsNonEmptyCollection() {
        let providers = SwiftargoGetSupportedTradingProviders()

        XCTAssertNotNil(providers, "Supported providers should not be nil")
        XCTAssertGreaterThan(providers!.size(), 0, "Should have at least one supported provider")
    }

    func testGetSupportedTradingProviders_ContainsBinancePaper() {
        let providers = SwiftargoGetSupportedTradingProviders()!

        var hasBinancePaper = false
        for i in 0..<providers.size() {
            if providers.get(i) == "binance-paper" {
                hasBinancePaper = true
                break
            }
        }

        XCTAssertTrue(hasBinancePaper, "Supported providers should include 'binance-paper'")
    }

    func testGetSupportedTradingProviders_ContainsBinanceLive() {
        let providers = SwiftargoGetSupportedTradingProviders()!

        var hasBinanceLive = false
        for i in 0..<providers.size() {
            if providers.get(i) == "binance-live" {
                hasBinanceLive = true
                break
            }
        }

        XCTAssertTrue(hasBinanceLive, "Supported providers should include 'binance-live'")
    }

    func testGetSupportedTradingProviders_ReturnsConsistentResults() {
        let providers1 = SwiftargoGetSupportedTradingProviders()!
        let providers2 = SwiftargoGetSupportedTradingProviders()!

        XCTAssertEqual(
            providers1.size(), providers2.size(),
            "Should return same number of providers on multiple calls")
    }

    // MARK: - GetTradingProviderKeychainFields Tests

    func testGetTradingProviderKeychainFields_BinancePaperReturnsFields() {
        let fields = SwiftargoGetTradingProviderKeychainFields("binance-paper")

        XCTAssertNotNil(fields, "Binance paper should have keychain fields")
        XCTAssertEqual(fields!.size(), 2, "Binance paper should have 2 keychain fields")
        XCTAssertEqual(fields!.get(0), "apiKey", "First keychain field should be apiKey")
        XCTAssertEqual(fields!.get(1), "secretKey", "Second keychain field should be secretKey")
    }

    func testGetTradingProviderKeychainFields_BinanceLiveReturnsFields() {
        let fields = SwiftargoGetTradingProviderKeychainFields("binance-live")

        XCTAssertNotNil(fields, "Binance live should have keychain fields")
        XCTAssertEqual(fields!.size(), 2, "Binance live should have 2 keychain fields")
        XCTAssertEqual(fields!.get(0), "apiKey", "First keychain field should be apiKey")
        XCTAssertEqual(fields!.get(1), "secretKey", "Second keychain field should be secretKey")
    }

    func testGetTradingProviderKeychainFields_InvalidProviderReturnsNil() {
        let fields = SwiftargoGetTradingProviderKeychainFields("invalid")

        XCTAssertNil(fields, "Invalid provider should return nil")
    }

    // MARK: - GetTradingProviderSchema Tests

    func testGetTradingProviderSchema_BinancePaperReturnsValidJSON() {
        let schema = SwiftargoGetTradingProviderSchema("binance-paper")

        XCTAssertFalse(schema.isEmpty, "Binance paper schema should not be empty")
        XCTAssertTrue(isValidJSON(schema), "Binance paper schema should be valid JSON")
    }

    func testGetTradingProviderSchema_BinancePaperContainsRequiredFields() {
        let schema = SwiftargoGetTradingProviderSchema("binance-paper")

        // Binance provider config should have these fields
        XCTAssertTrue(
            schema.contains("apiKey") || schema.contains("api_key") || schema.contains("ApiKey"),
            "Binance paper schema should contain apiKey field")
        XCTAssertTrue(
            schema.contains("secretKey") || schema.contains("secret_key") || schema.contains("SecretKey"),
            "Binance paper schema should contain secretKey field")
    }

    func testGetTradingProviderSchema_BinanceLiveReturnsValidJSON() {
        let schema = SwiftargoGetTradingProviderSchema("binance-live")

        XCTAssertFalse(schema.isEmpty, "Binance live schema should not be empty")
        XCTAssertTrue(isValidJSON(schema), "Binance live schema should be valid JSON")
    }

    func testGetTradingProviderSchema_BinanceLiveContainsRequiredFields() {
        let schema = SwiftargoGetTradingProviderSchema("binance-live")

        // Binance provider config should have these fields
        XCTAssertTrue(
            schema.contains("apiKey") || schema.contains("api_key") || schema.contains("ApiKey"),
            "Binance live schema should contain apiKey field")
        XCTAssertTrue(
            schema.contains("secretKey") || schema.contains("secret_key") || schema.contains("SecretKey"),
            "Binance live schema should contain secretKey field")
    }

    func testGetTradingProviderSchema_InvalidProviderReturnsEmptyString() {
        let schema = SwiftargoGetTradingProviderSchema("invalid_provider")

        XCTAssertEqual(schema, "", "Invalid provider should return empty string")
    }

    func testGetTradingProviderSchema_EmptyProviderReturnsEmptyString() {
        let schema = SwiftargoGetTradingProviderSchema("")

        XCTAssertEqual(schema, "", "Empty provider should return empty string")
    }

    func testGetTradingProviderSchema_CaseSensitive() {
        let schemaLower = SwiftargoGetTradingProviderSchema("binance-paper")
        let schemaUpper = SwiftargoGetTradingProviderSchema("BINANCE-PAPER")
        let schemaMixed = SwiftargoGetTradingProviderSchema("Binance-Paper")

        // Provider names should be lowercase with hyphen
        XCTAssertFalse(schemaLower.isEmpty, "Lowercase 'binance-paper' should work")
        XCTAssertTrue(schemaUpper.isEmpty, "Uppercase 'BINANCE-PAPER' should not work")
        XCTAssertTrue(schemaMixed.isEmpty, "Mixed case 'Binance-Paper' should not work")
    }

    // MARK: - GetTradingProviderInfo Tests

    func testGetTradingProviderInfo_BinancePaperReturnsCorrectInfo() {
        var error: NSError?
        let info = SwiftargoGetTradingProviderInfo("binance-paper", &error)

        XCTAssertNil(error, "Should not return error for valid provider")
        XCTAssertNotNil(info, "Should return provider info")
        XCTAssertEqual(info?.name, "binance-paper")
        XCTAssertTrue(info?.isPaperTrading ?? false, "Binance paper should be paper trading")
        XCTAssertFalse(info?.displayName.isEmpty ?? true, "Display name should not be empty")
        XCTAssertFalse(info?.description.isEmpty ?? true, "Description should not be empty")
    }

    func testGetTradingProviderInfo_BinanceLiveReturnsCorrectInfo() {
        var error: NSError?
        let info = SwiftargoGetTradingProviderInfo("binance-live", &error)

        XCTAssertNil(error, "Should not return error for valid provider")
        XCTAssertNotNil(info, "Should return provider info")
        XCTAssertEqual(info?.name, "binance-live")
        XCTAssertFalse(info?.isPaperTrading ?? true, "Binance live should not be paper trading")
        XCTAssertFalse(info?.displayName.isEmpty ?? true, "Display name should not be empty")
        XCTAssertFalse(info?.description.isEmpty ?? true, "Description should not be empty")
    }

    func testGetTradingProviderInfo_InvalidProviderReturnsError() {
        var error: NSError?
        let info = SwiftargoGetTradingProviderInfo("invalid", &error)

        XCTAssertNotNil(error, "Should return error for invalid provider")
        XCTAssertNil(info, "Should not return info for invalid provider")
    }

    // MARK: - GetLiveTradingEngineConfigSchema Tests

    func testGetLiveTradingEngineConfigSchema_ReturnsValidJSON() {
        let schema = SwiftargoGetLiveTradingEngineConfigSchema()

        XCTAssertFalse(schema.isEmpty, "Engine config schema should not be empty")
        XCTAssertTrue(isValidJSON(schema), "Engine config schema should be valid JSON")
    }

    func testGetLiveTradingEngineConfigSchema_ContainsRequiredFields() {
        let schema = SwiftargoGetLiveTradingEngineConfigSchema()

        // Engine config should have these fields
        XCTAssertTrue(schema.contains("symbols"), "Schema should contain symbols field")
        XCTAssertTrue(schema.contains("interval"), "Schema should contain interval field")
    }

    // MARK: - GetSupportedMarketDataProviders Tests

    func testGetSupportedMarketDataProviders_ReturnsNonEmptyCollection() {
        let providers = SwiftargoGetSupportedMarketDataProviders()

        XCTAssertNotNil(providers, "Supported market data providers should not be nil")
        XCTAssertGreaterThan(providers!.size(), 0, "Should have at least one supported provider")
    }

    func testGetSupportedMarketDataProviders_ContainsBinance() {
        let providers = SwiftargoGetSupportedMarketDataProviders()!

        var hasBinance = false
        for i in 0..<providers.size() {
            if providers.get(i) == "binance" {
                hasBinance = true
                break
            }
        }

        XCTAssertTrue(hasBinance, "Supported market data providers should include 'binance'")
    }

    func testGetSupportedMarketDataProviders_ContainsPolygon() {
        let providers = SwiftargoGetSupportedMarketDataProviders()!

        var hasPolygon = false
        for i in 0..<providers.size() {
            if providers.get(i) == "polygon" {
                hasPolygon = true
                break
            }
        }

        XCTAssertTrue(hasPolygon, "Supported market data providers should include 'polygon'")
    }

    // MARK: - NewTradingEngine Tests

    func testNewTradingEngine_CreatesValidInstance() throws {
        let helper = MockTradingEngineHelper()
        var error: NSError?
        let engine = SwiftargoNewTradingEngine(helper, &error)

        XCTAssertNil(error, "Should not return an error: \(error?.localizedDescription ?? "")")
        XCTAssertNotNil(engine, "Should create a valid TradingEngine instance")
    }

    func testNewTradingEngine_WithNilHelper_Succeeds() throws {
        var error: NSError?
        let engine = SwiftargoNewTradingEngine(nil, &error)

        XCTAssertNil(error, "Creating TradingEngine with nil helper should not error")
        XCTAssertNotNil(engine, "Should return non-nil TradingEngine instance")
    }

    // MARK: - Cancel Tests

    func testCancel_WithNoRunInProgress_ReturnsFalse() throws {
        let helper = MockTradingEngineHelper()
        var error: NSError?
        let engine = SwiftargoNewTradingEngine(helper, &error)!

        let cancelled = engine.cancel()

        XCTAssertFalse(cancelled, "Cancel should return false when no run is in progress")
    }

    func testCancel_MultipleCalls_AreSafe() throws {
        let helper = MockTradingEngineHelper()
        var error: NSError?
        let engine = SwiftargoNewTradingEngine(helper, &error)!

        // Multiple cancel calls should not crash
        _ = engine.cancel()
        _ = engine.cancel()
        _ = engine.cancel()

        // If we get here without crashing, the test passes
        XCTAssertTrue(true, "Multiple cancel calls should be safe")
    }

    // MARK: - Initialize Tests

    func testInitialize_WithValidConfig_Succeeds() throws {
        let helper = MockTradingEngineHelper()
        var error: NSError?
        let engine = SwiftargoNewTradingEngine(helper, &error)!

        let configJSON = """
        {
            "symbols": ["BTCUSDT"],
            "interval": "1m",
            "market_data_cache_size": 1000,
            "enable_logging": false
        }
        """

        XCTAssertNoThrow(try engine.initialize(configJSON))
    }

    func testInitialize_WithInvalidJSON_ThrowsError() throws {
        let helper = MockTradingEngineHelper()
        var error: NSError?
        let engine = SwiftargoNewTradingEngine(helper, &error)!

        let invalidConfig = "not valid json"

        XCTAssertThrowsError(try engine.initialize(invalidConfig)) { err in
            XCTAssertNotNil(err, "Should throw error for invalid JSON")
        }
    }

    // MARK: - SetWasm Tests

    func testSetWasm_WithInvalidPath_ThrowsError() throws {
        let helper = MockTradingEngineHelper()
        var error: NSError?
        let engine = SwiftargoNewTradingEngine(helper, &error)!

        let invalidPath = "/nonexistent/strategy.wasm"

        XCTAssertThrowsError(try engine.setWasm(invalidPath)) { err in
            XCTAssertNotNil(err, "Should throw error for invalid WASM path")
        }
    }

    func testSetWasm_WithValidPath_Succeeds() throws {
        let helper = MockTradingEngineHelper()
        var error: NSError?
        let engine = SwiftargoNewTradingEngine(helper, &error)!

        let wasmPath = TestResources.wasmStrategyPath

        guard FileManager.default.fileExists(atPath: wasmPath) else {
            throw XCTSkip("Test WASM file not found at \(wasmPath)")
        }

        XCTAssertNoThrow(try engine.setWasm(wasmPath))
    }

    // MARK: - SetTradingProvider Tests

    func testSetTradingProvider_WithInvalidProvider_ThrowsError() throws {
        let helper = MockTradingEngineHelper()
        var error: NSError?
        let engine = SwiftargoNewTradingEngine(helper, &error)!

        XCTAssertThrowsError(try engine.setTradingProvider("invalid", configJSON: "{}")) { err in
            XCTAssertNotNil(err, "Should throw error for invalid provider")
        }
    }

    func testSetTradingProvider_WithInvalidJSON_ThrowsError() throws {
        let helper = MockTradingEngineHelper()
        var error: NSError?
        let engine = SwiftargoNewTradingEngine(helper, &error)!

        XCTAssertThrowsError(try engine.setTradingProvider("binance-paper", configJSON: "invalid json")) { err in
            XCTAssertNotNil(err, "Should throw error for invalid JSON")
        }
    }

    func testSetTradingProvider_WithMissingRequiredFields_ThrowsError() throws {
        let helper = MockTradingEngineHelper()
        var error: NSError?
        let engine = SwiftargoNewTradingEngine(helper, &error)!

        // Missing apiKey and secretKey
        XCTAssertThrowsError(try engine.setTradingProvider("binance-paper", configJSON: "{}")) { err in
            XCTAssertNotNil(err, "Should throw error for missing required fields")
        }
    }

    // MARK: - SetMarketDataProvider Tests

    func testSetMarketDataProvider_WithBinance_Succeeds() throws {
        let helper = MockTradingEngineHelper()
        var error: NSError?
        let engine = SwiftargoNewTradingEngine(helper, &error)!

        // Binance doesn't require config
        XCTAssertNoThrow(try engine.setMarketDataProvider("binance", configJSON: "{}"))
    }

    func testSetMarketDataProvider_WithInvalidProvider_ThrowsError() throws {
        let helper = MockTradingEngineHelper()
        var error: NSError?
        let engine = SwiftargoNewTradingEngine(helper, &error)!

        XCTAssertThrowsError(try engine.setMarketDataProvider("invalid", configJSON: "{}")) { err in
            XCTAssertNotNil(err, "Should throw error for invalid provider")
        }
    }

    func testSetMarketDataProvider_PolygonMissingApiKey_ThrowsError() throws {
        let helper = MockTradingEngineHelper()
        var error: NSError?
        let engine = SwiftargoNewTradingEngine(helper, &error)!

        // Polygon requires apiKey
        XCTAssertThrowsError(try engine.setMarketDataProvider("polygon", configJSON: "{}")) { err in
            XCTAssertNotNil(err, "Should throw error for missing apiKey")
        }
    }

    // MARK: - SetStrategyConfig Tests

    func testSetStrategyConfig_Succeeds() throws {
        let helper = MockTradingEngineHelper()
        var error: NSError?
        let engine = SwiftargoNewTradingEngine(helper, &error)!

        let config = """
        {"fastPeriod": 10, "slowPeriod": 20, "symbol": "BTCUSDT"}
        """

        XCTAssertNoThrow(try engine.setStrategyConfig(config))
    }
}
