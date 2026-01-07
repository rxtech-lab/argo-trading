// swift-tools-version:5.9
import PackageDescription

let package = Package(
    name: "ArgoTradingE2ETests",
    platforms: [
        .macOS(.v12)
    ],
    products: [
        .library(
            name: "ArgoTradingE2ETests",
            targets: ["ArgoTradingE2ETests"]
        )
    ],
    dependencies: [
        .package(url: "https://github.com/duckdb/duckdb-swift.git", from: "1.0.0")
    ],
    targets: [
        .binaryTarget(
            name: "ArgoTrading",
            path: "../../pkg/swift-argo/ArgoTrading.xcframework"
        ),
        .target(
            name: "ArgoTradingE2ETests",
            dependencies: [
                "ArgoTrading",
                .product(name: "DuckDB", package: "duckdb-swift")
            ],
            path: "Sources/ArgoTradingE2ETests"
        ),
        .testTarget(
            name: "ArgoTradingE2ETestsTests",
            dependencies: [
                "ArgoTrading",
                "ArgoTradingE2ETests",
                .product(name: "DuckDB", package: "duckdb-swift")
            ],
            path: "Tests/ArgoTradingE2ETests"
        )
    ]
)
