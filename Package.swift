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
            url: "https://github.com/rxtech-lab/argo-trading/releases/download/v1.7.2/ArgoTrading.xcframework.zip",
            checksum: "df51d830d67624c23511be03df7a2c0d56fdd230d73ebe5264caffbb81360aba"
        )
    ]
)
