import ArgoTrading
import Foundation

/// Mock implementation of ArgoHelper for testing backtest callbacks
class MockArgoHelper: NSObject, SwiftargoArgoHelperProtocol {
    // Track callback invocations
    var backtestStartCalled = false
    var backtestEndCalled = false
    var strategyStartCalled = false
    var strategyEndCalled = false
    var runStartCalled = false
    var runEndCalled = false
    var processDataCalled = false

    // Store callback parameters for verification
    var lastTotalStrategies: Int = 0
    var lastTotalConfigs: Int = 0
    var lastTotalDataFiles: Int = 0
    var lastError: Error?
    var lastStrategyName: String = ""
    var lastRunID: String = ""
    var lastDataFilePath: String = ""
    var lastResultFolderPath: String = ""
    var processDataCount: Int = 0

    // Control callback behavior
    var shouldFailOnBacktestStart = false
    var shouldFailOnStrategyStart = false
    var shouldFailOnRunStart = false
    var shouldFailOnProcessData = false

    func onBacktestEnd(_ err: Error?) {
        backtestEndCalled = true
        lastError = err
    }

    func onBacktestStart(_ totalStrategies: Int, totalConfigs: Int, totalDataFiles: Int) throws {
        backtestStartCalled = true
        lastTotalStrategies = totalStrategies
        lastTotalConfigs = totalConfigs
        lastTotalDataFiles = totalDataFiles

        if shouldFailOnBacktestStart {
            throw NSError(
                domain: "TestError", code: 1,
                userInfo: [NSLocalizedDescriptionKey: "Test error in onBacktestStart"])
        }
    }

    func onProcessData(_ current: Int, total: Int) throws {
        processDataCalled = true
        processDataCount += 1

        if shouldFailOnProcessData {
            throw NSError(
                domain: "TestError", code: 4,
                userInfo: [NSLocalizedDescriptionKey: "Test error in onProcessData"])
        }
    }

    func onRunEnd(
        _ configIndex: Int, configName: String?, dataFileIndex: Int, dataFilePath: String?,
        resultFolderPath: String?
    ) {
        runEndCalled = true
        lastResultFolderPath = resultFolderPath ?? ""
    }

    func onRunStart(
        _ runID: String?, configIndex: Int, configName: String?, dataFileIndex: Int,
        dataFilePath: String?, totalDataPoints: Int
    ) throws {
        runStartCalled = true
        lastRunID = runID ?? ""
        lastDataFilePath = dataFilePath ?? ""

        if shouldFailOnRunStart {
            throw NSError(
                domain: "TestError", code: 3,
                userInfo: [NSLocalizedDescriptionKey: "Test error in onRunStart"])
        }
    }

    func onStrategyEnd(_ strategyIndex: Int, strategyName: String?) {
        strategyEndCalled = true
    }

    func onStrategyStart(_ strategyIndex: Int, strategyName: String?, totalStrategies: Int) throws {
        strategyStartCalled = true
        lastStrategyName = strategyName ?? ""

        if shouldFailOnStrategyStart {
            throw NSError(
                domain: "TestError", code: 2,
                userInfo: [NSLocalizedDescriptionKey: "Test error in onStrategyStart"])
        }
    }

    /// Reset all tracking state
    func reset() {
        backtestStartCalled = false
        backtestEndCalled = false
        strategyStartCalled = false
        strategyEndCalled = false
        runStartCalled = false
        runEndCalled = false
        processDataCalled = false
        lastTotalStrategies = 0
        lastTotalConfigs = 0
        lastTotalDataFiles = 0
        lastError = nil
        lastStrategyName = ""
        lastRunID = ""
        lastDataFilePath = ""
        lastResultFolderPath = ""
        processDataCount = 0
        shouldFailOnBacktestStart = false
        shouldFailOnStrategyStart = false
        shouldFailOnRunStart = false
        shouldFailOnProcessData = false
    }
}

/// Mock implementation of MarketDownloaderHelper for testing download progress callbacks
class MockMarketDownloaderHelper: NSObject, SwiftargoMarketDownloaderHelperProtocol {
    var progressCalled = false
    var lastCurrent: Double = 0
    var lastTotal: Double = 0
    var lastMessage: String = ""
    var progressCallCount: Int = 0

    func onDownloadProgress(_ current: Double, total: Double, message: String?) {
        progressCalled = true
        progressCallCount += 1
        lastCurrent = current
        lastTotal = total
        lastMessage = message ?? ""
    }

    func reset() {
        progressCalled = false
        lastCurrent = 0
        lastTotal = 0
        lastMessage = ""
        progressCallCount = 0
    }
}

/// Helper to get the path to test resources
/// Uses #file to locate the project root relative to this source file's location
enum TestResources {
    /// Get the project root directory by navigating up from this file's location
    /// This file is at: <project>/e2e/swift-pkg/Tests/ArgoTradingE2ETests/TestHelpers.swift
    /// Project root is 5 directories up
    static var projectRoot: String {
        let thisFile = #file
        var url = URL(fileURLWithPath: thisFile)

        // Navigate up: TestHelpers.swift -> ArgoTradingE2ETests -> Tests -> swift-pkg -> e2e -> project_root
        for _ in 0..<5 {
            url = url.deletingLastPathComponent()
        }

        return url.path
    }

