import SwiftUI

struct ContentView: View {
    @Environment(\.colorScheme) private var colorScheme
    @ObservedObject var model: NativeHostModel

    var body: some View {
        ZStack {
            LinearGradient(
                colors: [
                    Color(red: 0.97, green: 0.95, blue: 0.90),
                    Color(red: 0.90, green: 0.93, blue: 0.95),
                    Color(red: 0.84, green: 0.89, blue: 0.94),
                ],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )
            .ignoresSafeArea()

            VStack(spacing: 16) {
                headerBar

                if let activeProfile = model.activeProfile {
                    sessionBar(profile: activeProfile)
                }

                Group {
                    if model.shouldKeepWebViewMounted {
                        workspaceShell
                    } else if model.isError, let activeProfile = model.activeProfile {
                        ScrollView(showsIndicators: false) {
                            VStack(spacing: 16) {
                                errorCard(profile: activeProfile)
                                recentSessionsSection
                            }
                            .padding(.bottom, 20)
                        }
                    } else {
                        ScrollView(showsIndicators: false) {
                            VStack(spacing: 16) {
                                if !model.hasSavedProfiles {
                                    welcomeHero
                                }
                                recentSessionsSection
                            }
                            .padding(.bottom, 20)
                        }
                    }
                }
            }
            .padding(.horizontal, 16)
            .padding(.top, 12)
            .padding(.bottom, 8)
        }
        .onAppear {
            model.handleColorSchemeChanged(colorScheme)
        }
        .onChange(of: colorScheme) { _, newValue in
            model.handleColorSchemeChanged(newValue)
        }
        .sheet(isPresented: $model.sessionsSheetPresented) {
            SessionsSheet(model: model)
                .presentationDetents([.medium, .large])
        }
        .sheet(isPresented: $model.settingsSheetPresented) {
            SettingsSheet(model: model)
                .presentationDetents([.medium])
        }
        .fullScreenCover(isPresented: $model.scannerPresented) {
            NativeHostScannerView(
                onCodeScanned: { payload in
                    model.handleScannedPayload(payload)
                },
                onCancel: {}
            )
        }
        .alert(item: $model.profilePendingReset) { profile in
            Alert(
                title: Text("Reset Enrollment"),
                message: Text("Forget \(profile.name) on this device?"),
                primaryButton: .destructive(Text("Reset")) {
                    model.resetEnrollment(profile)
                },
                secondaryButton: .cancel {
                    model.profilePendingReset = nil
                }
            )
        }
    }

    private var headerBar: some View {
        HStack(alignment: .top, spacing: 12) {
            VStack(alignment: .leading, spacing: 4) {
                Text("Codex Mobile")
                    .font(.system(.largeTitle, design: .rounded, weight: .bold))
                    .foregroundStyle(Color(red: 0.13, green: 0.19, blue: 0.27))
                Text(model.statusMessage)
                    .font(.subheadline.weight(.medium))
                    .foregroundStyle(Color(red: 0.29, green: 0.37, blue: 0.47))
                    .lineLimit(2)
            }
            Spacer()
            HStack(spacing: 10) {
                CircleIconButton(systemImage: "gearshape.fill") {
                    model.presentSettingsSheet()
                }
                CircleIconButton(systemImage: "square.grid.2x2.fill") {
                    model.presentSessionsSheet()
                }
            }
        }
    }

