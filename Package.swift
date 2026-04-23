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
                "https://github.com/rxtech-lab/argo-trading/releases/download/v1.20.0/ArgoTrading.xcframework.zip",
            checksum: "4b07efd5f96dab3f9994521f12cc3da7ddfc543454dcc5f669efec2d67c5a090"
        )
    ]
)
