// swift-tools-version:5.9
import PackageDescription

let package = Package(
    name: "WebHost",
    platforms: [
        .macOS(.v13)
    ],
    dependencies: [
        .package(url: "https://github.com/SnapKit/SnapKit.git", from: "5.7.1")
    ],
    targets: [
        .executableTarget(
            name: "WebHost",
            dependencies: ["SnapKit"],
            path: "Sources"
        ),
        .testTarget(
            name: "WebHostTests",
            dependencies: ["WebHost"],
            path: "Tests"
        )
    ]
)