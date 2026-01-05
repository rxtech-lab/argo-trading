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
            url: "https://github.com/rxtech-lab/argo-trading/releases/download/v1.9.0/ArgoTrading.xcframework.zip",
            checksum: "97ccdd675b52e93a122903c3fce8c92248bfa0bcbad28568a06d1a31a9139121"
        )
    ]
)
