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
                "https://github.com/rxtech-lab/argo-trading/releases/download/v1.16.0/ArgoTrading.xcframework.zip",
            checksum: "ec249eb93044a671810306da56db349098cc9b836a2701cceccb492f2d28d847"
        )
    ]
)
