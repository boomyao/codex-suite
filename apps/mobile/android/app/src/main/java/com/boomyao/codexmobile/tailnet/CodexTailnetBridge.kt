package com.boomyao.codexmobile.tailnet

import android.content.Context
import org.json.JSONObject

object CodexTailnetBridge {
    fun setDefaultRouteInterface(interfaceName: String?) {
        CodexTailnetNative.setDefaultRouteInterface(interfaceName)
    }

    fun setInterfaceSnapshot(snapshotJson: String?) {
        CodexTailnetNative.setInterfaceSnapshot(snapshotJson)
    }

    fun stage(context: Context, payload: TailnetEnrollmentPayload): TailnetStatusSnapshot {
        val store = EnrollmentStore(context)
        store.saveEnrollment(payload)
        val snapshot = TailnetStatusSnapshot(
            state = "starting",
            mode = "native-shell",
            message = "Android tailnet shell is starting.",
            bridgeName = payload.bridgeName,
            bridgeServerEndpoint = payload.bridgeServerEndpoint,
            localProxyUrl = null,
            rawEnrollmentType = "codex-mobile-enrollment",
            auth = null,
        )
        store.writeStatus(snapshot)
        return snapshot
    }

    fun installVpnService(service: CodexTailnetService) {
        CodexTailnetNative.installVpnService(service)
    }

    fun clearVpnService() {
        CodexTailnetNative.clearVpnService()
    }

    fun start(context: Context, payload: TailnetEnrollmentPayload, tunFd: Int): TailnetStatusSnapshot {
        val store = EnrollmentStore(context)
        store.saveEnrollment(payload)
        val stateDir = java.io.File(context.filesDir, "codex-tailnet").absolutePath
        val snapshot = responseToSnapshot(
            responseJson = CodexTailnetNative.start(payload.rawPayload, stateDir, tunFd),
            fallbackBridgeName = payload.bridgeName,
            fallbackBridgeServerEndpoint = payload.bridgeServerEndpoint,
        )
        store.writeStatus(snapshot)
        return snapshot
    }

    fun stop(context: Context): TailnetStatusSnapshot {
        val store = EnrollmentStore(context)
        val current = store.readStatus()
        val snapshot = responseToSnapshot(
            responseJson = CodexTailnetNative.stop(),
            fallbackBridgeName = current.bridgeName,
            fallbackBridgeServerEndpoint = current.bridgeServerEndpoint,
        )
        store.writeStatus(snapshot)
        return snapshot
    }

    fun configureBridgeProxy(
        context: Context,
        bridgeEndpoint: String,
        authToken: String?,
    ): TailnetStatusSnapshot {
        val store = EnrollmentStore(context)
        val current = store.readStatus()
        val snapshot = responseToSnapshot(
            responseJson = CodexTailnetNative.configureBridgeProxy(bridgeEndpoint, authToken.orEmpty()),
            fallbackBridgeName = current.bridgeName,
            fallbackBridgeServerEndpoint = if (bridgeEndpoint.isBlank()) current.bridgeServerEndpoint else bridgeEndpoint,
        )
        store.writeStatus(snapshot)
        return snapshot
    }

    fun status(context: Context): TailnetStatusSnapshot {
        val current = EnrollmentStore(context).readStatus()
        return responseToSnapshot(
            responseJson = CodexTailnetNative.status(),
            fallbackBridgeName = current.bridgeName,
            fallbackBridgeServerEndpoint = current.bridgeServerEndpoint,
        )
    }

    private fun responseToSnapshot(
        responseJson: String,
        fallbackBridgeName: String?,
        fallbackBridgeServerEndpoint: String?,
    ): TailnetStatusSnapshot {
        return try {
            val root = JSONObject(responseJson)
            val ok = root.optBoolean("ok", false)
            val running = root.optBoolean("running", false)
            val baseMessage = root.optString("message").trim()
            val errorMessage = root.optString("error").trim()
            val message = when {
                !ok && baseMessage.isNotEmpty() && errorMessage.isNotEmpty() && errorMessage != baseMessage ->
                    "$baseMessage: $errorMessage"
                baseMessage.isNotEmpty() -> baseMessage
                errorMessage.isNotEmpty() -> errorMessage
                running -> "Embedded Android tailnet runtime is running."
                else -> "Embedded Android tailnet runtime is idle."
            }
            val data = root.optJSONObject("data")
            TailnetStatusSnapshot(
                state = when {
                    ok && running -> "running"
                    ok -> "stopped"
                    else -> "error"
                },
                mode = "native-bridge",
                message = message,
                bridgeName = data?.optString("bridgeName")?.ifBlank { fallbackBridgeName } ?: fallbackBridgeName,
                bridgeServerEndpoint =
                    data?.optString("bridgeServerEndpoint")?.ifBlank { fallbackBridgeServerEndpoint }
                        ?: fallbackBridgeServerEndpoint,
                localProxyUrl = data?.optString("localProxyUrl")?.ifBlank { null },
                rawEnrollmentType = "codex-mobile-enrollment",
                auth = parseTailnetAuthStatus(data?.optJSONObject("auth")),
            )
        } catch (_: Exception) {
            TailnetStatusSnapshot(
                state = "error",
                mode = "native-bridge",
                message = "Native tailnet bridge returned an invalid response.",
                bridgeName = fallbackBridgeName,
                bridgeServerEndpoint = fallbackBridgeServerEndpoint,
                localProxyUrl = null,
                rawEnrollmentType = "codex-mobile-enrollment",
                auth = null,
            )
        }
    }
}
