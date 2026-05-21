import Foundation
import WebKit

class McpServer {
    var sessions: [String: WebKitSession] = [:]
    let sessionManager: SessionManager = SessionManager()
    let lock = NSLock()

    func handleSync(request: JsonRpcRequest) -> JsonRpcResponse {
        guard request.method == "tools/call",
              let params = request.params,
              let toolName = params.name else {
            return JsonRpcResponse(id: request.id, error: .invalidParams)
        }

        let arguments = params.arguments

        switch toolName {
        case "web_session_create":
            return createSession(arguments: arguments, requestId: request.id)
        case "page_open":
            return navigate(arguments: arguments, requestId: request.id)
        case "script_run":
            return evaluate(arguments: arguments, requestId: request.id)
        case "page_screenshot":
            return screenshot(arguments: arguments, requestId: request.id)
        case "web_set_interactive":
            return setInteractive(arguments: arguments, requestId: request.id)
        case "web_capabilities":
            return getCapabilities(arguments: arguments, requestId: request.id)
        case "web_session_close":
            return closeSession(arguments: arguments, requestId: request.id)
        case "web_inject":
            return injectContent(arguments: arguments, requestId: request.id)
        case "web_wait":
            return waitForElement(arguments: arguments, requestId: request.id)
        case "web_dialog_response":
            return dialogResponse(arguments: arguments, requestId: request.id)
        default:
            return JsonRpcResponse(id: request.id, error: .methodNotFound)
        }
    }

    private func createSession(arguments: [String: AnyCodable]?, requestId: Any?) -> JsonRpcResponse {
        lock.lock()
        defer { lock.unlock() }

        let viewport = parseViewport(arguments)
        let sessionId = UUID().uuidString
        let session = WebKitSession(id: sessionId, viewport: viewport)

        sessions[sessionId] = session
        sessionManager.add(sessionId: sessionId)

        DispatchQueue.main.async {
            session.showWindow()
        }

        return JsonRpcResponse(id: requestId, result: JsonRpcResult(success: true, sessionId: sessionId))
    }

    private func navigate(arguments: [String: AnyCodable]?, requestId: Any?) -> JsonRpcResponse {
        guard let sessionId = arguments?["sessionId"]?.value as? String else {
            return JsonRpcResponse(id: requestId, error: .invalidParams)
        }

        guard let urlString = arguments?["url"]?.value as? String,
              let url = URL(string: urlString) else {
            return JsonRpcResponse(id: requestId, error: .invalidParams)
        }

        lock.lock()
        let session = sessions[sessionId]
        lock.unlock()

        guard let session = session else {
            return JsonRpcResponse(id: requestId, error: .sessionNotFound)
        }

        DispatchQueue.main.async {
            session.navigate(to: url)
        }

        let title = session.getTitle()
        return JsonRpcResponse(id: requestId, result: JsonRpcResult(
            success: true,
            url: urlString,
            title: title,
            status: "loading"
        ))
    }

    private func evaluate(arguments: [String: AnyCodable]?, requestId: Any?) -> JsonRpcResponse {
        guard let sessionId = arguments?["sessionId"]?.value as? String else {
            return JsonRpcResponse(id: requestId, error: .invalidParams)
        }

        guard let script = arguments?["script"]?.value as? String else {
            return JsonRpcResponse(id: requestId, error: .invalidParams)
        }

        lock.lock()
        let session = sessions[sessionId]
        lock.unlock()

        guard let session = session else {
            return JsonRpcResponse(id: requestId, error: .sessionNotFound)
        }

        let timeoutMs = (arguments?["timeout"]?.value as? Int) ?? 10000

        do {
            let result = try session.evaluateSync(script: script, timeout: timeoutMs)
            return JsonRpcResponse(id: requestId, result: JsonRpcResult(success: true, value: result))
        } catch {
            return JsonRpcResponse(id: requestId, error: .internalError)
        }
    }

    private func screenshot(arguments: [String: AnyCodable]?, requestId: Any?) -> JsonRpcResponse {
        guard let sessionId = arguments?["sessionId"]?.value as? String else {
            return JsonRpcResponse(id: requestId, error: .invalidParams)
        }

        lock.lock()
        let session = sessions[sessionId]
        lock.unlock()

        guard let session = session else {
            return JsonRpcResponse(id: requestId, error: .sessionNotFound)
        }

        do {
            let data = try session.screenshotSync()
            let base64 = data.base64EncodedString()
            return JsonRpcResponse(id: requestId, result: JsonRpcResult(success: true, data: base64))
        } catch {
            return JsonRpcResponse(id: requestId, error: .internalError)
        }
    }

