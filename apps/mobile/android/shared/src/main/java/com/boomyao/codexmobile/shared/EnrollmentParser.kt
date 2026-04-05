package com.boomyao.codexmobile.shared

import org.json.JSONObject

private val ENROLLMENT_WRAPPER_KEYS = listOf(
    "payload",
    "mobileEnrollmentPayload",
    "enrollmentPayload",
    "data",
    "result",
)

private fun unwrapEnrollmentPayloadValue(candidate: Any?): JSONObject? {
    return when (candidate) {
        is JSONObject -> candidate
        is String -> runCatching { JSONObject(candidate) }.getOrNull()
        else -> null
    }
}

private fun normalizedEnrollmentPayloadRoot(rawPayload: String): JSONObject {
    val root = try {
        JSONObject(rawPayload)
    } catch (error: Exception) {
        throw IllegalArgumentException("Enrollment payload is not valid JSON.", error)
    }
    var current = root
    repeat(4) {
        val payloadType = current.optString("type").trim()
        if (payloadType.isNotEmpty()) {
            return current
        }
        val next =
            ENROLLMENT_WRAPPER_KEYS.firstNotNullOfOrNull { key ->
                unwrapEnrollmentPayloadValue(current.opt(key))
            }
        if (next == null) {
            return current
        }
        current = next
    }
    return current
}

private fun normalizedEnrollmentPayloadJson(rawPayload: String): String {
    return normalizedEnrollmentPayloadRoot(rawPayload).toString()
}

private fun unsupportedEnrollmentPayloadTypeError(root: JSONObject): IllegalArgumentException {
    val payloadType = root.optString("type").trim()
    val typeDescription =
        if (payloadType.isEmpty()) {
            "missing"
        } else {
            payloadType
        }
    val keys = mutableListOf<String>()
    val iterator = root.keys()
    while (iterator.hasNext()) {
        keys += iterator.next()
    }
    val keysDescription = keys.sorted().joinToString(",").ifEmpty { "<none>" }
    return IllegalArgumentException(
        "Unsupported enrollment payload type ($typeDescription). Top-level keys: $keysDescription.",
    )
}

object EnrollmentParser {
    fun parse(rawJson: String): EnrollmentPayload {
        val normalizedRawJson = normalizedEnrollmentPayloadJson(rawJson)
        val root = JSONObject(normalizedRawJson)

        return when (root.optString("type")) {
            "codex-mobile-bridge" -> EnrollmentPayload.Bridge(
                bridgeId = root.optString("bridgeId").trim().ifEmpty { null },
                name = root.optString("name").trim().ifEmpty { "Codex Bridge" },
                serverEndpoint = BridgeEndpoint.normalizeEndpoint(root.optString("serverEndpoint").trim()),
                pairingCode = root.optString("pairingCode").trim().ifEmpty { null },
            )

            "codex-mobile-enrollment" -> EnrollmentPayload.Bridge(
                bridgeId = root.optString("bridgeId").trim().ifEmpty { null },
                name = root.optString("bridgeName").trim().ifEmpty { "Codex Bridge" },
                serverEndpoint = BridgeEndpoint.normalizeEndpoint(
                    root.optString("bridgeServerEndpoint").trim(),
                ),
                pairingCode = root.optString("pairingCode").trim().ifEmpty { null },
            )

            else -> throw unsupportedEnrollmentPayloadTypeError(root)
        }
    }
}
