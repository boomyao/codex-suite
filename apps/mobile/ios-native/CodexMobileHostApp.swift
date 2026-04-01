import SwiftUI

@main
struct CodexMobileHostApp: App {
    @StateObject private var model = NativeHostModel()

    var body: some Scene {
        WindowGroup {
            ContentView(model: model)
                .preferredColorScheme(model.preferredColorScheme)
        }
    }
}
