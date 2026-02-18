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
                "https://github.com/rxtech-lab/argo-trading/releases/download/v1.14.2/ArgoTrading.xcframework.zip",
            checksum: "ad906f178a89da937135c1819e5515fa28d716d293ddb428139556c7c2ea4ad3"
        )
    ]
)
