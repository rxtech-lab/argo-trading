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
            url: "https://github.com/rxtech-lab/argo-trading/releases/download/v1.6.0/ArgoTrading.xcframework.zip",
            checksum: "68301ef6a01932ff6fb99e90c60ad0bf1d1696577f61bab6af610cebe69e2dec"
        )
    ]
)
