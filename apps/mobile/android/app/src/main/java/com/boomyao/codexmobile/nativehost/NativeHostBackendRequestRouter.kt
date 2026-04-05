package com.boomyao.codexmobile.nativehost

import android.util.Log
import org.json.JSONArray
import org.json.JSONObject
import java.io.BufferedReader
import java.io.InputStreamReader
import java.net.HttpURLConnection
import java.net.URL
import kotlin.concurrent.thread

class NativeHostBackendRequestRouter(
    private val hostId: String,
    private val activeProfileProvider: () -> BridgeProfile?,
    private val activeLoadTargetProvider: () -> BridgeLoadTarget?,
    private val resolveBridgeLoadTarget: (BridgeProfile) -> BridgeLoadTarget,
    private val resolveFetchMethodPayload: (String, JSONObject?) -> Any,
    private val unhandledLocalMethod: Any,
    private val sendHostMessage: (JSONObject) -> Unit,
    private val performAppServerMcpRequest: (BridgeLoadTarget, String?, String, JSONObject) -> JSONObject,
    private val resetAppServerWebSocketClient: () -> Unit,
    private val integrateDirectRpcResult: (String, JSONObject) -> Unit,
    private val rememberPendingTurnCompletion: (String?, String?) -> Unit,
    private val scheduleTurnCompletionFallback: (String, String?, BridgeLoadTarget, String?) -> Unit,
    private val reconcileThreadSnapshotIfNeeded: (String, JSONObject, BridgeLoadTarget, String?) -> Unit,
    private val onBridgeConnectionIssue: (Throwable) -> Unit,
    private val normalizeErrorMessage: (Throwable) -> String,
) {
    fun handleFetchMessage(message: JSONObject) {
        val requestId = message.optString("requestId").trim()
        val rawUrl = message.optString("url").trim()
        if (requestId.isEmpty() || rawUrl.isEmpty()) {
            return
        }
        val profile = activeProfileProvider() ?: return
        val loadTarget = activeLoadTargetProvider() ?: resolveBridgeLoadTarget(profile)
        thread {
            try {
                val method = resolveRequestMethodName(rawUrl)
                if (method != null) {
                    val body = parseJsonBodyObject(message.optString("body"))
                    val localResult = resolveFetchMethodPayload(method, body)
                    if (localResult !== unhandledLocalMethod) {
                        sendHostMessage(buildFetchSuccessResponse(requestId, localResult, 200, JSONObject()))
                        return@thread
                    }
                    val result = BridgeApi.performDirectRpcByBaseUrl(
                        baseUrl = loadTarget.baseUrl,
                        method = method,
                        params = body ?: JSONObject(),
                        authToken = resolveRequestAuthToken(loadTarget, profile),
                    )
                    integrateDirectRpcResult(method, result)
                    sendHostMessage(buildFetchSuccessResponse(requestId, result, 200, JSONObject()))
                    return@thread
                }

                val proxyUrl = resolveServerFetchUrl(rawUrl, loadTarget.baseUrl)
                if (proxyUrl == null) {
                    sendHostMessage(
                        buildFetchErrorResponse(
                            requestId = requestId,
                            error = "Unsupported fetch URL: $rawUrl",
                            status = 501,
                        ),
                    )
                    return@thread
                }

                val proxiedResponse = proxyHttpFetch(
                    url = proxyUrl,
                    method = message.optString("method"),
                    requestBody = message.optString("body").takeIf { it.isNotBlank() },
                    requestHeaders = message.optJSONObject("headers"),
                    authToken = resolveRequestAuthToken(loadTarget, profile),
                )
                sendHostMessage(
                    buildFetchSuccessResponse(
                        requestId = requestId,
                        body = proxiedResponse.body,
                        status = proxiedResponse.status,
                        headers = proxiedResponse.headers,
                    ),
                )
            } catch (error: Exception) {
                Log.w("CodexMobile", "fetch handler failed for $rawUrl", error)
                if (resolveRequestMethodName(rawUrl) != null) {
                    onBridgeConnectionIssue(error)
                }
                sendHostMessage(
                    buildFetchErrorResponse(
                        requestId = requestId,
                        error = normalizeErrorMessage(error),
                        status = 500,
                    ),
                )
            }
        }
    }

    fun handleMcpRequestMessage(message: JSONObject) {
        val request = message.optJSONObject("request") ?: return
        val requestId = request.opt("id")?.toString()?.trim().orEmpty()
        val method = request.optString("method").trim()
        if (requestId.isEmpty() || method.isEmpty()) {
            return
        }
        Log.d("CodexMobile", "mcp request $method id=$requestId")
        val profile = activeProfileProvider() ?: return
        val loadTarget = activeLoadTargetProvider() ?: resolveBridgeLoadTarget(profile)
        thread {
            try {
                val requestParams = request.optJSONObject("params")
                val localResult = resolveFetchMethodPayload(method, requestParams)
                if (localResult !== unhandledLocalMethod) {
                    val localObject =
                        when (localResult) {
                            is JSONObject -> localResult
                            else -> JSONObject.wrap(localResult) as? JSONObject ?: JSONObject()
                        }
                    sendHostMessage(
                        JSONObject()
                            .put("type", "mcp-response")
                            .put("hostId", hostId)
                            .put("id", request.opt("id"))
                            .put("result", localObject)
                            .put("message", JSONObject().put("id", request.opt("id")).put("result", localObject))
                            .put("response", JSONObject().put("id", request.opt("id")).put("result", localObject)),
                    )
                    return@thread
                }

                val authToken = resolveRequestAuthToken(loadTarget, profile)
                val nonNullParams = requestParams ?: JSONObject()
                val result =
                    try {
                        performAppServerMcpRequest(loadTarget, authToken, method, nonNullParams)
                    } catch (socketError: Exception) {
                        Log.w(
                            "CodexMobile",
                            "app server websocket request failed; falling back to direct RPC for $method",
                            socketError,
                        )
                        resetAppServerWebSocketClient()
                        BridgeApi.performDirectRpcByBaseUrl(
                            baseUrl = loadTarget.baseUrl,
                            method = method,
                            params = nonNullParams,
                            authToken = authToken,
                        )
                    }
                integrateDirectRpcResult(method, result)
                Log.d("CodexMobile", "mcp response ok $method id=$requestId")
                sendHostMessage(
                    JSONObject()
                        .put("type", "mcp-response")
                        .put("hostId", hostId)
                        .put("id", request.opt("id"))
                        .put("result", result)
                        .put("message", JSONObject().put("id", request.opt("id")).put("result", result))
                        .put("response", JSONObject().put("id", request.opt("id")).put("result", result)),
                )
                if (method == "turn/start") {
                    val requestThreadId = requestParams?.optString("threadId")?.trim().orEmpty()
                    val responseTurnId = result.optJSONObject("turn")?.optString("id")?.trim().orEmpty()
                    rememberPendingTurnCompletion(
                        requestThreadId.ifEmpty { null },
                        responseTurnId.ifEmpty { null },
                    )
                    if (requestThreadId.isNotEmpty()) {
                        Log.d(
                            "CodexMobile",
                            "schedule turn fallback thread=$requestThreadId turn=$responseTurnId",
                        )
                        scheduleTurnCompletionFallback(
                            requestThreadId,
                            responseTurnId.ifEmpty { null },
                            loadTarget,
                            authToken,
                        )
                    }
                } else {
                    reconcileThreadSnapshotIfNeeded(method, result, loadTarget, authToken)
                }
            } catch (error: Exception) {
                Log.w("CodexMobile", "mcp response error $method id=$requestId", error)
                onBridgeConnectionIssue(error)
                val errorPayload = JSONObject().put("message", error.message ?: "MCP request failed.")
                sendHostMessage(
                    JSONObject()
                        .put("type", "mcp-response")
                        .put("hostId", hostId)
                        .put("id", request.opt("id"))
                        .put("error", errorPayload)
                        .put("message", JSONObject().put("id", request.opt("id")).put("error", errorPayload))
                        .put("response", JSONObject().put("id", request.opt("id")).put("error", errorPayload)),
                )
            }
        }
    }

    private fun resolveRequestAuthToken(loadTarget: BridgeLoadTarget, profile: BridgeProfile): String? {
        return if (loadTarget.usesLocalProxy) {
            null
        } else {
            profile.authToken
        }
    }

    private fun resolveRequestMethodName(rawUrl: String): String? {
        val prefix = "vscode://codex/"
        if (!rawUrl.startsWith(prefix)) {
            return null
        }
        return rawUrl.removePrefix(prefix).substringBefore('?').trim().ifBlank { null }
    }

    private fun resolveServerFetchUrl(rawUrl: String, baseUrl: String): String? {
        val trimmed = rawUrl.trim()
        if (trimmed.isEmpty()) {
            return null
        }
        if (trimmed.startsWith("http://") || trimmed.startsWith("https://")) {
            return trimmed
        }
        return try {
            URL(URL("${BridgeApi.normalizeEndpoint(baseUrl)}/"), trimmed).toString()
        } catch (_: Exception) {
            null
        }
    }

    private fun parseJsonBodyObject(value: String?): JSONObject? {
        if (value == null || value.isBlank()) {
            return null
        }
        return try {
            JSONObject(value)
        } catch (_: Exception) {
            null
        }
    }

    private fun proxyHttpFetch(
        url: String,
        method: String?,
        requestBody: String?,
        requestHeaders: JSONObject?,
        authToken: String?,
    ): HttpProxyResponse {
        val connection = URL(url).openConnection() as HttpURLConnection
        connection.requestMethod = method?.trim()?.uppercase().takeUnless { it.isNullOrBlank() } ?: "GET"
        requestHeaders?.keys()?.forEach { key ->
            val value = requestHeaders.opt(key)
            if (value is String && value.isNotBlank()) {
                connection.setRequestProperty(key, value)
            }
        }
        if (!authToken.isNullOrBlank() && connection.getRequestProperty("Authorization").isNullOrBlank()) {
            connection.setRequestProperty("Authorization", "Bearer ${authToken.trim()}")
        }
        if (!requestBody.isNullOrBlank()) {
            connection.doOutput = true
            if (connection.getRequestProperty("Content-Type").isNullOrBlank()) {
                connection.setRequestProperty("Content-Type", "application/json")
            }
            connection.outputStream.use { stream ->
                stream.write(requestBody.toByteArray())
            }
        }

        val status = connection.responseCode
        val stream = if (status in 200..299) {
            connection.inputStream
        } else {
            connection.errorStream ?: connection.inputStream
        }
        val responseText = BufferedReader(InputStreamReader(stream)).use { reader ->
            buildString {
                while (true) {
                    val line = reader.readLine() ?: break
                    append(line)
                }
            }
        }
        val headers = JSONObject()
        connection.headerFields.forEach { (key, values) ->
            if (key != null && !values.isNullOrEmpty()) {
                headers.put(key, values.joinToString(", "))
            }
        }
        return HttpProxyResponse(
            body = parseJsonValue(responseText),
            status = status,
            headers = headers,
        )
    }

    private fun parseJsonValue(value: String): Any? {
        val trimmed = value.trim()
        if (trimmed.isEmpty()) {
            return null
        }
        return try {
            when {
                trimmed.startsWith("{") -> JSONObject(trimmed)
                trimmed.startsWith("[") -> JSONArray(trimmed)
                else -> trimmed
            }
        } catch (_: Exception) {
            trimmed
        }
    }

    private fun buildFetchSuccessResponse(
        requestId: String,
        body: Any?,
        status: Int,
        headers: JSONObject,
    ): JSONObject {
        return JSONObject()
            .put("type", "fetch-response")
            .put("requestId", requestId)
            .put("responseType", "success")
            .put("status", status)
            .put("headers", headers)
            .put("bodyJsonString", JSONObject.wrap(body)?.toString() ?: "null")
    }

    private fun buildFetchErrorResponse(
        requestId: String,
        error: String,
        status: Int,
    ): JSONObject {
        return JSONObject()
            .put("type", "fetch-response")
            .put("requestId", requestId)
            .put("responseType", "error")
            .put("status", status)
            .put("error", error)
    }
}
