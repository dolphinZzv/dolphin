import Foundation

struct Viewport: Codable, Sendable {
    var width: Int
    var height: Int

    init(width: Int = 1920, height: Int = 1080) {
        self.width = width
        self.height = height
    }
}
