import AppKit
import CoreGraphics
import CoreMedia
import CoreVideo
import Foundation
import ScreenCaptureKit
import VideoToolbox

private struct HelperErrorMessage: Encodable {
    let type = "error"
    let message: String
}

private struct HelperReadyMessage: Encodable {
    let type = "ready"
    let codec = "h264"
    let width: Int
    let height: Int
    let scale: Double
    let inputWidth: Int
    let inputHeight: Int
    let inputOriginX: Int
    let inputOriginY: Int
    let frameRate: Int
}

private struct HelperFormatPayload: Encodable {
    let codec = "h264"
    let width: Int
    let height: Int
    let scale: Double
    let inputWidth: Int
    let inputHeight: Int
    let inputOriginX: Int
    let inputOriginY: Int
    let frameRate: Int
    let nalUnitHeaderLength: Int
    let spsBase64: String
    let ppsBase64: String
}

private struct HelperFormatMessage: Encodable {
    let type = "format"
    let payload: HelperFormatPayload
}

private enum HelperPacketType: UInt8 {
    case json = 1
    case sample = 2
}

private final class FramedStdoutWriter {
    private let encoder = JSONEncoder()
    private let lock = NSLock()
    private let stdout = FileHandle.standardOutput

    func writeJSON<T: Encodable>(_ value: T) {
        do {
            let data = try encoder.encode(value)
            writePacket(type: .json, payload: data)
        } catch {
            fputs("desktop_video_streamer encode failed: \(error)\n", stderr)
        }
    }

    func writeSample(ptsUs: Int64, durationUs: Int64, keyFrame: Bool, frameData: Data) {
        var payload = Data()
        payload.reserveCapacity(17 + frameData.count)
        payload.append(bigEndianBytes(UInt64(bitPattern: ptsUs)))
        payload.append(bigEndianBytes(UInt64(bitPattern: durationUs)))
        payload.append(keyFrame ? 1 : 0)
        payload.append(frameData)
        writePacket(type: .sample, payload: payload)
    }

    private func writePacket(type: HelperPacketType, payload: Data) {
        var framed = Data()
        framed.reserveCapacity(5 + payload.count)
        framed.append(type.rawValue)
        framed.append(bigEndianBytes(UInt32(payload.count)))
        framed.append(payload)

        lock.lock()
        defer { lock.unlock() }
        stdout.write(framed)
    }

    private func bigEndianBytes<T: FixedWidthInteger>(_ value: T) -> Data {
        var bigEndianValue = value.bigEndian
        return withUnsafeBytes(of: &bigEndianValue) { Data($0) }
    }
}

private final class DesktopVideoStreamer: NSObject, SCStreamOutput, SCStreamDelegate {
    private let writer = FramedStdoutWriter()
    private let sampleHandlerQueue = DispatchQueue(label: "codex.desktop-video.sample-handler")
    private let encodeQueue = DispatchQueue(label: "codex.desktop-video.encode")
    private let frameRate = 30

    private var stream: SCStream?
    private var compressionSession: VTCompressionSession?
    private var width = 0
    private var height = 0
    private var scale = 1.0
    private var inputWidth = 0
    private var inputHeight = 0
    private var inputOriginX = 0
    private var inputOriginY = 0
    private var parameterSetSignature = ""

    func start() async throws {
        let content = try await SCShareableContent.current
        guard let display = selectDisplay(from: content.displays) else {
            throw NSError(domain: "DesktopVideoStreamer", code: 1, userInfo: [
                NSLocalizedDescriptionKey: "No shareable display is available for desktop video streaming.",
            ])
        }

        let dimensions = displayDimensions(for: display)
        width = max(dimensions.width, 1)
        height = max(dimensions.height, 1)
        scale = dimensions.scale
        inputWidth = max(dimensions.inputWidth, 1)
        inputHeight = max(dimensions.inputHeight, 1)
        inputOriginX = dimensions.inputOriginX
        inputOriginY = dimensions.inputOriginY

        try configureCompressionSession()

        let configuration = SCStreamConfiguration()
        configuration.width = width
        configuration.height = height
        configuration.minimumFrameInterval = CMTime(value: 1, timescale: CMTimeScale(frameRate))
        configuration.queueDepth = 3
        configuration.pixelFormat = kCVPixelFormatType_420YpCbCr8BiPlanarVideoRange
        configuration.showsCursor = true
        if #available(macOS 15.0, *) {
            configuration.showMouseClicks = true
        }

