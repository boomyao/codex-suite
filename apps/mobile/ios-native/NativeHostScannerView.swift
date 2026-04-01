import AVFoundation
import SwiftUI
import UIKit

@MainActor
final class NativeHostScannerCoordinator: NSObject, ObservableObject {
    @Published var isCameraAuthorized = true
    @Published var errorMessage: String?

    private var hasScanned = false
    private let onCodeScanned: (String) -> Void

    init(onCodeScanned: @escaping (String) -> Void) {
        self.onCodeScanned = onCodeScanned
        super.init()
    }

    func requestCameraAccess() async {
        switch AVCaptureDevice.authorizationStatus(for: .video) {
        case .authorized:
            isCameraAuthorized = true
        case .notDetermined:
            isCameraAuthorized = await AVCaptureDevice.requestAccess(for: .video)
        default:
            isCameraAuthorized = false
        }
    }

    func reset() {
        hasScanned = false
        errorMessage = nil
    }

    fileprivate func handleScannedValue(_ value: String) {
        guard !hasScanned else {
            return
        }
        hasScanned = true
        onCodeScanned(value)
    }
}

private final class ScannerPreviewViewController: UIViewController, AVCaptureMetadataOutputObjectsDelegate {
    private let session = AVCaptureSession()
    private var previewLayer: AVCaptureVideoPreviewLayer?
    private let coordinator: NativeHostScannerCoordinator

    init(coordinator: NativeHostScannerCoordinator) {
        self.coordinator = coordinator
        super.init(nibName: nil, bundle: nil)
    }

    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = .black
        configureSession()
    }

    override func viewDidLayoutSubviews() {
        super.viewDidLayoutSubviews()
        previewLayer?.frame = view.bounds
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        if !session.isRunning {
            DispatchQueue.global(qos: .userInitiated).async {
                self.session.startRunning()
            }
        }
    }

    override func viewWillDisappear(_ animated: Bool) {
        super.viewWillDisappear(animated)
        if session.isRunning {
            DispatchQueue.global(qos: .userInitiated).async {
                self.session.stopRunning()
            }
        }
    }

    private func configureSession() {
        guard let device = AVCaptureDevice.default(for: .video),
              let input = try? AVCaptureDeviceInput(device: device),
              session.canAddInput(input) else {
            coordinator.errorMessage = "The camera is unavailable on this device."
            return
        }
        session.beginConfiguration()
        session.sessionPreset = .high
        session.addInput(input)

        let output = AVCaptureMetadataOutput()
        guard session.canAddOutput(output) else {
            session.commitConfiguration()
            coordinator.errorMessage = "The QR scanner could not be configured."
            return
        }
        session.addOutput(output)
        output.setMetadataObjectsDelegate(self, queue: .main)
        output.metadataObjectTypes = [.qr]
        session.commitConfiguration()

        let previewLayer = AVCaptureVideoPreviewLayer(session: session)
        previewLayer.videoGravity = .resizeAspectFill
        view.layer.addSublayer(previewLayer)
        self.previewLayer = previewLayer
    }

    func metadataOutput(
        _ output: AVCaptureMetadataOutput,
        didOutput metadataObjects: [AVMetadataObject],
        from connection: AVCaptureConnection
    ) {
        guard let code = metadataObjects.compactMap({ $0 as? AVMetadataMachineReadableCodeObject }).first?.stringValue else {
            return
        }
        coordinator.handleScannedValue(code)
    }
}

private struct ScannerPreviewContainer: UIViewControllerRepresentable {
    @ObservedObject var coordinator: NativeHostScannerCoordinator

    func makeUIViewController(context: Context) -> ScannerPreviewViewController {
        ScannerPreviewViewController(coordinator: coordinator)
    }

    func updateUIViewController(_ uiViewController: ScannerPreviewViewController, context: Context) {}
}

struct NativeHostScannerView: View {
    @Environment(\.dismiss) private var dismiss
    @StateObject private var coordinator: NativeHostScannerCoordinator

    private let onCancel: () -> Void

    init(
        onCodeScanned: @escaping (String) -> Void,
        onCancel: @escaping () -> Void
    ) {
        _coordinator = StateObject(wrappedValue: NativeHostScannerCoordinator(onCodeScanned: onCodeScanned))
        self.onCancel = onCancel
    }

    var body: some View {
        ZStack {
            if coordinator.isCameraAuthorized {
                ScannerPreviewContainer(coordinator: coordinator)
                    .ignoresSafeArea()
                scannerOverlay
            } else {
                cameraPermissionView
            }
        }
        .task {
            await coordinator.requestCameraAccess()
        }
        .onAppear {
            coordinator.reset()
        }
    }

    private var scannerOverlay: some View {
        ZStack {
            Color.black.opacity(0.34)
                .ignoresSafeArea()

            VStack(spacing: 0) {
                HStack {
                    Button("Close") {
                        dismiss()
                        onCancel()
                    }
                    .font(.headline)
                    .foregroundStyle(.white)
                    Spacer()
                }
                .padding(.horizontal, 20)
                .padding(.top, 18)

                Spacer()

                RoundedRectangle(cornerRadius: 28, style: .continuous)
                    .strokeBorder(.white.opacity(0.92), lineWidth: 3)
                    .frame(width: 280, height: 280)
                    .overlay(alignment: .top) {
                        VStack(spacing: 10) {
                            Text("Scan the desktop QR code")
                                .font(.title3.weight(.semibold))
                                .foregroundStyle(.white)
                            Text("Open Codex on desktop, show the mobile QR code, then point this iPad at it.")
                                .font(.callout)
                                .multilineTextAlignment(.center)
                                .foregroundStyle(.white.opacity(0.84))
                                .frame(maxWidth: 280)
                        }
                        .padding(.top, -120)
                    }

                Spacer()

                if let errorMessage = coordinator.errorMessage, !errorMessage.isEmpty {
                    Text(errorMessage)
                        .font(.callout)
                        .foregroundStyle(.white)
                        .padding(.horizontal, 20)
                        .padding(.bottom, 24)
                } else {
                    Text("The setup code will open a trusted session on this iPad.")
                        .font(.callout)
                        .foregroundStyle(.white.opacity(0.84))
                        .padding(.horizontal, 20)
                        .padding(.bottom, 24)
                }
            }
        }
    }

    private var cameraPermissionView: some View {
        ZStack {
            LinearGradient(
                colors: [Color(red: 0.11, green: 0.13, blue: 0.18), Color(red: 0.03, green: 0.04, blue: 0.07)],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )
            .ignoresSafeArea()

            VStack(spacing: 18) {
                Image(systemName: "camera.viewfinder")
                    .font(.system(size: 46, weight: .medium))
                    .foregroundStyle(.white)
                Text("Camera access is required")
                    .font(.title2.weight(.semibold))
                    .foregroundStyle(.white)
                Text("Allow camera access to scan the desktop QR code.")
                    .font(.body)
                    .multilineTextAlignment(.center)
                    .foregroundStyle(.white.opacity(0.82))
                    .frame(maxWidth: 340)

                VStack(spacing: 12) {
                    Button("Open Settings") {
                        if let settingsURL = URL(string: UIApplication.openSettingsURLString) {
                            UIApplication.shared.open(settingsURL)
                        }
                    }
                    .buttonStyle(.borderedProminent)

                    Button("Close") {
                        dismiss()
                        onCancel()
                    }
                    .buttonStyle(.plain)
                    .foregroundStyle(.white.opacity(0.8))
                }
            }
            .padding(24)
        }
    }
}
