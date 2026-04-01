import Foundation
import SwiftUI
import WebKit

@MainActor
protocol NativeHostWebBridgeDelegate: AnyObject {
    func webBridgeDidReceive(message: String)
    func webBridgeDidStartNavigation(url: URL?)
    func webBridgeDidFinishNavigation(url: URL?)
    func webBridgeDidFailNavigation(message: String)
    func webBridgeCurrentBaseURL() -> URL?
    func webBridgeOpenExternalURL(_ url: URL)
}

@MainActor
final class NativeHostWebBridge: NSObject {
    weak var delegate: NativeHostWebBridgeDelegate?

    private weak var webView: WKWebView?
    private var pendingLoad: (url: URL, headers: [String: String])?

    func makeWebView() -> WKWebView {
        if let webView {
            return webView
        }

        let controller = WKUserContentController()
        controller.add(self, name: "CodexMobileNativeBridge")
        controller.addUserScript(
            WKUserScript(
                source: Self.bootstrapScript,
                injectionTime: .atDocumentStart,
                forMainFrameOnly: false
            )
        )

        let configuration = WKWebViewConfiguration()
        configuration.userContentController = controller
        configuration.defaultWebpagePreferences.preferredContentMode = .mobile
        configuration.websiteDataStore = .default()

        let webView = WKWebView(frame: .zero, configuration: configuration)
        webView.navigationDelegate = self
        webView.uiDelegate = self
        webView.scrollView.contentInsetAdjustmentBehavior = .never
        webView.allowsBackForwardNavigationGestures = false
        webView.isOpaque = false
        webView.backgroundColor = .clear
        webView.scrollView.backgroundColor = .clear
        if #available(iOS 16.4, *) {
            webView.isInspectable = true
        }
        self.webView = webView

        if let pendingLoad {
            load(url: pendingLoad.url, headers: pendingLoad.headers)
        }

        return webView
    }

    func attach(webView: WKWebView) {
        self.webView = webView
    }

    func load(url: URL, headers: [String: String]) {
        pendingLoad = (url, headers)
        guard let webView else {
            return
        }
        var request = URLRequest(url: url)
        headers.forEach { request.setValue($1, forHTTPHeaderField: $0) }
        webView.load(request)
    }

    func loadBlank() {
        pendingLoad = nil
        webView?.loadHTMLString("", baseURL: nil)
    }

    func evaluateJavaScript(_ script: String) {
        webView?.evaluateJavaScript(script, completionHandler: nil)
    }

    func configureCookies(baseURL: URL, authToken: String?, usesLocalProxy: Bool) async {
        guard let host = baseURL.host else {
            return
        }
        let cookieStore = makeWebView().configuration.websiteDataStore.httpCookieStore
        let normalizedBaseURL = URL(string: BridgeAPI.normalizeEndpoint(baseURL.absoluteString)) ?? baseURL

        if let mobileHostCookie = HTTPCookie(properties: [
            .domain: host,
            .path: "/",
            .name: "codex_mobile_host",
            .value: "ios-native",
            .secure: normalizedBaseURL.scheme == "https",
        ]) {
            await setCookie(mobileHostCookie, in: cookieStore)
        }

        if !usesLocalProxy, let authToken, !authToken.isEmpty {
            if let authCookie = HTTPCookie(properties: [
                .domain: host,
                .path: "/",
                .name: "codex_bridge_token",
                .value: authToken,
                .secure: normalizedBaseURL.scheme == "https",
            ]) {
                await setCookie(authCookie, in: cookieStore)
            }
        } else if let clearedCookie = HTTPCookie(properties: [
            .domain: host,
            .path: "/",
            .name: "codex_bridge_token",
            .value: "",
            .expires: Date(timeIntervalSince1970: 0),
            .secure: normalizedBaseURL.scheme == "https",
        ]) {
            await setCookie(clearedCookie, in: cookieStore)
        }
    }

    private func setCookie(_ cookie: HTTPCookie, in store: WKHTTPCookieStore) async {
        await withCheckedContinuation { continuation in
            store.setCookie(cookie) {
                continuation.resume()
            }
        }
    }

    private static let bootstrapScript =
        """
        (function () {
          window.CodexMobileNativeBridge = window.CodexMobileNativeBridge || {};
          window.CodexMobileNativeBridge.postMessage = function (message) {
            try {
              var payload = typeof message === "string" ? message : JSON.stringify(message);
              window.webkit.messageHandlers.CodexMobileNativeBridge.postMessage(payload);
            } catch (error) {}
          };
        })();
        """
}

extension NativeHostWebBridge: WKScriptMessageHandler {
    func userContentController(_ userContentController: WKUserContentController, didReceive message: WKScriptMessage) {
        guard message.name == "CodexMobileNativeBridge" else {
            return
        }
        let text: String
        if let stringBody = message.body as? String {
            text = stringBody
        } else {
            text = jsonEncodedString(["value": message.body])
        }
        delegate?.webBridgeDidReceive(message: text)
    }
}

extension NativeHostWebBridge: WKNavigationDelegate, WKUIDelegate {
    func webView(_ webView: WKWebView, didStartProvisionalNavigation navigation: WKNavigation!) {
        delegate?.webBridgeDidStartNavigation(url: webView.url)
    }

    func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
        delegate?.webBridgeDidFinishNavigation(url: webView.url)
    }

    func webView(_ webView: WKWebView, didFail navigation: WKNavigation!, withError error: Error) {
        delegate?.webBridgeDidFailNavigation(message: (error as NSError).localizedDescription)
    }

    func webView(_ webView: WKWebView, didFailProvisionalNavigation navigation: WKNavigation!, withError error: Error) {
        delegate?.webBridgeDidFailNavigation(message: (error as NSError).localizedDescription)
    }

    func webView(
        _ webView: WKWebView,
        decidePolicyFor navigationAction: WKNavigationAction,
        decisionHandler: @escaping (WKNavigationActionPolicy) -> Void
    ) {
        Task { @MainActor [weak self] in
            guard let self else {
                decisionHandler(.cancel)
                return
            }
            if navigationAction.navigationType == .linkActivated,
               let targetURL = navigationAction.request.url,
               let currentBaseURL = delegate?.webBridgeCurrentBaseURL(),
               targetURL.host != currentBaseURL.host {
                delegate?.webBridgeOpenExternalURL(targetURL)
                decisionHandler(.cancel)
                return
            }
            decisionHandler(.allow)
        }
    }
}

struct CodexWebView: UIViewRepresentable {
    let bridge: NativeHostWebBridge

    func makeUIView(context: Context) -> WKWebView {
        bridge.makeWebView()
    }

    func updateUIView(_ uiView: WKWebView, context: Context) {
        bridge.attach(webView: uiView)
    }
}
