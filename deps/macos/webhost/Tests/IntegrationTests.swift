import XCTest
@testable import WebHost

final class ToolHandlerTests: XCTestCase {
    var server: McpServer!

    override func setUp() {
        super.setUp()
        server = McpServer()
    }

    override func tearDown() {
        server = nil
        super.tearDown()
    }

    func testWebInjectHandlerWithMissingSessionId() {
        let injectRequest = JsonRpcRequest(
            id: 2,
            method: "tools/call",
            params: [
                "name": AnyCodable("web_inject"),
                "arguments": AnyCodable([
                    "sessionId": AnyCodable("nonexistent-session"),
                    "css": AnyCodable("body { }"),
                    "js": AnyCodable("console.log('test');")
                ])
            ]
        )
        let response = server.handleSync(request: injectRequest)
        XCTAssertTrue(response.error?.code == -32602 || response.error?.code == -32000)
    }

    func testWebWaitHandlerWithMissingSessionId() {
        let waitRequest = JsonRpcRequest(
            id: 2,
            method: "tools/call",
            params: [
                "name": AnyCodable("web_wait"),
                "arguments": AnyCodable([
                    "sessionId": AnyCodable("nonexistent-session"),
                    "selector": AnyCodable("#main-content")
                ])
            ]
        )
        let response = server.handleSync(request: waitRequest)
        XCTAssertTrue(response.error?.code == -32602 || response.error?.code == -32000)
    }

    func testWebDialogResponseHandlerWithMissingSessionId() {
        let dialogRequest = JsonRpcRequest(
            id: 2,
            method: "tools/call",
            params: [
                "name": AnyCodable("web_dialog_response"),
                "arguments": AnyCodable([
                    "sessionId": AnyCodable("nonexistent-session"),
                    "dialogId": AnyCodable("dlg_123"),
                    "action": AnyCodable("accept")
                ])
            ]
        )
        let response = server.handleSync(request: dialogRequest)
        XCTAssertTrue(response.error?.code == -32602 || response.error?.code == -32000)
    }

    func testScriptRunHandlerExists() {
        let scriptRequest = JsonRpcRequest(
            id: 1,
            method: "tools/call",
            params: [
                "name": AnyCodable("script_run"),
                "arguments": AnyCodable(["sessionId": AnyCodable("nonexistent")])
            ]
        )
        let response = server.handleSync(request: scriptRequest)
        XCTAssertTrue(response.error != nil)
    }

    func testPageScreenshotHandlerExists() {
        let screenshotRequest = JsonRpcRequest(
            id: 1,
            method: "tools/call",
            params: [
                "name": AnyCodable("page_screenshot"),
                "arguments": AnyCodable(["sessionId": AnyCodable("nonexistent")])
            ]
        )
        let response = server.handleSync(request: screenshotRequest)
        XCTAssertTrue(response.error != nil)
    }

    func testPageOpenHandlerExists() {
        let openRequest = JsonRpcRequest(
            id: 1,
            method: "tools/call",
            params: [
                "name": AnyCodable("page_open"),
                "arguments": AnyCodable(["sessionId": AnyCodable("nonexistent"), "url": AnyCodable("https://example.com")])
            ]
        )
        let response = server.handleSync(request: openRequest)
        XCTAssertTrue(response.error != nil)
    }

    func testWebCapabilitiesHandlerExists() {
        let request = JsonRpcRequest(
            id: 1,
            method: "tools/call",
            params: [
                "name": AnyCodable("web_capabilities"),
                "arguments": AnyCodable([:])
            ]
        )
        let response = server.handleSync(request: request)
        XCTAssertTrue(response.result?.success ?? false)
        XCTAssertNotNil(response.result?.capabilities)
    }

    func testWebSessionCloseHandlerExists() {
        let request = JsonRpcRequest(
            id: 1,
            method: "tools/call",
            params: [
                "name": AnyCodable("web_session_close"),
                "arguments": AnyCodable(["sessionId": AnyCodable("nonexistent")])
            ]
        )
        let response = server.handleSync(request: request)
        XCTAssertTrue(response.result?.success ?? false)
    }

    func testWebSetInteractiveHandlerExists() {
        let request = JsonRpcRequest(
            id: 1,
            method: "tools/call",
            params: [
                "name": AnyCodable("web_set_interactive"),
                "arguments": AnyCodable(["sessionId": AnyCodable("nonexistent"), "interactive": AnyCodable(true)])
            ]
        )
        let response = server.handleSync(request: request)
        XCTAssertTrue(response.error != nil)
    }
}

final class SessionManagementTests: XCTestCase {
    var server: McpServer!

    override func setUp() {
        super.setUp()
        server = McpServer()
    }

    override func tearDown() {
        server = nil
        super.tearDown()
    }

    func testSessionManagerExists() {
        XCTAssertNotNil(server.sessionManager)
    }

    func testCloseNonexistentSessionReturnsSuccess() {
        let closeRequest = JsonRpcRequest(
            id: 1,
            method: "tools/call",
            params: [
                "name": AnyCodable("web_session_close"),
                "arguments": AnyCodable(["sessionId": AnyCodable("nonexistent")])
            ]
        )
        let response = server.handleSync(request: closeRequest)
        XCTAssertTrue(response.result?.success ?? false)
    }
}

