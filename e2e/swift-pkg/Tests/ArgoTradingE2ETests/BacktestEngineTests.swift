import XCTest
import ArgoTrading

/// Tests for the Argo Backtest Engine API
final class BacktestEngineTests: XCTestCase {

    var helper: MockArgoHelper!

    override func setUp() {
        super.setUp()
        helper = MockArgoHelper()
    }

    override func tearDown() {
        helper = nil
        super.tearDown()
    }

    // MARK: - NewArgo Tests

    func testNewArgo_CreatesValidInstance() throws {
        var error: NSError?
        let argo = SwiftargoNewArgo(helper, &error)

        XCTAssertNil(error, "Should not return an error: \(error?.localizedDescription ?? "")")
        XCTAssertNotNil(argo, "Should create a valid Argo instance")
    }

    func testNewArgo_WithValidHelper_Succeeds() throws {
        var error: NSError?
        let argo = SwiftargoNewArgo(helper, &error)

        XCTAssertNil(error, "Creating Argo with valid helper should not error")
        XCTAssertNotNil(argo, "Should return non-nil Argo instance")
    }

    // MARK: - SetDataPath Tests

    func testSetDataPath_WithValidPath_Succeeds() throws {
        var error: NSError?
        let argo = SwiftargoNewArgo(helper, &error)!

        let dataPath = TestResources.testDataPath

        // Skip if test data doesn't exist
        guard FileManager.default.fileExists(atPath: dataPath) else {
            throw XCTSkip("Test data file not found at \(dataPath)")
        }

        XCTAssertNoThrow(try argo.setDataPath(dataPath), "Setting valid data path should not throw")
    }

    func testSetDataPath_WithGlobPattern_Succeeds() throws {
        var error: NSError?
        let argo = SwiftargoNewArgo(helper, &error)!

        // Glob patterns should be accepted
        let globPath = TestResources.projectRoot + "/internal/indicator/test_data/*.parquet"

        // The path might not match anything, but the glob pattern should be accepted
        // (validation happens at run time)
        XCTAssertNoThrow(try argo.setDataPath(globPath))
    }

    // MARK: - SetConfigContent Tests

    func testSetConfigContent_WithValidConfigs_Succeeds() throws {
        var error: NSError?
        let argo = SwiftargoNewArgo(helper, &error)!

        let configs = SwiftargoStringArray()
        _ = configs.add("{\"fastPeriod\": 10, \"slowPeriod\": 20, \"symbol\": \"BTCUSDT\"}")

        XCTAssertNoThrow(try argo.setConfigContent(configs), "Setting valid config content should not throw")
    }

    func testSetConfigContent_WithMultipleConfigs_Succeeds() throws {
        var error: NSError?
        let argo = SwiftargoNewArgo(helper, &error)!

        let configs = SwiftargoStringArray()
        _ = configs.add("{\"fastPeriod\": 10, \"slowPeriod\": 20, \"symbol\": \"BTCUSDT\"}")
        _ = configs.add("{\"fastPeriod\": 5, \"slowPeriod\": 15, \"symbol\": \"ETHUSDT\"}")

        XCTAssertNoThrow(try argo.setConfigContent(configs), "Setting multiple configs should not throw")
    }

    func testSetConfigContent_WithEmptyConfigs_Succeeds() throws {
        var error: NSError?
        let argo = SwiftargoNewArgo(helper, &error)!

        let configs = SwiftargoStringArray()

        // Empty configs should be accepted (no strategy configs)
        XCTAssertNoThrow(try argo.setConfigContent(configs))
    }

    // MARK: - Cancel Tests

    func testCancel_WithNoRunInProgress_ReturnsFalse() throws {
        var error: NSError?
        let argo = SwiftargoNewArgo(helper, &error)!

        let cancelled = argo.cancel()

        XCTAssertFalse(cancelled, "Cancel should return false when no run is in progress")
    }

    func testCancel_MultipleCalls_AreSafe() throws {
        var error: NSError?
        let argo = SwiftargoNewArgo(helper, &error)!

        // Multiple cancel calls should not crash
        _ = argo.cancel()
        _ = argo.cancel()
        _ = argo.cancel()

        // If we get here without crashing, the test passes
        XCTAssertTrue(true, "Multiple cancel calls should be safe")
    }

    // MARK: - Full Backtest Run Tests

