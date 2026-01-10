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
            url: "https://github.com/rxtech-lab/argo-trading/releases/download/v1.12.0/ArgoTrading.xcframework.zip",
            checksum: "e458913d46d19a07e4589cfddfcc2803c1d6f27a4b4796e5805f238138e3d06e"
        )
    ]
)