        let filter = SCContentFilter(display: display, excludingWindows: [])
        let stream = SCStream(filter: filter, configuration: configuration, delegate: self)
        try stream.addStreamOutput(self, type: .screen, sampleHandlerQueue: sampleHandlerQueue)
        try await stream.startCapture()
        self.stream = stream

        writer.writeJSON(
            HelperReadyMessage(
                width: width,
                height: height,
                scale: scale,
                inputWidth: inputWidth,
                inputHeight: inputHeight,
                inputOriginX: inputOriginX,
                inputOriginY: inputOriginY,
                frameRate: frameRate
            )
        )
    }

    func stop() {
        if let stream {
            Task {
                try? await stream.stopCapture()
            }
        }
        if let session = compressionSession {
            VTCompressionSessionCompleteFrames(session, untilPresentationTimeStamp: .invalid)
            VTCompressionSessionInvalidate(session)
            compressionSession = nil
        }
    }

    func stream(_ stream: SCStream, didOutputSampleBuffer sampleBuffer: CMSampleBuffer, of type: SCStreamOutputType) {
        guard type == .screen,
              CMSampleBufferDataIsReady(sampleBuffer),
              shouldEncode(sampleBuffer),
              let imageBuffer = CMSampleBufferGetImageBuffer(sampleBuffer),
              let compressionSession else {
            return
        }

        let pts = CMSampleBufferGetPresentationTimeStamp(sampleBuffer)
        let duration = durationForSampleBuffer(sampleBuffer)

        encodeQueue.async {
            var infoFlags = VTEncodeInfoFlags()
            let status = VTCompressionSessionEncodeFrame(
                compressionSession,
                imageBuffer: imageBuffer,
                presentationTimeStamp: pts,
                duration: duration,
                frameProperties: nil,
                sourceFrameRefcon: nil,
                infoFlagsOut: &infoFlags
            )
            if status != noErr {
                fputs("desktop_video_streamer encode failed: \(status)\n", stderr)
            }
        }
    }

    func stream(_ stream: SCStream, didStopWithError error: Error) {
        writer.writeJSON(HelperErrorMessage(message: error.localizedDescription))
    }

    private func configureCompressionSession() throws {
        var session: VTCompressionSession?
        let status = VTCompressionSessionCreate(
            allocator: nil,
            width: Int32(width),
            height: Int32(height),
            codecType: kCMVideoCodecType_H264,
            encoderSpecification: nil,
            imageBufferAttributes: nil,
            compressedDataAllocator: nil,
            outputCallback: compressionOutputCallback,
            refcon: Unmanaged.passUnretained(self).toOpaque(),
            compressionSessionOut: &session
        )
        guard status == noErr, let session else {
            throw NSError(domain: "DesktopVideoStreamer", code: Int(status), userInfo: [
                NSLocalizedDescriptionKey: "Failed to create H264 compression session (\(status)).",
            ])
        }

        compressionSession = session
        VTSessionSetProperty(session, key: kVTCompressionPropertyKey_RealTime, value: kCFBooleanTrue)
        VTSessionSetProperty(session, key: kVTCompressionPropertyKey_AllowFrameReordering, value: kCFBooleanFalse)
        VTSessionSetProperty(session, key: kVTCompressionPropertyKey_MaxFrameDelayCount, value: NSNumber(value: 0))
        VTSessionSetProperty(session, key: kVTCompressionPropertyKey_PrioritizeEncodingSpeedOverQuality, value: kCFBooleanTrue)
        VTSessionSetProperty(session, key: kVTCompressionPropertyKey_ProfileLevel, value: kVTProfileLevel_H264_Main_AutoLevel)
        VTSessionSetProperty(session, key: kVTCompressionPropertyKey_ExpectedFrameRate, value: NSNumber(value: frameRate))
        VTSessionSetProperty(session, key: kVTCompressionPropertyKey_MaxKeyFrameInterval, value: NSNumber(value: frameRate))
        VTSessionSetProperty(session, key: kVTCompressionPropertyKey_MaxKeyFrameIntervalDuration, value: NSNumber(value: 2))
        VTSessionSetProperty(session, key: kVTCompressionPropertyKey_AverageBitRate, value: NSNumber(value: targetBitRate()))
        VTCompressionSessionPrepareToEncodeFrames(session)
    }

    fileprivate func handleEncodedSampleBuffer(_ sampleBuffer: CMSampleBuffer) {
        guard CMSampleBufferDataIsReady(sampleBuffer),
              let formatDescription = CMSampleBufferGetFormatDescription(sampleBuffer),
              let blockBuffer = CMSampleBufferGetDataBuffer(sampleBuffer) else {
            return
        }

        let keyFrame = sampleBufferIsKeyFrame(sampleBuffer)
        if let parameterSets = h264ParameterSets(from: formatDescription) {
            let signature = parameterSets.signature
            if signature != parameterSetSignature {
                parameterSetSignature = signature
                writer.writeJSON(
                    HelperFormatMessage(
                        payload: HelperFormatPayload(
                            width: width,
                            height: height,
                            scale: scale,
                            inputWidth: inputWidth,
                            inputHeight: inputHeight,
                            inputOriginX: inputOriginX,
                            inputOriginY: inputOriginY,
                            frameRate: frameRate,
                            nalUnitHeaderLength: parameterSets.nalUnitHeaderLength,
                            spsBase64: parameterSets.sps.base64EncodedString(),
                            ppsBase64: parameterSets.pps.base64EncodedString()
                        )
                    )
                )
            }
        }

        let dataLength = CMBlockBufferGetDataLength(blockBuffer)
        guard dataLength > 0 else {
            return
        }

        var data = Data(count: dataLength)
        let copyStatus = data.withUnsafeMutableBytes { rawBuffer in
            guard let baseAddress = rawBuffer.baseAddress else {
                return kCMBlockBufferBadCustomBlockSourceErr
            }
            return CMBlockBufferCopyDataBytes(blockBuffer, atOffset: 0, dataLength: dataLength, destination: baseAddress)
        }
        guard copyStatus == noErr else {
            fputs("desktop_video_streamer block copy failed: \(copyStatus)\n", stderr)
            return
        }

        writer.writeSample(
            ptsUs: timeValueInMicroseconds(CMSampleBufferGetPresentationTimeStamp(sampleBuffer)),
            durationUs: timeValueInMicroseconds(durationForSampleBuffer(sampleBuffer)),
            keyFrame: keyFrame,
            frameData: data
        )
    }

    private func durationForSampleBuffer(_ sampleBuffer: CMSampleBuffer) -> CMTime {
        let duration = CMSampleBufferGetDuration(sampleBuffer)
        if duration.isValid, duration.isNumeric, duration.seconds > 0 {
            return duration
        }
        return CMTime(value: 1, timescale: CMTimeScale(frameRate))
    }

    private func shouldEncode(_ sampleBuffer: CMSampleBuffer) -> Bool {
        guard let attachments = CMSampleBufferGetSampleAttachmentsArray(sampleBuffer, createIfNecessary: false) as? [[SCStreamFrameInfo: Any]],
              let attachment = attachments.first,
              let rawStatus = attachment[.status] as? Int else {
            return true
        }
        let frameStatus = SCFrameStatus(rawValue: rawStatus)
        return frameStatus == .complete || frameStatus == .started
    }

    private func sampleBufferIsKeyFrame(_ sampleBuffer: CMSampleBuffer) -> Bool {
        guard let attachments = CMSampleBufferGetSampleAttachmentsArray(sampleBuffer, createIfNecessary: false) as? [[CFString: Any]],
              let firstAttachment = attachments.first else {
            return true
        }
        let notSync = firstAttachment[kCMSampleAttachmentKey_NotSync] as? Bool ?? false
        return !notSync
    }

    private func targetBitRate() -> Int {
        let pixels = max(width * height, 1)
        return max(2_500_000, min(pixels * 6, 14_000_000))
    }

    private func selectDisplay(from displays: [SCDisplay]) -> SCDisplay? {
        let mainDisplayID = CGMainDisplayID()
        if let match = displays.first(where: { $0.displayID == mainDisplayID }) {
            return match
        }
        return displays.sorted { lhs, rhs in
            if lhs.frame.minX == rhs.frame.minX {
                return lhs.frame.minY < rhs.frame.minY
            }
            return lhs.frame.minX < rhs.frame.minX
        }.first
    }

    private func displayDimensions(for display: SCDisplay) -> (width: Int, height: Int, scale: Double, inputWidth: Int, inputHeight: Int, inputOriginX: Int, inputOriginY: Int) {
        let inputWidth = max(Int(display.frame.width.rounded()), 1)
        let inputHeight = max(Int(display.frame.height.rounded()), 1)
        let inputOriginX = Int(display.frame.minX.rounded())
        let inputOriginY = Int(display.frame.minY.rounded())
        if let mode = CGDisplayCopyDisplayMode(display.displayID) {
            let scale = max(Double(mode.pixelWidth) / max(Double(display.width), 1), 1)
            return (
                max(mode.pixelWidth, 1),
                max(mode.pixelHeight, 1),
                scale,
                inputWidth,
                inputHeight,
                inputOriginX,
                inputOriginY
            )
        }
        return (
            max(display.width, 1),
            max(display.height, 1),
            1,
            inputWidth,
            inputHeight,
            inputOriginX,
            inputOriginY
        )
    }
}

