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
            url: "https://github.com/rxtech-lab/argo-trading/releases/download/v1.7.0/ArgoTrading.xcframework.zip",
            checksum: "84c31b8da131ae8d4c2720e7ed6948b08a6893aa10fd830a678a204ceec18d5c"
        )
    ]
)