    private func setInteractive(arguments: [String: AnyCodable]?, requestId: Any?) -> JsonRpcResponse {
        guard let sessionId = arguments?["sessionId"]?.value as? String else {
            return JsonRpcResponse(id: requestId, error: .invalidParams)
        }

        let interactive = arguments?["interactive"]?.value as? Bool ?? false

        lock.lock()
        let session = sessions[sessionId]
        lock.unlock()

        guard let session = session else {
            return JsonRpcResponse(id: requestId, error: .sessionNotFound)
        }

        session.setInteractive(interactive)

        return JsonRpcResponse(id: requestId, result: JsonRpcResult(success: true))
    }

    private func getCapabilities(arguments: [String: AnyCodable]?, requestId: Any?) -> JsonRpcResponse {
        let capabilities: [String: AnyCodable] = [
            "dialog": AnyCodable(true),
            "popup": AnyCodable(true),
            "screenshot": AnyCodable(true),
            "fullPage": AnyCodable(true),
            "console": AnyCodable(true),
            "navigation": AnyCodable(true),
            "upload": AnyCodable(true),
            "download": AnyCodable(false)
        ]

        return JsonRpcResponse(id: requestId, result: JsonRpcResult(success: true, capabilities: capabilities))
    }

    private func closeSession(arguments: [String: AnyCodable]?, requestId: Any?) -> JsonRpcResponse {
        guard let sessionId = arguments?["sessionId"]?.value as? String else {
            return JsonRpcResponse(id: requestId, error: .invalidParams)
        }

        lock.lock()
        sessions.removeValue(forKey: sessionId)
        lock.unlock()

        sessionManager.remove(sessionId: sessionId)

        return JsonRpcResponse(id: requestId, result: JsonRpcResult(success: true))
    }

    private func injectContent(arguments: [String: AnyCodable]?, requestId: Any?) -> JsonRpcResponse {
        guard let sessionId = arguments?["sessionId"]?.value as? String else {
            return JsonRpcResponse(id: requestId, error: .invalidParams)
        }

        lock.lock()
        let session = sessions[sessionId]
        lock.unlock()

        guard let session = session else {
            return JsonRpcResponse(id: requestId, error: .sessionNotFound)
        }

        let css = arguments?["css"]?.value as? String
        let js = arguments?["js"]?.value as? String

        session.injectContent(css: css, js: js)

        return JsonRpcResponse(id: requestId, result: JsonRpcResult(success: true))
    }

    private func waitForElement(arguments: [String: AnyCodable]?, requestId: Any?) -> JsonRpcResponse {
        guard let sessionId = arguments?["sessionId"]?.value as? String else {
            return JsonRpcResponse(id: requestId, error: .invalidParams)
        }

        guard let selector = arguments?["selector"]?.value as? String else {
            return JsonRpcResponse(id: requestId, error: .invalidParams)
        }

        lock.lock()
        let session = sessions[sessionId]
        lock.unlock()

        guard let session = session else {
            return JsonRpcResponse(id: requestId, error: .sessionNotFound)
        }

        let timeout = (arguments?["timeout"]?.value as? Int) ?? 30000

        do {
            let found = try session.waitForElement(selector: selector, timeout: timeout)
            return JsonRpcResponse(id: requestId, result: JsonRpcResult(success: true, found: found))
        } catch {
            return JsonRpcResponse(id: requestId, error: .navigationTimeout)
        }
    }

    private func dialogResponse(arguments: [String: AnyCodable]?, requestId: Any?) -> JsonRpcResponse {
        guard let sessionId = arguments?["sessionId"]?.value as? String else {
            return JsonRpcResponse(id: requestId, error: .invalidParams)
        }

        lock.lock()
        let session = sessions[sessionId]
        lock.unlock()

        guard let session = session else {
            return JsonRpcResponse(id: requestId, error: .sessionNotFound)
        }

        let dialogId = arguments?["dialogId"]?.value as? String
        let action = arguments?["action"]?.value as? String
        let text = arguments?["text"]?.value as? String

        session.resolveDialog(dialogId: dialogId ?? "", action: action ?? "dismiss", text: text)

        return JsonRpcResponse(id: requestId, result: JsonRpcResult(success: true))
    }

    private func parseViewport(_ arguments: [String: AnyCodable]?) -> Viewport {
        guard let args = arguments,
              let vp = args["viewport"]?.value as? [String: Any],
              let width = vp["width"] as? Int,
              let height = vp["height"] as? Int else {
            return Viewport()
        }
        return Viewport(width: width, height: height)
    }
}

class SessionManager: Sendable {
    private var activeSessions: Set<String> = []
    private let lock = NSLock()
    let maxCount = 10

    func add(sessionId: String) {
        lock.lock()
        activeSessions.insert(sessionId)
        lock.unlock()
    }

    func remove(sessionId: String) {
        lock.lock()
        activeSessions.remove(sessionId)
        lock.unlock()
    }

    func count() -> Int {
        lock.lock()
        let c = activeSessions.count
        lock.unlock()
        return c
    }
}