import XCTest
import ArgoTrading

/// Mock implementation of MarketDownloaderHelper for testing download callbacks
class MockMarketDownloaderHelper: NSObject, SwiftargoMarketDownloaderHelperProtocol {
    var progressCalls: [(current: Double, total: Double, message: String)] = []

    func onDownloadProgress(_ current: Double, total: Double, message: String?) {
        progressCalls.append((current: current, total: total, message: message ?? ""))
    }
}

final class MarketDownloaderTests: XCTestCase {

    /// Test that we can get the list of supported download clients
    func testGetSupportedDownloadClients() throws {
        let clients = SwiftargoGetSupportedDownloadClients()

        XCTAssertNotNil(clients)
        XCTAssertGreaterThan(clients!.size(), 0)

        // Verify expected providers are available
        var hasPolygon = false
        var hasBinance = false

        for i in 0..<clients!.size() {
            let client = clients!.get(i)
            if client == "polygon" {
                hasPolygon = true
            }
            if client == "binance" {
                hasBinance = true
            }
        }

        XCTAssertTrue(hasPolygon, "Polygon provider should be available")
        XCTAssertTrue(hasBinance, "Binance provider should be available")
    }

    /// Test that we can get the Polygon download client schema
    func testGetPolygonDownloadClientSchema() throws {
        var error: NSError?
        let schema = SwiftargoGetDownloadClientSchema("polygon", &error)

        XCTAssertNil(error)
        XCTAssertNotNil(schema)
        XCTAssertFalse(schema!.isEmpty)

        // Verify it's valid JSON
        let data = schema!.data(using: .utf8)!
        let json = try JSONSerialization.jsonObject(with: data)
        XCTAssertNotNil(json)
    }

    /// Test that we can get the Binance download client schema
    func testGetBinanceDownloadClientSchema() throws {
        var error: NSError?
        let schema = SwiftargoGetDownloadClientSchema("binance", &error)

        XCTAssertNil(error)
        XCTAssertNotNil(schema)
        XCTAssertFalse(schema!.isEmpty)

        // Verify it's valid JSON
        let data = schema!.data(using: .utf8)!
        let json = try JSONSerialization.jsonObject(with: data)
        XCTAssertNotNil(json)
    }

    /// Test that getting schema for unknown provider returns error
    func testGetUnknownDownloadClientSchema() throws {
        var error: NSError?
        let schema = SwiftargoGetDownloadClientSchema("unknown_provider", &error)

        XCTAssertNotNil(error)
        XCTAssertNil(schema)
    }

    /// Test that we can create a MarketDownloader instance
    func testCreateMarketDownloader() throws {
        let helper = MockMarketDownloaderHelper()
        let downloader = SwiftargoNewMarketDownloader(helper)

        XCTAssertNotNil(downloader)
    }

    /// Test that cancellation when no download is running returns false
    func testMarketDownloaderCancellation() throws {
        let helper = MockMarketDownloaderHelper()
        let downloader = SwiftargoNewMarketDownloader(helper)!

        // Cancel when no download is running should return false
        let cancelled = downloader.cancel()
        XCTAssertFalse(cancelled)
    }
}
