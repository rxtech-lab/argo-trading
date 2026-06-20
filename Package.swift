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
                "https://github.com/rxtech-lab/argo-trading/releases/download/v1.24.0/ArgoTrading.xcframework.zip",
            checksum: "3eab025fa58ceaa2662d7a0edf1ee4f9477f62f1cec34a21cf3b1bf2816ccbf6"
        )
    ]
)
