import AppKit
import ApplicationServices
import Foundation

private struct EditableStatePayload: Encodable {
    let active: Bool
    let text: String
    let placeholder: String
    let selectionStart: Int
    let selectionEnd: Int
    let role: String?
    let error: String?
}

private struct DebugPayload: Encodable {
    let role: String?
    let subrole: String?
    let title: String?
    let description: String?
    let help: String?
    let placeholder: String?
    let value: String?
    let selectedText: String?
    let selectedTextRange: String?
    let editable: Bool?
    let focused: Bool?
    let enabled: Bool?
}

private enum HelperCommand: String {
    case focusedEditableState = "focused-editable-state"
    case setFocusedTextState = "set-focused-text-state"
    case debugFocusedElement = "debug-focused-element"
}

private final class DesktopAccessibilityHelper {
    func run(arguments: [String]) throws {
        _ = NSApplication.shared
        NSApp.setActivationPolicy(.prohibited)

        guard let rawCommand = arguments.dropFirst().first,
              let command = HelperCommand(rawValue: rawCommand) else {
            throw HelperError("A helper command is required.")
        }

        switch command {
        case .focusedEditableState:
            try writePayload(focusedEditableState())
        case .setFocusedTextState:
            let payload = try setFocusedTextState(arguments: Array(arguments.dropFirst(2)))
            try writePayload(payload)
        case .debugFocusedElement:
            try writeDebugPayload(debugFocusedElement())
        }
    }

    private func writePayload(_ payload: EditableStatePayload) throws {
        let data = try JSONEncoder().encode(payload)
        FileHandle.standardOutput.write(data)
    }

    private func writeDebugPayload(_ payload: DebugPayload) throws {
        let data = try JSONEncoder().encode(payload)
        FileHandle.standardOutput.write(data)
    }

    private func focusedEditableState() -> EditableStatePayload {
        guard AXIsProcessTrusted() else {
            return EditableStatePayload(
                active: false,
                text: "",
                placeholder: "",
                selectionStart: 0,
                selectionEnd: 0,
                role: nil,
                error: "Accessibility permission is required for desktop text focus detection."
            )
        }

        guard let element = focusedElement() else {
            return EditableStatePayload(
                active: false,
                text: "",
                placeholder: "",
                selectionStart: 0,
                selectionEnd: 0,
                role: nil,
                error: nil
            )
        }

        return editableState(for: element)
    }

    private func debugFocusedElement() -> DebugPayload {
        guard AXIsProcessTrusted() else {
            return DebugPayload(
                role: nil,
                subrole: nil,
                title: nil,
                description: nil,
                help: nil,
                placeholder: nil,
                value: nil,
                selectedText: nil,
                selectedTextRange: nil,
                editable: nil,
                focused: nil,
                enabled: nil
            )
        }

        guard let element = focusedElement() else {
            return DebugPayload(
                role: nil,
                subrole: nil,
                title: nil,
                description: nil,
                help: nil,
                placeholder: nil,
                value: nil,
                selectedText: nil,
                selectedTextRange: nil,
                editable: nil,
                focused: nil,
                enabled: nil
            )
        }

        let selectedRange = copySelectionRange(element: element)
        return DebugPayload(
            role: copyAttributeValue(element: element, attribute: kAXRoleAttribute as CFString) as? String,
            subrole: copyAttributeValue(element: element, attribute: kAXSubroleAttribute as CFString) as? String,
            title: copyAttributeValue(element: element, attribute: kAXTitleAttribute as CFString) as? String,
            description: copyAttributeValue(element: element, attribute: kAXDescriptionAttribute as CFString) as? String,
            help: copyAttributeValue(element: element, attribute: kAXHelpAttribute as CFString) as? String,
            placeholder: copyAttributeValue(element: element, attribute: kAXPlaceholderValueAttribute as CFString) as? String,
            value: copyAttributeValue(element: element, attribute: kAXValueAttribute as CFString) as? String,
            selectedText: copyAttributeValue(element: element, attribute: kAXSelectedTextAttribute as CFString) as? String,
            selectedTextRange: selectedRange.map { "{\($0.location),\($0.length)}" },
            editable: copyAttributeValue(element: element, attribute: "AXEditable" as CFString) as? Bool,
            focused: copyAttributeValue(element: element, attribute: kAXFocusedAttribute as CFString) as? Bool,
            enabled: copyAttributeValue(element: element, attribute: kAXEnabledAttribute as CFString) as? Bool
        )
    }

