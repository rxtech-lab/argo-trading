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
            url: "https://github.com/rxtech-lab/argo-trading/releases/download/v1.14.0/ArgoTrading.xcframework.zip",
            checksum: "4d6670cbf37e1b3a6a5005080d0cf201bdf6de8cb2706d2a6911a96c7701279b"
        )
    ]
)
