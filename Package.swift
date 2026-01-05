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
            url: "https://github.com/rxtech-lab/argo-trading/releases/download/v1.7.1/ArgoTrading.xcframework.zip",
            checksum: "05aecd4967e0f92b6e5e69de97961cab7535cb5adad2f81a266494e5b777c46f"
        )
    ]
)
