import XCTest
import ArgoTrading

/// Tests for market-related APIs (excluding actual download operations)
final class MarketApiTests: XCTestCase {

    // MARK: - GetSupportedDownloadClients Tests

    func testGetSupportedDownloadClients_ReturnsNonEmptyCollection() {
        let clients = SwiftargoGetSupportedDownloadClients()

        XCTAssertNotNil(clients, "Supported clients should not be nil")
        XCTAssertGreaterThan(clients!.size(), 0, "Should have at least one supported client")
    }

    func testGetSupportedDownloadClients_ContainsPolygon() {
        let clients = SwiftargoGetSupportedDownloadClients()!

        var hasPolygon = false
        for i in 0..<clients.size() {
            if clients.get(i) == "polygon" {
                hasPolygon = true
                break
            }
        }

        XCTAssertTrue(hasPolygon, "Supported clients should include 'polygon'")
    }

    func testGetSupportedDownloadClients_ContainsBinance() {
        let clients = SwiftargoGetSupportedDownloadClients()!

        var hasBinance = false
        for i in 0..<clients.size() {
            if clients.get(i) == "binance" {
                hasBinance = true
                break
            }
        }

        XCTAssertTrue(hasBinance, "Supported clients should include 'binance'")
    }

    func testGetSupportedDownloadClients_ReturnsConsistentResults() {
        let clients1 = SwiftargoGetSupportedDownloadClients()!
        let clients2 = SwiftargoGetSupportedDownloadClients()!

        XCTAssertEqual(clients1.size(), clients2.size(),
                       "Should return same number of clients on multiple calls")

        // Compare all items
        for i in 0..<clients1.size() {
            XCTAssertEqual(clients1.get(i), clients2.get(i),
                           "Client at index \(i) should be consistent")
        }
    }

    // MARK: - GetDownloadClientSchema Tests

    func testGetDownloadClientSchema_PolygonReturnsValidJSON() {
        let schema = SwiftargoGetDownloadClientSchema("polygon")

        XCTAssertFalse(schema.isEmpty, "Polygon schema should not be empty")
        XCTAssertTrue(isValidJSON(schema), "Polygon schema should be valid JSON")
    }

    func testGetDownloadClientSchema_PolygonContainsRequiredFields() {
        let schema = SwiftargoGetDownloadClientSchema("polygon")

        // Polygon config should have these fields
        XCTAssertTrue(schema.contains("ticker") || schema.contains("Ticker"),
                      "Polygon schema should contain ticker field")
        XCTAssertTrue(schema.contains("api_key") || schema.contains("apiKey") || schema.contains("ApiKey"),
                      "Polygon schema should contain api_key field")
        XCTAssertTrue(schema.contains("start_date") || schema.contains("startDate") || schema.contains("StartDate"),
                      "Polygon schema should contain start_date field")
        XCTAssertTrue(schema.contains("end_date") || schema.contains("endDate") || schema.contains("EndDate"),
                      "Polygon schema should contain end_date field")
    }

    func testGetDownloadClientSchema_BinanceReturnsValidJSON() {
        let schema = SwiftargoGetDownloadClientSchema("binance")

        XCTAssertFalse(schema.isEmpty, "Binance schema should not be empty")
        XCTAssertTrue(isValidJSON(schema), "Binance schema should be valid JSON")
    }

    func testGetDownloadClientSchema_BinanceContainsRequiredFields() {
        let schema = SwiftargoGetDownloadClientSchema("binance")

        // Binance config should have these fields
        XCTAssertTrue(schema.contains("ticker") || schema.contains("Ticker"),
                      "Binance schema should contain ticker field")
        XCTAssertTrue(schema.contains("start_date") || schema.contains("startDate") || schema.contains("StartDate"),
                      "Binance schema should contain start_date field")
        XCTAssertTrue(schema.contains("end_date") || schema.contains("endDate") || schema.contains("EndDate"),
                      "Binance schema should contain end_date field")
    }

    func testGetDownloadClientSchema_InvalidProviderReturnsEmptyString() {
        let schema = SwiftargoGetDownloadClientSchema("invalid_provider")

        XCTAssertEqual(schema, "", "Invalid provider should return empty string")
    }

    func testGetDownloadClientSchema_EmptyProviderReturnsEmptyString() {
        let schema = SwiftargoGetDownloadClientSchema("")

        XCTAssertEqual(schema, "", "Empty provider should return empty string")
    }

    func testGetDownloadClientSchema_CaseSensitive() {
        let schemaLower = SwiftargoGetDownloadClientSchema("polygon")
        let schemaUpper = SwiftargoGetDownloadClientSchema("POLYGON")
        let schemaMixed = SwiftargoGetDownloadClientSchema("Polygon")

        // Provider names should be lowercase
        XCTAssertFalse(schemaLower.isEmpty, "Lowercase 'polygon' should work")
        XCTAssertTrue(schemaUpper.isEmpty, "Uppercase 'POLYGON' should not work")
        XCTAssertTrue(schemaMixed.isEmpty, "Mixed case 'Polygon' should not work")
    }

    // MARK: - NewMarketDownloader Tests

    func testNewMarketDownloader_CreatesValidInstance() {
        let helper = MockMarketDownloaderHelper()
        let downloader = SwiftargoNewMarketDownloader(helper)

        XCTAssertNotNil(downloader, "Should create a valid market downloader instance")
    }

    func testNewMarketDownloader_CancelWithNoDownloadReturnsFalse() {
        let helper = MockMarketDownloaderHelper()
        let downloader = SwiftargoNewMarketDownloader(helper)!

        // Cancel should return false when no download is in progress
        let cancelled = downloader.cancel()

        XCTAssertFalse(cancelled, "Cancel should return false when no download is in progress")
    }

    func testNewMarketDownloader_MultipleCancelCallsAreSafe() {
        let helper = MockMarketDownloaderHelper()
        let downloader = SwiftargoNewMarketDownloader(helper)!

        // Multiple cancel calls should not crash
        _ = downloader.cancel()
        _ = downloader.cancel()
        _ = downloader.cancel()

        // If we get here without crashing, the test passes
        XCTAssertTrue(true, "Multiple cancel calls should be safe")
    }
}
