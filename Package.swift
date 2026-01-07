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
            url: "https://github.com/rxtech-lab/argo-trading/releases/download/v1.11.0/ArgoTrading.xcframework.zip",
            checksum: "fdf864ba9207e5c23d59b9ac10e4b6b6e88f73d26568bab5097e1233d95520ee"
        )
    ]
)