    private func sessionBar(profile: BridgeProfile) -> some View {
        HStack(spacing: 14) {
            ZStack {
                RoundedRectangle(cornerRadius: 16, style: .continuous)
                    .fill(Color(red: 0.13, green: 0.18, blue: 0.25))
                Text(profileAvatar(profile))
                    .font(.title3.weight(.bold))
                    .foregroundStyle(.white)
            }
            .frame(width: 56, height: 56)

            VStack(alignment: .leading, spacing: 4) {
                Text(displayProfileLabel(profile))
                    .font(.headline.weight(.semibold))
                    .foregroundStyle(.primary)
                Text(displayProfileDetail(profile))
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
            }

            Spacer()

            VStack(alignment: .trailing, spacing: 8) {
                StatusPill(
                    label: model.isConnected ? "Live" : (model.isLoading ? "Connecting" : "Needs Attention"),
                    tone: model.isConnected ? .success : (model.isLoading ? .active : .warning)
                )
                Button("Sessions") {
                    model.presentSessionsSheet()
                }
                .font(.subheadline.weight(.semibold))
            }
        }
        .padding(16)
        .background(
            RoundedRectangle(cornerRadius: 28, style: .continuous)
                .fill(.ultraThinMaterial)
                .overlay(
                    RoundedRectangle(cornerRadius: 28, style: .continuous)
                        .stroke(Color.white.opacity(0.6), lineWidth: 1)
                )
        )
    }

    private var workspaceShell: some View {
        ZStack {
            RoundedRectangle(cornerRadius: 34, style: .continuous)
                .fill(Color.white.opacity(0.56))
                .overlay(
                    RoundedRectangle(cornerRadius: 34, style: .continuous)
                        .stroke(Color.white.opacity(0.74), lineWidth: 1)
                )

            CodexWebView(bridge: model.webBridge)
                .clipShape(RoundedRectangle(cornerRadius: 30, style: .continuous))
                .padding(4)
                .opacity(model.isConnected ? 1 : 0.01)

            if model.isLoading {
                loadingOverlay
            }
        }
    }

    private var loadingOverlay: some View {
        VStack(spacing: 22) {
            Spacer()
            VStack(alignment: .leading, spacing: 16) {
                Text("Preparing your Codex session")
                    .font(.title2.weight(.bold))
                    .foregroundStyle(.primary)
                Text(model.loadingSummary)
                    .font(.body)
                    .foregroundStyle(.secondary)

                ProgressView(value: model.connectionProgressValue)
                    .tint(Color(red: 0.18, green: 0.51, blue: 0.73))

                FlowLabels(labels: model.connectionStageLabels, activeLabel: model.currentConnectionStage?.title)
            }
            .padding(22)
            .frame(maxWidth: 420)
            .background(
                RoundedRectangle(cornerRadius: 28, style: .continuous)
                    .fill(.regularMaterial)
            )
            .padding(20)
            Spacer()
        }
    }

    private var welcomeHero: some View {
        VStack(alignment: .leading, spacing: 20) {
            HStack(alignment: .top) {
                VStack(alignment: .leading, spacing: 8) {
                    Text("Codex Mobile")
                        .font(.subheadline.weight(.bold))
                        .foregroundStyle(Color(red: 0.17, green: 0.43, blue: 0.67))
                    Text("Bring your desktop workspace to this device")
                        .font(.system(.title, design: .rounded, weight: .bold))
                        .foregroundStyle(Color(red: 0.14, green: 0.18, blue: 0.25))
                    Text("Scan a desktop QR code to open a Codex session here and pick up the workspace from this device.")
                        .font(.body)
                        .foregroundStyle(Color(red: 0.31, green: 0.39, blue: 0.49))
                }
                Spacer()
                StatusPill(
                    label: model.hasSavedProfiles ? "\(model.profiles.count) Saved" : "Ready",
                    tone: .active
                )
            }

            ActionButton(title: "Scan Desktop QR", systemImage: "viewfinder") {
                model.openScanner()
            }

            HStack(spacing: 14) {
                SceneNode(title: "Desktop Bridge", systemImage: "desktopcomputer")
                SceneConnector()
                SceneNode(title: "This Device", systemImage: "ipad.landscape")
            }

            Text("Pair once, then jump back into trusted sessions from this device.")
                .font(.subheadline)
                .foregroundStyle(Color(red: 0.31, green: 0.39, blue: 0.49))

            VStack(alignment: .leading, spacing: 10) {
                Text("Before You Scan")
                    .font(.headline.weight(.semibold))
                    .foregroundStyle(Color(red: 0.14, green: 0.18, blue: 0.25))
                Text("1. Open Codex on desktop.\n2. Show the mobile QR code.\n3. Scan it here to start a trusted session.")
                    .font(.subheadline)
                    .foregroundStyle(Color(red: 0.31, green: 0.39, blue: 0.49))
            }
            .padding(18)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(
                RoundedRectangle(cornerRadius: 24, style: .continuous)
                    .fill(Color.white.opacity(0.72))
            )
        }
        .padding(24)
        .background(
            RoundedRectangle(cornerRadius: 34, style: .continuous)
                .fill(
                    LinearGradient(
                        colors: [
                            Color(red: 0.99, green: 0.97, blue: 0.93),
                            Color(red: 0.92, green: 0.95, blue: 0.98),
                        ],
                        startPoint: .topLeading,
                        endPoint: .bottomTrailing
                    )
                )
                .overlay(
                    RoundedRectangle(cornerRadius: 34, style: .continuous)
                        .stroke(Color.white.opacity(0.82), lineWidth: 1)
                )
        )
    }

