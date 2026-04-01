package com.boomyao.codexmobile.nativehost

import org.json.JSONObject
import java.io.BufferedReader
import java.io.InputStreamReader
import java.net.HttpURLConnection
import java.net.URL

object BridgeApi {
    private const val DEFAULT_TIMEOUT_MS = 10_000

    data class BridgeReadyProbe(
        val ready: Boolean,
        val errorMessage: String? = null,
    )

    fun normalizeEndpoint(value: String): String = value.trim().trimEnd('/')

    fun deriveServerHttpBaseUrl(endpoint: String): String {
        val normalized = normalizeEndpoint(endpoint)
        return when {
            normalized.startsWith("ws://") -> "http://${normalized.removePrefix("ws://")}"
            normalized.startsWith("wss://") -> "https://${normalized.removePrefix("wss://")}"
            normalized.startsWith("http://") || normalized.startsWith("https://") -> normalized
            else -> "http://$normalized"
        }
    }

    fun buildRemoteShellUrlFromBaseUrl(baseUrl: String): String {
        return "${normalizeEndpoint(baseUrl)}/ui/index.html"
    }

    fun fetchConnectionTargetByBaseUrl(
        baseUrl: String,
        authToken: String?,
    ): ConnectionTargetResponse {
        val responseText = requestJson(
            url = "${normalizeEndpoint(baseUrl)}/codex-mobile/connect",
            authToken = authToken,
        )
        val payload = JSONObject(responseText)
        val connection = payload.optJSONObject("connection")
        val auth = payload.optJSONObject("auth")
        val recommendedServerEndpoint =
            connection?.optString("recommendedServerEndpoint")?.trim().orEmpty().ifEmpty {
                normalizeEndpoint(baseUrl)
            }
        val localAuthPage = payload.optString("localAuthPage").trim().ifEmpty { null }
        return ConnectionTargetResponse(
            bridgeId = connection?.optString("bridgeId")?.trim()?.ifEmpty { null },
            recommendedServerEndpoint = recommendedServerEndpoint,
            authMode = if (auth?.optString("mode") == "device-token") "device-token" else "none",
            localAuthPage = localAuthPage,
        )
    }

    fun completeDevicePairingByBaseUrl(
        baseUrl: String,
        pairingCode: String,
        authToken: String?,
    ): PairingResponse {
        val body = JSONObject()
            .put("code", pairingCode.trim())
            .put("deviceName", "Codex Mobile (Android Native Host)")
            .toString()
        val responseText = requestJson(
            url = "${normalizeEndpoint(baseUrl)}/auth/pair/complete",
            method = "POST",
            authToken = authToken,
            requestBody = body,
        )
        val payload = JSONObject(responseText)
        val accessToken = payload.optString("accessToken").trim()
        if (accessToken.isEmpty()) {
            throw IllegalStateException("Pairing response did not include an access token.")
        }
        return PairingResponse(
            accessToken = accessToken,
            approved = payload.optBoolean("approved", false),
        )
    }

    fun performDirectRpcByBaseUrl(
        baseUrl: String,
        method: String,
        params: JSONObject?,
        authToken: String?,
    ): JSONObject {
        val body = JSONObject()
            .put("method", method)
            .put("params", params ?: JSONObject())
            .toString()
        val responseText = requestJson(
            url = "${normalizeEndpoint(baseUrl)}/codex-mobile/rpc",
            method = "POST",
            authToken = authToken,
            requestBody = body,
        )
        val payload = JSONObject(responseText)
        if (!payload.optBoolean("ok", false)) {
            throw IllegalStateException(
                payload.optString("error").trim().ifEmpty { "Direct RPC failed." },
            )
        }
        return payload.optJSONObject("result") ?: JSONObject()
    }

    fun fetchGlobalStateSnapshotByBaseUrl(
        baseUrl: String,
        authToken: String?,
    ): JSONObject {
        var lastError: Exception? = null
        repeat(6) { attempt ->
            try {
                return performDirectRpcByBaseUrl(
                    baseUrl = baseUrl,
                    method = "get-global-state-snapshot",
                    params = JSONObject(),
                    authToken = authToken,
                )
            } catch (error: Exception) {
                lastError = error
                if (attempt == 5) {
                    throw error
                }
                Thread.sleep(250L)
            }
        }
        throw lastError ?: IllegalStateException("Failed to fetch bridge global state snapshot.")
    }

    fun isBridgeReadyByBaseUrl(
        baseUrl: String,
        authToken: String?,
    ): Boolean {
        return probeBridgeReadyByBaseUrl(baseUrl, authToken).ready
    }

    fun probeBridgeReadyByBaseUrl(
        baseUrl: String,
        authToken: String?,
    ): BridgeReadyProbe {
        return try {
            requestJson(
                url = "${normalizeEndpoint(baseUrl)}/readyz",
                authToken = authToken,
            )
            BridgeReadyProbe(ready = true)
        } catch (error: Exception) {
            BridgeReadyProbe(
                ready = false,
                errorMessage = error.message?.trim().orEmpty().ifEmpty { null },
            )
        }
    }

    private fun requestJson(
        url: String,
        method: String = "GET",
        authToken: String? = null,
        requestBody: String? = null,
    ): String {
        val connection = URL(url).openConnection() as HttpURLConnection
        connection.connectTimeout = DEFAULT_TIMEOUT_MS
        connection.readTimeout = DEFAULT_TIMEOUT_MS
        connection.requestMethod = method
        connection.setRequestProperty("Accept", "application/json")
        if (!authToken.isNullOrBlank()) {
            connection.setRequestProperty("Authorization", "Bearer ${authToken.trim()}")
        }
        if (requestBody != null) {
            connection.doOutput = true
            connection.setRequestProperty("Content-Type", "application/json")
            connection.outputStream.use { stream ->
                stream.write(requestBody.toByteArray())
            }
        }

        val responseCode = connection.responseCode
        val stream = if (responseCode in 200..299) {
            connection.inputStream
        } else {
            connection.errorStream ?: connection.inputStream
        }
        val text = BufferedReader(InputStreamReader(stream)).use { reader ->
            buildString {
                while (true) {
                    val line = reader.readLine() ?: break
                    append(line)
                }
            }
        }
        if (responseCode !in 200..299) {
            val message = runCatching {
                JSONObject(text).optString("error").trim()
            }.getOrNull().orEmpty()
            if (message.isNotEmpty()) {
                throw IllegalStateException(message)
            }
            val fallbackText = text.trim()
            if (fallbackText.isNotEmpty()) {
                throw IllegalStateException(fallbackText)
            }
            throw IllegalStateException("Request failed with HTTP $responseCode.")
        }
        return text
    }
}