private struct H264ParameterSets {
    let sps: Data
    let pps: Data
    let nalUnitHeaderLength: Int

    var signature: String {
        sps.base64EncodedString() + "|" + pps.base64EncodedString() + "|" + String(nalUnitHeaderLength)
    }
}

private func h264ParameterSets(from formatDescription: CMFormatDescription) -> H264ParameterSets? {
    var spsPointer: UnsafePointer<UInt8>?
    var spsSize = 0
    var ppsPointer: UnsafePointer<UInt8>?
    var ppsSize = 0
    var parameterSetCount = 0
    var nalUnitHeaderLength: Int32 = 0

    let spsStatus = CMVideoFormatDescriptionGetH264ParameterSetAtIndex(
        formatDescription,
        parameterSetIndex: 0,
        parameterSetPointerOut: &spsPointer,
        parameterSetSizeOut: &spsSize,
        parameterSetCountOut: &parameterSetCount,
        nalUnitHeaderLengthOut: &nalUnitHeaderLength
    )
    guard spsStatus == noErr,
          let spsPointer,
          spsSize > 0 else {
        return nil
    }

    let ppsStatus = CMVideoFormatDescriptionGetH264ParameterSetAtIndex(
        formatDescription,
        parameterSetIndex: 1,
        parameterSetPointerOut: &ppsPointer,
        parameterSetSizeOut: &ppsSize,
        parameterSetCountOut: &parameterSetCount,
        nalUnitHeaderLengthOut: &nalUnitHeaderLength
    )
    guard ppsStatus == noErr,
          let ppsPointer,
          ppsSize > 0 else {
        return nil
    }

    return H264ParameterSets(
        sps: Data(bytes: spsPointer, count: spsSize),
        pps: Data(bytes: ppsPointer, count: ppsSize),
        nalUnitHeaderLength: Int(nalUnitHeaderLength)
    )
}

private func timeValueInMicroseconds(_ time: CMTime) -> Int64 {
    guard time.isValid, time.isNumeric else {
        return 0
    }
    let seconds = time.seconds
    guard seconds.isFinite else {
        return 0
    }
    return Int64((seconds * 1_000_000).rounded())
}

private let compressionOutputCallback: VTCompressionOutputCallback = { refCon, _, status, infoFlags, sampleBuffer in
    guard let refCon else {
        return
    }
    let streamer = Unmanaged<DesktopVideoStreamer>.fromOpaque(refCon).takeUnretainedValue()
    guard status == noErr,
          let sampleBuffer,
          infoFlags.contains(.frameDropped) == false else {
        if status != noErr {
            fputs("desktop_video_streamer callback failed: \(status)\n", stderr)
        }
        return
    }
    streamer.handleEncodedSampleBuffer(sampleBuffer)
}

private let streamer = DesktopVideoStreamer()
private let application = NSApplication.shared

application.setActivationPolicy(.prohibited)

Task {
    do {
        try await streamer.start()
    } catch {
        FramedStdoutWriter().writeJSON(HelperErrorMessage(message: error.localizedDescription))
        exit(1)
    }
}
dispatchMain()
