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
                "https://github.com/rxtech-lab/argo-trading/releases/download/v1.15.1/ArgoTrading.xcframework.zip",
            checksum: "6815eebda5adbfc7c514cd3cd415277a2677777f9129ea3b8b11db14879e9de5"
        )
    ]
)
