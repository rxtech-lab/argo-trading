import XCTest
import ArgoTrading

final class StrategyMetadataTests: XCTestCase {

    /// Test that we can create a StrategyApi instance
    func testCreateStrategyApi() throws {
        let api = SwiftargoNewStrategyApi()
        XCTAssertNotNil(api)
    }

    /// Test loading strategy metadata from a WASM file
    func testGetStrategyMetadata() throws {
        // Skip if resources are not available
        guard let resourcesPath = Bundle.module.resourcePath else {
            throw XCTSkip("Resources not available")
        }

        let wasmPath = resourcesPath + "/Resources/sma_plugin.wasm"

        // Check if WASM file exists
        guard FileManager.default.fileExists(atPath: wasmPath) else {
            throw XCTSkip("WASM file not found at \(wasmPath)")
        }

        let api = SwiftargoNewStrategyApi()!

        // GetStrategyMetadata returns (*StrategyMetadata, error) so it throws in Swift
        let metadata = try api.getStrategyMetadata(wasmPath)

        XCTAssertNotNil(metadata)

        // Verify metadata fields
        XCTAssertEqual(metadata!.name, "SimpleMAStrategy")
        XCTAssertEqual(metadata!.identifier, "com.argo-trading.e2e.simple-ma")
        XCTAssertFalse(metadata!.description_.isEmpty)
        XCTAssertFalse(metadata!.runtimeVersion.isEmpty)

        // Verify schema is valid JSON
        let schemaData = metadata!.schema.data(using: .utf8)!
        let schemaJson = try JSONSerialization.jsonObject(with: schemaData)
        XCTAssertNotNil(schemaJson)

        // Verify schema contains expected properties
        if let schemaDict = schemaJson as? [String: Any],
           let properties = schemaDict["properties"] as? [String: Any] {
            XCTAssertNotNil(properties["fastPeriod"])
            XCTAssertNotNil(properties["slowPeriod"])
            XCTAssertNotNil(properties["symbol"])
        }
    }

    /// Test that loading metadata from invalid path returns an error
    func testGetStrategyMetadataInvalidPath() throws {
        let api = SwiftargoNewStrategyApi()!

        // GetStrategyMetadata should throw for invalid path
        XCTAssertThrowsError(try api.getStrategyMetadata("/nonexistent/path/strategy.wasm"))
    }
}
