// swift-tools-version:5.9
import PackageDescription

let package = Package(
    name: "ArgoTrading",
    platforms: [
        .macOS(.v12)
    ],
    products: [
        .library(
            name: "ArgoTrading",
            targets: ["ArgoTrading"]
        )
    ],
    targets: [
        .binaryTarget(
            name: "ArgoTrading",
            url: "https://github.com/rxtech-lab/argo-trading/releases/download/v1.4.3/ArgoTrading.xcframework.zip",
            checksum: "7eca276739be7d13289e65b3a17bee1d3a9bb5cbbfff78162301ecf274e30a40"
        )
    ]
)
