// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "swift-worker",
    platforms: [
        .macOS(.v10_15)
    ],
    dependencies: [
        .package(url: "https://github.com/swift-server/RediStack.git", from: "1.6.0")
    ],
    targets: [
        .executableTarget(
            name: "swift-worker",
            dependencies: [
                .product(name: "RediStack", package: "RediStack")
            ],
            path: "Sources"
        )
    ]
)
