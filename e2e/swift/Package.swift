// swift-tools-version:5.9
import PackageDescription

let package = Package(
    name: "ArgoTradingE2ETests",
    platforms: [.macOS(.v12)],
    products: [],
    dependencies: [],
    targets: [
        .binaryTarget(
            name: "ArgoTrading",
            path: "../../pkg/swift-argo/ArgoTrading.xcframework"
        ),
        .testTarget(
            name: "ArgoTradingE2ETests",
            dependencies: ["ArgoTrading"],
            resources: [
                .copy("Resources")
            ]
        )
    ]
)
