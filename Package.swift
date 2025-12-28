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
            url: "https://github.com/rxtech-lab/argo-trading/releases/download/v1.4.0/ArgoTrading.xcframework.zip",
            checksum: "a32f543294a26e1dbecb82d2c14271c6aab982b0e8b7f5c200ee5098a407087e"
        )
    ]
)
