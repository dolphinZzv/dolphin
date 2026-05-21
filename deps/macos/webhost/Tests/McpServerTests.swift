import XCTest
@testable import WebHost

final class McpServerTests: XCTestCase {
    var server: McpServer!

    override func setUp() {
        super.setUp()
        server = McpServer()
    }

    override func tearDown() {
        server = nil
        super.tearDown()
    }

    func testCreateSession() {
        let request = JsonRpcRequest(
            id: 1,
            method: "tools/call",
            params: [
                "name": AnyCodable("web_session_create"),
                "arguments": AnyCodable(["viewport": AnyCodable(["width": 1920, "height": 1080])])
            ]
        )

        let response = server.handleSync(request: request)

        XCTAssertTrue(response.result?.success ?? false)
        XCTAssertNotNil(response.result?.sessionId)
    }

    func testGetCapabilities() {
        let request = JsonRpcRequest(
            id: 2,
            method: "tools/call",
            params: [
                "name": AnyCodable("web_capabilities"),
                "arguments": AnyCodable(["sessionId": AnyCodable("any-session")])
            ]
        )

        let response = server.handleSync(request: request)

        XCTAssertTrue(response.result?.success ?? false)
        XCTAssertNotNil(response.result?.capabilities)
        XCTAssertEqual(response.result?.capabilities?["screenshot"]?.value as? Bool, true)
        XCTAssertEqual(response.result?.capabilities?["dialog"]?.value as? Bool, true)
    }

    func testSessionNotFound() {
        let request = JsonRpcRequest(
            id: 3,
            method: "tools/call",
            params: [
                "name": AnyCodable("page_open"),
                "arguments": AnyCodable([
                    "sessionId": AnyCodable("nonexistent-session"),
                    "url": AnyCodable("https://example.com")
                ])
            ]
        )

        let response = server.handleSync(request: request)

        XCTAssertNotNil(response.error)
        XCTAssertEqual(response.error?.code, -32000)
    }

    func testInvalidParams() {
        let request = JsonRpcRequest(
            id: 4,
            method: "tools/call",
            params: [
                "name": AnyCodable("page_open"),
                "arguments": AnyCodable([:])
            ]
        )

        let response = server.handleSync(request: request)

        XCTAssertNotNil(response.error)
    }

    func testUnknownTool() {
        let request = JsonRpcRequest(
            id: 5,
            method: "tools/call",
            params: [
                "name": AnyCodable("unknown_tool"),
                "arguments": AnyCodable([:])
            ]
        )

        let response = server.handleSync(request: request)

        XCTAssertNotNil(response.error)
        XCTAssertEqual(response.error?.code, -32601)
    }
}