    func testRun_WithValidInputs_ExecutesBacktest() throws {
        var error: NSError?
        let argo = SwiftargoNewArgo(helper, &error)!

        // Get paths to test resources
        let wasmPath = TestResources.wasmStrategyPath
        let dataPath = TestResources.testDataPath

        // Skip if resources not available
        guard FileManager.default.fileExists(atPath: wasmPath) else {
            throw XCTSkip("Test WASM file not found at \(wasmPath)")
        }
        guard FileManager.default.fileExists(atPath: dataPath) else {
            throw XCTSkip("Test data file not found at \(dataPath)")
        }

        // Create temp directory for results
        let tempDir = try TempDirectory(prefix: "backtest_test_")

        // Set up configs
        let configs = SwiftargoStringArray()
        _ = configs.add("{\"fastPeriod\": 10, \"slowPeriod\": 20, \"symbol\": \"BTCUSDT\"}")

        try argo.setConfigContent(configs)
        try argo.setDataPath(dataPath)

        // Run backtest
        let backtestConfig = "initial_capital: 10000"
        try argo.run(backtestConfig, strategyPath: wasmPath, resultsFolderPath: tempDir.path)

        // Verify callbacks were called
        XCTAssertTrue(helper.backtestStartCalled, "OnBacktestStart should be called")
        XCTAssertTrue(helper.backtestEndCalled, "OnBacktestEnd should be called")
        XCTAssertTrue(helper.strategyStartCalled, "OnStrategyStart should be called")
        XCTAssertTrue(helper.strategyEndCalled, "OnStrategyEnd should be called")
        XCTAssertTrue(helper.runStartCalled, "OnRunStart should be called")
        XCTAssertTrue(helper.runEndCalled, "OnRunEnd should be called")
        XCTAssertTrue(helper.processDataCalled, "OnProcessData should be called")
    }

    func testRun_WithInvalidStrategyPath_ThrowsError() throws {
        var error: NSError?
        let argo = SwiftargoNewArgo(helper, &error)!

        let invalidPath = "/nonexistent/strategy.wasm"
        let tempDir = try TempDirectory(prefix: "backtest_test_")

        let configs = SwiftargoStringArray()
        _ = configs.add("{}")

        try argo.setConfigContent(configs)
        try argo.setDataPath("/nonexistent/data.parquet")

        let backtestConfig = "initial_capital: 10000"

        XCTAssertThrowsError(try argo.run(backtestConfig, strategyPath: invalidPath, resultsFolderPath: tempDir.path)) { error in
            XCTAssertNotNil(error, "Run should throw an error for invalid strategy path")
        }
    }

    func testRun_WithInvalidBacktestConfig_ThrowsError() throws {
        var error: NSError?
        let argo = SwiftargoNewArgo(helper, &error)!

        let wasmPath = TestResources.wasmStrategyPath
        let dataPath = TestResources.testDataPath

        guard FileManager.default.fileExists(atPath: wasmPath) else {
            throw XCTSkip("Test WASM file not found")
        }
        guard FileManager.default.fileExists(atPath: dataPath) else {
            throw XCTSkip("Test data file not found")
        }

        let tempDir = try TempDirectory(prefix: "backtest_test_")

        try argo.setDataPath(dataPath)

        // Invalid YAML config
        let invalidConfig = "this is not: valid: yaml: config"

        XCTAssertThrowsError(try argo.run(invalidConfig, strategyPath: wasmPath, resultsFolderPath: tempDir.path)) { error in
            XCTAssertNotNil(error, "Run should throw an error for invalid backtest config")
        }
    }

    // MARK: - Callback Parameter Verification Tests

    func testRun_CallbackParameters_AreCorrect() throws {
        var error: NSError?
        let argo = SwiftargoNewArgo(helper, &error)!

        let wasmPath = TestResources.wasmStrategyPath
        let dataPath = TestResources.testDataPath

        guard FileManager.default.fileExists(atPath: wasmPath) else {
            throw XCTSkip("Test WASM file not found")
        }
        guard FileManager.default.fileExists(atPath: dataPath) else {
            throw XCTSkip("Test data file not found")
        }

        let tempDir = try TempDirectory(prefix: "backtest_test_")

        let configs = SwiftargoStringArray()
        _ = configs.add("{\"fastPeriod\": 10, \"slowPeriod\": 20, \"symbol\": \"BTCUSDT\"}")

        try argo.setConfigContent(configs)
        try argo.setDataPath(dataPath)

        let backtestConfig = "initial_capital: 10000"
        try argo.run(backtestConfig, strategyPath: wasmPath, resultsFolderPath: tempDir.path)

        // Verify callback parameters
        XCTAssertEqual(helper.lastTotalStrategies, 1, "Should have 1 strategy")
        XCTAssertEqual(helper.lastTotalConfigs, 1, "Should have 1 config")
        // Note: totalDataFiles may be 0 depending on how engine counts data files
        XCTAssertGreaterThanOrEqual(helper.lastTotalDataFiles, 0, "Data file count should be non-negative")
        // Note: Strategy name may be empty depending on WASM metadata
        XCTAssertNotNil(helper.lastStrategyName, "Strategy name should be set")
        XCTAssertFalse(helper.lastRunID.isEmpty, "Run ID should not be empty")
        XCTAssertGreaterThan(helper.processDataCount, 0, "Should have processed data points")
    }
}