    private func setFocusedTextState(arguments: [String]) throws -> EditableStatePayload {
        guard AXIsProcessTrusted() else {
            throw HelperError("Accessibility permission is required for desktop text sync.")
        }
        guard arguments.count >= 3 else {
            throw HelperError("set-focused-text-state requires text, selectionStart, and selectionEnd.")
        }
        guard let textData = Data(base64Encoded: arguments[0]),
              let text = String(data: textData, encoding: .utf8) else {
            throw HelperError("set-focused-text-state received invalid text data.")
        }
        guard let selectionStart = Int(arguments[1]),
              let selectionEnd = Int(arguments[2]) else {
            throw HelperError("set-focused-text-state received invalid selection bounds.")
        }
        guard let element = focusedElement() else {
            throw HelperError("No focused desktop element is available.")
        }

        let currentState = editableState(for: element)
        guard currentState.active else {
            throw HelperError("The focused desktop element is not editable.")
        }

        if setStringAttribute(element: element, attribute: kAXValueAttribute as CFString, value: text) == false {
            throw HelperError("Failed to update the focused desktop element text.")
        }

        let textLength = (text as NSString).length
        let clampedStart = min(max(selectionStart, 0), textLength)
        let clampedEnd = min(max(selectionEnd, clampedStart), textLength)
        let selectionLength = max(clampedEnd - clampedStart, 0)
        _ = setSelectionRange(element: element, location: clampedStart, length: selectionLength)

        return settledEditableState(
            for: element,
            expectedText: text,
            selectionStart: clampedStart,
            selectionEnd: clampedEnd
        )
    }

    private func settledEditableState(
        for element: AXUIElement,
        expectedText: String,
        selectionStart: Int,
        selectionEnd: Int
    ) -> EditableStatePayload {
        var latestState = editableState(for: element)
        let deadline = Date().addingTimeInterval(0.45)

        while Date() < deadline {
            if latestState.active,
               latestState.text == expectedText,
               latestState.selectionStart == selectionStart,
               latestState.selectionEnd == selectionEnd {
                return latestState
            }

            if latestState.active,
               latestState.text == expectedText,
               (latestState.selectionStart != selectionStart || latestState.selectionEnd != selectionEnd) {
                let selectionLength = max(selectionEnd - selectionStart, 0)
                _ = setSelectionRange(element: element, location: selectionStart, length: selectionLength)
            }

            Thread.sleep(forTimeInterval: 0.02)
            latestState = editableState(for: element)
        }

        return latestState
    }

    private func focusedElement() -> AXUIElement? {
        let systemWide = AXUIElementCreateSystemWide()
        guard let value = copyAttributeValue(element: systemWide, attribute: kAXFocusedUIElementAttribute as CFString) else {
            return nil
        }
        return (value as! AXUIElement)
    }

    private func editableState(for element: AXUIElement) -> EditableStatePayload {
        let role = copyAttributeValue(element: element, attribute: kAXRoleAttribute as CFString) as? String
        let valueString = copyAttributeValue(element: element, attribute: kAXValueAttribute as CFString) as? String ?? ""
        let placeholderString = copyAttributeValue(element: element, attribute: kAXPlaceholderValueAttribute as CFString) as? String ?? ""
        let effectiveText = normalizedEditableText(value: valueString, placeholder: placeholderString)
        let selectionRange = normalizedSelectionRange(
            originalValue: valueString,
            effectiveText: effectiveText,
            proposedRange: copySelectionRange(element: element)
        )
        let active = isEditable(element: element, role: role, text: effectiveText)
        let selectionStart = max(selectionRange.location, 0)
        let selectionEnd = max(selectionRange.location + selectionRange.length, selectionStart)
        return EditableStatePayload(
            active: active,
            text: active ? effectiveText : "",
            placeholder: active ? placeholderString.replacingOccurrences(of: "\r\n", with: "\n") : "",
            selectionStart: active ? selectionStart : 0,
            selectionEnd: active ? selectionEnd : 0,
            role: role,
            error: nil
        )
    }

