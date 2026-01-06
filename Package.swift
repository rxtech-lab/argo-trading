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
            url: "https://github.com/rxtech-lab/argo-trading/releases/download/v1.10.0/ArgoTrading.xcframework.zip",
            checksum: "76de30656d4caf2f41a87bac4223d3c3477fbca61c92f398e4e3262712e16b27"
        )
    ]
)