    @ViewBuilder
    private var recentSessionsSection: some View {
        if model.hasSavedProfiles {
        VStack(alignment: .leading, spacing: 14) {
            HStack(alignment: .firstTextBaseline) {
                VStack(alignment: .leading, spacing: 4) {
                    Text("On This Device")
                        .font(.headline.weight(.semibold))
                    Text(
                        model.isError
                            ? "Open another saved session before rescanning from desktop."
                            : "Tap Open to reconnect, or use Scan QR for another desktop."
                    )
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }
                Spacer()
                Button("All Sessions") {
                    model.presentSessionsSheet()
                }
                .font(.subheadline.weight(.semibold))
            }

            VStack(spacing: 12) {
                ForEach(model.savedProfilesPreview) { profile in
                    SessionCard(
                        profile: profile,
                        presentation: makeSessionCardPresentation(model: model, profile: profile),
                        onOpen: {
                            model.activateProfile(profile)
                        }
                    )
                }
            }
        }
        .padding(22)
        .background(
            RoundedRectangle(cornerRadius: 30, style: .continuous)
                .fill(Color.white.opacity(0.56))
                .overlay(
                    RoundedRectangle(cornerRadius: 30, style: .continuous)
                        .stroke(Color.white.opacity(0.72), lineWidth: 1)
                )
        )
        }
    }

    private func errorCard(profile: BridgeProfile) -> some View {
        VStack(alignment: .leading, spacing: 16) {
            HStack {
                StatusPill(label: "Connection Needs Attention", tone: .warning)
                Spacer()
            }
            Text(model.workspaceErrorTitle(model.statusMessage))
                .font(.title3.weight(.bold))
            Text(model.summarizeWorkspaceError(model.statusMessage))
                .font(.body)
                .foregroundStyle(.secondary)

            HStack(spacing: 12) {
                ActionButton(
                    title: model.isTransientProfile(profile) ? "Scan Again" : "Try Again",
                    systemImage: model.isTransientProfile(profile) ? "viewfinder" : "arrow.clockwise"
                ) {
                    if model.isTransientProfile(profile) {
                        model.openScanner()
                    } else {
                        model.reloadActiveBridge()
                    }
                }
                ActionButton(title: "Other Sessions", systemImage: "rectangle.stack.fill.badge.person.crop") {
                    model.presentSessionsSheet()
                }
            }
        }
        .padding(22)
        .background(
            RoundedRectangle(cornerRadius: 30, style: .continuous)
                .fill(Color(red: 1.0, green: 0.95, blue: 0.92).opacity(0.92))
                .overlay(
                    RoundedRectangle(cornerRadius: 30, style: .continuous)
                        .stroke(Color(red: 0.86, green: 0.54, blue: 0.41), lineWidth: 1)
                )
        )
    }
}

