import SwiftUI

@main
struct CodexMobileHostApp: App {
    @StateObject private var model = NativeHostViewModel()

    var body: some Scene {
        WindowGroup {
            ContentView(model: model)
        }
    }
}
