import SwiftUI

@MainActor
final class NativeHostViewModel: ObservableObject {
    @Published var profile: BridgeProfile?
    @Published var enrollmentDraft = ""
    @Published var statusMessage = "Ready to enroll"

    private let store = BridgeProfileStore()

    init() {
        profile = store.read()
    }

    var remoteShellURL: URL? {
        guard let profile else { return nil }
        return BridgeAPI.buildRemoteShellURL(profile.serverEndpoint)
    }

    func importEnrollment() {
        let rawJSON = enrollmentDraft.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !rawJSON.isEmpty else {
            statusMessage = "Paste a bridge enrollment payload first."
            return
        }

        do {
            let payload = try EnrollmentParser.parse(rawJSON: rawJSON)
            let nextProfile: BridgeProfile
            switch payload {
            case let .bridge(name, serverEndpoint, _, _):
                nextProfile = BridgeProfile(
                    name: name,
                    serverEndpoint: BridgeAPI.normalizeEndpoint(serverEndpoint),
                    authToken: nil
                )
                statusMessage = "Bridge enrollment imported."
            case let .tailnet(bridgeName, bridgeServerEndpoint, _, _):
                nextProfile = BridgeProfile(
                    name: bridgeName,
                    serverEndpoint: BridgeAPI.normalizeEndpoint(bridgeServerEndpoint),
                    authToken: nil
                )
                statusMessage = "Tailnet enrollment imported. Network Extension wiring is still pending."
            }
            store.write(nextProfile)
            profile = nextProfile
            enrollmentDraft = ""
        } catch {
            statusMessage = error.localizedDescription
        }
    }

    func reset() {
        store.clear()
        profile = nil
        statusMessage = "Enrollment cleared."
    }
}

struct ContentView: View {
    @ObservedObject var model: NativeHostViewModel
    @State private var isShowingEnrollmentSheet = false

    var body: some View {
        NavigationStack {
            Group {
                if let url = model.remoteShellURL, let profile = model.profile {
                    VStack(spacing: 0) {
                        CodexWebView(url: url, authToken: profile.authToken)
                        Text(model.statusMessage)
                            .font(.footnote)
                            .foregroundStyle(.secondary)
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .padding(.horizontal, 16)
                            .padding(.vertical, 12)
                            .background(Color.secondary.opacity(0.12))
                    }
                } else {
                    VStack(alignment: .leading, spacing: 12) {
                        Text("No bridge enrolled.")
                            .font(.title3.weight(.semibold))
                        Text("Import a bridge enrollment payload instead of typing endpoints by hand. This native iOS host will eventually pair that flow with a Network Extension backed tailnet runtime.")
                            .foregroundStyle(.secondary)
                        Text(model.statusMessage)
                            .font(.footnote)
                            .foregroundStyle(.secondary)
                    }
                    .padding(24)
                }
            }
            .navigationTitle("Codex Mobile")
            .toolbar {
                ToolbarItem(placement: .primaryAction) {
                    Button("Enroll") {
                        isShowingEnrollmentSheet = true
                    }
                }
                ToolbarItem(placement: .automatic) {
                    if model.profile != nil {
                        Button("Reset", role: .destructive) {
                            model.reset()
                        }
                    }
                }
            }
            .sheet(isPresented: $isShowingEnrollmentSheet) {
                NavigationStack {
                    VStack(alignment: .leading, spacing: 16) {
                        Text("Paste a `codex-mobile-enrollment` or `codex-mobile-bridge` payload from the bridge host.")
                            .foregroundStyle(.secondary)
                        TextEditor(text: $model.enrollmentDraft)
                            .frame(minHeight: 240)
                            .padding(8)
                            .overlay {
                                RoundedRectangle(cornerRadius: 12)
                                    .stroke(Color.secondary.opacity(0.25), lineWidth: 1)
                            }
                        Text(model.statusMessage)
                            .font(.footnote)
                            .foregroundStyle(.secondary)
                        Spacer()
                    }
                    .padding(20)
                    .navigationTitle("Enrollment")
                    .toolbar {
                        ToolbarItem(placement: .cancellationAction) {
                            Button("Cancel") {
                                isShowingEnrollmentSheet = false
                            }
                        }
                        ToolbarItem(placement: .confirmationAction) {
                            Button("Import") {
                                model.importEnrollment()
                                if model.profile != nil {
                                    isShowingEnrollmentSheet = false
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