private struct SessionsSheet: View {
    @Environment(\.dismiss) private var dismiss
    @ObservedObject var model: NativeHostModel

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 18) {
                    Text(model.activeSessionForSheet == nil
                         ? "Scan a QR code or reopen a trusted session on this device."
                         : "Switch sessions, reload the current one, or reset this device if the saved setup has gone stale.")
                        .font(.body)
                        .foregroundStyle(.secondary)

                    VStack(spacing: 12) {
                        if model.activeSessionForSheet != nil {
                            sheetAction(title: "Reload Workspace", systemImage: "arrow.clockwise") {
                                dismiss()
                                model.reloadActiveBridge()
                            }
                            sheetAction(title: "Scan New QR Code", systemImage: "viewfinder") {
                                dismiss()
                                model.openScanner()
                            }
                            sheetAction(title: "Reset This Device", systemImage: "trash", tint: .red) {
                                dismiss()
                                model.profilePendingReset = model.activeSessionForSheet
                            }
                        } else {
                            sheetAction(title: "Scan QR Code", systemImage: "viewfinder") {
                                dismiss()
                                model.openScanner()
                            }
                        }
                    }

                    Text("Trusted Sessions on This Device")
                        .font(.headline.weight(.semibold))

                    if model.profiles.isEmpty {
                        Text("No trusted sessions yet.")
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                    } else {
                        VStack(spacing: 12) {
                            ForEach(model.profiles) { profile in
                                SessionCard(
                                    profile: profile,
                                    presentation: makeSessionCardPresentation(model: model, profile: profile),
                                    onOpen: {
                                        dismiss()
                                        model.activateProfile(profile)
                                    }
                                )
                            }
                        }
                    }
                }
                .padding(20)
            }
            .navigationTitle(model.activeSessionForSheet == nil ? "Sessions" : "Session Control")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Done") {
                        dismiss()
                    }
                }
            }
        }
    }

    private func sheetAction(title: String, systemImage: String, tint: Color = .accentColor, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            HStack {
                Label(title, systemImage: systemImage)
                    .font(.headline.weight(.semibold))
                Spacer()
                Image(systemName: "chevron.right")
                    .font(.caption.weight(.bold))
            }
            .padding(16)
            .background(
                RoundedRectangle(cornerRadius: 22, style: .continuous)
                    .fill(Color(.secondarySystemBackground))
            )
        }
        .foregroundStyle(tint)
    }
}

private struct SettingsSheet: View {
    @Environment(\.dismiss) private var dismiss
    @ObservedObject var model: NativeHostModel

    var body: some View {
        NavigationStack {
            Form {
                Section("Appearance") {
                    Picker("Theme Mode", selection: $model.themeMode) {
                        ForEach(ThemeMode.allCases) { mode in
                            Text(mode.displayName).tag(mode)
                        }
                    }
                    .pickerStyle(.inline)
                    .onChange(of: model.themeMode) { _, newValue in
                        model.saveThemeMode(newValue)
                    }
                }
            }
            .navigationTitle("Settings")
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("Done") {
                        dismiss()
                    }
                }
            }
        }
    }
}

private struct SessionCard: View {
    struct Badge {
        let label: String
        let tone: StatusPill.Tone
    }

    struct Presentation {
        let detail: String
        let badge: Badge?
        let isActionEnabled: Bool
        let showsProgress: Bool
    }

    let profile: BridgeProfile
    let presentation: Presentation
    let onOpen: () -> Void

