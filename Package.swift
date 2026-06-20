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
                "https://github.com/rxtech-lab/argo-trading/releases/download/v1.23.0/ArgoTrading.xcframework.zip",
            checksum: "a096444f554575bb9e3bc9c053b7e6fef530a266e57e767fe1d0b4ed1cf80f77"
        )
    ]
)
