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
            url: "https://github.com/rxtech-lab/argo-trading/releases/download/v1.7.3/ArgoTrading.xcframework.zip",
            checksum: "26e9bb38433b70e7c85fb9f001150a27b0890c83dcf2c196ec8e420c895b948d"
        )
    ]
)
