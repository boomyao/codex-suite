package com.boomyao.codexmobile.nativehost

import com.boomyao.codexmobile.tailnet.normalizedEnrollmentPayloadJson
import org.json.JSONObject

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
                serverEndpoint = BridgeApi.normalizeEndpoint(root.optString("serverEndpoint").trim()),
                pairingCode = root.optString("pairingCode").trim().ifEmpty { null },
                rawJson = normalizedRawJson,
            )

            "codex-mobile-enrollment" -> EnrollmentPayload.Tailnet(
                bridgeId = root.optString("bridgeId").trim().ifEmpty { null },
                bridgeName = root.optString("bridgeName").trim().ifEmpty { "Codex Bridge" },
                bridgeServerEndpoint = BridgeApi.normalizeEndpoint(
                    root.optString("bridgeServerEndpoint").trim(),
                ),
                pairingCode = root.optString("pairingCode").trim().ifEmpty { null },
                rawJson = normalizedRawJson,
            )

            else -> throw unsupportedEnrollmentPayloadTypeError(root)
        }
    }
}
