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
            Color.black.opacity(0.4)
                .ignoresSafeArea()

            VStack(spacing: 0) {
                HStack {
                    Button {
                        dismiss()
                        onCancel()
                    } label: {
                        Image(systemName: "xmark")
                            .font(.body.weight(.semibold))
                            .foregroundStyle(.white)
                            .frame(width: 36, height: 36)
                            .background(.ultraThinMaterial, in: Circle())
                    }
                    Spacer()
                }
                .padding(.horizontal, 20)
                .padding(.top, 16)

                Spacer()

                RoundedRectangle(cornerRadius: 20, style: .continuous)
                    .strokeBorder(.white.opacity(0.6), lineWidth: 2)
                    .frame(width: 240, height: 240)

                Spacer()

                VStack(spacing: 6) {
                    if let errorMessage = coordinator.errorMessage, !errorMessage.isEmpty {
                        Text(errorMessage)
                            .font(.subheadline)
                            .foregroundStyle(.white)
                    } else {
                        Text("Point at the desktop QR code")
                            .font(.subheadline.weight(.medium))
                            .foregroundStyle(.white)
                    }
                }
                .padding(.horizontal, 20)
                .padding(.bottom, 32)
            }
        }
    }

    private var cameraPermissionView: some View {
        ZStack {
            Color.black
                .ignoresSafeArea()

            VStack(spacing: 20) {
                Image(systemName: "camera")
                    .font(.system(size: 36, weight: .medium))
                    .foregroundStyle(.white.opacity(0.6))
                Text("Camera access required")
                    .font(.headline)
                    .foregroundStyle(.white)
                Text("Allow camera access to scan the QR code.")
                    .font(.subheadline)
                    .multilineTextAlignment(.center)
                    .foregroundStyle(.white.opacity(0.6))
                    .frame(maxWidth: 280)

                VStack(spacing: 12) {
                    Button("Open Settings") {
                        if let settingsURL = URL(string: UIApplication.openSettingsURLString) {
                            UIApplication.shared.open(settingsURL)
                        }
                    }
                    .buttonStyle(.borderedProminent)
                    .tint(.white)
                    .foregroundStyle(.black)

                    Button("Close") {
                        dismiss()
                        onCancel()
                    }
                    .foregroundStyle(.white.opacity(0.5))
                }
            }
            .padding(24)
        }
    }
}
