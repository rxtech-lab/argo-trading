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
                "https://github.com/rxtech-lab/argo-trading/releases/download/v1.21.0/ArgoTrading.xcframework.zip",
            checksum: "2a96b0387a51301878a0ed27a929fdec9110ff4a0373924d6bb1f2801d248d06"
        )
    ]
)
