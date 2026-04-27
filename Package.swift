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
            url:
                "https://github.com/rxtech-lab/argo-trading/releases/download/v1.22.0/ArgoTrading.xcframework.zip",
            checksum: "1b69dd976b5e83a64a78c628f272a3912e94c9979f0c2c759105b43e6c2e678a"
        )
    ]
)
