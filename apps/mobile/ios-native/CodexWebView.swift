import SwiftUI
import Foundation

#if canImport(UIKit) && canImport(WebKit)
import UIKit
import WebKit

struct CodexWebView: UIViewRepresentable {
    let url: URL
    let authToken: String?

    func makeUIView(context: Context) -> WKWebView {
        let view = WKWebView(frame: .zero)
        view.navigationDelegate = context.coordinator
        return view
    }

    func updateUIView(_ webView: WKWebView, context: Context) {
        var request = URLRequest(url: url)
        if let authToken, !authToken.isEmpty {
            request.setValue("Bearer \(authToken)", forHTTPHeaderField: "Authorization")
        }
        webView.load(request)
    }

    func makeCoordinator() -> Coordinator {
        Coordinator()
    }

    final class Coordinator: NSObject, WKNavigationDelegate {}
}
#else
struct CodexWebView: View {
    let url: URL
    let authToken: String?

    var body: some View {
        Text("WKWebView host is only available on iOS.")
    }
}
#endif
