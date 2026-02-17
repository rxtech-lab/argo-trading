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
            url: "https://github.com/rxtech-lab/argo-trading/releases/download/v1.13.0/ArgoTrading.xcframework.zip",
            checksum: "4cc9f44ae1f0a3926c62d6edd418b2fffeacaf1ada90139c9a66cc63d17d839f"
        )
    ]
)
