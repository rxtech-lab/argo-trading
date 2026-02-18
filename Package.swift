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
            url: "https://github.com/rxtech-lab/argo-trading/releases/download/v1.14.1/ArgoTrading.xcframework.zip",
            checksum: "b575ac064e958762d89e379d50ab700ea1c7562f6afc5e15628c4af6a626cdb1"
        )
    ]
)
