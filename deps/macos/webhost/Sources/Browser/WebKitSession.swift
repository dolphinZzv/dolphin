import AppKit
import Foundation
import WebKit

class WebKitSession: NSObject, Sendable {
    let id: String
    let webView: WKWebView
    var interactive: Bool = false
    private var blockerView: BlockerView?
    private let eventBuffer = EventBuffer()
    private let lock = NSLock()
    private let hostWindow: NSWindow

    init(id: String, viewport: Viewport) {
        self.id = id
        let config = WKWebViewConfiguration()
        webView = WKWebView(frame: NSRect(x: 0, y: 0, width: viewport.width, height: viewport.height), configuration: config)
        hostWindow = NSWindow(contentRect: NSRect(x: 0, y: 0, width: viewport.width, height: viewport.height),
                              styleMask: [.titled, .closable, .miniaturizable, .resizable],
                              backing: .buffered,
                              defer: false)
        hostWindow.title = "Browser - \(id.prefix(8))"
        hostWindow.backgroundColor = .white
        hostWindow.setFrame(NSRect(x: 100, y: 100, width: viewport.width, height: viewport.height), display: true)
        hostWindow.contentView?.addSubview(webView)
        super.init()
        webView.uiDelegate = self
        webView.navigationDelegate = self
    }

    func showWindow() {
        hostWindow.makeKeyAndOrderFront(nil)
    }

    func evaluate(script: String, timeout: Int = 10000) async throws -> String {
        try await withCheckedThrowingContinuation { continuation in
            webView.evaluateJavaScript(script) { result, error in
                if let error = error {
                    continuation.resume(throwing: error)
                } else {
                    continuation.resume(returning: result as? String ?? "")
                }
            }
        }
    }

    func evaluateSync(script: String, timeout: Int = 10000) throws -> String {
        var result: String = ""
        var evalError: Error?
        let semaphore = DispatchSemaphore(value: 0)

        webView.evaluateJavaScript(script) { res, err in
            if let err = err {
                evalError = err
            } else {
                result = res as? String ?? ""
            }
            semaphore.signal()
        }

        let timeout_ns = Int64(timeout) * Int64(NSEC_PER_MSEC)
        let didTimeout = semaphore.wait(timeout: .now() + Double(timeout_ns) / Double(NSEC_PER_SEC)) == .timedOut

        if didTimeout {
            throw WebHostError.scriptTimeout
        }
        if let err = evalError {
            throw err
        }
        return result
    }

    @MainActor func screenshot() throws -> Data {
        let bounds = webView.bounds
        webView.layoutSubtreeIfNeeded()

        guard let bitmapRep = hostWindow.contentView?.bitmapImageRepForCachingDisplay(in: bounds) else {
            throw WebHostError.captureFailed
        }
        hostWindow.contentView?.cacheDisplay(in: bounds, to: bitmapRep)

        guard let pngData = bitmapRep.representation(using: .png, properties: [:]) else {
            throw WebHostError.pngConversionFailed
        }
        return pngData
    }

    func screenshotSync() throws -> Data {
        var screenshotData: Data?
        var screenshotError: Error?
        let semaphore = DispatchSemaphore(value: 0)

        DispatchQueue.main.async { [weak self] in
            guard let self = self else {
                semaphore.signal()
                return
            }
            do {
                let data = try self.screenshot()
                screenshotData = data
            } catch {
                screenshotError = error
            }
            semaphore.signal()
        }

        semaphore.wait()

        if let err = screenshotError {
            throw err
        }
        guard let data = screenshotData else {
            throw WebHostError.captureFailed
        }
        return data
    }

    func setInteractive(_ enabled: Bool) {
        interactive = enabled
        if !enabled {
            DispatchQueue.main.async { [weak self] in
                guard let self = self else { return }
                let blocker = BlockerView(frame: self.webView.bounds)
                blocker.autoresizingMask = [.width, .height]
                self.webView.addSubview(blocker)
                self.blockerView = blocker
            }

            webView.evaluateJavaScript("""
                document.addEventListener('mousedown', e => e.stopPropagation(), true);
                document.addEventListener('mouseup', e => e.stopPropagation(), true);
                document.addEventListener('click', e => e.stopPropagation(), true);
                document.addEventListener('keydown', e => e.stopPropagation(), true);
                document.addEventListener('keyup', e => e.stopPropagation(), true);
            """, completionHandler: nil)
        } else {
            DispatchQueue.main.async { [weak self] in
                self?.blockerView?.removeFromSuperview()
                self?.blockerView = nil
                self?.webView.window?.makeFirstResponder(self?.webView)
            }

            webView.evaluateJavaScript("""
                document.querySelectorAll('[data-dolphin-block]').forEach(el => el.remove());
            """, completionHandler: nil)
        }
    }

