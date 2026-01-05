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
            url: "https://github.com/rxtech-lab/argo-trading/releases/download/v1.8.0/ArgoTrading.xcframework.zip",
            checksum: "e44fcf79fddcb5e6885e3987490f1f0aded4237b7318f2759e5870570bb56fd4"
        )
    ]
)
