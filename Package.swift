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
                "https://github.com/rxtech-lab/argo-trading/releases/download/v1.17.0/ArgoTrading.xcframework.zip",
            checksum: "d221f968228e6a762c775cc72a5ce09bbaa3d80ea06baa2bcf39d5fd49d91c51"
        )
    ]
)