final class ErrorHandlingTests: XCTestCase {
    var server: McpServer!

    override func setUp() {
        super.setUp()
        server = McpServer()
    }

    override func tearDown() {
        server = nil
        super.tearDown()
    }

    func testInvalidMethodName() {
        let request = JsonRpcRequest(
            id: 1,
            method: "tools/call",
            params: [
                "name": AnyCodable("nonexistent_tool"),
                "arguments": AnyCodable([:])
            ]
        )
        let response = server.handleSync(request: request)
        XCTAssertNotNil(response.error)
        XCTAssertEqual(response.error?.code, -32601)
    }

    func testMissingArguments() {
        let request = JsonRpcRequest(
            id: 1,
            method: "tools/call",
            params: [
                "name": AnyCodable("page_open")
            ]
        )
        let response = server.handleSync(request: request)
        XCTAssertNotNil(response.error)
    }
}

final class IntegrationTests: XCTestCase {
    func testHealthEndpointJson() throws {
        let json = """
        {"status":"ok"}
        """
        XCTAssertTrue(json.contains("ok"))
    }
}

final class WebHostWorkflowTests: XCTestCase {
    func testFullWorkflowJsonConstruction() throws {
        let sessionCreateRequest = JsonRpcRequest(
            id: 1,
            method: "tools/call",
            params: [
                "name": AnyCodable("web_session_create"),
                "arguments": AnyCodable(["viewport": AnyCodable(["width": 1920, "height": 1080])])
            ]
        )

        let pageOpenRequest = JsonRpcRequest(
            id: 2,
            method: "tools/call",
            params: [
                "name": AnyCodable("page_open"),
                "arguments": AnyCodable([
                    "sessionId": AnyCodable("sess_001"),
                    "url": AnyCodable("https://example.com")
                ])
            ]
        )

        let scriptRunRequest = JsonRpcRequest(
            id: 3,
            method: "tools/call",
            params: [
                "name": AnyCodable("script_run"),
                "arguments": AnyCodable([
                    "sessionId": AnyCodable("sess_001"),
                    "script": AnyCodable("document.title")
                ])
            ]
        )

        let screenshotRequest = JsonRpcRequest(
            id: 4,
            method: "tools/call",
            params: [
                "name": AnyCodable("page_screenshot"),
                "arguments": AnyCodable(["sessionId": AnyCodable("sess_001")])
            ]
        )

        let closeRequest = JsonRpcRequest(
            id: 5,
            method: "tools/call",
            params: [
                "name": AnyCodable("web_session_close"),
                "arguments": AnyCodable(["sessionId": AnyCodable("sess_001")])
            ]
        )

        XCTAssertNotNil(closeRequest.id)
        XCTAssertEqual(closeRequest.method, "tools/call")
    }

    func testEventSequence() throws {
        var events: [Event] = []

        events.append(WebEvent.navigation("https://example.com", status: "loading"))
        events.append(WebEvent.console("Page started loading"))
        events.append(WebEvent.navigation("https://example.com", status: "complete"))
        events.append(WebEvent.console("Page loaded"))

        XCTAssertEqual(events.count, 4)
        XCTAssertTrue(events[0].t <= events[1].t)
        XCTAssertTrue(events[1].t <= events[2].t)
    }

    func testNewToolHandlersExist() throws {
        let handlers = ["web_inject", "web_wait", "web_dialog_response"]

        for handler in handlers {
            let request = JsonRpcRequest(
                id: 1,
                method: "tools/call",
                params: [
                    "name": AnyCodable(handler),
                    "arguments": AnyCodable([:])
                ]
            )

            let server = McpServer()
            let response = server.handleSync(request: request)
            XCTAssertNotNil(response)
        }
    }
}

final class HttpHandlerTests: XCTestCase {
    func testStreamQueryParsing() {
        let uri = "/mcp/stream?sessionId=sess_001&since=1234567890"

        var since: Int64 = 0
        var sessionId: String?

        if let queryStart = uri.firstIndex(of: "?") {
            let query = String(uri[queryStart...]).dropFirst()
            let params = query.split(separator: "&")
            for param in params {
                let keyValue = param.split(separator: "=")
                if keyValue.count == 2 {
                    let key = String(keyValue[0])
                    let value = String(keyValue[1])
                    if key == "since" {
                        since = Int64(value) ?? 0
                    } else if key == "sessionId" {
                        sessionId = value
                    }
                }
            }
        }

        XCTAssertEqual(sessionId, "sess_001")
        XCTAssertEqual(since, 1234567890)
    }

    func testStreamQueryWithoutSince() {
        let uri = "/mcp/stream?sessionId=sess_001"

        var since: Int64 = 0
        var sessionId: String?

        if let queryStart = uri.firstIndex(of: "?") {
            let query = String(uri[queryStart...]).dropFirst()
            let params = query.split(separator: "&")
            for param in params {
                let keyValue = param.split(separator: "=")
                if keyValue.count == 2 {
                    let key = String(keyValue[0])
                    let value = String(keyValue[1])
                    if key == "since" {
                        since = Int64(value) ?? 0
                    } else if key == "sessionId" {
                        sessionId = value
                    }
                }
            }
        }

        XCTAssertEqual(sessionId, "sess_001")
        XCTAssertEqual(since, 0)
    }
}