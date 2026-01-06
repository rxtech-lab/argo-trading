import XCTest
import ArgoTrading

/// Mock implementation of ArgoHelper for testing backtest callbacks
class MockArgoHelper: NSObject, SwiftargoArgoHelperProtocol {
    var backtestStartCalled = false
    var backtestEndCalled = false
    var backtestEndError: Error?
    var strategyStartCalled = false
    var strategyEndCalled = false
    var runStartCalled = false
    var runEndCalled = false
    var processDataCalled = false

    var totalStrategies = 0
    var totalConfigs = 0
    var totalDataFiles = 0
    var strategyName = ""
    var runID = ""
    var resultFolderPath = ""
    var currentDataPoint = 0
    var totalDataPoints = 0

    func onBacktestEnd(_ err: (any Error)?) {
        backtestEndCalled = true
        backtestEndError = err
    }

    func onBacktestStart(_ totalStrategies: Int, totalConfigs: Int, totalDataFiles: Int) throws {
        backtestStartCalled = true
        self.totalStrategies = totalStrategies
        self.totalConfigs = totalConfigs
        self.totalDataFiles = totalDataFiles
    }

    func onProcessData(_ current: Int, total: Int) throws {
        processDataCalled = true
        currentDataPoint = current
        totalDataPoints = total
    }

    func onRunEnd(_ configIndex: Int, configName: String?, dataFileIndex: Int, dataFilePath: String?, resultFolderPath: String?) {
        runEndCalled = true
        self.resultFolderPath = resultFolderPath ?? ""
    }

    func onRunStart(_ runID: String?, configIndex: Int, configName: String?, dataFileIndex: Int, dataFilePath: String?, totalDataPoints: Int) throws {
        runStartCalled = true
        self.runID = runID ?? ""
        self.totalDataPoints = totalDataPoints
    }

    func onStrategyEnd(_ strategyIndex: Int, strategyName: String?) {
        strategyEndCalled = true
    }

    func onStrategyStart(_ strategyIndex: Int, strategyName: String?, totalStrategies: Int) throws {
        strategyStartCalled = true
        self.strategyName = strategyName ?? ""
        self.totalStrategies = totalStrategies
    }
}

final class ArgoBacktestTests: XCTestCase {

    /// Test that we can get the backtest engine config schema
    func testGetBacktestEngineConfigSchema() throws {
        let schema = SwiftargoGetBacktestEngineConfigSchema()

        XCTAssertNotNil(schema)
        XCTAssertFalse(schema!.isEmpty)

        // Verify it's valid JSON
        let data = schema!.data(using: .utf8)!
        let json = try JSONSerialization.jsonObject(with: data)
        XCTAssertNotNil(json)

        // Verify it contains expected fields
        if let jsonDict = json as? [String: Any] {
            XCTAssertNotNil(jsonDict["properties"])
        }
    }

    /// Test that we can get the backtest engine version
    func testGetBacktestEngineVersion() throws {
        let version = SwiftargoGetBacktestEngineVersion()

        XCTAssertNotNil(version)
        XCTAssertFalse(version!.isEmpty)
    }

    /// Test that we can create an Argo instance
    func testCreateArgoInstance() throws {
        let helper = MockArgoHelper()
        var error: NSError?

        let argo = SwiftargoNewArgo(helper, &error)

        XCTAssertNil(error)
        XCTAssertNotNil(argo)
    }

    /// Test backtest with a simple strategy
    func testBacktestWithStrategy() throws {
        // Skip if resources are not available
        guard let resourcesPath = Bundle.module.resourcePath else {
            throw XCTSkip("Resources not available")
        }

        let wasmPath = resourcesPath + "/Resources/sma_plugin.wasm"
        let dataPath = resourcesPath + "/Resources/test_data.parquet"
        let configPath = resourcesPath + "/Resources/config.json"

        // Check if all required files exist
        guard FileManager.default.fileExists(atPath: wasmPath) else {
            throw XCTSkip("WASM file not found at \(wasmPath)")
        }
        guard FileManager.default.fileExists(atPath: dataPath) else {
            throw XCTSkip("Data file not found at \(dataPath)")
        }
        guard FileManager.default.fileExists(atPath: configPath) else {
            throw XCTSkip("Config file not found at \(configPath)")
        }

        let helper = MockArgoHelper()
        var error: NSError?

        let argo = SwiftargoNewArgo(helper, &error)
        XCTAssertNil(error, "Failed to create Argo instance: \(error?.localizedDescription ?? "unknown")")
        XCTAssertNotNil(argo)

        // Set data path
        try argo!.setDataPath(dataPath)

        // Set config content
        let configs = SwiftargoNewStringArray()!
        let configContent = try String(contentsOfFile: configPath, encoding: .utf8)
        configs.add(configContent)
        try argo!.setConfigContent(configs)

        // Create temp directory for results
        let tempDir = FileManager.default.temporaryDirectory
            .appendingPathComponent("argo_test_\(UUID().uuidString)")
        try FileManager.default.createDirectory(at: tempDir, withIntermediateDirectories: true)
        defer {
            try? FileManager.default.removeItem(at: tempDir)
        }

        // Run backtest with engine config
        let engineConfig = """
        initial_capital: 10000
        """

        try argo!.run(engineConfig, strategyPath: wasmPath, resultsFolderPath: tempDir.path)

        // Verify callbacks were called
        XCTAssertTrue(helper.backtestStartCalled, "onBacktestStart should have been called")
        XCTAssertTrue(helper.backtestEndCalled, "onBacktestEnd should have been called")
        XCTAssertTrue(helper.strategyStartCalled, "onStrategyStart should have been called")
        XCTAssertTrue(helper.strategyEndCalled, "onStrategyEnd should have been called")
        XCTAssertTrue(helper.runStartCalled, "onRunStart should have been called")
        XCTAssertTrue(helper.runEndCalled, "onRunEnd should have been called")
        XCTAssertTrue(helper.processDataCalled, "onProcessData should have been called")

        // Verify no error occurred
        XCTAssertNil(helper.backtestEndError)

        // Verify result folder was set
        XCTAssertFalse(helper.resultFolderPath.isEmpty)
    }

    /// Test cancellation of backtest
    func testBacktestCancellation() throws {
        let helper = MockArgoHelper()
        var error: NSError?

        let argo = SwiftargoNewArgo(helper, &error)
        XCTAssertNil(error)
        XCTAssertNotNil(argo)

        // Cancel when no backtest is running should return false
        let cancelled = argo!.cancel()
        XCTAssertFalse(cancelled)
    }
}