    var body: some View {
        Button(action: onOpen) {
            VStack(alignment: .leading, spacing: 12) {
                HStack(spacing: 14) {
                    ZStack {
                        RoundedRectangle(cornerRadius: 16, style: .continuous)
                            .fill(Color(red: 0.17, green: 0.22, blue: 0.29))
                        Text(profileAvatar(profile))
                            .font(.headline.weight(.bold))
                            .foregroundStyle(.white)
                    }
                    .frame(width: 52, height: 52)

                    VStack(alignment: .leading, spacing: 4) {
                        Text(displayProfileLabel(profile))
                            .font(.headline.weight(.semibold))
                        Text(displayProfileDetail(profile))
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                        Text(presentation.detail)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                            .multilineTextAlignment(.leading)
                    }

                    Spacer()

                    VStack(alignment: .trailing, spacing: 8) {
                        if let badge = presentation.badge {
                            StatusPill(label: badge.label, tone: badge.tone)
                        } else {
                            Text("Open")
                                .font(.subheadline.weight(.semibold))
                                .foregroundStyle(Color(red: 0.17, green: 0.43, blue: 0.67))
                        }

                        if presentation.isActionEnabled {
                            Image(systemName: "chevron.right")
                                .font(.caption.weight(.bold))
                                .foregroundStyle(.secondary)
                        }
                    }
                }

                if presentation.showsProgress {
                    ProgressView()
                        .tint(Color(red: 0.18, green: 0.51, blue: 0.73))
                        .frame(maxWidth: .infinity, alignment: .leading)
                }
            }
        }
        .padding(16)
        .background(
            RoundedRectangle(cornerRadius: 24, style: .continuous)
                .fill(Color.white.opacity(0.72))
                .overlay(
                    RoundedRectangle(cornerRadius: 24, style: .continuous)
                        .stroke(cardBorderColor, lineWidth: cardBorderWidth)
                )
        )
        .buttonStyle(.plain)
        .disabled(!presentation.isActionEnabled)
    }

    private var cardBorderColor: Color {
        if presentation.showsProgress {
            return Color(red: 0.17, green: 0.43, blue: 0.67)
        }
        switch presentation.badge?.tone {
        case .warning:
            return Color(red: 0.86, green: 0.54, blue: 0.41)
        default:
            return Color.white.opacity(0.18)
        }
    }

    private var cardBorderWidth: CGFloat {
        switch presentation.badge?.tone {
        case .warning:
            return 2
        default:
            return presentation.showsProgress ? 2 : 1
        }
    }
}

@MainActor
private func makeSessionCardPresentation(model: NativeHostModel, profile: BridgeProfile) -> SessionCard.Presentation {
    let isCurrent = profile.id == model.activeSessionForSheet?.id
    let isConnected = isCurrent && model.isConnected
    let isConnecting = isCurrent && model.isLoading
    let isErrored = isCurrent && model.isError

    let badge: SessionCard.Badge?
    if isConnecting {
        badge = SessionCard.Badge(label: "Connecting", tone: .active)
    } else if isErrored {
        badge = SessionCard.Badge(label: "Retry", tone: .warning)
    } else if isConnected {
        badge = SessionCard.Badge(label: "Current", tone: .success)
    } else {
        badge = nil
    }

    let detail: String
    if isConnecting {
        detail = model.describeConnectingStatus(profile)
    } else if isErrored {
        detail = model.summarizeWorkspaceError(model.statusMessage)
    } else {
        detail = describeProfileStatus(profile, isCurrent: isCurrent)
    }

    return SessionCard.Presentation(
        detail: detail,
        badge: badge,
        isActionEnabled: !(isConnected || isConnecting),
        showsProgress: isConnecting
    )
}

private struct CircleIconButton: View {
    let systemImage: String
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            Image(systemName: systemImage)
                .font(.headline.weight(.bold))
                .foregroundStyle(Color(red: 0.13, green: 0.19, blue: 0.27))
                .frame(width: 42, height: 42)
                .background(
                    Circle()
                        .fill(Color.white.opacity(0.82))
                )
        }
    }
}

private struct ActionButton: View {
    let title: String
    let systemImage: String
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            Label(title, systemImage: systemImage)
                .font(.headline.weight(.semibold))
                .frame(maxWidth: .infinity)
                .padding(.vertical, 14)
        }
        .buttonStyle(.borderedProminent)
        .tint(Color(red: 0.17, green: 0.43, blue: 0.67))
    }
}

private struct SceneNode: View {
    let title: String
    let systemImage: String

