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
                "https://github.com/rxtech-lab/argo-trading/releases/download/v1.18.0/ArgoTrading.xcframework.zip",
            checksum: "72f2d7958411b2fa938528bd14d8908b74c630aecdaac5502b348f47fb73eced"
        )
    ]
)