    static var wasmStrategyPath: String {
        return projectRoot + "/e2e/backtest/wasm/sma/sma_plugin.wasm"
    }

    static var testDataPath: String {
        return projectRoot + "/internal/indicator/test_data/test_data.parquet"
    }

    /// Path to any WASM strategy in the e2e test directory
    static func wasmPath(for strategy: String) -> String {
        return projectRoot + "/e2e/backtest/wasm/\(strategy)"
    }

    /// Check if a resource exists
    static func exists(_ path: String) -> Bool {
        return FileManager.default.fileExists(atPath: path)
    }
}

/// Helper to create temporary directories for test outputs
class TempDirectory {
    let path: String

    init(prefix: String = "argo_test_") throws {
        let tempDir = FileManager.default.temporaryDirectory
        let uniqueName = prefix + UUID().uuidString
        let dirPath = tempDir.appendingPathComponent(uniqueName)
        try FileManager.default.createDirectory(at: dirPath, withIntermediateDirectories: true)
        self.path = dirPath.path
    }

    deinit {
        try? FileManager.default.removeItem(atPath: path)
    }
}

/// Mock implementation of TradingEngineHelper for testing live trading callbacks
class MockTradingEngineHelper: NSObject, SwiftargoTradingEngineHelperProtocol {
    // Track callback invocations
    var engineStartCalled = false
    var engineStopCalled = false
    var marketDataCalled = false
    var orderPlacedCalled = false
    var orderFilledCalled = false
    var errorCalled = false
    var strategyErrorCalled = false

    // Store callback parameters for verification
    var lastSymbols: [String] = []
    var lastInterval: String = ""
    var lastPreviousDataPath: String = ""
    var lastRunId: String = ""
    var lastError: Error?
    var marketDataCount: Int = 0
    var orderPlacedCount: Int = 0
    var orderFilledCount: Int = 0
    var lastOrderJSON: String = ""

    // Control callback behavior
    var shouldFailOnEngineStart = false
    var shouldFailOnMarketData = false
    var shouldFailOnOrderPlaced = false
    var shouldFailOnOrderFilled = false

    func onEngineStart(
        _ symbols: (any SwiftargoStringCollectionProtocol)?, interval: String?,
        previousDataPath: String?
    ) throws {
        engineStartCalled = true
        lastInterval = interval ?? ""
        lastPreviousDataPath = previousDataPath ?? ""
        if let syms = symbols {
            lastSymbols = []
            for i in 0..<syms.size() {
                lastSymbols.append(syms.get(i))
            }
        }

        if shouldFailOnEngineStart {
            throw NSError(
                domain: "TestError", code: 1,
                userInfo: [NSLocalizedDescriptionKey: "Test error in onEngineStart"])
        }
    }

    func onEngineStop(_ err: Error?) {
        engineStopCalled = true
        lastError = err
    }

    func onMarketData(
        _ runId: String?, symbol: String?, timestamp: Int64, open: Double, high: Double, low: Double, close: Double,
        volume: Double
    ) throws {
        lastRunId = runId ?? ""
        marketDataCalled = true
        marketDataCount += 1

        if shouldFailOnMarketData {
            throw NSError(
                domain: "TestError", code: 2,
                userInfo: [NSLocalizedDescriptionKey: "Test error in onMarketData"])
        }
    }

    func onOrderPlaced(_ orderJSON: String?) throws {
        orderPlacedCalled = true
        orderPlacedCount += 1
        lastOrderJSON = orderJSON ?? ""

        if shouldFailOnOrderPlaced {
            throw NSError(
                domain: "TestError", code: 3,
                userInfo: [NSLocalizedDescriptionKey: "Test error in onOrderPlaced"])
        }
    }

    func onOrderFilled(_ orderJSON: String?) throws {
        orderFilledCalled = true
        orderFilledCount += 1
        lastOrderJSON = orderJSON ?? ""

        if shouldFailOnOrderFilled {
            throw NSError(
                domain: "TestError", code: 4,
                userInfo: [NSLocalizedDescriptionKey: "Test error in onOrderFilled"])
        }
    }

    func onError(_ err: Error?) {
        errorCalled = true
        lastError = err
    }

    func onStrategyError(_ symbol: String?, timestamp: Int64, err: Error?) {
        strategyErrorCalled = true
        lastError = err
    }

    /// Reset all tracking state
    func reset() {
        engineStartCalled = false
        engineStopCalled = false
        marketDataCalled = false
        orderPlacedCalled = false
        orderFilledCalled = false
        errorCalled = false
        strategyErrorCalled = false
        lastSymbols = []
        lastInterval = ""
        lastPreviousDataPath = ""
        lastRunId = ""
        lastError = nil
        marketDataCount = 0
        orderPlacedCount = 0
        orderFilledCount = 0
        lastOrderJSON = ""
        shouldFailOnEngineStart = false
        shouldFailOnMarketData = false
        shouldFailOnOrderPlaced = false
        shouldFailOnOrderFilled = false
    }
}

/// Helper to validate JSON strings
func isValidJSON(_ string: String) -> Bool {
    guard let data = string.data(using: .utf8) else { return false }
    do {
        _ = try JSONSerialization.jsonObject(with: data, options: [])
        return true
    } catch {
        return false
    }
}