    var body: some View {
        VStack(spacing: 10) {
            ZStack {
                RoundedRectangle(cornerRadius: 18, style: .continuous)
                    .fill(Color.white.opacity(0.92))
                Image(systemName: systemImage)
                    .font(.title2.weight(.bold))
                    .foregroundStyle(Color(red: 0.18, green: 0.36, blue: 0.56))
            }
            .frame(height: 76)
            Text(title)
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(.primary)
        }
        .frame(maxWidth: .infinity)
    }
}

private struct SceneConnector: View {
    var body: some View {
        Capsule()
            .fill(
                LinearGradient(
                    colors: [Color(red: 0.20, green: 0.46, blue: 0.70), Color(red: 0.61, green: 0.79, blue: 0.89)],
                    startPoint: .leading,
                    endPoint: .trailing
                )
            )
            .frame(width: 42, height: 8)
    }
}

private struct FlowLabels: View {
    let labels: [String]
    let activeLabel: String?

    var body: some View {
        FlexibleView(data: labels, spacing: 10, alignment: .leading) { label in
            Text(label)
                .font(.caption.weight(.semibold))
                .padding(.horizontal, 10)
                .padding(.vertical, 8)
                .background(
                    Capsule()
                        .fill((label == activeLabel ? Color(red: 0.18, green: 0.51, blue: 0.73) : Color.white).opacity(label == activeLabel ? 1 : 0.78))
                )
                .foregroundStyle(label == activeLabel ? .white : Color(red: 0.21, green: 0.29, blue: 0.38))
        }
    }
}

private struct StatusPill: View {
    enum Tone {
        case active
        case success
        case warning

        var background: Color {
            switch self {
            case .active:
                return Color(red: 0.82, green: 0.92, blue: 1.0)
            case .success:
                return Color(red: 0.86, green: 0.96, blue: 0.90)
            case .warning:
                return Color(red: 1.0, green: 0.90, blue: 0.83)
            }
        }

        var foreground: Color {
            switch self {
            case .active:
                return Color(red: 0.14, green: 0.38, blue: 0.62)
            case .success:
                return Color(red: 0.13, green: 0.46, blue: 0.24)
            case .warning:
                return Color(red: 0.72, green: 0.32, blue: 0.18)
            }
        }
    }

    let label: String
    let tone: Tone

    var body: some View {
        Text(label)
            .font(.caption.weight(.bold))
            .padding(.horizontal, 10)
            .padding(.vertical, 6)
            .background(Capsule().fill(tone.background))
            .foregroundStyle(tone.foreground)
    }
}

private struct FlexibleView<Data: Collection, Content: View>: View where Data.Element: Hashable {
    let data: Data
    let spacing: CGFloat
    let alignment: HorizontalAlignment
    let content: (Data.Element) -> Content

    init(data: Data, spacing: CGFloat, alignment: HorizontalAlignment, @ViewBuilder content: @escaping (Data.Element) -> Content) {
        self.data = data
        self.spacing = spacing
        self.alignment = alignment
        self.content = content
    }

    var body: some View {
        GeometryReader { geometry in
            self.generateContent(in: geometry)
        }
        .frame(minHeight: 10)
    }

    private func generateContent(in geometry: GeometryProxy) -> some View {
        var width = CGFloat.zero
        var height = CGFloat.zero
        let items = Array(data)

        return ZStack(alignment: Alignment(horizontal: alignment, vertical: .top)) {
            ForEach(Array(items.enumerated()), id: \.offset) { index, item in
                content(item)
                    .padding(.trailing, spacing)
                    .alignmentGuide(.leading) { dimension in
                        if abs(width - dimension.width) > geometry.size.width {
                            width = 0
                            height -= dimension.height + spacing
                        }
                        let result = width
                        if index == items.indices.last {
                            width = 0
                        } else {
                            width -= dimension.width + spacing
                        }
                        return result
                    }
                    .alignmentGuide(.top) { _ in
                        let result = height
                        if index == items.indices.last {
                            height = 0
                        }
                        return result
                    }
            }
        }
    }
}
