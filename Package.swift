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
                "https://github.com/rxtech-lab/argo-trading/releases/download/v1.19.0/ArgoTrading.xcframework.zip",
            checksum: "9fa5700ac61ed11fa62b845560f44427e9b6572aff2b719ca4272d2f083d5215"
        )
    ]
)