    private func normalizedEditableText(value: String, placeholder: String) -> String {
        let normalizedValue = value.replacingOccurrences(of: "\r\n", with: "\n")
        let normalizedPlaceholder = placeholder.replacingOccurrences(of: "\r\n", with: "\n")
        let lines = normalizedValue.split(separator: "\n", omittingEmptySubsequences: false).map(String.init)

        if normalizedPlaceholder.isEmpty == false {
            if normalizedValue == normalizedPlaceholder || normalizedValue == "\n" + normalizedPlaceholder {
                return ""
            }

            if let lastLine = lines.last, lastLine == normalizedPlaceholder {
                return Array(lines.dropLast()).joined(separator: "\n")
            }
        }

        guard let placeholderLine = lines.last(where: { knownComposerPlaceholderLines.contains($0) }) else {
            return normalizedValue
        }

        if normalizedValue == placeholderLine || normalizedValue == "\n" + placeholderLine {
            return ""
        }

        if let lastLine = lines.last, lastLine == placeholderLine {
            let trimmedLines = Array(lines.dropLast())
            return trimmedLines.joined(separator: "\n")
        }

        return normalizedValue
    }

    private func normalizedSelectionRange(
        originalValue: String,
        effectiveText: String,
        proposedRange: CFRange?
    ) -> CFRange {
        let fallback = CFRange(location: effectiveText.utf16.count, length: 0)
        guard var range = proposedRange else {
            return fallback
        }

        let textLength = effectiveText.utf16.count
        if originalValue != effectiveText && range.location == 0 && range.length == 0 {
            return fallback
        }

        range.location = min(max(range.location, 0), textLength)
        range.length = min(max(range.length, 0), max(textLength - range.location, 0))
        return range
    }

    private func isEditable(element: AXUIElement, role: String?, text: String) -> Bool {
        if let editable = copyAttributeValue(element: element, attribute: "AXEditable" as CFString) as? Bool,
           editable {
            return true
        }

        var settable = DarwinBoolean(false)
        if AXUIElementIsAttributeSettable(element, kAXValueAttribute as CFString, &settable) == .success,
           settable.boolValue {
            return true
        }
        if AXUIElementIsAttributeSettable(element, kAXSelectedTextRangeAttribute as CFString, &settable) == .success,
           settable.boolValue {
            return true
        }

        guard let role else {
            return false
        }
        if text.isEmpty == false {
            return editableRoles.contains(role)
        }
        return editableRoles.contains(role) && copySelectionRange(element: element) != nil
    }

    private func copyAttributeValue(element: AXUIElement, attribute: CFString) -> AnyObject? {
        var value: CFTypeRef?
        let status = AXUIElementCopyAttributeValue(element, attribute, &value)
        guard status == .success, let value else {
            return nil
        }
        return value
    }

    private func copySelectionRange(element: AXUIElement) -> CFRange? {
        guard let value = copyAttributeValue(element: element, attribute: kAXSelectedTextRangeAttribute as CFString) else {
            return nil
        }
        guard CFGetTypeID(value) == AXValueGetTypeID() else {
            return nil
        }
        let axValue = value as! AXValue
        guard AXValueGetType(axValue) == .cfRange else {
            return nil
        }
        var range = CFRange(location: 0, length: 0)
        guard AXValueGetValue(axValue, .cfRange, &range) else {
            return nil
        }
        return range
    }

    private func setStringAttribute(element: AXUIElement, attribute: CFString, value: String) -> Bool {
        AXUIElementSetAttributeValue(element, attribute, value as CFTypeRef) == .success
    }

    private func setSelectionRange(element: AXUIElement, location: Int, length: Int) -> Bool {
        var range = CFRange(location: location, length: length)
        guard let axValue = AXValueCreate(.cfRange, &range) else {
            return false
        }
        return AXUIElementSetAttributeValue(element, kAXSelectedTextRangeAttribute as CFString, axValue) == .success
    }
}

private struct HelperError: Error, LocalizedError {
    let message: String

    init(_ message: String) {
        self.message = message
    }

    var errorDescription: String? { message }
}

private let editableRoles: Set<String> = [
    kAXComboBoxRole as String,
    kAXTextAreaRole as String,
    kAXTextFieldRole as String,
    "AXSearchField",
]

private let knownComposerPlaceholderLines: Set<String> = [
    "Ask for follow-up changes",
]

do {
    let helper = DesktopAccessibilityHelper()
    try helper.run(arguments: CommandLine.arguments)
} catch {
    let payload = EditableStatePayload(
        active: false,
        text: "",
        placeholder: "",
        selectionStart: 0,
        selectionEnd: 0,
        role: nil,
        error: error.localizedDescription
    )
    if let data = try? JSONEncoder().encode(payload) {
        FileHandle.standardOutput.write(data)
    }
    exit(1)
}
