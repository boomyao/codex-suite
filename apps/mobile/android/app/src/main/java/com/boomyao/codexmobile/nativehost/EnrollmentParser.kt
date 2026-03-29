package com.boomyao.codexmobile.nativehost

import org.json.JSONObject

object EnrollmentParser {
    fun parse(rawJson: String): EnrollmentPayload {
        val root = try {
            JSONObject(rawJson)
        } catch (error: Exception) {
            throw IllegalArgumentException("Enrollment payload is not valid JSON.", error)
        }

        return when (root.optString("type")) {
            "codex-mobile-bridge" -> EnrollmentPayload.Bridge(
                name = root.optString("name").trim().ifEmpty { "Codex Bridge" },
                serverEndpoint = BridgeApi.normalizeEndpoint(root.optString("serverEndpoint").trim()),
                pairingCode = root.optString("pairingCode").trim().ifEmpty { null },
                rawJson = rawJson,
            )

            "codex-mobile-enrollment" -> EnrollmentPayload.Tailnet(
                bridgeName = root.optString("bridgeName").trim().ifEmpty { "Codex Bridge" },
                bridgeServerEndpoint = BridgeApi.normalizeEndpoint(
                    root.optString("bridgeServerEndpoint").trim(),
                ),
                pairingCode = root.optString("pairingCode").trim().ifEmpty { null },
                rawJson = rawJson,
            )

            else -> throw IllegalArgumentException("Unsupported enrollment payload type.")
        }
    }
}
