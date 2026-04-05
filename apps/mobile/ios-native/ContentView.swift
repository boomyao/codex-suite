import SwiftUI
import UIKit
import WebKit

private protocol BrowserKeyboardTextViewDelegate: AnyObject {
    func browserKeyboardTextView(
        _ textView: BrowserKeyboardTextView,
        didCommitTextState text: String,
        selectedRange: NSRange
    )
    func browserKeyboardTextViewDidPressEnter(_ textView: BrowserKeyboardTextView)
}

private final class BrowserKeyboardTextView: UITextView {
    weak var bridgeDelegate: BrowserKeyboardTextViewDelegate?
    private var mirroredCommittedText = ""
    private var mirroredCommittedSelection = NSRange(location: 0, length: 0)
    private var suppressBridgeSync = false
    private var appliedSeedVersion = -1

    override var canBecomeFirstResponder: Bool { true }

    override init(frame: CGRect, textContainer: NSTextContainer?) {
        super.init(frame: frame, textContainer: textContainer)
        backgroundColor = .clear
        textColor = .clear
        tintColor = .clear
        isOpaque = false
        isScrollEnabled = false
        autocorrectionType = .no
        autocapitalizationType = .none
        smartDashesType = .no
        smartQuotesType = .no
        spellCheckingType = .no
        text = ""
        accessibilityLabel = "Browser Keyboard Bridge"
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }

    override func insertText(_ text: String) {
        if text == "\n" {
            bridgeDelegate?.browserKeyboardTextViewDidPressEnter(self)
            return
        }

        super.insertText(text)
    }

    override func deleteBackward() {
        super.deleteBackward()
    }

    override func paste(_ sender: Any?) {
        super.paste(sender)
    }

    func synchronizeCommittedTextStateIfNeeded() {
        guard suppressBridgeSync == false, markedTextRange == nil else {
            return
        }

        let currentText = text ?? ""
        let currentSelection = clampedSelectedRange(for: currentText)
        guard currentText != mirroredCommittedText || currentSelection != mirroredCommittedSelection else {
            return
        }

        mirroredCommittedText = currentText
        mirroredCommittedSelection = currentSelection
        bridgeDelegate?.browserKeyboardTextView(
            self,
            didCommitTextState: currentText,
            selectedRange: currentSelection
        )
    }

    func applyRemoteTextState(text: String, selectedRange: NSRange, seedVersion: Int) {
        guard appliedSeedVersion != seedVersion else {
            return
        }
        suppressBridgeSync = true
        self.text = text
        let clampedRange = clampedSelectedRange(for: text, proposedRange: selectedRange)
        self.selectedRange = clampedRange
        mirroredCommittedText = text
        mirroredCommittedSelection = clampedRange
        appliedSeedVersion = seedVersion
        suppressBridgeSync = false
    }

    func clearTransientBuffer() {
        suppressBridgeSync = true
        if text.isEmpty == false {
            text = ""
        }
        selectedRange = NSRange(location: 0, length: 0)
        mirroredCommittedText = ""
        mirroredCommittedSelection = NSRange(location: 0, length: 0)
        appliedSeedVersion = -1
        suppressBridgeSync = false
    }

    private func clampedSelectedRange(for text: String, proposedRange: NSRange? = nil) -> NSRange {
        let length = (text as NSString).length
        let range = proposedRange ?? selectedRange
        let location = min(max(range.location, 0), length)
        let maxLength = max(length - location, 0)
        let clampedLength = min(max(range.length, 0), maxLength)
        return NSRange(location: location, length: clampedLength)
    }
}

private struct BrowserKeyboardInputBridge: UIViewRepresentable {
    @Binding var isFocused: Bool
    let seedText: String
    let seedSelection: NSRange
    let seedVersion: Int
    let onTextState: (String, NSRange) -> Void
    let onEnter: () -> Void

    func makeCoordinator() -> Coordinator {
        Coordinator(onTextState: onTextState, onEnter: onEnter)
    }

    func makeUIView(context: Context) -> BrowserKeyboardTextView {
        let view = BrowserKeyboardTextView(frame: .zero)
        view.bridgeDelegate = context.coordinator
        view.delegate = context.coordinator
        return view
    }

    func updateUIView(_ uiView: BrowserKeyboardTextView, context: Context) {
        uiView.bridgeDelegate = context.coordinator
        uiView.delegate = context.coordinator
        uiView.applyRemoteTextState(text: seedText, selectedRange: seedSelection, seedVersion: seedVersion)
        if isFocused {
            if uiView.isFirstResponder == false {
                DispatchQueue.main.async {
                    uiView.becomeFirstResponder()
                }
            }
        } else if uiView.isFirstResponder {
            uiView.clearTransientBuffer()
            DispatchQueue.main.async {
                uiView.resignFirstResponder()
            }
        }
    }

    final class Coordinator: NSObject, BrowserKeyboardTextViewDelegate, UITextViewDelegate {
        private let onTextState: (String, NSRange) -> Void
        private let onEnter: () -> Void

        init(
            onTextState: @escaping (String, NSRange) -> Void,
            onEnter: @escaping () -> Void
        ) {
            self.onTextState = onTextState
            self.onEnter = onEnter
        }

        func browserKeyboardTextView(
            _ textView: BrowserKeyboardTextView,
            didCommitTextState text: String,
            selectedRange: NSRange
        ) {
            onTextState(text, selectedRange)
        }

        func browserKeyboardTextViewDidPressEnter(_ textView: BrowserKeyboardTextView) {
            onEnter()
        }

        func textViewDidChange(_ textView: UITextView) {
            (textView as? BrowserKeyboardTextView)?.synchronizeCommittedTextStateIfNeeded()
        }

        func textViewDidChangeSelection(_ textView: UITextView) {
            (textView as? BrowserKeyboardTextView)?.synchronizeCommittedTextStateIfNeeded()
        }
    }
}

struct ContentView: View {
    @Environment(\.colorScheme) private var colorScheme
    @ObservedObject var model: NativeHostModel

