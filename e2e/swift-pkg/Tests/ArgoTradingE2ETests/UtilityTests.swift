import XCTest
import ArgoTrading

/// Tests for utility functions exposed by ArgoTrading
final class UtilityTests: XCTestCase {

    // MARK: - GetBacktestEngineConfigSchema Tests

    func testGetBacktestEngineConfigSchema_ReturnsNonEmptyString() {
        let schema = SwiftargoGetBacktestEngineConfigSchema()

        XCTAssertFalse(schema.isEmpty, "Config schema should not be empty")
    }

    func testGetBacktestEngineConfigSchema_ReturnsValidJSON() {
        let schema = SwiftargoGetBacktestEngineConfigSchema()

        XCTAssertTrue(isValidJSON(schema), "Config schema should be valid JSON")
    }

    func testGetBacktestEngineConfigSchema_ContainsRequiredFields() {
        let schema = SwiftargoGetBacktestEngineConfigSchema()

        // The schema should contain expected field names
        XCTAssertTrue(schema.contains("initial_capital") || schema.contains("initialCapital"),
                      "Schema should contain initial_capital field")
        XCTAssertTrue(schema.contains("broker"),
                      "Schema should contain broker field")
    }

    // MARK: - StringCollection Tests

    func testStringArray_AddAndGet() {
        let array = SwiftargoStringArray()

        _ = array.add("item1")
        _ = array.add("item2")
        _ = array.add("item3")

        XCTAssertEqual(array.size(), 3, "Array should have 3 items")
        XCTAssertEqual(array.get(0), "item1", "First item should be 'item1'")
        XCTAssertEqual(array.get(1), "item2", "Second item should be 'item2'")
        XCTAssertEqual(array.get(2), "item3", "Third item should be 'item3'")
    }

    func testStringArray_GetOutOfBounds_ReturnsEmptyString() {
        let array = SwiftargoStringArray()

        _ = array.add("item1")

        // Out of bounds access should return empty string (not crash)
        XCTAssertEqual(array.get(-1), "", "Negative index should return empty string")
        XCTAssertEqual(array.get(1), "", "Index beyond size should return empty string")
        XCTAssertEqual(array.get(100), "", "Large index should return empty string")
    }

    func testStringArray_EmptyArray() {
        let array = SwiftargoStringArray()

        XCTAssertEqual(array.size(), 0, "New array should have size 0")
        XCTAssertEqual(array.get(0), "", "Getting from empty array should return empty string")
    }

    func testStringArray_ChainedAdd() {
        // Test that Add returns self for chaining
        let array = SwiftargoStringArray()

        let result = array.add("a")?.add("b")?.add("c")

        XCTAssertNotNil(result, "Chained add should return non-nil")
        XCTAssertEqual(array.size(), 3, "Chained array should have 3 items")
    }
}
