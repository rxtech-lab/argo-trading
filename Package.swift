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
            url: "https://github.com/rxtech-lab/argo-trading/releases/download/v1.5.0/ArgoTrading.xcframework.zip",
            checksum: "b91475a6afd9dd46dfe5e02f1acb617a7c4f015cb63f6ca708704025c3054252"
        )
    ]
)