    var body: some View {
        ZStack {
            Color(.systemBackground)
                .ignoresSafeArea()

            if model.shouldKeepWebViewMounted {
                connectedView
            } else if model.isError, let activeProfile = model.activeProfile {
                disconnectedView {
                    errorCard(profile: activeProfile)
                }
            } else {
                disconnectedView {
                    EmptyView()
                }
            }
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
        .sheet(isPresented: $model.browserSessionSheetPresented) {
            BrowserSessionSheet(model: model)
        }
        .sheet(isPresented: $model.desktopSessionSheetPresented) {
            DesktopSessionSheet(model: model)
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

    // MARK: - Connected State

    private var connectedView: some View {
        VStack(spacing: 0) {
            CodexWebView(bridge: model.webBridge)
                .opacity(model.isLoading ? 0.01 : 1)
                .allowsHitTesting(model.isConnected)
                .overlay {
                    if model.isLoading {
                        loadingOverlay
                    } else if model.isAutoReconnectPending {
                        reconnectOverlay
                    }
                }

            if model.isConnected {
                floatingBar
            }
        }
    }

    private var floatingBar: some View {
        HStack(spacing: 14) {
            Circle()
                .fill(.green)
                .frame(width: 6, height: 6)
            if let profile = model.activeProfile {
                Text(displayProfileLabel(profile))
                    .font(.caption.weight(.medium))
                    .foregroundStyle(.secondary)
            }
            Spacer()
            Button {
                model.presentDesktopSessionSheet()
            } label: {
                Image(systemName: "display")
                    .font(.caption)
                    .foregroundStyle(.tertiary)
            }
            Button {
                model.presentSessionsSheet()
            } label: {
                Image(systemName: "arrow.triangle.2.circlepath")
                    .font(.caption)
                    .foregroundStyle(.tertiary)
            }
            Button {
                model.presentSettingsSheet()
            } label: {
                Image(systemName: "gearshape")
                    .font(.caption)
                    .foregroundStyle(.tertiary)
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 6)
        .background(Color(.secondarySystemBackground))
    }

    private var loadingOverlay: some View {
        ZStack {
            Color(.systemBackground).opacity(0.92)
                .ignoresSafeArea()

            VStack(spacing: 16) {
                ProgressView()
                    .controlSize(.regular)
                Text(model.currentConnectionStage?.title ?? "Connecting")
                    .font(.subheadline.weight(.medium))
                    .foregroundStyle(.secondary)
                if let profile = model.activeProfile {
                    Text(displayProfileLabel(profile))
                        .font(.caption)
                        .foregroundStyle(.tertiary)
                }
            }
        }
    }

    private var reconnectOverlay: some View {
        ZStack {
            Color(.systemBackground).opacity(0.92)
                .ignoresSafeArea()

            VStack(spacing: 20) {
                Image(systemName: "arrow.triangle.2.circlepath")
                    .font(.title2)
                    .foregroundStyle(.secondary)
                Text("Reconnecting")
                    .font(.headline)
                ProgressView()
                    .controlSize(.small)
                HStack(spacing: 16) {
                    Button("Retry Now") {
                        model.reloadActiveBridge()
                    }
                    .buttonStyle(.borderedProminent)
                    .tint(.primary.opacity(0.15))
                    .foregroundStyle(.primary)
                    Button("Switch") {
                        model.presentSessionsSheet()
                    }
                    .foregroundStyle(.secondary)
                }
                .font(.subheadline.weight(.medium))
            }
        }
        .transition(.opacity)
    }

    // MARK: - Disconnected State

    private func disconnectedView<ErrorContent: View>(@ViewBuilder errorContent: () -> ErrorContent) -> some View {
        VStack(spacing: 0) {
            // Top bar
            HStack {
                Spacer()
                Button {
                    model.presentSettingsSheet()
                } label: {
                    Image(systemName: "gearshape")
                        .font(.body.weight(.medium))
                        .foregroundStyle(.secondary)
                        .frame(width: 40, height: 40)
                }
            }
            .padding(.horizontal, 20)
            .padding(.top, 8)

            Spacer()

            // Center content
            VStack(spacing: 24) {
                errorContent()

                Text("Codex")
                    .font(.system(size: 34, weight: .bold, design: .default))
                    .foregroundStyle(.primary)

                Button {
                    model.openScanner()
                } label: {
                    HStack(spacing: 8) {
                        Image(systemName: "qrcode.viewfinder")
                        Text("Scan QR Code")
                    }
                    .font(.headline.weight(.semibold))
                    .foregroundStyle(Color(.systemBackground))
                    .padding(.horizontal, 32)
                    .padding(.vertical, 14)
                    .background(Capsule().fill(.primary))
                }
            }

            Spacer()

            // Saved profiles
            if model.hasSavedProfiles {
                savedProfilesList
            }
        }
    }

    @ViewBuilder
    private var savedProfilesList: some View {
        VStack(spacing: 0) {
            HStack {
                Text("Saved")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(.tertiary)
                    .textCase(.uppercase)
                    .tracking(0.5)
                Spacer()
                if model.profiles.count > 2 {
                    Button("All") {
                        model.presentSessionsSheet()
                    }
                    .font(.caption.weight(.medium))
                    .foregroundStyle(.secondary)
                }
            }
            .padding(.horizontal, 24)
            .padding(.bottom, 8)

            VStack(spacing: 1) {
                ForEach(model.savedProfilesPreview) { profile in
                    SessionRow(
                        profile: profile,
                        presentation: makeSessionCardPresentation(model: model, profile: profile),
                        onOpen: {
                            model.activateProfile(profile)
                        }
                    )
                }
            }
            .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
            .padding(.horizontal, 20)
        }
        .padding(.bottom, 20)
    }

    private func errorCard(profile: BridgeProfile) -> some View {
        VStack(spacing: 12) {
            Image(systemName: "exclamationmark.triangle")
                .font(.title2)
                .foregroundStyle(.orange)
            Text(model.workspaceErrorTitle(model.statusMessage))
                .font(.subheadline.weight(.semibold))
            Text(model.summarizeWorkspaceError(model.statusMessage))
                .font(.caption)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)
                .frame(maxWidth: 320)
            HStack(spacing: 16) {
                Button {
                    if model.isTransientProfile(profile) {
                        model.openScanner()
                    } else {
                        model.reloadActiveBridge()
                    }
                } label: {
                    Text(model.isTransientProfile(profile) ? "Scan Again" : "Retry")
                        .font(.subheadline.weight(.semibold))
                }
                .buttonStyle(.borderedProminent)
                .tint(.primary.opacity(0.15))
                .foregroundStyle(.primary)

                Button("Sessions") {
                    model.presentSessionsSheet()
                }
                .font(.subheadline.weight(.medium))
                .foregroundStyle(.secondary)
            }
        }
        .padding(.bottom, 16)
    }
}

private struct SessionsSheet: View {
    @Environment(\.dismiss) private var dismiss
    @ObservedObject var model: NativeHostModel

    var body: some View {
        NavigationStack {
            List {
                Section {
                    if model.activeSessionForSheet != nil {
                        Button {
                            dismiss()
                            model.reloadActiveBridge()
                        } label: {
                            Label("Reload", systemImage: "arrow.clockwise")
                        }
                        Button {
                            dismiss()
                            model.openScanner()
                        } label: {
                            Label("Scan New QR", systemImage: "qrcode.viewfinder")
                        }
                        Button(role: .destructive) {
                            dismiss()
                            model.profilePendingReset = model.activeSessionForSheet
                        } label: {
                            Label("Reset Device", systemImage: "trash")
                        }
                    } else {
                        Button {
                            dismiss()
                            model.openScanner()
                        } label: {
                            Label("Scan QR Code", systemImage: "qrcode.viewfinder")
                        }
                    }
                }

                Section("Saved Sessions") {
                    if model.profiles.isEmpty {
                        Text("No sessions yet.")
                            .foregroundStyle(.secondary)
                    } else {
                        ForEach(model.profiles) { profile in
                            SessionRow(
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
            .navigationTitle("Sessions")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Done") {
                        dismiss()
                    }
                }
            }
        }
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

                Section("Desktop Files") {
                    Toggle("Full Desktop File Access", isOn: $model.fullDesktopFileAccessEnabled)
                        .onChange(of: model.fullDesktopFileAccessEnabled) { _, newValue in
                            model.saveFullDesktopFileAccessEnabled(newValue)
                        }
                    Text("Lets this iPad browse any regular file or folder the bridge process can read. Special files stay blocked, and macOS permissions still apply.")
                        .font(.footnote)
                        .foregroundStyle(.secondary)
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

private struct BrowserCanvasInteractionOverlay: UIViewRepresentable {
    let onTap: (CGPoint) -> Void
    let onSinglePanBegan: (CGPoint) -> Void
    let onSinglePanChanged: (CGPoint, CGPoint, CGSize) -> Void
    let onSinglePanEnded: (CGPoint, CGPoint, CGSize) -> Void
    let onTwoFingerPan: (CGPoint, CGSize) -> Void
    let onPinch: (CGFloat, UIGestureRecognizer.State) -> Void

    func makeCoordinator() -> Coordinator {
        Coordinator(
            onTap: onTap,
            onSinglePanBegan: onSinglePanBegan,
            onSinglePanChanged: onSinglePanChanged,
            onSinglePanEnded: onSinglePanEnded,
            onTwoFingerPan: onTwoFingerPan,
            onPinch: onPinch
        )
    }

    func makeUIView(context: Context) -> UIView {
        let view = UIView(frame: .zero)
        view.backgroundColor = .clear

        let tapRecognizer = UITapGestureRecognizer(target: context.coordinator, action: #selector(Coordinator.handleTap(_:)))
        let singlePanRecognizer = UIPanGestureRecognizer(target: context.coordinator, action: #selector(Coordinator.handleSinglePan(_:)))
        singlePanRecognizer.minimumNumberOfTouches = 1
        singlePanRecognizer.maximumNumberOfTouches = 1

        let twoFingerPanRecognizer = UIPanGestureRecognizer(target: context.coordinator, action: #selector(Coordinator.handleTwoFingerPan(_:)))
        twoFingerPanRecognizer.minimumNumberOfTouches = 2
        twoFingerPanRecognizer.maximumNumberOfTouches = 2

        let pinchRecognizer = UIPinchGestureRecognizer(target: context.coordinator, action: #selector(Coordinator.handlePinch(_:)))

        tapRecognizer.require(toFail: singlePanRecognizer)

        singlePanRecognizer.delegate = context.coordinator
        twoFingerPanRecognizer.delegate = context.coordinator
        pinchRecognizer.delegate = context.coordinator

        view.addGestureRecognizer(tapRecognizer)
        view.addGestureRecognizer(singlePanRecognizer)
        view.addGestureRecognizer(twoFingerPanRecognizer)
        view.addGestureRecognizer(pinchRecognizer)
        return view
    }

    func updateUIView(_ uiView: UIView, context: Context) {
        context.coordinator.onTap = onTap
        context.coordinator.onSinglePanBegan = onSinglePanBegan
        context.coordinator.onSinglePanChanged = onSinglePanChanged
        context.coordinator.onSinglePanEnded = onSinglePanEnded
        context.coordinator.onTwoFingerPan = onTwoFingerPan
        context.coordinator.onPinch = onPinch
    }

    final class Coordinator: NSObject, UIGestureRecognizerDelegate {
        var onTap: (CGPoint) -> Void
        var onSinglePanBegan: (CGPoint) -> Void
        var onSinglePanChanged: (CGPoint, CGPoint, CGSize) -> Void
        var onSinglePanEnded: (CGPoint, CGPoint, CGSize) -> Void
        var onTwoFingerPan: (CGPoint, CGSize) -> Void
        var onPinch: (CGFloat, UIGestureRecognizer.State) -> Void

        private var singlePanStartLocation: CGPoint = .zero
        private var twoFingerLastTranslation: CGPoint = .zero

        init(
            onTap: @escaping (CGPoint) -> Void,
            onSinglePanBegan: @escaping (CGPoint) -> Void,
            onSinglePanChanged: @escaping (CGPoint, CGPoint, CGSize) -> Void,
            onSinglePanEnded: @escaping (CGPoint, CGPoint, CGSize) -> Void,
            onTwoFingerPan: @escaping (CGPoint, CGSize) -> Void,
            onPinch: @escaping (CGFloat, UIGestureRecognizer.State) -> Void
        ) {
            self.onTap = onTap
            self.onSinglePanBegan = onSinglePanBegan
            self.onSinglePanChanged = onSinglePanChanged
            self.onSinglePanEnded = onSinglePanEnded
            self.onTwoFingerPan = onTwoFingerPan
            self.onPinch = onPinch
        }

        @objc
        func handleTap(_ recognizer: UITapGestureRecognizer) {
            guard let view = recognizer.view, recognizer.state == .ended else {
                return
            }
            onTap(recognizer.location(in: view))
        }

        @objc
        func handleSinglePan(_ recognizer: UIPanGestureRecognizer) {
            guard let view = recognizer.view else {
                return
            }
            let location = recognizer.location(in: view)
            let translationPoint = recognizer.translation(in: view)
            let translation = CGSize(width: translationPoint.x, height: translationPoint.y)

            switch recognizer.state {
            case .began:
                singlePanStartLocation = location
                onSinglePanBegan(location)
            case .changed:
                onSinglePanChanged(location, singlePanStartLocation, translation)
            case .ended, .cancelled, .failed:
                onSinglePanEnded(location, singlePanStartLocation, translation)
                singlePanStartLocation = .zero
            default:
                break
            }
        }

        @objc
        func handleTwoFingerPan(_ recognizer: UIPanGestureRecognizer) {
            guard let view = recognizer.view else {
                return
            }
            let location = recognizer.location(in: view)
            let translation = recognizer.translation(in: view)

            switch recognizer.state {
            case .began:
                twoFingerLastTranslation = .zero
            case .changed:
                let delta = CGSize(
                    width: translation.x - twoFingerLastTranslation.x,
                    height: translation.y - twoFingerLastTranslation.y
                )
                twoFingerLastTranslation = translation
                onTwoFingerPan(location, delta)
            case .ended, .cancelled, .failed:
                twoFingerLastTranslation = .zero
            default:
                break
            }
        }

        @objc
        func handlePinch(_ recognizer: UIPinchGestureRecognizer) {
            onPinch(recognizer.scale, recognizer.state)
            recognizer.scale = 1
        }

        func gestureRecognizer(_ gestureRecognizer: UIGestureRecognizer, shouldRecognizeSimultaneouslyWith otherGestureRecognizer: UIGestureRecognizer) -> Bool {
            gestureRecognizer is UIPinchGestureRecognizer || otherGestureRecognizer is UIPinchGestureRecognizer
        }
    }
}

private enum BrowserCanvasGestureMode {
    case idle
    case pendingTap
    case scrolling
    case dragging
}

private enum BrowserCanvasInteractionMode: String, CaseIterable, Identifiable {
    case scroll
    case drag

    var id: String { rawValue }

    var title: String {
        switch self {
        case .scroll:
            return "Scroll"
        case .drag:
            return "Drag"
        }
    }
}

private struct BrowserSessionSheet: View {
    @Environment(\.dismiss) private var dismiss
    @ObservedObject var model: NativeHostModel
    @State private var source: NativeHostBrowserSessionSource = .preview
    @State private var preset: NativeHostBrowserViewportPreset = .desktop
    @State private var previewURL = ""
    @State private var attachTargets: [NativeHostBrowserAttachTarget] = []
    @State private var selectedAttachTargetID: String?
    @State private var sessionSummary: NativeHostBrowserSessionSummary?
    @State private var sessionSnapshot: NativeHostBrowserSessionSnapshot?
    @State private var navigationURL = ""
    @State private var textInput = ""
    @State private var errorMessage: String?
    @State private var isWorking = false
    @State private var diagnosticsExpanded = false
    @State private var interactionBoostUntil = Date.distantPast
    @State private var keyboardBridgeFocused = false
    @State private var canvasInteractionMode: BrowserCanvasInteractionMode = .scroll
    @State private var canvasGestureMode: BrowserCanvasGestureMode = .idle
    @State private var canvasGestureStartLocation: CGPoint = .zero
    @State private var canvasGestureCurrentLocation: CGPoint = .zero
    @State private var canvasZoomScale: CGFloat = 1
    @State private var scrollLastTranslation: CGSize = .zero
    @State private var scrollLastDispatchAt = Date.distantPast
    @State private var dragLastMoveDispatchAt = Date.distantPast
    @State private var pendingLongPressDragTask: Task<Void, Never>?
    @State private var keyboardOperationChain: Task<Void, Never>?
    @State private var keyboardBridgeSeedText = ""
    @State private var keyboardBridgeSeedSelection = NSRange(location: 0, length: 0)
    @State private var keyboardBridgeSeedVersion = 0
    @State private var keyboardQueuedOperationCount = 0
    @State private var keyboardMirroredText = ""
    @State private var keyboardMirroredSelection = NSRange(location: 0, length: 0)
    @State private var pendingEditableStateSyncTask: Task<Void, Never>?
    @State private var frameStreamClient: BrowserSessionFrameStreamClient?
    @State private var streamedFrameImageData: Data?

    var body: some View {
        NavigationStack {
            Group {
                if let sessionSummary {
                    liveSessionView(summary: sessionSummary)
                } else {
                    configurationView
                }
            }
            .navigationTitle(sessionSummary?.title ?? "Browser")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Done") {
                        closeSheet()
                    }
                }
                if sessionSummary != nil {
                    ToolbarItem(placement: .confirmationAction) {
                        Button("End") {
                            Task {
                                await endSession()
                            }
                        }
                    }
                }
            }
        }
        .overlay(alignment: .topLeading) {
            BrowserKeyboardInputBridge(
                isFocused: $keyboardBridgeFocused,
                seedText: keyboardBridgeSeedText,
                seedSelection: keyboardBridgeSeedSelection,
                seedVersion: keyboardBridgeSeedVersion,
                onTextState: { text, selectedRange in
                    handleKeyboardCommittedTextState(text: text, selectedRange: selectedRange)
                },
                onEnter: {
                    handleKeyboardEnter()
                }
            )
            .frame(width: 1, height: 1)
            .opacity(0.01)
            .allowsHitTesting(false)
        }
        .presentationDetents([.large])
        .task(id: source) {
            guard source == .attach, sessionSummary == nil else {
                return
            }
            await refreshAttachTargets()
        }
        .task(id: sessionSummary?.sessionID) {
            guard let sessionID = sessionSummary?.sessionID else {
                return
            }
            await pollSnapshots(sessionID: sessionID)
        }
        .onChange(of: canvasInteractionMode) { _, _ in
            resetCanvasGestureState()
        }
        .onDisappear {
            keyboardBridgeFocused = false
            resetCanvasGestureState()
            pendingEditableStateSyncTask?.cancel()
            pendingEditableStateSyncTask = nil
            resetKeyboardOperationChain()
            let streamClient = frameStreamClient
            frameStreamClient = nil
            streamedFrameImageData = nil
            if let streamClient {
                Task {
                    await streamClient.close()
                }
            }
            if let sessionID = sessionSummary?.sessionID {
                Task {
                    await model.stopBrowserSession(sessionID: sessionID)
                }
            }
        }
    }

    private var configurationView: some View {
        Form {
            Section("Mode") {
                Picker("Source", selection: $source) {
                    ForEach(NativeHostBrowserSessionSource.allCases) { mode in
                        Text(mode.displayName).tag(mode)
                    }
                }
                .pickerStyle(.segmented)

                Text(source.subtitle)
                    .font(.footnote)
                    .foregroundStyle(.secondary)

                Picker("Viewport", selection: $preset) {
                    ForEach(NativeHostBrowserViewportPreset.allCases) { preset in
                        Text(preset.displayName).tag(preset)
                    }
                }
                .pickerStyle(.segmented)
            }

            if source == .preview {
                Section("Preview") {
                    TextField("https://example.com or localhost:3000", text: $previewURL)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()

                    Text("Launches a dedicated browser owned by the bridge. This is the safest mode for previews and local dev servers.")
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                }
            } else {
                Section("Chrome Tabs") {
                    Button {
                        Task {
                            await refreshAttachTargets()
                        }
                    } label: {
                        Label("Refresh Chrome Tabs", systemImage: "arrow.clockwise")
                    }

                    if attachTargets.isEmpty {
                        Text(errorMessage ?? "No attachable Chrome tabs were found yet. Start Chrome with remote debugging enabled, then refresh.")
                            .font(.footnote)
                            .foregroundStyle(.secondary)
                    } else {
                        ForEach(attachTargets) { target in
                            Button {
                                selectedAttachTargetID = target.id
                            } label: {
                                HStack(alignment: .top, spacing: 12) {
                                    Image(systemName: selectedAttachTargetID == target.id ? "checkmark.circle.fill" : "circle")
                                        .foregroundStyle(selectedAttachTargetID == target.id ? Color.accentColor : Color.secondary)
                                    VStack(alignment: .leading, spacing: 4) {
                                        Text(target.title)
                                            .font(.headline)
                                            .foregroundStyle(.primary)
                                        if target.url.isEmpty == false {
                                            Text(target.url)
                                                .font(.caption)
                                                .foregroundStyle(.secondary)
                                                .lineLimit(2)
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }

            Section {
                Button {
                    Task {
                        await startSession()
                    }
                } label: {
                    HStack {
                        if isWorking {
                            ProgressView()
                                .controlSize(.small)
                        }
                        Text(source == .preview ? "Start Preview Browser" : "Attach Selected Tab")
                            .font(.headline.weight(.semibold))
                    }
                    .frame(maxWidth: .infinity, alignment: .center)
                }
                .disabled(isWorking || !canStartSession)
            }

            if let errorMessage, sessionSummary == nil {
                Section("Status") {
                    Text(errorMessage)
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                }
            }
        }
    }

    private func liveSessionView(summary: NativeHostBrowserSessionSummary) -> some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                sessionToolbar(summary: summary)
                sessionCanvas(summary: summary)
                sessionInputBar(summary: summary)
                sessionDiagnostics
            }
            .padding(16)
        }
        .background(Color(.systemGroupedBackground))
    }

    private func sessionToolbar(summary: NativeHostBrowserSessionSummary) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack(spacing: 10) {
                StatusPill(label: summary.source.displayName, tone: .active)
                if summary.isStreaming {
                    StatusPill(label: "Live", tone: .active)
                }
                if summary.loading {
                    StatusPill(label: "Loading", tone: .warning)
                }
                Spacer()
            }

            TextField("Navigate", text: $navigationURL)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
                .textFieldStyle(.roundedBorder)
                .onSubmit {
                    Task {
                        await navigate(action: "navigate", url: navigationURL)
                    }
                }

            HStack(spacing: 10) {
                Button {
                    Task {
                        await navigate(action: "back")
                    }
                } label: {
                    Label("Back", systemImage: "chevron.left")
                }
                .disabled(isWorking || summary.canGoBack == false)

                Button {
                    Task {
                        await navigate(action: "forward")
                    }
                } label: {
                    Label("Forward", systemImage: "chevron.right")
                }
                .disabled(isWorking || summary.canGoForward == false)

                Button {
                    Task {
                        await navigate(action: "reload")
                    }
                } label: {
                    Label("Reload", systemImage: "arrow.clockwise")
                }
                .disabled(isWorking)

                Spacer()

                Text("\(summary.width)×\(summary.height)")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .padding(16)
        .background(
            RoundedRectangle(cornerRadius: 24, style: .continuous)
                .fill(Color(.secondarySystemBackground))
        )
    }

    private func sessionCanvas(summary: NativeHostBrowserSessionSummary) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            if let errorMessage {
                Text(errorMessage)
                    .font(.footnote)
                    .foregroundStyle(.secondary)
            }

            Picker("Interaction", selection: $canvasInteractionMode) {
                ForEach(BrowserCanvasInteractionMode.allCases) { mode in
                    Text(mode.title).tag(mode)
                }
            }
            .pickerStyle(.segmented)

            GeometryReader { geometry in
                let uiImage = displayedBrowserFrameImage
                ZStack {
                    RoundedRectangle(cornerRadius: 28, style: .continuous)
                        .fill(Color.black.opacity(0.88))
                    if let uiImage {
                        let imageRect = fittedImageRect(for: uiImage.size, in: geometry.size)
                        let displayedImageRect = scaledImageRect(from: imageRect)
                        Image(uiImage: uiImage)
                            .resizable()
                            .aspectRatio(contentMode: .fit)
                            .frame(width: imageRect.width, height: imageRect.height)
                            .scaleEffect(canvasZoomScale, anchor: .center)
                            .position(x: imageRect.midX, y: imageRect.midY)
                        BrowserCanvasInteractionOverlay(
                            onTap: { location in
                                guard displayedImageRect.contains(location),
                                      let normalizedPoint = normalizedBrowserPoint(
                                        for: location,
                                        in: displayedImageRect,
                                        allowClampingOutside: false
                                      ) else {
                                    return
                                }
                                Task {
                                    await sendTap(xNorm: normalizedPoint.x, yNorm: normalizedPoint.y)
                                }
                            },
                            onSinglePanBegan: { startLocation in
                                guard displayedImageRect.contains(startLocation) else {
                                    return
                                }
                                handleCanvasPanBegan(startLocation, imageRect: displayedImageRect)
                            },
                            onSinglePanChanged: { currentLocation, startLocation, translation in
                                guard displayedImageRect.contains(startLocation) else {
                                    return
                                }
                                handleCanvasPanChanged(
                                    currentLocation: currentLocation,
                                    startLocation: startLocation,
                                    translation: translation,
                                    imageRect: displayedImageRect
                                )
                            },
                            onSinglePanEnded: { currentLocation, startLocation, translation in
                                guard displayedImageRect.contains(startLocation) else {
                                    resetCanvasGestureState()
                                    return
                                }
                                handleCanvasPanEnded(
                                    currentLocation: currentLocation,
                                    startLocation: startLocation,
                                    translation: translation,
                                    imageRect: displayedImageRect
                                )
                            },
                            onTwoFingerPan: { location, delta in
                                guard let normalizedPoint = normalizedBrowserPoint(
                                    for: location,
                                    in: displayedImageRect,
                                    allowClampingOutside: true
                                ) else {
                                    return
                                }
                                Task {
                                    await scroll(
                                        deltaX: -Double(delta.width) * scrollWheelScale,
                                        deltaY: -Double(delta.height) * scrollWheelScale,
                                        xNorm: normalizedPoint.x,
                                        yNorm: normalizedPoint.y
                                    )
                                }
                            },
                            onPinch: { scaleDelta, state in
                                handleCanvasPinch(scaleDelta: scaleDelta, state: state)
                            }
                        )
                        .frame(width: geometry.size.width, height: geometry.size.height)
                    } else {
                        VStack(spacing: 12) {
                            if isWorking {
                                ProgressView()
                            }
                            Text("Waiting for browser frame…")
                                .font(.body)
                                .foregroundStyle(.white.opacity(0.85))
                        }
                    }
                }
            }
            .frame(minHeight: 280, idealHeight: 420)
            .clipShape(RoundedRectangle(cornerRadius: 28, style: .continuous))

            HStack(spacing: 10) {
                Button {
                    Task {
                        await scroll(deltaY: -420)
                    }
                } label: {
                    Label("Scroll Up", systemImage: "arrow.up")
                }
                .disabled(sessionSummary == nil)

                Button {
                    Task {
                        await scroll(deltaY: 420)
                    }
                } label: {
                    Label("Scroll Down", systemImage: "arrow.down")
                }
                .disabled(sessionSummary == nil)

                Button("Reset Zoom") {
                    canvasZoomScale = 1
                }
                .disabled(canvasZoomScale <= 1.01)

                Spacer()

                Text(canvasInteractionMode == .scroll
                     ? "Tap to click or focus. Drag to scroll. Long-press then drag for sliders, selection, or canvas work. Pinch zoom and use two fingers to scroll."
                     : "Tap to click or focus. Drag mode sends pointer down/move/up for sliders, selection, and canvas work.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
    }

    private func sessionInputBar(summary: NativeHostBrowserSessionSummary) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("Input")
                .font(.headline)

            Text(keyboardBridgeFocused ? "System keyboard is attached to the remote field." : "Tap a field in the browser frame first to raise the iPad keyboard.")
                .font(.caption)
                .foregroundStyle(.secondary)

            TextEditor(text: $textInput)
                .font(.body)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
                .frame(minHeight: 92)
                .padding(8)
                .background(
                    RoundedRectangle(cornerRadius: 16, style: .continuous)
                        .fill(Color(.tertiarySystemBackground))
                )

            HStack(spacing: 10) {
                Button("Send Text") {
                    Task {
                        await sendText()
                    }
                }
                .disabled(textInput.isEmpty)

                Button("Enter") {
                    Task {
                        await sendKey("Enter")
                    }
                }
                Button("Tab") {
                    Task {
                        await sendKey("Tab")
                    }
                }
                Button("Esc") {
                    Task {
                        await sendKey("Escape")
                    }
                }
                Button("Backspace") {
                    Task {
                        await sendKey("Backspace")
                    }
                }
                Spacer()
            }
            .font(.subheadline.weight(.semibold))
        }
        .padding(16)
        .background(
            RoundedRectangle(cornerRadius: 24, style: .continuous)
                .fill(Color(.secondarySystemBackground))
        )
    }

    private var sessionDiagnostics: some View {
        DisclosureGroup("Diagnostics", isExpanded: $diagnosticsExpanded) {
            VStack(alignment: .leading, spacing: 10) {
                let snapshot = sessionSnapshot
                Text("Console: \(snapshot?.consoleErrorCount ?? 0) errors, \(snapshot?.consoleWarnCount ?? 0) warnings")
                    .font(.subheadline.weight(.semibold))
                if let consoleLines = snapshot?.consoleLines, consoleLines.isEmpty == false {
                    ForEach(consoleLines, id: \.self) { line in
                        Text(line)
                            .font(.caption.monospaced())
                            .foregroundStyle(.secondary)
                            .frame(maxWidth: .infinity, alignment: .leading)
                    }
                }
                Text("Network: \(snapshot?.networkInflightCount ?? 0) inflight, \(snapshot?.networkFailedCount ?? 0) failed")
                    .font(.subheadline.weight(.semibold))
                if let networkLines = snapshot?.networkFailedLines, networkLines.isEmpty == false {
                    ForEach(networkLines, id: \.self) { line in
                        Text(line)
                            .font(.caption.monospaced())
                            .foregroundStyle(.secondary)
                            .frame(maxWidth: .infinity, alignment: .leading)
                    }
                }
            }
            .padding(.top, 8)
        }
        .padding(16)
        .background(
            RoundedRectangle(cornerRadius: 24, style: .continuous)
                .fill(Color(.secondarySystemBackground))
        )
    }

    private var canStartSession: Bool {
        switch source {
        case .preview:
            return true
        case .attach:
            return selectedAttachTarget != nil
        }
    }

    private var selectedAttachTarget: NativeHostBrowserAttachTarget? {
        guard let selectedAttachTargetID else {
            return nil
        }
        return attachTargets.first(where: { $0.id == selectedAttachTargetID })
    }

    private func startSession() async {
        guard isWorking == false else {
            return
        }
        isWorking = true
        errorMessage = nil
        do {
            let summary: NativeHostBrowserSessionSummary
            switch source {
            case .preview:
                summary = try await model.startPreviewBrowserSession(
                    url: previewURL,
                    preset: preset
                )
            case .attach:
                guard let selectedAttachTarget else {
                    throw NativeHostBridgeError.requestFailed("Choose a Chrome tab to attach first.")
                }
                summary = try await model.startAttachedBrowserSession(
                    target: selectedAttachTarget,
                    preset: preset
                )
            }
            sessionSummary = summary
            syncKeyboardFocus(with: summary)
            navigationURL = summary.url
            sessionSnapshot = nil
            streamedFrameImageData = nil
            diagnosticsExpanded = false
            advanceInteractionBoostWindow()
            await refreshSnapshot(sessionID: summary.sessionID)
        } catch {
            errorMessage = (error as NSError).localizedDescription
        }
        isWorking = false
    }

    private func refreshAttachTargets() async {
        guard source == .attach, isWorking == false else {
            return
        }
        isWorking = true
        defer { isWorking = false }
        do {
            let targets = try await model.loadBrowserAttachTargets()
            attachTargets = targets
            if let selectedAttachTargetID,
               targets.contains(where: { $0.id == selectedAttachTargetID }) == false {
                self.selectedAttachTargetID = targets.first?.id
            } else if self.selectedAttachTargetID == nil {
                self.selectedAttachTargetID = targets.first?.id
            }
            errorMessage = nil
        } catch {
            attachTargets = []
            selectedAttachTargetID = nil
            errorMessage = (error as NSError).localizedDescription
        }
    }

    private func pollSnapshots(sessionID: String) async {
        while Task.isCancelled == false {
            if sessionSummary?.sessionID != sessionID {
                return
            }
            if sessionSummary?.isStreaming == true {
                await ensureFrameStreamStarted(sessionID: sessionID)
                if frameStreamClient == nil {
                    if let currentRevision = sessionSnapshot?.revision {
                        await refreshNextFrame(sessionID: sessionID, afterRevision: currentRevision)
                    } else {
                        await refreshSnapshot(sessionID: sessionID)
                    }
                } else {
                    try? await Task.sleep(nanoseconds: 1_000_000_000)
                }
                continue
            }
            if frameStreamClient != nil {
                await closeFrameStream()
            }
            await refreshSnapshot(sessionID: sessionID)
            try? await Task.sleep(nanoseconds: snapshotPollIntervalNanoseconds)
        }
    }

    private func navigate(action: String, url: String? = nil) async {
        guard let sessionSummary else {
            return
        }
        isWorking = true
        defer { isWorking = false }
        do {
            let nextSummary = try await model.navigateBrowserSession(
                sessionID: sessionSummary.sessionID,
                action: action,
                url: url
            )
            self.sessionSummary = nextSummary
            syncKeyboardFocus(with: nextSummary)
            advanceInteractionBoostWindow()
            if action != "navigate" || (url?.isEmpty == false) {
                navigationURL = nextSummary.url
            }
            errorMessage = nil
            if shouldUseSnapshotRefresh {
                await refreshSnapshot(sessionID: nextSummary.sessionID)
            }
        } catch {
            errorMessage = (error as NSError).localizedDescription
        }
    }

    private func sendTap(xNorm: Double, yNorm: Double) async {
        guard let sessionSummary else {
            return
        }
        do {
            let nextSummary = try await model.sendBrowserSessionTap(
                sessionID: sessionSummary.sessionID,
                xNorm: xNorm,
                yNorm: yNorm
            )
            self.sessionSummary = nextSummary
            syncKeyboardFocus(with: nextSummary, forceRemoteSeed: true)
            advanceInteractionBoostWindow()
            if nextSummary.textInputActive {
                scheduleEditableStateSync(sessionID: nextSummary.sessionID, forceRemoteSeed: true)
            }
            if shouldUseSnapshotRefresh {
                await refreshSnapshot(sessionID: sessionSummary.sessionID)
            }
        } catch {
            errorMessage = (error as NSError).localizedDescription
        }
    }

    private func sendPointerDown(xNorm: Double, yNorm: Double) async {
        guard let sessionSummary else {
            return
        }
        do {
            let nextSummary = try await model.sendBrowserSessionPointerDown(
                sessionID: sessionSummary.sessionID,
                xNorm: xNorm,
                yNorm: yNorm
            )
            self.sessionSummary = nextSummary
            syncKeyboardFocus(with: nextSummary)
            advanceInteractionBoostWindow()
            errorMessage = nil
        } catch {
            errorMessage = (error as NSError).localizedDescription
        }
    }

    private func sendPointerMove(xNorm: Double, yNorm: Double) async {
        guard let sessionSummary else {
            return
        }
        do {
            let nextSummary = try await model.sendBrowserSessionPointerMove(
                sessionID: sessionSummary.sessionID,
                xNorm: xNorm,
                yNorm: yNorm
            )
            self.sessionSummary = nextSummary
            advanceInteractionBoostWindow()
            errorMessage = nil
        } catch {
            errorMessage = (error as NSError).localizedDescription
        }
    }

    private func sendPointerUp(xNorm: Double, yNorm: Double) async {
        guard let sessionSummary else {
            return
        }
        do {
            let nextSummary = try await model.sendBrowserSessionPointerUp(
                sessionID: sessionSummary.sessionID,
                xNorm: xNorm,
                yNorm: yNorm
            )
            self.sessionSummary = nextSummary
            syncKeyboardFocus(with: nextSummary, forceRemoteSeed: true)
            advanceInteractionBoostWindow()
            errorMessage = nil
            if nextSummary.textInputActive {
                scheduleEditableStateSync(sessionID: nextSummary.sessionID, forceRemoteSeed: true)
            }
            if shouldUseSnapshotRefresh {
                await refreshSnapshot(sessionID: sessionSummary.sessionID)
            }
        } catch {
            errorMessage = (error as NSError).localizedDescription
        }
    }

    private func scroll(
        deltaX: Double = 0,
        deltaY: Double,
        xNorm: Double = 0.5,
        yNorm: Double = 0.5
    ) async {
        guard let sessionSummary else {
            return
        }
        do {
            self.sessionSummary = try await model.sendBrowserSessionScroll(
                sessionID: sessionSummary.sessionID,
                deltaX: deltaX,
                deltaY: deltaY,
                xNorm: xNorm,
                yNorm: yNorm
            )
            if let sessionSummary = self.sessionSummary {
                syncKeyboardFocus(with: sessionSummary)
            }
            advanceInteractionBoostWindow()
            if shouldUseSnapshotRefresh {
                await refreshSnapshot(sessionID: sessionSummary.sessionID)
            }
        } catch {
            errorMessage = (error as NSError).localizedDescription
        }
    }

    private func sendText() async {
        guard let sessionSummary else {
            return
        }
        guard textInput.isEmpty == false else {
            return
        }
        do {
            self.sessionSummary = try await model.sendBrowserSessionText(
                sessionID: sessionSummary.sessionID,
                text: textInput
            )
            if let sessionSummary = self.sessionSummary {
                syncKeyboardFocus(with: sessionSummary)
            }
            textInput = ""
            advanceInteractionBoostWindow()
            errorMessage = nil
            if shouldUseSnapshotRefresh {
                await refreshSnapshot(sessionID: sessionSummary.sessionID)
            }
        } catch {
            errorMessage = (error as NSError).localizedDescription
        }
    }

    private func sendKey(_ key: String) async {
        guard let sessionSummary else {
            return
        }
        do {
            self.sessionSummary = try await model.sendBrowserSessionKey(
                sessionID: sessionSummary.sessionID,
                key: key
            )
            if let sessionSummary = self.sessionSummary {
                syncKeyboardFocus(
                    with: sessionSummary,
                    forceRemoteSeed: shouldForceKeyboardRemoteSeed(
                        with: sessionSummary,
                        allowedPendingOperationCount: 0
                    )
                )
            }
            advanceInteractionBoostWindow()
            errorMessage = nil
            if shouldUseSnapshotRefresh {
                await refreshSnapshot(sessionID: sessionSummary.sessionID)
            }
        } catch {
            errorMessage = (error as NSError).localizedDescription
        }
    }

    private func endSession() async {
        guard let sessionID = sessionSummary?.sessionID else {
            return
        }
        pendingEditableStateSyncTask?.cancel()
        pendingEditableStateSyncTask = nil
        await closeFrameStream()
        await model.stopBrowserSession(sessionID: sessionID)
        sessionSummary = nil
        sessionSnapshot = nil
        streamedFrameImageData = nil
        navigationURL = ""
        keyboardBridgeFocused = false
        resetCanvasGestureState()
        resetKeyboardOperationChain()
    }

    private func closeSheet() {
        Task {
            await endSession()
            dismiss()
        }
    }

    private func fittedImageRect(for imageSize: CGSize, in containerSize: CGSize) -> CGRect {
        guard imageSize.width > 0, imageSize.height > 0,
              containerSize.width > 0, containerSize.height > 0 else {
            return CGRect(origin: .zero, size: containerSize)
        }
        let scale = min(containerSize.width / imageSize.width, containerSize.height / imageSize.height)
        let width = imageSize.width * scale
        let height = imageSize.height * scale
        let originX = (containerSize.width - width) / 2
        let originY = (containerSize.height - height) / 2
        return CGRect(x: originX, y: originY, width: width, height: height)
    }

    private var pointerMoveDispatchInterval: TimeInterval {
        sessionSummary?.isStreaming == true ? (1.0 / 45.0) : (1.0 / 24.0)
    }

    private var dragTapThreshold: CGFloat {
        10
    }

    private var scrollDispatchDistanceThreshold: CGFloat {
        3
    }

    private var scrollWheelScale: Double {
        2.0
    }

    private var longPressToDragDelayNanoseconds: UInt64 {
        350_000_000
    }

    private var maximumCanvasZoomScale: CGFloat {
        3
    }

    private func resetCanvasGestureState() {
        pendingLongPressDragTask?.cancel()
        pendingLongPressDragTask = nil
        canvasGestureMode = .idle
        canvasGestureStartLocation = .zero
        canvasGestureCurrentLocation = .zero
        scrollLastTranslation = .zero
        scrollLastDispatchAt = Date.distantPast
        dragLastMoveDispatchAt = Date.distantPast
    }

    private func normalizedBrowserPoint(
        for location: CGPoint,
        in imageRect: CGRect,
        allowClampingOutside: Bool
    ) -> CGPoint? {
        guard imageRect.width > 0, imageRect.height > 0 else {
            return nil
        }
        if allowClampingOutside == false, imageRect.contains(location) == false {
            return nil
        }
        let clampedX = min(max(location.x, imageRect.minX), imageRect.maxX)
        let clampedY = min(max(location.y, imageRect.minY), imageRect.maxY)
        let normalizedX = (clampedX - imageRect.minX) / max(imageRect.width, 1)
        let normalizedY = (clampedY - imageRect.minY) / max(imageRect.height, 1)
        return CGPoint(x: normalizedX, y: normalizedY)
    }

    private func scaledImageRect(from imageRect: CGRect) -> CGRect {
        guard imageRect.width > 0, imageRect.height > 0 else {
            return imageRect
        }
        let width = imageRect.width * canvasZoomScale
        let height = imageRect.height * canvasZoomScale
        return CGRect(
            x: imageRect.midX - (width / 2),
            y: imageRect.midY - (height / 2),
            width: width,
            height: height
        )
    }

    private func handleCanvasPinch(scaleDelta: CGFloat, state: UIGestureRecognizer.State) {
        guard state == .began || state == .changed else {
            return
        }
        let nextScale = min(max(canvasZoomScale * scaleDelta, 1), maximumCanvasZoomScale)
        canvasZoomScale = nextScale
    }

    private func handleCanvasPanBegan(_ startLocation: CGPoint, imageRect: CGRect) {
        canvasGestureStartLocation = startLocation
        canvasGestureCurrentLocation = startLocation
        canvasGestureMode = .pendingTap
        scrollLastTranslation = .zero
        scrollLastDispatchAt = Date.distantPast
        dragLastMoveDispatchAt = Date.distantPast
        scheduleLongPressDragActivationIfNeeded(imageRect: imageRect)
    }

    private func handleCanvasPanChanged(
        currentLocation: CGPoint,
        startLocation: CGPoint,
        translation: CGSize,
        imageRect: CGRect
    ) {
        let translationDistance = hypot(translation.width, translation.height)

        switch canvasGestureMode {
        case .idle:
            handleCanvasPanBegan(startLocation, imageRect: imageRect)
        case .pendingTap:
            switch canvasInteractionMode {
            case .scroll:
                canvasGestureCurrentLocation = currentLocation
                guard translationDistance >= dragTapThreshold else {
                    return
                }
                pendingLongPressDragTask?.cancel()
                pendingLongPressDragTask = nil
                canvasGestureMode = .scrolling
                scrollLastTranslation = .zero
                scrollLastDispatchAt = Date.distantPast
                dispatchIncrementalScroll(
                    currentLocation: currentLocation,
                    translation: translation,
                    imageRect: imageRect
                )
            case .drag:
                guard translationDistance >= dragTapThreshold else {
                    return
                }
                pendingLongPressDragTask?.cancel()
                pendingLongPressDragTask = nil
                canvasGestureMode = .dragging
                dragLastMoveDispatchAt = Date.distantPast
                beginPointerDrag(
                    startLocation: startLocation,
                    currentLocation: currentLocation,
                    imageRect: imageRect
                )
            }
        case .scrolling:
            dispatchIncrementalScroll(
                currentLocation: currentLocation,
                translation: translation,
                imageRect: imageRect
            )
        case .dragging:
            canvasGestureCurrentLocation = currentLocation
            dispatchIncrementalPointerMove(
                currentLocation: currentLocation,
                imageRect: imageRect
            )
        }
    }

    private func handleCanvasPanEnded(
        currentLocation: CGPoint,
        startLocation: CGPoint,
        translation: CGSize,
        imageRect: CGRect
    ) {
        switch canvasGestureMode {
        case .idle:
            break
        case .pendingTap:
            guard let normalizedPoint = normalizedBrowserPoint(
                for: currentLocation,
                in: imageRect,
                allowClampingOutside: false
            ) else {
                break
            }
            Task {
                await sendTap(xNorm: normalizedPoint.x, yNorm: normalizedPoint.y)
            }
        case .scrolling:
            dispatchIncrementalScroll(
                currentLocation: currentLocation,
                translation: translation,
                imageRect: imageRect
            )
        case .dragging:
            guard let normalizedPoint = normalizedBrowserPoint(
                for: currentLocation,
                in: imageRect,
                allowClampingOutside: true
            ) else {
                break
            }
            Task {
                await sendPointerUp(xNorm: normalizedPoint.x, yNorm: normalizedPoint.y)
            }
        }

        resetCanvasGestureState()
    }

    private func dispatchIncrementalScroll(
        currentLocation: CGPoint,
        translation: CGSize,
        imageRect: CGRect
    ) {
        guard let normalizedPoint = normalizedBrowserPoint(
            for: currentLocation,
            in: imageRect,
            allowClampingOutside: true
        ) else {
            return
        }
        let deltaWidth = translation.width - scrollLastTranslation.width
        let deltaHeight = translation.height - scrollLastTranslation.height
        let deltaDistance = hypot(deltaWidth, deltaHeight)
        let now = Date()
        guard deltaDistance >= scrollDispatchDistanceThreshold ||
                now.timeIntervalSince(scrollLastDispatchAt) >= pointerMoveDispatchInterval else {
            return
        }
        scrollLastTranslation = translation
        scrollLastDispatchAt = now
        Task {
            await scroll(
                deltaX: -Double(deltaWidth) * scrollWheelScale,
                deltaY: -Double(deltaHeight) * scrollWheelScale,
                xNorm: normalizedPoint.x,
                yNorm: normalizedPoint.y
            )
        }
    }

    private func beginPointerDrag(
        startLocation: CGPoint,
        currentLocation: CGPoint,
        imageRect: CGRect
    ) {
        guard let startPoint = normalizedBrowserPoint(
            for: startLocation,
            in: imageRect,
            allowClampingOutside: false
        ), let currentPoint = normalizedBrowserPoint(
            for: currentLocation,
            in: imageRect,
            allowClampingOutside: true
        ) else {
            return
        }
        dragLastMoveDispatchAt = Date()
        Task {
            await sendPointerDown(xNorm: startPoint.x, yNorm: startPoint.y)
            await sendPointerMove(xNorm: currentPoint.x, yNorm: currentPoint.y)
        }
    }

    private func scheduleLongPressDragActivationIfNeeded(imageRect: CGRect) {
        pendingLongPressDragTask?.cancel()
        pendingLongPressDragTask = nil
        guard canvasInteractionMode == .scroll else {
            return
        }
        pendingLongPressDragTask = Task { @MainActor in
            try? await Task.sleep(nanoseconds: longPressToDragDelayNanoseconds)
            guard Task.isCancelled == false,
                  canvasInteractionMode == .scroll,
                  canvasGestureMode == .pendingTap else {
                return
            }
            canvasGestureMode = .dragging
            dragLastMoveDispatchAt = Date.distantPast
            beginPointerDrag(
                startLocation: canvasGestureStartLocation,
                currentLocation: canvasGestureCurrentLocation,
                imageRect: imageRect
            )
            pendingLongPressDragTask = nil
        }
    }

    private func dispatchIncrementalPointerMove(
        currentLocation: CGPoint,
        imageRect: CGRect
    ) {
        guard let currentPoint = normalizedBrowserPoint(
            for: currentLocation,
            in: imageRect,
            allowClampingOutside: true
        ) else {
            return
        }
        let now = Date()
        guard now.timeIntervalSince(dragLastMoveDispatchAt) >= pointerMoveDispatchInterval else {
            return
        }
        dragLastMoveDispatchAt = now
        Task {
            await sendPointerMove(xNorm: currentPoint.x, yNorm: currentPoint.y)
        }
    }

    private var snapshotPollIntervalNanoseconds: UInt64 {
        let isInteractive = Date() < interactionBoostUntil || sessionSummary?.loading == true
        let isStreaming = sessionSummary?.isStreaming == true
        if isStreaming {
            return isInteractive ? 120_000_000 : 220_000_000
        }
        return isInteractive ? 250_000_000 : 700_000_000
    }

    private func advanceInteractionBoostWindow() {
        interactionBoostUntil = Date().addingTimeInterval(2.0)
    }

    private func refreshSnapshot(sessionID: String) async {
        do {
            let snapshot = try await model.fetchBrowserSessionSnapshot(sessionID: sessionID)
            guard sessionSummary?.sessionID == sessionID else {
                return
            }
            sessionSnapshot = snapshot
            if streamedFrameImageData == nil || shouldUseSnapshotRefresh {
                streamedFrameImageData = snapshot.imageData
            }
            sessionSummary = snapshot.summary
            syncKeyboardFocus(with: snapshot.summary)
            navigationURL = snapshot.summary.url
            errorMessage = nil
        } catch {
            guard sessionSummary?.sessionID == sessionID else {
                return
            }
            errorMessage = (error as NSError).localizedDescription
        }
    }

    private func refreshNextFrame(sessionID: String, afterRevision: Int) async {
        do {
            let snapshot = try await model.waitForNextBrowserSessionSnapshot(
                sessionID: sessionID,
                afterRevision: afterRevision
            )
            guard sessionSummary?.sessionID == sessionID else {
                return
            }
            sessionSnapshot = snapshot
            streamedFrameImageData = snapshot.imageData
            sessionSummary = snapshot.summary
            syncKeyboardFocus(with: snapshot.summary)
            navigationURL = snapshot.summary.url
            errorMessage = nil
        } catch {
            guard sessionSummary?.sessionID == sessionID else {
                return
            }
            errorMessage = (error as NSError).localizedDescription
            try? await Task.sleep(nanoseconds: 300_000_000)
        }
    }

    private var displayedBrowserFrameImage: UIImage? {
        if let streamedFrameImageData,
           let image = UIImage(data: streamedFrameImageData) {
            return image
        }
        return sessionSnapshot.flatMap { UIImage(data: $0.imageData) }
    }

    private var shouldUseSnapshotRefresh: Bool {
        sessionSummary?.isStreaming != true || frameStreamClient == nil
    }

    private func ensureFrameStreamStarted(sessionID: String) async {
        guard sessionSummary?.isStreaming == true else {
            if frameStreamClient != nil {
                await closeFrameStream()
            }
            return
        }
        guard frameStreamClient == nil else {
            return
        }
        do {
            let connection = try await model.browserSessionStreamConnection(sessionID: sessionID)
            let client = BrowserSessionFrameStreamClient(
                webSocketURL: connection.url,
                headers: connection.headers,
                onFrame: { data in
                    await MainActor.run {
                        guard self.sessionSummary?.sessionID == sessionID else {
                            return
                        }
                        self.streamedFrameImageData = data
                    }
                },
                onStatus: { payload in
                    do {
                        let summary = try await self.model.decodeBrowserSessionSummary(from: payload)
                        await MainActor.run {
                            guard self.sessionSummary?.sessionID == sessionID else {
                                return
                            }
                            self.sessionSummary = summary
                            self.navigationURL = summary.url
                            self.syncKeyboardFocus(with: summary)
                            self.errorMessage = nil
                        }
                    } catch {
                        NativeHostDebugLog.write("browser-frame-stream status-parse-error=\((error as NSError).localizedDescription)")
                    }
                },
                onDisconnect: { error in
                    Task { @MainActor in
                        guard self.sessionSummary?.sessionID == sessionID else {
                            return
                        }
                        self.frameStreamClient = nil
                        if let error {
                            NativeHostDebugLog.write("browser-frame-stream disconnected=\((error as NSError).localizedDescription)")
                        } else {
                            NativeHostDebugLog.write("browser-frame-stream disconnected")
                        }
                    }
                }
            )
            frameStreamClient = client
            await client.connect()
            errorMessage = nil
        } catch {
            frameStreamClient = nil
            NativeHostDebugLog.write("browser-frame-stream start-failed=\((error as NSError).localizedDescription)")
        }
    }

    private func closeFrameStream() async {
        let client = frameStreamClient
        frameStreamClient = nil
        if let client {
            await client.close()
        }
    }

    private func syncKeyboardFocus(
        with summary: NativeHostBrowserSessionSummary,
        forceRemoteSeed: Bool = false
    ) {
        let previouslyFocused = keyboardBridgeFocused
        keyboardBridgeFocused = summary.textInputActive

        guard summary.textInputActive else {
            pendingEditableStateSyncTask?.cancel()
            pendingEditableStateSyncTask = nil
            if previouslyFocused {
                keyboardBridgeSeedText = ""
                keyboardBridgeSeedSelection = NSRange(location: 0, length: 0)
                keyboardMirroredText = ""
                keyboardMirroredSelection = NSRange(location: 0, length: 0)
                keyboardBridgeSeedVersion += 1
            }
            return
        }

        let remoteSelection = browserSelectionRange(from: summary)
        let shouldSeed = forceRemoteSeed
            || previouslyFocused == false
            || shouldForceKeyboardRemoteSeed(
                with: summary,
                allowedPendingOperationCount: 0,
                remoteSelection: remoteSelection
            )
        guard shouldSeed else {
            return
        }

        let text = summary.editableText
        keyboardBridgeSeedText = text
        keyboardBridgeSeedSelection = remoteSelection
        updateKeyboardMirror(text: text, selectedRange: remoteSelection)
        keyboardBridgeSeedVersion += 1
    }

    private func handleKeyboardCommittedTextState(text: String, selectedRange: NSRange) {
        guard let sessionID = sessionSummary?.sessionID else {
            return
        }
        updateKeyboardMirror(text: text, selectedRange: selectedRange)
        enqueueKeyboardOperation(for: sessionID) {
            do {
                let nextSummary = try await model.sendBrowserSessionTextState(
                    sessionID: sessionID,
                    text: text,
                    selectionStart: selectedRange.location,
                    selectionEnd: selectedRange.location + selectedRange.length
                )
                guard self.sessionSummary?.sessionID == sessionID else {
                    return
                }
                let shouldForceSeed = shouldForceKeyboardRemoteSeed(
                    with: nextSummary,
                    allowedPendingOperationCount: 1
                )
                self.sessionSummary = nextSummary
                syncKeyboardFocus(with: nextSummary, forceRemoteSeed: shouldForceSeed)
                advanceInteractionBoostWindow()
                errorMessage = nil
            } catch {
                guard self.sessionSummary?.sessionID == sessionID else {
                    return
                }
                errorMessage = (error as NSError).localizedDescription
            }
        }
    }

    private func handleKeyboardEnter() {
        guard let sessionID = sessionSummary?.sessionID else {
            return
        }
        enqueueKeyboardOperation(for: sessionID) {
            await sendKey("Enter")
        }
    }

    private func enqueueKeyboardOperation(
        for sessionID: String,
        operation: @escaping @MainActor () async -> Void
    ) {
        keyboardQueuedOperationCount += 1
        let previousTask = keyboardOperationChain
        let nextTask = Task { @MainActor in
            defer {
                keyboardQueuedOperationCount = max(keyboardQueuedOperationCount - 1, 0)
            }
            _ = await previousTask?.result
            guard Task.isCancelled == false,
                  self.sessionSummary?.sessionID == sessionID else {
                return
            }
            await operation()
        }
        keyboardOperationChain = nextTask
    }

    private func resetKeyboardOperationChain() {
        keyboardOperationChain?.cancel()
        keyboardOperationChain = nil
        keyboardQueuedOperationCount = 0
    }

    private func browserSelectionRange(from summary: NativeHostBrowserSessionSummary) -> NSRange {
        let length = (summary.editableText as NSString).length
        let location = min(max(summary.selectionStart, 0), length)
        let maxSelectionLength = max(length - location, 0)
        let selectionLength = min(max(summary.selectionEnd - summary.selectionStart, 0), maxSelectionLength)
        return NSRange(location: location, length: selectionLength)
    }

    private func updateKeyboardMirror(text: String, selectedRange: NSRange) {
        keyboardMirroredText = text
        keyboardMirroredSelection = selectedRange
    }

    private func shouldForceKeyboardRemoteSeed(
        with summary: NativeHostBrowserSessionSummary,
        allowedPendingOperationCount: Int,
        remoteSelection: NSRange? = nil
    ) -> Bool {
        guard keyboardBridgeFocused,
              keyboardQueuedOperationCount <= allowedPendingOperationCount else {
            return false
        }
        let normalizedSelection = remoteSelection ?? browserSelectionRange(from: summary)
        return summary.editableText != keyboardMirroredText || normalizedSelection != keyboardMirroredSelection
    }

    private func scheduleEditableStateSync(
        sessionID: String,
        forceRemoteSeed: Bool,
        delayMilliseconds: Int = 90
    ) {
        pendingEditableStateSyncTask?.cancel()
        pendingEditableStateSyncTask = Task {
            if delayMilliseconds > 0 {
                try? await Task.sleep(nanoseconds: UInt64(delayMilliseconds) * 1_000_000)
            }
            guard Task.isCancelled == false,
                  self.sessionSummary?.sessionID == sessionID else {
                return
            }
            do {
                let nextSummary = try await model.syncBrowserSessionEditableState(sessionID: sessionID)
                guard Task.isCancelled == false,
                      self.sessionSummary?.sessionID == sessionID else {
                    return
                }
                self.sessionSummary = nextSummary
                syncKeyboardFocus(
                    with: nextSummary,
                    forceRemoteSeed: forceRemoteSeed || shouldForceKeyboardRemoteSeed(
                        with: nextSummary,
                        allowedPendingOperationCount: 0
                    )
                )
            } catch {
                guard Task.isCancelled == false else {
                    return
                }
                NativeHostDebugLog.write("browser-editable-sync failed=\((error as NSError).localizedDescription)")
            }
        }
    }
}

@MainActor
private final class DesktopWebRTCPlayerController: NSObject, ObservableObject, NativeHostWebBridgeDelegate {
    struct StartRequest: Equatable {
        let sessionID: String
        let peerID: String
        let offerSdpBase64: String
    }

    @Published private(set) var connectionState = "idle"
    @Published private(set) var lastError: String?

    var onAnswer: ((String, String) -> Void)?
    var onStateChange: ((String, String?) -> Void)?

    private let webBridge = NativeHostWebBridge()
    private var htmlLoaded = false
    private var playerReady = false
    private var pendingStart: StartRequest?

    override init() {
        super.init()
        webBridge.delegate = self
    }

    func makeWebView() -> WKWebView {
        let webView = webBridge.makeWebView()
        webView.scrollView.isScrollEnabled = false
        webView.scrollView.bounces = false
        webView.backgroundColor = .black
        webView.scrollView.backgroundColor = .black
        return webView
    }

    func prepareIfNeeded() {
        guard !htmlLoaded else {
            return
        }
        htmlLoaded = true
        webBridge.loadHTMLString(Self.playerHTML, baseURL: nil)
    }

    func start(sessionID: String, offer: NativeHostDesktopWebRTCOffer) {
        prepareIfNeeded()
        let offerSdpBase64 = Data(offer.sdp.utf8).base64EncodedString()
        pendingStart = StartRequest(sessionID: sessionID, peerID: offer.peerID, offerSdpBase64: offerSdpBase64)
        lastError = nil
        connectionState = "connecting"
        dispatchPendingStartIfPossible()
    }

    func stop() {
        pendingStart = nil
        connectionState = "idle"
        lastError = nil
        webBridge.evaluateJavaScript("window.CodexDesktopWebRTC && window.CodexDesktopWebRTC.stop();")
    }

    private func dispatchPendingStartIfPossible() {
        guard playerReady, let pendingStart else {
            return
        }
        let payload: [String: Any] = [
            "sessionId": pendingStart.sessionID,
            "peerId": pendingStart.peerID,
            "offerSdpBase64": pendingStart.offerSdpBase64,
        ]
        self.pendingStart = nil
        webBridge.evaluateJavaScript(
            "window.CodexDesktopWebRTC && window.CodexDesktopWebRTC.start(\(jsonEncodedString(payload)));"
        )
    }

    func webBridgeDidReceive(message: String) {
        guard let payload = parseJSONValue(from: message) as? JSONDictionary,
              let type = (payload["type"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty else {
            return
        }

        switch type {
        case "desktop-webrtc-ready":
            playerReady = true
            dispatchPendingStartIfPossible()
        case "desktop-webrtc-answer":
            guard let peerID = (payload["peerId"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty,
                  let sdp = decodedSDP(from: payload) else {
                return
            }
            onAnswer?(peerID, sdp)
        case "desktop-webrtc-state":
            let state = (payload["state"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty ?? "unknown"
            connectionState = state
            if let message = (payload["message"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty {
                lastError = message
            }
            onStateChange?(state, lastError)
        case "desktop-webrtc-error":
            let message = (payload["message"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty ??
                "Desktop WebRTC failed."
            lastError = message
            connectionState = "failed"
            onStateChange?("failed", message)
        default:
            return
        }
    }

    func webBridgeDidStartNavigation(url: URL?) {}

    func webBridgeDidFinishNavigation(url: URL?) {}

    func webBridgeDidFailNavigation(message: String) {
        let trimmed = message.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            return
        }
        lastError = trimmed
        connectionState = "failed"
        onStateChange?("failed", trimmed)
    }

    func webBridgeCurrentBaseURL() -> URL? { nil }

    func webBridgeOpenExternalURL(_ url: URL) {}

    func webBridgePickOpenPanelFiles(title: String, allowsMultipleSelection: Bool) async throws -> [URL] { [] }

    private static let playerHTML =
        """
        <!doctype html>
        <html>
          <head>
            <meta charset="utf-8">
            <meta name="viewport" content="width=device-width,initial-scale=1,maximum-scale=1,user-scalable=no">
            <style>
              html, body {
                margin: 0;
                padding: 0;
                width: 100%;
                height: 100%;
                background: #000;
                overflow: hidden;
              }
              video {
                width: 100%;
                height: 100%;
                object-fit: contain;
                background: #000;
              }
            </style>
          </head>
          <body>
            <video id="desktop-video" autoplay playsinline muted></video>
            <script>
              (function () {
                var video = document.getElementById("desktop-video");
                var currentPeer = null;
                var currentPeerId = null;

                function post(payload) {
                  try {
                    window.CodexMobileNativeBridge.postMessage(JSON.stringify(payload));
                  } catch (error) {}
                }

                function decodeBase64Text(value) {
                  if (!value) {
                    return "";
                  }
                  try {
                    return atob(value);
                  } catch (error) {
                    return "";
                  }
                }

                function encodeBase64Text(value) {
                  try {
                    return btoa(value || "");
                  } catch (error) {
                    return "";
                  }
                }

                function waitForIceGatheringComplete(peer) {
                  if (peer.iceGatheringState === "complete") {
                    return Promise.resolve();
                  }
                  return new Promise(function (resolve) {
                    function handleChange() {
                      if (peer.iceGatheringState === "complete") {
                        peer.removeEventListener("icegatheringstatechange", handleChange);
                        resolve();
                      }
                    }
                    peer.addEventListener("icegatheringstatechange", handleChange);
                  });
                }

                async function start(config) {
                  try {
                    stop();
                    if (!window.RTCPeerConnection) {
                      throw new Error("RTCPeerConnection is unavailable in this WebView.");
                    }

                    currentPeerId = config.peerId || "";
                    currentPeer = new RTCPeerConnection({
                      bundlePolicy: "max-bundle",
                      rtcpMuxPolicy: "require"
                    });

                    currentPeer.addEventListener("track", function (event) {
                      var stream = (event.streams && event.streams[0]) || new MediaStream([event.track]);
                      video.srcObject = stream;
                      var playPromise = video.play && video.play();
                      if (playPromise && playPromise.catch) {
                        playPromise.catch(function () {});
                      }
                    });

                    currentPeer.addEventListener("connectionstatechange", function () {
                      post({
                        type: "desktop-webrtc-state",
                        state: currentPeer ? currentPeer.connectionState : "closed"
                      });
                    });

                    await currentPeer.setRemoteDescription({
                      type: "offer",
                      sdp: decodeBase64Text(config.offerSdpBase64)
                    });
                    var answer = await currentPeer.createAnswer();
                    await currentPeer.setLocalDescription(answer);
                    await waitForIceGatheringComplete(currentPeer);

                    if (!currentPeer || !currentPeer.localDescription || !currentPeer.localDescription.sdp) {
                      throw new Error("WebRTC answer was not generated.");
                    }

                    post({
                      type: "desktop-webrtc-answer",
                      peerId: currentPeerId,
                      sdpBase64: encodeBase64Text(currentPeer.localDescription.sdp)
                    });
                  } catch (error) {
                    post({
                      type: "desktop-webrtc-error",
                      message: error && error.message ? error.message : String(error)
                    });
                  }
                }

                function stop() {
                  if (currentPeer) {
                    try {
                      currentPeer.close();
                    } catch (error) {}
                    currentPeer = null;
                  }
                  currentPeerId = null;
                  if (video) {
                    video.srcObject = null;
                  }
                }

                window.CodexDesktopWebRTC = {
                  start: start,
                  stop: stop
                };

                post({ type: "desktop-webrtc-ready" });
              })();
            </script>
          </body>
        </html>
        """

    private func decodedSDP(from payload: JSONDictionary) -> String? {
        if let base64 = (payload["sdpBase64"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines),
           let data = Data(base64Encoded: base64),
           let sdp = String(data: data, encoding: .utf8),
           sdp.isEmpty == false {
            return sdp
        }
        if let sdp = payload["sdp"] as? String, sdp.isEmpty == false {
            return sdp
        }
        return nil
    }
}

private struct DesktopWebRTCPlayerView: UIViewRepresentable {
    @ObservedObject var controller: DesktopWebRTCPlayerController

    func makeUIView(context: Context) -> WKWebView {
        controller.prepareIfNeeded()
        return controller.makeWebView()
    }

    func updateUIView(_ uiView: WKWebView, context: Context) {
        controller.prepareIfNeeded()
    }
}

private struct DesktopModifierOption: Identifiable {
    let id: String
    let label: String
}

private struct DesktopKeyAction: Identifiable {
    let id: String
    let label: String
    let key: String
}

private struct DesktopShortcutAction: Identifiable {
    let id: String
    let label: String
    let key: String
    let modifiers: [String]
}

private let knownDesktopComposerPlaceholderLines: Set<String> = [
    "Ask for follow-up changes",
]

private struct DesktopSessionSheet: View {
    @Environment(\.dismiss) private var dismiss
    @ObservedObject var model: NativeHostModel

    @State private var sessionSummary: NativeHostDesktopSessionSummary?
    @State private var textInput = ""
    @State private var errorMessage: String?
    @State private var isWorking = false
    @State private var canvasGestureMode: BrowserCanvasGestureMode = .idle
    @State private var canvasGestureStartLocation: CGPoint = .zero
    @State private var canvasGestureCurrentLocation: CGPoint = .zero
    @State private var canvasZoomScale: CGFloat = 1
    @State private var dragLastMoveDispatchAt = Date.distantPast
    @State private var desktopWebRTCAttempted = false
    @State private var desktopWebRTCFailed = false
    @State private var desktopWebRTCConnected = false
    @State private var desktopPendingModifiers = Set<String>()
    @State private var keyboardBridgeFocused = false
    @State private var keyboardBridgeSeedText = ""
    @State private var keyboardBridgeSeedSelection = NSRange(location: 0, length: 0)
    @State private var keyboardBridgeSeedVersion = 0
    @State private var keyboardOperationChain: Task<Void, Never>?
    @State private var keyboardQueuedOperationCount = 0
    @State private var keyboardMirroredText = ""
    @State private var keyboardMirroredSelection = NSRange(location: 0, length: 0)
    @State private var keyboardRemoteSeedSuppressedUntil = Date.distantPast
    @State private var pendingDesktopFocusSyncTask: Task<Void, Never>?
    @StateObject private var desktopWebRTCPlayer = DesktopWebRTCPlayerController()

    var body: some View {
        NavigationStack {
            Group {
                if let sessionSummary {
                    desktopLiveView(summary: sessionSummary)
                } else {
                    desktopConfigurationView
                }
            }
            .navigationTitle(sessionSummary?.title ?? "Desktop")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Done") {
                        closeSheet()
                    }
                }
                if sessionSummary != nil {
                    ToolbarItem(placement: .confirmationAction) {
                        Button("End") {
                            Task {
                                await endSession()
                            }
                        }
                    }
                }
            }
        }
        .overlay(alignment: .topLeading) {
            BrowserKeyboardInputBridge(
                isFocused: $keyboardBridgeFocused,
                seedText: keyboardBridgeSeedText,
                seedSelection: keyboardBridgeSeedSelection,
                seedVersion: keyboardBridgeSeedVersion,
                onTextState: { text, selectedRange in
                    handleDesktopKeyboardCommittedTextState(text: text, selectedRange: selectedRange)
                },
                onEnter: {
                    handleDesktopKeyboardEnter()
                }
            )
            .frame(width: 1, height: 1)
            .opacity(0.01)
            .allowsHitTesting(false)
        }
        .presentationDetents([.large])
        .onAppear {
            configureDesktopWebRTCPlayer()
        }
        .task(id: sessionSummary?.sessionID) {
            guard let sessionID = sessionSummary?.sessionID else {
                return
            }
            await pollDesktopStatus(sessionID: sessionID)
        }
        .onDisappear {
            desktopWebRTCAttempted = false
            desktopWebRTCFailed = false
            desktopWebRTCConnected = false
            keyboardBridgeFocused = false
            pendingDesktopFocusSyncTask?.cancel()
            pendingDesktopFocusSyncTask = nil
            resetDesktopKeyboardOperationChain()
            resetDesktopKeyboardBuffer()
            desktopWebRTCPlayer.stop()
            if let sessionID = sessionSummary?.sessionID {
                Task {
                    await model.stopDesktopSession(sessionID: sessionID)
                }
            }
        }
    }

    private var desktopConfigurationView: some View {
        Form {
            Section("Desktop Fallback") {
                Text("Open a full desktop fallback session from this iPad. This mode is intended as the escape hatch when structured mobile tools are not enough.")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
                Button {
                    Task {
                        await startSession()
                    }
                } label: {
                    HStack {
                        if isWorking {
                            ProgressView()
                        }
                        Text("Start Desktop")
                    }
                }
                .disabled(isWorking)
            }
            if let errorMessage {
                Section("Error") {
                    Text(errorMessage)
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                }
            }
        }
    }

    private func desktopLiveView(summary: NativeHostDesktopSessionSummary) -> some View {
        ScrollView(showsIndicators: false) {
            VStack(alignment: .leading, spacing: 18) {
                desktopToolbar(summary: summary)
                desktopCanvas(summary: summary)
                desktopInputBar
            }
            .padding(16)
        }
        .background(Color(.systemGroupedBackground))
    }

    private func desktopToolbar(summary: NativeHostDesktopSessionSummary) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(spacing: 10) {
                StatusPill(
                    label: desktopWebRTCConnected ? "WebRTC Live" : (desktopWebRTCAttempted ? "WebRTC Connecting" : (summary.videoReady ? "Offer Ready" : "Preparing")),
                    tone: desktopWebRTCConnected ? .success : .active
                )
                if summary.width > 0, summary.height > 0 {
                    Text("\(summary.width) × \(summary.height)")
                        .font(.caption.weight(.medium))
                        .foregroundStyle(.secondary)
                }
                Text(String(format: "@ %.1fx", summary.scale))
                    .font(.caption.weight(.medium))
                    .foregroundStyle(.secondary)
                if let videoCodec = summary.videoCodec {
                    Text(videoCodec.uppercased())
                        .font(.caption.weight(.medium))
                        .foregroundStyle(.secondary)
                }
            }
            if let webRTCError = desktopWebRTCPlayer.lastError, !desktopWebRTCConnected {
                Text(webRTCError)
                    .font(.footnote)
                    .foregroundStyle(.secondary)
            } else if let videoError = summary.videoError {
                Text(videoError)
                    .font(.footnote)
                    .foregroundStyle(.secondary)
            } else if let lastError = summary.lastError {
                Text(lastError)
                    .font(.footnote)
                    .foregroundStyle(.secondary)
            } else {
                Text("This desktop fallback now runs on WebRTC only. Tap to click, drag to move the pointer, and use the text panel below for keyboard input.")
                    .font(.footnote)
                    .foregroundStyle(.secondary)
            }
        }
    }

    private func desktopCanvas(summary: NativeHostDesktopSessionSummary) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            if let errorMessage {
                Text(errorMessage)
                    .font(.footnote)
                    .foregroundStyle(.secondary)
            }

            GeometryReader { geometry in
                let contentSize = currentDesktopCanvasContentSize(summary: summary, containerSize: geometry.size)
                ZStack {
                    RoundedRectangle(cornerRadius: 28, style: .continuous)
                        .fill(Color.black.opacity(0.9))
                    if contentSize.width > 0, contentSize.height > 0 {
                        let imageRect = fittedImageRect(for: contentSize, in: geometry.size)
                        let displayedImageRect = scaledImageRect(from: imageRect)
                        if shouldDisplayDesktopWebRTC(summary: summary) {
                            DesktopWebRTCPlayerView(controller: desktopWebRTCPlayer)
                                .frame(width: imageRect.width, height: imageRect.height)
                                .scaleEffect(canvasZoomScale, anchor: .center)
                                .position(x: imageRect.midX, y: imageRect.midY)
                        } else {
                            VStack(spacing: 12) {
                                if isWorking || (summary.videoReady && desktopWebRTCAttempted && !desktopWebRTCFailed) {
                                    ProgressView()
                                }
                                Text(summary.videoReady ? "Starting WebRTC desktop stream…" : "Preparing desktop stream…")
                                    .font(.body)
                                    .foregroundStyle(.white.opacity(0.85))
                            }
                        }
                        BrowserCanvasInteractionOverlay(
                            onTap: { location in
                                guard displayedImageRect.contains(location),
                                      let normalizedPoint = normalizedPoint(
                                        for: location,
                                        in: displayedImageRect,
                                        allowClampingOutside: false
                                      ) else {
                                    return
                                }
                                Task {
                                    await sendTap(xNorm: normalizedPoint.x, yNorm: normalizedPoint.y)
                                }
                            },
                            onSinglePanBegan: { startLocation in
                                guard displayedImageRect.contains(startLocation) else {
                                    return
                                }
                                handlePanBegan(startLocation, imageRect: displayedImageRect)
                            },
                            onSinglePanChanged: { currentLocation, startLocation, translation in
                                guard displayedImageRect.contains(startLocation) else {
                                    return
                                }
                                handlePanChanged(
                                    currentLocation: currentLocation,
                                    startLocation: startLocation,
                                    translation: translation,
                                    imageRect: displayedImageRect
                                )
                            },
                            onSinglePanEnded: { currentLocation, startLocation, translation in
                                guard displayedImageRect.contains(startLocation) else {
                                    resetGestureState()
                                    return
                                }
                                handlePanEnded(
                                    currentLocation: currentLocation,
                                    startLocation: startLocation,
                                    translation: translation,
                                    imageRect: displayedImageRect
                                )
                            },
                            onTwoFingerPan: { _, delta in
                                Task {
                                    await sendScroll(deltaY: -Double(delta.height) * 5)
                                }
                            },
                            onPinch: { scaleDelta, state in
                                guard state == .began || state == .changed else {
                                    return
                                }
                                let nextScale = min(max(canvasZoomScale * scaleDelta, 1), 3)
                                canvasZoomScale = nextScale
                            }
                        )
                        .frame(width: geometry.size.width, height: geometry.size.height)
                    }
                }
            }
            .frame(minHeight: 300, idealHeight: 460)
            .clipShape(RoundedRectangle(cornerRadius: 28, style: .continuous))

            HStack(spacing: 10) {
                Button("Page Up") {
                    Task {
                        await sendScroll(deltaY: -600)
                    }
                }
                Button("Page Down") {
                    Task {
                        await sendScroll(deltaY: 600)
                    }
                }
                Button("Reset Zoom") {
                    canvasZoomScale = 1
                }
                .disabled(canvasZoomScale <= 1.01)
                Spacer()
                Text("Tap to click. Drag to move the pointer. Two-finger pan maps to coarse page scrolling.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
    }

    private var desktopInputBar: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("Keyboard")
                .font(.headline)

            Text(keyboardBridgeFocused
                 ? "System keyboard is attached to the desktop session."
                 : "Tap a text field on the desktop canvas, or press the button below, to raise the iPad keyboard.")
                .font(.footnote)
                .foregroundStyle(.secondary)

            HStack(spacing: 10) {
                Button(keyboardBridgeFocused ? "Hide iPad Keyboard" : "Use iPad Keyboard") {
                    if keyboardBridgeFocused {
                        keyboardBridgeFocused = false
                    } else {
                        activateDesktopKeyboardBridge(resetBuffer: true)
                    }
                }
                .buttonStyle(.borderedProminent)

                Text("Modifier toggles apply to the next key press. Shortcut buttons send real desktop combinations like Cmd-C and Cmd-S.")
                    .font(.footnote)
                    .foregroundStyle(.secondary)
            }

            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 10) {
                    ForEach(desktopModifierOptions) { option in
                        Group {
                            if desktopPendingModifiers.contains(option.id) {
                                Button(option.label) {
                                    toggleDesktopModifier(option.id)
                                }
                                .buttonStyle(.borderedProminent)
                            } else {
                                Button(option.label) {
                                    toggleDesktopModifier(option.id)
                                }
                                .buttonStyle(.bordered)
                            }
                        }
                    }
                    if desktopPendingModifiers.isEmpty == false {
                        Button("Clear") {
                            desktopPendingModifiers.removeAll()
                        }
                        .buttonStyle(.bordered)
                    }
                }
            }

            TextEditor(text: $textInput)
                .font(.body)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
                .frame(minHeight: 92)
                .padding(8)
                .background(
                    RoundedRectangle(cornerRadius: 16, style: .continuous)
                        .fill(Color(.tertiarySystemBackground))
                )

            HStack(spacing: 10) {
                Button("Send Text") {
                    Task {
                        await sendText()
                    }
                }
                .disabled(textInput.isEmpty)

                Button("Space") {
                    Task {
                        await sendKey("Space")
                    }
                }
            }
            .buttonStyle(.borderedProminent)

            LazyVGrid(columns: desktopInputGridColumns, alignment: .leading, spacing: 10) {
                ForEach(desktopKeyActions) { action in
                    Button(action.label) {
                        Task {
                            await sendKey(action.key)
                        }
                    }
                    .buttonStyle(.bordered)
                    .controlSize(.small)
                }
            }

            LazyVGrid(columns: desktopInputGridColumns, alignment: .leading, spacing: 10) {
                ForEach(desktopShortcutActions) { action in
                    Button(action.label) {
                        Task {
                            await sendShortcut(action)
                        }
                    }
                    .buttonStyle(.bordered)
                    .controlSize(.small)
                }
            }
        }
    }

    private func startSession() async {
        guard isWorking == false else {
            return
        }
        isWorking = true
        defer { isWorking = false }
        do {
            let summary = try await model.startDesktopSession()
            sessionSummary = summary
            desktopWebRTCAttempted = false
            desktopWebRTCFailed = false
            desktopWebRTCConnected = false
            desktopPendingModifiers.removeAll()
            keyboardBridgeFocused = false
            pendingDesktopFocusSyncTask?.cancel()
            pendingDesktopFocusSyncTask = nil
            resetDesktopKeyboardOperationChain()
            resetDesktopKeyboardBuffer()
            desktopWebRTCPlayer.stop()
            errorMessage = nil
        } catch {
            errorMessage = (error as NSError).localizedDescription
        }
    }

    private func pollDesktopStatus(sessionID: String) async {
        while Task.isCancelled == false {
            guard sessionSummary?.sessionID == sessionID else {
                return
            }

            await refreshDesktopStatus(sessionID: sessionID)

            guard sessionSummary?.sessionID == sessionID else {
                return
            }

            if shouldAttemptDesktopWebRTC(summary: sessionSummary) {
                await ensureDesktopWebRTCStarted(sessionID: sessionID)
            }

            try? await Task.sleep(nanoseconds: desktopStatusPollIntervalNanoseconds)
        }
    }

    private var desktopStatusPollIntervalNanoseconds: UInt64 {
        desktopWebRTCConnected ? 1_000_000_000 : 250_000_000
    }

    private func refreshDesktopStatus(sessionID: String) async {
        do {
            let summary = try await model.fetchDesktopSessionStatus(sessionID: sessionID)
            guard sessionSummary?.sessionID == sessionID else {
                return
            }
            sessionSummary = summary
            syncDesktopKeyboardFocus(with: summary)
            if desktopWebRTCPlayer.lastError == nil {
                errorMessage = nil
            }
        } catch {
            guard sessionSummary?.sessionID == sessionID else {
                return
            }
            errorMessage = (error as NSError).localizedDescription
        }
    }

    private func configureDesktopWebRTCPlayer() {
        desktopWebRTCPlayer.onAnswer = { peerID, sdp in
            guard let sessionID = self.sessionSummary?.sessionID else {
                return
            }
            Task {
                do {
                    let summary = try await self.model.submitDesktopSessionWebRTCAnswer(
                        sessionID: sessionID,
                        peerID: peerID,
                        sdp: sdp
                    )
                    await MainActor.run {
                        guard self.sessionSummary?.sessionID == sessionID else {
                            return
                        }
                        self.sessionSummary = summary
                        self.errorMessage = nil
                    }
                } catch {
                    await MainActor.run {
                        guard self.sessionSummary?.sessionID == sessionID else {
                            return
                        }
                        self.desktopWebRTCFailed = true
                        self.desktopWebRTCConnected = false
                        self.errorMessage = (error as NSError).localizedDescription
                    }
                }
            }
        }
        desktopWebRTCPlayer.onStateChange = { state, message in
            switch state {
            case "connected":
                self.desktopWebRTCConnected = true
                self.errorMessage = nil
            case "failed":
                self.desktopWebRTCFailed = true
                self.desktopWebRTCConnected = false
                if let message, !message.isEmpty {
                    self.errorMessage = message
                }
            case "closed", "disconnected":
                self.desktopWebRTCConnected = false
            default:
                break
            }
        }
    }

    private func ensureDesktopWebRTCStarted(sessionID: String) async {
        guard desktopWebRTCAttempted == false, desktopWebRTCFailed == false else {
            return
        }
        desktopWebRTCAttempted = true
        do {
            let offer = try await model.createDesktopSessionWebRTCOffer(sessionID: sessionID)
            await MainActor.run {
                guard self.sessionSummary?.sessionID == sessionID else {
                    return
                }
                self.desktopWebRTCPlayer.start(sessionID: sessionID, offer: offer)
                self.errorMessage = nil
            }
        } catch {
            await MainActor.run {
                guard self.sessionSummary?.sessionID == sessionID else {
                    return
                }
                self.desktopWebRTCFailed = true
                self.desktopWebRTCConnected = false
                self.errorMessage = (error as NSError).localizedDescription
            }
        }
    }

    private func shouldAttemptDesktopWebRTC(summary: NativeHostDesktopSessionSummary?) -> Bool {
        guard let summary else {
            return false
        }
        return summary.preferredTransport == "webrtc" && summary.videoReady
    }

    private func shouldDisplayDesktopWebRTC(summary: NativeHostDesktopSessionSummary?) -> Bool {
        guard let summary else {
            return false
        }
        return summary.preferredTransport == "webrtc" && desktopWebRTCAttempted && desktopWebRTCFailed == false
    }

    private func currentDesktopCanvasContentSize(
        summary: NativeHostDesktopSessionSummary,
        containerSize: CGSize
    ) -> CGSize {
        if summary.width > 0, summary.height > 0 {
            return CGSize(width: summary.width, height: summary.height)
        }
        return CGSize(width: max(containerSize.width, 1), height: max(containerSize.height, 1))
    }

    private func fittedImageRect(for imageSize: CGSize, in containerSize: CGSize) -> CGRect {
        guard imageSize.width > 0, imageSize.height > 0,
              containerSize.width > 0, containerSize.height > 0 else {
            return CGRect(origin: .zero, size: containerSize)
        }
        let scale = min(containerSize.width / imageSize.width, containerSize.height / imageSize.height)
        let width = imageSize.width * scale
        let height = imageSize.height * scale
        let originX = (containerSize.width - width) / 2
        let originY = (containerSize.height - height) / 2
        return CGRect(x: originX, y: originY, width: width, height: height)
    }

    private func scaledImageRect(from imageRect: CGRect) -> CGRect {
        guard imageRect.width > 0, imageRect.height > 0 else {
            return imageRect
        }
        let width = imageRect.width * canvasZoomScale
        let height = imageRect.height * canvasZoomScale
        return CGRect(
            x: imageRect.midX - (width / 2),
            y: imageRect.midY - (height / 2),
            width: width,
            height: height
        )
    }

    private func normalizedPoint(
        for location: CGPoint,
        in imageRect: CGRect,
        allowClampingOutside: Bool
    ) -> CGPoint? {
        guard imageRect.width > 0, imageRect.height > 0 else {
            return nil
        }
        if allowClampingOutside == false, imageRect.contains(location) == false {
            return nil
        }
        let clampedX = min(max(location.x, imageRect.minX), imageRect.maxX)
        let clampedY = min(max(location.y, imageRect.minY), imageRect.maxY)
        let normalizedX = (clampedX - imageRect.minX) / max(imageRect.width, 1)
        let normalizedY = (clampedY - imageRect.minY) / max(imageRect.height, 1)
        return CGPoint(x: normalizedX, y: normalizedY)
    }

    private var pointerMoveDispatchInterval: TimeInterval { 1.0 / 30.0 }

    private var dragTapThreshold: CGFloat {
        10
    }

    private func handlePanBegan(_ startLocation: CGPoint, imageRect: CGRect) {
        canvasGestureStartLocation = startLocation
        canvasGestureCurrentLocation = startLocation
        canvasGestureMode = .pendingTap
        dragLastMoveDispatchAt = Date.distantPast
    }

    private func handlePanChanged(
        currentLocation: CGPoint,
        startLocation: CGPoint,
        translation: CGSize,
        imageRect: CGRect
    ) {
        let translationDistance = hypot(translation.width, translation.height)
        canvasGestureCurrentLocation = currentLocation
        switch canvasGestureMode {
        case .idle:
            handlePanBegan(startLocation, imageRect: imageRect)
        case .pendingTap:
            guard translationDistance >= dragTapThreshold else {
                return
            }
            canvasGestureMode = .dragging
            dragLastMoveDispatchAt = Date.distantPast
            beginPointerDrag(startLocation: startLocation, currentLocation: currentLocation, imageRect: imageRect)
        case .dragging:
            dispatchIncrementalPointerMove(currentLocation: currentLocation, imageRect: imageRect)
        case .scrolling:
            break
        }
    }

    private func handlePanEnded(
        currentLocation: CGPoint,
        startLocation: CGPoint,
        translation: CGSize,
        imageRect: CGRect
    ) {
        switch canvasGestureMode {
        case .idle:
            break
        case .pendingTap:
            guard let normalizedPoint = normalizedPoint(
                for: currentLocation,
                in: imageRect,
                allowClampingOutside: false
            ) else {
                break
            }
            Task {
                await sendTap(xNorm: normalizedPoint.x, yNorm: normalizedPoint.y)
            }
        case .dragging:
            guard let normalizedPoint = normalizedPoint(
                for: currentLocation,
                in: imageRect,
                allowClampingOutside: true
            ) else {
                break
            }
            Task {
                await sendPointerUp(xNorm: normalizedPoint.x, yNorm: normalizedPoint.y)
            }
        case .scrolling:
            break
        }
        resetGestureState()
    }

    private func beginPointerDrag(
        startLocation: CGPoint,
        currentLocation: CGPoint,
        imageRect: CGRect
    ) {
        guard let startPoint = normalizedPoint(
            for: startLocation,
            in: imageRect,
            allowClampingOutside: false
        ), let currentPoint = normalizedPoint(
            for: currentLocation,
            in: imageRect,
            allowClampingOutside: true
        ) else {
            return
        }
        dragLastMoveDispatchAt = Date()
        Task {
            await sendPointerDown(xNorm: startPoint.x, yNorm: startPoint.y)
            await sendPointerMove(xNorm: currentPoint.x, yNorm: currentPoint.y)
        }
    }

    private func dispatchIncrementalPointerMove(
        currentLocation: CGPoint,
        imageRect: CGRect
    ) {
        guard let currentPoint = normalizedPoint(
            for: currentLocation,
            in: imageRect,
            allowClampingOutside: true
        ) else {
            return
        }
        let now = Date()
        guard now.timeIntervalSince(dragLastMoveDispatchAt) >= pointerMoveDispatchInterval else {
            return
        }
        dragLastMoveDispatchAt = now
        Task {
            await sendPointerMove(xNorm: currentPoint.x, yNorm: currentPoint.y)
        }
    }

    private func resetGestureState() {
        canvasGestureMode = .idle
        canvasGestureStartLocation = .zero
        canvasGestureCurrentLocation = .zero
        dragLastMoveDispatchAt = Date.distantPast
    }

    private func sendTap(xNorm: Double, yNorm: Double) async {
        guard let sessionSummary else {
            return
        }
        do {
            let nextSummary = try await model.sendDesktopSessionTap(sessionID: sessionSummary.sessionID, xNorm: xNorm, yNorm: yNorm)
            self.sessionSummary = nextSummary
            prepareDesktopKeyboardFocusRefresh()
            scheduleDesktopFocusSync(sessionID: nextSummary.sessionID, delayMilliseconds: 120)
            errorMessage = nil
        } catch {
            errorMessage = (error as NSError).localizedDescription
        }
    }

    private func sendPointerDown(xNorm: Double, yNorm: Double) async {
        guard let sessionSummary else {
            return
        }
        do {
            let nextSummary = try await model.sendDesktopSessionPointerDown(sessionID: sessionSummary.sessionID, xNorm: xNorm, yNorm: yNorm)
            self.sessionSummary = nextSummary
            syncDesktopKeyboardFocus(with: nextSummary)
            errorMessage = nil
        } catch {
            errorMessage = (error as NSError).localizedDescription
        }
    }

    private func sendPointerMove(xNorm: Double, yNorm: Double) async {
        guard let sessionSummary else {
            return
        }
        do {
            let nextSummary = try await model.sendDesktopSessionPointerMove(sessionID: sessionSummary.sessionID, xNorm: xNorm, yNorm: yNorm)
            self.sessionSummary = nextSummary
            errorMessage = nil
        } catch {
            errorMessage = (error as NSError).localizedDescription
        }
    }

    private func sendPointerUp(xNorm: Double, yNorm: Double) async {
        guard let sessionSummary else {
            return
        }
        do {
            let nextSummary = try await model.sendDesktopSessionPointerUp(sessionID: sessionSummary.sessionID, xNorm: xNorm, yNorm: yNorm)
            self.sessionSummary = nextSummary
            prepareDesktopKeyboardFocusRefresh()
            scheduleDesktopFocusSync(sessionID: nextSummary.sessionID, delayMilliseconds: 120)
            errorMessage = nil
        } catch {
            errorMessage = (error as NSError).localizedDescription
        }
    }

    private func sendText() async {
        guard let sessionSummary else {
            return
        }
        let payload = textInput
        guard payload.isEmpty == false else {
            return
        }
        do {
            let nextSummary = try await model.sendDesktopSessionText(sessionID: sessionSummary.sessionID, text: payload)
            self.sessionSummary = nextSummary
            syncDesktopKeyboardFocus(with: nextSummary)
            textInput = ""
            errorMessage = nil
        } catch {
            errorMessage = (error as NSError).localizedDescription
        }
    }

    private func sendKey(_ key: String) async {
        guard let sessionSummary else {
            return
        }
        let modifiers = activeDesktopModifiers()
        let shouldClearModifiers = modifiers.isEmpty == false
        do {
            let nextSummary = try await model.sendDesktopSessionKey(
                sessionID: sessionSummary.sessionID,
                key: key,
                modifiers: modifiers
            )
            self.sessionSummary = nextSummary
            syncDesktopKeyboardFocus(
                with: nextSummary,
                forceRemoteSeed: shouldForceDesktopKeyboardRemoteSeed(
                    with: nextSummary,
                    allowedPendingOperationCount: 0
                )
            )
            if shouldClearModifiers {
                desktopPendingModifiers.removeAll()
            }
            errorMessage = nil
        } catch {
            errorMessage = (error as NSError).localizedDescription
        }
    }

    private func sendShortcut(_ shortcut: DesktopShortcutAction) async {
        guard let sessionSummary else {
            return
        }
        do {
            let nextSummary = try await model.sendDesktopSessionKey(
                sessionID: sessionSummary.sessionID,
                key: shortcut.key,
                modifiers: shortcut.modifiers
            )
            self.sessionSummary = nextSummary
            syncDesktopKeyboardFocus(with: nextSummary)
            errorMessage = nil
        } catch {
            errorMessage = (error as NSError).localizedDescription
        }
    }

    private func sendScroll(deltaY: Double) async {
        guard let sessionSummary else {
            return
        }
        do {
            let nextSummary = try await model.sendDesktopSessionScroll(sessionID: sessionSummary.sessionID, deltaY: deltaY)
            self.sessionSummary = nextSummary
            syncDesktopKeyboardFocus(with: nextSummary)
            errorMessage = nil
        } catch {
            errorMessage = (error as NSError).localizedDescription
        }
    }

    private func endSession() async {
        guard let sessionID = sessionSummary?.sessionID else {
            return
        }
        desktopWebRTCPlayer.stop()
        desktopWebRTCAttempted = false
        desktopWebRTCFailed = false
        desktopWebRTCConnected = false
        desktopPendingModifiers.removeAll()
        keyboardBridgeFocused = false
        pendingDesktopFocusSyncTask?.cancel()
        pendingDesktopFocusSyncTask = nil
        resetDesktopKeyboardOperationChain()
        resetDesktopKeyboardBuffer()
        await model.stopDesktopSession(sessionID: sessionID)
        sessionSummary = nil
        textInput = ""
        resetGestureState()
    }

    private func closeSheet() {
        Task {
            await endSession()
            dismiss()
        }
    }

    private func toggleDesktopModifier(_ modifier: String) {
        if desktopPendingModifiers.contains(modifier) {
            desktopPendingModifiers.remove(modifier)
        } else {
            desktopPendingModifiers.insert(modifier)
        }
    }

    private func activeDesktopModifiers() -> [String] {
        desktopModifierOptions
            .map(\.id)
            .filter { desktopPendingModifiers.contains($0) }
    }

    private func activateDesktopKeyboardBridge(resetBuffer: Bool) {
        if resetBuffer {
            resetDesktopKeyboardBuffer()
        }
        keyboardBridgeFocused = true
    }

    private func resetDesktopKeyboardBuffer() {
        keyboardBridgeSeedText = ""
        keyboardBridgeSeedSelection = NSRange(location: 0, length: 0)
        keyboardBridgeSeedVersion += 1
        keyboardMirroredText = ""
        keyboardMirroredSelection = NSRange(location: 0, length: 0)
        keyboardRemoteSeedSuppressedUntil = Date.distantPast
    }

    private func resetDesktopKeyboardOperationChain() {
        keyboardOperationChain?.cancel()
        keyboardOperationChain = nil
        keyboardQueuedOperationCount = 0
    }

    private func enqueueDesktopKeyboardOperation(
        for sessionID: String,
        operation: @escaping @MainActor () async -> Void
    ) {
        keyboardQueuedOperationCount += 1
        let previousTask = keyboardOperationChain
        let nextTask = Task { @MainActor in
            defer {
                keyboardQueuedOperationCount = max(keyboardQueuedOperationCount - 1, 0)
            }
            _ = await previousTask?.result
            guard Task.isCancelled == false,
                  self.sessionSummary?.sessionID == sessionID else {
                return
            }
            await operation()
        }
        keyboardOperationChain = nextTask
    }

    private func handleDesktopKeyboardCommittedTextState(text: String, selectedRange: NSRange) {
        guard keyboardBridgeFocused, let sessionID = sessionSummary?.sessionID else {
            return
        }

        let previousText = keyboardMirroredText
        let previousSelection = keyboardMirroredSelection
        let shouldPreferIncrementalInput = sessionSummary.map { shouldUseIncrementalDesktopKeyboardInput(for: $0) } ?? false
        updateDesktopKeyboardMirror(text: text, selectedRange: selectedRange)
        keyboardRemoteSeedSuppressedUntil = Date().addingTimeInterval(0.9)
        enqueueDesktopKeyboardOperation(for: sessionID) {
            do {
                let nextSummary: NativeHostDesktopSessionSummary
                if shouldPreferIncrementalInput,
                   let incrementalSummary = try await applyIncrementalDesktopKeyboardUpdateIfPossible(
                    sessionID: sessionID,
                    previousText: previousText,
                    previousSelection: previousSelection,
                    nextText: text,
                    nextSelection: selectedRange
                   ) {
                    nextSummary = incrementalSummary
                } else {
                    nextSummary = try await model.sendDesktopSessionTextState(
                        sessionID: sessionID,
                        text: text,
                        selectionStart: selectedRange.location,
                        selectionEnd: selectedRange.location + selectedRange.length
                    )
                }
                guard self.sessionSummary?.sessionID == sessionID else {
                    return
                }
                self.sessionSummary = nextSummary
                if nextSummary.textInputActive == false {
                    syncDesktopKeyboardFocus(with: nextSummary, forceRemoteSeed: true)
                }
                errorMessage = nil
            } catch {
                errorMessage = (error as NSError).localizedDescription
            }
        }
    }

    private func shouldUseIncrementalDesktopKeyboardInput(for summary: NativeHostDesktopSessionSummary) -> Bool {
        summary.editableRole == "AXTextArea"
            && summary.editablePlaceholder.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    private func applyIncrementalDesktopKeyboardUpdateIfPossible(
        sessionID: String,
        previousText: String,
        previousSelection: NSRange,
        nextText: String,
        nextSelection: NSRange
    ) async throws -> NativeHostDesktopSessionSummary? {
        let previousLength = (previousText as NSString).length
        let nextLength = (nextText as NSString).length
        guard previousSelection.length == 0,
              nextSelection.length == 0,
              previousSelection.location == previousLength,
              nextSelection.location == nextLength else {
            return nil
        }

        let previousCharacters = Array(previousText)
        let nextCharacters = Array(nextText)
        var prefixLength = 0
        while prefixLength < previousCharacters.count,
              prefixLength < nextCharacters.count,
              previousCharacters[prefixLength] == nextCharacters[prefixLength] {
            prefixLength += 1
        }

        if prefixLength < previousCharacters.count,
           prefixLength < nextCharacters.count {
            return nil
        }

        let deletedCount = previousCharacters.count - prefixLength
        let insertedCharacters = nextCharacters.suffix(nextCharacters.count - prefixLength)
        let insertedText = String(insertedCharacters)

        var latestSummary = sessionSummary
        if deletedCount > 0 {
            for _ in 0..<deletedCount {
                latestSummary = try await model.sendDesktopSessionKey(sessionID: sessionID, key: "Backspace")
            }
        }
        if insertedText.isEmpty == false {
            latestSummary = try await model.sendDesktopSessionText(sessionID: sessionID, text: insertedText)
        }

        return latestSummary
    }

    private func handleDesktopKeyboardEnter() {
        guard keyboardBridgeFocused, let sessionID = sessionSummary?.sessionID else {
            return
        }
        enqueueDesktopKeyboardOperation(for: sessionID) {
            await sendKey("Enter")
        }
    }

    private func prepareDesktopKeyboardFocusRefresh() {
        keyboardBridgeFocused = false
        resetDesktopKeyboardBuffer()
    }

    private func syncDesktopKeyboardFocus(
        with summary: NativeHostDesktopSessionSummary,
        forceRemoteSeed: Bool = false
    ) {
        let previouslyFocused = keyboardBridgeFocused
        keyboardBridgeFocused = summary.textInputActive

        guard summary.textInputActive else {
            if previouslyFocused {
                resetDesktopKeyboardBuffer()
            }
            return
        }

        let remoteText = sanitizedDesktopEditableText(
            from: summary,
            treatMockPlaceholderOnFreshFocus: previouslyFocused == false
        )
        let remoteSelection = desktopSelectionRange(from: summary, text: remoteText)
        let shouldSeed = forceRemoteSeed
            || previouslyFocused == false
            || shouldForceDesktopKeyboardRemoteSeed(
                with: summary,
                allowedPendingOperationCount: 0,
                remoteSelection: remoteSelection,
                remoteText: remoteText
            )
        guard shouldSeed else {
            return
        }

        keyboardBridgeSeedText = remoteText
        keyboardBridgeSeedSelection = remoteSelection
        updateDesktopKeyboardMirror(text: remoteText, selectedRange: remoteSelection)
        keyboardBridgeSeedVersion += 1
    }

    private func desktopSelectionRange(from summary: NativeHostDesktopSessionSummary, text: String? = nil) -> NSRange {
        let effectiveText = text ?? sanitizedDesktopEditableText(from: summary)
        let length = (effectiveText as NSString).length
        let location = min(max(summary.selectionStart, 0), length)
        let maxSelectionLength = max(length - location, 0)
        let selectionLength = min(max(summary.selectionEnd - summary.selectionStart, 0), maxSelectionLength)
        return NSRange(location: location, length: selectionLength)
    }

    private func sanitizedDesktopEditableText(
        from summary: NativeHostDesktopSessionSummary,
        treatMockPlaceholderOnFreshFocus: Bool = false
    ) -> String {
        let normalizedText = summary.editableText.replacingOccurrences(of: "\r\n", with: "\n")
        let normalizedPlaceholder = summary.editablePlaceholder.replacingOccurrences(of: "\r\n", with: "\n")
        let lines = normalizedText.split(separator: "\n", omittingEmptySubsequences: false).map(String.init)
        if normalizedPlaceholder.isEmpty == false {
            if normalizedText == normalizedPlaceholder || normalizedText == "\n" + normalizedPlaceholder {
                return ""
            }

            if let lastLine = lines.last, lastLine == normalizedPlaceholder {
                return Array(lines.dropLast()).joined(separator: "\n")
            }
        }

        if treatMockPlaceholderOnFreshFocus,
           normalizedPlaceholder.isEmpty,
           summary.editableRole == "AXTextArea",
           summary.selectionStart == 0,
           summary.selectionEnd == 0,
           normalizedText.isEmpty == false {
            return ""
        }

        guard let placeholderLine = lines.last(where: { knownDesktopComposerPlaceholderLines.contains($0) }) else {
            return normalizedText
        }

        if normalizedText == placeholderLine || normalizedText == "\n" + placeholderLine {
            return ""
        }

        if let lastLine = lines.last, lastLine == placeholderLine {
            return Array(lines.dropLast()).joined(separator: "\n")
        }

        return normalizedText
    }

    private func updateDesktopKeyboardMirror(text: String, selectedRange: NSRange) {
        keyboardMirroredText = text
        keyboardMirroredSelection = selectedRange
    }

    private func shouldForceDesktopKeyboardRemoteSeed(
        with summary: NativeHostDesktopSessionSummary,
        allowedPendingOperationCount: Int,
        remoteSelection: NSRange? = nil,
        remoteText: String? = nil
    ) -> Bool {
        guard keyboardBridgeFocused,
              keyboardQueuedOperationCount <= allowedPendingOperationCount else {
            return false
        }
        if Date() < keyboardRemoteSeedSuppressedUntil {
            return false
        }
        let effectiveText = remoteText ?? sanitizedDesktopEditableText(from: summary)
        let normalizedSelection = remoteSelection ?? desktopSelectionRange(from: summary, text: effectiveText)
        return effectiveText != keyboardMirroredText || normalizedSelection != keyboardMirroredSelection
    }

    private func scheduleDesktopFocusSync(sessionID: String, delayMilliseconds: Int = 90) {
        pendingDesktopFocusSyncTask?.cancel()
        pendingDesktopFocusSyncTask = Task {
            if delayMilliseconds > 0 {
                try? await Task.sleep(nanoseconds: UInt64(delayMilliseconds) * 1_000_000)
            }
            guard Task.isCancelled == false,
                  self.sessionSummary?.sessionID == sessionID else {
                return
            }
            await refreshDesktopStatus(sessionID: sessionID)
        }
    }

    private var desktopInputGridColumns: [GridItem] {
        [GridItem(.adaptive(minimum: 88), spacing: 10, alignment: .leading)]
    }

    private var desktopModifierOptions: [DesktopModifierOption] {
        [
            DesktopModifierOption(id: "cmd", label: "Cmd"),
            DesktopModifierOption(id: "shift", label: "Shift"),
            DesktopModifierOption(id: "alt", label: "Option"),
            DesktopModifierOption(id: "ctrl", label: "Control"),
        ]
    }

    private var desktopKeyActions: [DesktopKeyAction] {
        [
            DesktopKeyAction(id: "enter", label: "Enter", key: "Enter"),
            DesktopKeyAction(id: "tab", label: "Tab", key: "Tab"),
            DesktopKeyAction(id: "esc", label: "Esc", key: "Escape"),
            DesktopKeyAction(id: "backspace", label: "Backspace", key: "Backspace"),
            DesktopKeyAction(id: "delete", label: "Delete", key: "Delete"),
            DesktopKeyAction(id: "left", label: "Left", key: "ArrowLeft"),
            DesktopKeyAction(id: "right", label: "Right", key: "ArrowRight"),
            DesktopKeyAction(id: "up", label: "Up", key: "ArrowUp"),
            DesktopKeyAction(id: "down", label: "Down", key: "ArrowDown"),
            DesktopKeyAction(id: "home", label: "Home", key: "Home"),
            DesktopKeyAction(id: "end", label: "End", key: "End"),
            DesktopKeyAction(id: "page-up", label: "Page Up", key: "PageUp"),
            DesktopKeyAction(id: "page-down", label: "Page Down", key: "PageDown"),
        ]
    }

    private var desktopShortcutActions: [DesktopShortcutAction] {
        [
            DesktopShortcutAction(id: "copy", label: "Cmd-C", key: "c", modifiers: ["cmd"]),
            DesktopShortcutAction(id: "paste", label: "Cmd-V", key: "v", modifiers: ["cmd"]),
            DesktopShortcutAction(id: "cut", label: "Cmd-X", key: "x", modifiers: ["cmd"]),
            DesktopShortcutAction(id: "undo", label: "Cmd-Z", key: "z", modifiers: ["cmd"]),
            DesktopShortcutAction(id: "redo", label: "Cmd-Shift-Z", key: "z", modifiers: ["cmd", "shift"]),
            DesktopShortcutAction(id: "save", label: "Cmd-S", key: "s", modifiers: ["cmd"]),
            DesktopShortcutAction(id: "find", label: "Cmd-F", key: "f", modifiers: ["cmd"]),
            DesktopShortcutAction(id: "select-all", label: "Cmd-A", key: "a", modifiers: ["cmd"]),
            DesktopShortcutAction(id: "new-tab", label: "Cmd-T", key: "t", modifiers: ["cmd"]),
            DesktopShortcutAction(id: "close-tab", label: "Cmd-W", key: "w", modifiers: ["cmd"]),
            DesktopShortcutAction(id: "refresh", label: "Cmd-R", key: "r", modifiers: ["cmd"]),
        ]
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
        SessionRow(profile: profile, presentation: self.presentation, onOpen: onOpen)
    }
}

private struct SessionRow: View {
    let profile: BridgeProfile
    let presentation: SessionCard.Presentation
    let onOpen: () -> Void

    var body: some View {
        Button(action: onOpen) {
            HStack(spacing: 12) {
                Text(profileAvatar(profile))
                    .font(.subheadline.weight(.bold))
                    .foregroundStyle(.white)
                    .frame(width: 36, height: 36)
                    .background(Circle().fill(Color(.tertiaryLabel)))

                VStack(alignment: .leading, spacing: 2) {
                    Text(displayProfileLabel(profile))
                        .font(.subheadline.weight(.medium))
                        .foregroundStyle(.primary)
                    Text(presentation.detail)
                        .font(.caption)
                        .foregroundStyle(.tertiary)
                        .lineLimit(1)
                }

                Spacer()

                if presentation.showsProgress {
                    ProgressView()
                        .controlSize(.small)
                } else if let badge = presentation.badge {
                    StatusPill(label: badge.label, tone: badge.tone)
                } else {
                    Image(systemName: "chevron.right")
                        .font(.caption2.weight(.semibold))
                        .foregroundStyle(.quaternary)
                }
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 12)
            .background(Color(.secondarySystemBackground))
        }
        .buttonStyle(.plain)
        .disabled(!presentation.isActionEnabled)
    }
}

@MainActor
private func makeSessionCardPresentation(model: NativeHostModel, profile: BridgeProfile) -> SessionCard.Presentation {
    let isCurrent = profile.id == model.activeSessionForSheet?.id
    let isConnected = isCurrent && model.isConnected
    let isConnecting = isCurrent && model.isLoading
    let isReconnecting = isCurrent && model.isAutoReconnectPending
    let isErrored = isCurrent && model.isError && !model.isAutoReconnectPending

    let badge: SessionCard.Badge?
    if isConnecting {
        badge = SessionCard.Badge(label: "Connecting", tone: .active)
    } else if isReconnecting {
        badge = SessionCard.Badge(label: "Reconnecting", tone: .active)
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
    } else if isReconnecting {
        detail = model.summarizeWorkspaceError(model.statusMessage)
    } else if isErrored {
        detail = model.summarizeWorkspaceError(model.statusMessage)
    } else {
        detail = describeProfileStatus(profile, isCurrent: isCurrent)
    }

    return SessionCard.Presentation(
        detail: detail,
        badge: badge,
        isActionEnabled: !(isConnected || isConnecting || isReconnecting),
        showsProgress: isConnecting || isReconnecting
    )
}

private struct StatusPill: View {
    enum Tone {
        case active
        case success
        case warning
    }

    let label: String
    let tone: Tone

    var body: some View {
        HStack(spacing: 4) {
            Circle()
                .fill(dotColor)
                .frame(width: 6, height: 6)
            Text(label)
                .font(.caption2.weight(.medium))
        }
        .foregroundStyle(dotColor)
    }

    private var dotColor: Color {
        switch tone {
        case .active: return .blue
        case .success: return .green
        case .warning: return .orange
        }
    }
}
