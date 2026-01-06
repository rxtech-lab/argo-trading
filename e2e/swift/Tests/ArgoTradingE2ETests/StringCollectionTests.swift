import XCTest
import ArgoTrading

final class StringCollectionTests: XCTestCase {

    /// Test creating a StringArray and adding/getting elements
    func testStringArrayOperations() throws {
        let array = SwiftargoNewStringArray()
        XCTAssertNotNil(array)

        // Initially empty
        XCTAssertEqual(array!.size(), 0)

        // Add elements
        array!.add("first")
        XCTAssertEqual(array!.size(), 1)
        XCTAssertEqual(array!.get(0), "first")

        array!.add("second")
        XCTAssertEqual(array!.size(), 2)
        XCTAssertEqual(array!.get(0), "first")
        XCTAssertEqual(array!.get(1), "second")

        array!.add("third")
        XCTAssertEqual(array!.size(), 3)
        XCTAssertEqual(array!.get(2), "third")
    }

    /// Test that getting out of bounds returns empty string
    func testStringArrayOutOfBounds() throws {
        let array = SwiftargoNewStringArray()!

        // Getting from empty array should return empty string
        XCTAssertEqual(array.get(0), "")
        XCTAssertEqual(array.get(10), "")

        array.add("element")

        // Getting beyond bounds should return empty string
        XCTAssertEqual(array.get(1), "")
        XCTAssertEqual(array.get(-1), "")
    }
}
