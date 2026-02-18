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
            url: "https://github.com/rxtech-lab/argo-trading/releases/download/v1.14.2/ArgoTrading.xcframework.zip",
            checksum: "a15d2113d7fa6d9767fb1844afc0a49f42cf98f0063338a0b8f3b135761dc7b9"
        )
    ]
)