    func pushEvent(_ event: Event) {
        lock.lock()
        eventBuffer.append(event)
        lock.unlock()
    }

    func getEvents(since: Int64) -> [Event] {
        lock.lock()
        let events = eventBuffer.getEvents(since: since)
        lock.unlock()
        return events
    }

    func navigate(to url: URL) {
        let request = URLRequest(url: url)
        webView.load(request)
    }

    func getTitle() -> String {
        return webView.title ?? ""
    }

    func injectContent(css: String?, js: String?) {
        if let css = css, !css.isEmpty {
            let escapedCss = css.replacingOccurrences(of: "'", with: "\\'")
            let script = """
            (function() {
                var style = document.createElement('style');
                style.textContent = '\(escapedCss)';
                document.head.appendChild(style);
            })();
            """
            webView.evaluateJavaScript(script, completionHandler: nil)
        }
        if let js = js, !js.isEmpty {
            webView.evaluateJavaScript(js, completionHandler: nil)
        }
    }

    func waitForElement(selector: String, timeout: Int) throws -> Bool {
        let semaphore = DispatchSemaphore(value: 0)
        var result = false
        var scriptError: Error?

        let script = """
        (function() {
            var el = document.querySelector('\(selector)');
            return el ? el.innerHTML : null;
        })();
        """

        let timeoutDate = Date().addingTimeInterval(TimeInterval(timeout) / 1000.0)

        func checkElement() {
            if Date().compare(timeoutDate) == .orderedDescending {
                semaphore.signal()
                return
            }

            webView.evaluateJavaScript(script) { _, error in
                if let error = error {
                    scriptError = error
                    semaphore.signal()
                    return
                }

                result = true
                semaphore.signal()
            }
        }

        checkElement()
        semaphore.wait()

        if let error = scriptError {
            throw error
        }

        return result
    }

    func resolveDialog(dialogId: String, action: String, text: String?) {
    }
}

extension WebKitSession: WKUIDelegate {
    func webView(_ webView: WKWebView,
                 runJavaScriptConfirmPanelWithMessage message: String,
                 initiatedByFrame frame: WKFrameInfo,
                 completionHandler: @escaping (Bool) -> Void) {
        let event = WebEvent.dialog("confirm", message: message, dialogId: UUID().uuidString)
        pushEvent(event)
        completionHandler(false)
    }

    func webView(_ webView: WKWebView,
                 runJavaScriptTextInputPanelWithPrompt prompt: String,
                 defaultText: String?,
                 initiatedByFrame frame: WKFrameInfo,
                 completionHandler: @escaping (String?) -> Void) {
        let event = WebEvent.dialog("prompt", message: prompt, dialogId: UUID().uuidString)
        pushEvent(event)
        completionHandler(nil)
    }

    func webView(_ webView: WKWebView,
                 createWebViewWith configuration: WKWebViewConfiguration,
                 for navigationAction: WKNavigationAction,
                 windowFeatures: WKWindowFeatures) -> WKWebView? {
        let url = navigationAction.request.url?.absoluteString ?? ""
        let event = WebEvent.popup(url, popupId: UUID().uuidString)
        pushEvent(event)
        return nil
    }
}

extension WebKitSession: WKNavigationDelegate {
    func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
        let url = webView.url?.absoluteString ?? ""
        pushEvent(WebEvent.navigation(url, status: "complete"))
    }

    func webView(_ webView: WKWebView, didStartProvisionalNavigation navigation: WKNavigation!) {
        let url = webView.url?.absoluteString ?? ""
        pushEvent(WebEvent.navigation(url, status: "loading"))
    }
}

enum WebHostError: Error {
    case captureFailed
    case pngConversionFailed
    case scriptTimeout
}

class EventBuffer: Sendable {
    private var events: [Event] = []
    private let maxSize = 1000
    private let lock = NSLock()

    func append(_ event: Event) {
        lock.lock()
        if events.count >= maxSize {
            events.removeFirst()
        }
        events.append(event)
        lock.unlock()
    }

    func getEvents(since: Int64) -> [Event] {
        lock.lock()
        let result = events.filter { $0.t > since }
        lock.unlock()
        return result
    }
}