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
            url: "https://github.com/rxtech-lab/argo-trading/releases/download/v1.4.2/ArgoTrading.xcframework.zip",
            checksum: "0af44afafd6379ecbeaa9bdedb7257d1243979836e3b757465109755dd119980"
        )
    ]
)
