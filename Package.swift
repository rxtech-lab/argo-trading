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
            url: "https://github.com/rxtech-lab/argo-trading/releases/download/v1.4.1/ArgoTrading.xcframework.zip",
            checksum: "0949a6eaffaddccf5aec930d8c81b69cd77a11bcd34db245a0c1accda83a430b"
        )
    ]
)
