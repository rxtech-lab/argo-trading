import XCTest
import ArgoTrading

/// Tests for Strategy API - loading WASM strategies and reading metadata
final class StrategyApiTests: XCTestCase {

    var strategyApi: SwiftargoStrategyApi!

    override func setUp() {
        super.setUp()
        strategyApi = SwiftargoNewStrategyApi()
    }

    override func tearDown() {
        strategyApi = nil
        super.tearDown()
    }

    // MARK: - NewStrategyApi Tests

    func testNewStrategyApi_CreatesValidInstance() {
        let api = SwiftargoNewStrategyApi()

        XCTAssertNotNil(api, "Should create a valid StrategyApi instance")
    }

    func testNewStrategyApi_MultipleInstancesAreIndependent() {
        let api1 = SwiftargoNewStrategyApi()
        let api2 = SwiftargoNewStrategyApi()

        XCTAssertNotNil(api1, "First instance should be valid")
        XCTAssertNotNil(api2, "Second instance should be valid")
        // Both should work independently
    }

    // MARK: - GetStrategyMetadata Tests

    func testGetStrategyMetadata_WithValidWasm_ReturnsMetadata() throws {
        // Get path to test WASM file
        let wasmPath = TestResources.wasmStrategyPath

        // Skip test if resource not available
        guard FileManager.default.fileExists(atPath: wasmPath) else {
            throw XCTSkip("Test WASM file not found at \(wasmPath)")
        }

        let metadata = try strategyApi.getStrategyMetadata(wasmPath)

        XCTAssertNotNil(metadata, "Should return metadata for valid WASM")
    }

    func testGetStrategyMetadata_WithValidWasm_HasNonEmptyName() throws {
        let wasmPath = TestResources.wasmStrategyPath

        guard FileManager.default.fileExists(atPath: wasmPath) else {
            throw XCTSkip("Test WASM file not found at \(wasmPath)")
        }

        let metadata = try strategyApi.getStrategyMetadata(wasmPath)

        XCTAssertFalse(metadata.name.isEmpty, "Strategy name should not be empty")
    }

    func testGetStrategyMetadata_WithValidWasm_HasNonEmptyIdentifier() throws {
        let wasmPath = TestResources.wasmStrategyPath

        guard FileManager.default.fileExists(atPath: wasmPath) else {
            throw XCTSkip("Test WASM file not found at \(wasmPath)")
        }

        let metadata = try strategyApi.getStrategyMetadata(wasmPath)

        XCTAssertFalse(metadata.identifier.isEmpty, "Strategy identifier should not be empty")
    }

    func testGetStrategyMetadata_WithValidWasm_HasValidSchema() throws {
        let wasmPath = TestResources.wasmStrategyPath

        guard FileManager.default.fileExists(atPath: wasmPath) else {
            throw XCTSkip("Test WASM file not found at \(wasmPath)")
        }

        let metadata = try strategyApi.getStrategyMetadata(wasmPath)

        if !metadata.schema.isEmpty {
            XCTAssertTrue(isValidJSON(metadata.schema), "Strategy schema should be valid JSON")
        }
    }

    func testGetStrategyMetadata_WithInvalidPath_ThrowsError() {
        let invalidPath = "/nonexistent/path/to/strategy.wasm"

        XCTAssertThrowsError(try strategyApi.getStrategyMetadata(invalidPath)) { error in
            XCTAssertNotNil(error, "Should throw an error for invalid path")
        }
    }

    func testGetStrategyMetadata_WithEmptyPath_ThrowsError() {
        XCTAssertThrowsError(try strategyApi.getStrategyMetadata("")) { error in
            XCTAssertNotNil(error, "Should throw an error for empty path")
        }
    }

    func testGetStrategyMetadata_WithNonWasmFile_ThrowsError() throws {
        // Create a temporary non-WASM file
        let tempDir = FileManager.default.temporaryDirectory
        let fakePath = tempDir.appendingPathComponent("fake_strategy.wasm")

        try "not a wasm file".write(to: fakePath, atomically: true, encoding: .utf8)
        defer {
            try? FileManager.default.removeItem(at: fakePath)
        }

        XCTAssertThrowsError(try strategyApi.getStrategyMetadata(fakePath.path)) { error in
            XCTAssertNotNil(error, "Should throw an error for non-WASM file")
        }
    }
}
