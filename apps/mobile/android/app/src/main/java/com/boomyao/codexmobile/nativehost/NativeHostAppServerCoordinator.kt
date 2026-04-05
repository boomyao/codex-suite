package com.boomyao.codexmobile.nativehost

import org.json.JSONObject
import java.util.concurrent.ConcurrentHashMap
import kotlin.concurrent.thread

class NativeHostAppServerCoordinator(
    private val hostId: String,
    private val sendHostMessage: (JSONObject) -> Unit,
    private val integrateDirectRpcResult: (String, JSONObject) -> Unit,
    private val logWarning: (String, Throwable?) -> Unit,
    private val onConnectionLost: (Throwable?) -> Unit,
) {
    private val pendingTurnCompletions = ConcurrentHashMap<String, String>()
    private val activeTurnReconciliations = ConcurrentHashMap.newKeySet<String>()
    private val emittedTurnCompletions = ConcurrentHashMap<String, String>()

    private var appServerWebSocketClient: AppServerWebSocketClient? = null

    fun reset() {
        appServerWebSocketClient?.close()
        appServerWebSocketClient = null
    }

    fun performRequest(
        loadTarget: BridgeLoadTarget,
        authToken: String?,
        method: String,
        params: JSONObject,
    ): JSONObject {
        return getAppServerWebSocketClient(loadTarget, authToken).request(method, params)
    }

    fun rememberPendingTurnCompletion(threadId: String?, turnId: String?) {
        if (threadId.isNullOrBlank() || turnId.isNullOrBlank()) {
            return
        }
        pendingTurnCompletions[threadId] = turnId
    }

    fun scheduleTurnCompletionFallback(
        threadId: String,
        turnId: String?,
        loadTarget: BridgeLoadTarget,
        authToken: String?,
        delayMs: Long = 1500L,
    ) {
        thread(name = "codex-turn-fallback-$threadId") {
            try {
                Thread.sleep(delayMs)
                if (!hasPendingTurnCompletion(threadId, turnId)) {
                    return@thread
                }
                reconcileCompletedThreadState(threadId, loadTarget, authToken)
            } catch (_: InterruptedException) {
                Thread.currentThread().interrupt()
            }
        }
    }

    fun reconcileThreadSnapshotIfNeeded(
        method: String,
        result: JSONObject,
        loadTarget: BridgeLoadTarget,
        authToken: String?,
    ) {
        if (method != "thread/read" && method != "thread/resume" && method != "thread/start") {
            return
        }
        val threadObject = result.optJSONObject("thread") ?: return
        val threadId = threadObject.optString("id").trim()
        if (threadId.isEmpty()) {
            return
        }
        val statusObject = threadObject.optJSONObject("status")
        if (!isThreadStatusTerminal(statusObject) && method != "thread/start") {
            return
        }
        reconcileCompletedThreadState(threadId, loadTarget, authToken)
    }

    private fun webSocketUrlForBaseUrl(baseUrl: String): String {
        val normalized = BridgeApi.normalizeEndpoint(baseUrl)
        return when {
            normalized.startsWith("https://") -> "wss://${normalized.removePrefix("https://")}"
            normalized.startsWith("http://") -> "ws://${normalized.removePrefix("http://")}"
            else -> normalized
        }
    }

    private fun appServerAuthHeaders(loadTarget: BridgeLoadTarget, authToken: String?): Map<String, String> {
        if (loadTarget.usesLocalProxy || authToken.isNullOrBlank()) {
            return emptyMap()
        }
        return mapOf("Authorization" to "Bearer $authToken")
    }

    private fun getAppServerWebSocketClient(
        loadTarget: BridgeLoadTarget,
        authToken: String?,
    ): AppServerWebSocketClient {
        val webSocketUrl = webSocketUrlForBaseUrl(loadTarget.baseUrl)
        val headers = appServerAuthHeaders(loadTarget, authToken)
        val existing = appServerWebSocketClient
        if (existing != null && existing.matches(webSocketUrl, headers)) {
            return existing
        }

        existing?.close()
        return AppServerWebSocketClient(
            webSocketUrl = webSocketUrl,
            headers = headers,
            onNotification = { method, params ->
                handleAppServerNotification(
                    method = method,
                    params = params,
                    loadTarget = loadTarget,
                    authToken = authToken,
                )
            },
            onLog = logWarning,
            onDisconnected = { error ->
                appServerWebSocketClient = null
                thread(name = "codex-app-server-disconnect-probe") {
                    val probe = BridgeApi.probeBridgeReadyByBaseUrl(loadTarget.baseUrl, authToken)
                    if (!probe.ready) {
                        onConnectionLost(error)
                    }
                }
            },
        ).also {
            appServerWebSocketClient = it
        }
    }

    private fun handleAppServerNotification(
        method: String,
        params: JSONObject,
        loadTarget: BridgeLoadTarget,
        authToken: String?,
    ) {
        when (method) {
            "thread/status/changed" -> {
                val threadId = params.optString("threadId").trim()
                val status = params.optJSONObject("status")
                if (
                    threadId.isNotEmpty() &&
                    status != null &&
                    isThreadStatusTerminal(status) &&
                    hasPendingTurnCompletion(threadId)
                ) {
                    reconcileCompletedThreadState(threadId, loadTarget, authToken)
                }
            }
            "turn/started" -> {
                val threadId = params.optString("threadId").trim()
                val turnId = params.optJSONObject("turn")?.optString("id")?.trim().orEmpty()
                rememberPendingTurnCompletion(
                    threadId = threadId.ifEmpty { null },
                    turnId = turnId.ifEmpty { null },
                )
            }
            "turn/completed" -> {
                val threadId = params.optString("threadId").trim()
                val turnId = params.optJSONObject("turn")?.optString("id")?.trim().orEmpty()
                clearPendingTurnCompletion(
                    threadId = threadId.ifEmpty { null },
                    turnId = turnId.ifEmpty { null },
                )
            }
        }

        emitMcpNotification(method, deepCopyJsonObject(params))
    }

    private fun clearPendingTurnCompletion(threadId: String?, turnId: String? = null) {
        if (threadId.isNullOrBlank()) {
            return
        }
        val pendingTurnId = pendingTurnCompletions[threadId] ?: return
        if (!turnId.isNullOrBlank() && pendingTurnId != turnId) {
            return
        }
        pendingTurnCompletions.remove(threadId)
    }

    private fun hasPendingTurnCompletion(threadId: String?, turnId: String? = null): Boolean {
        if (threadId.isNullOrBlank()) {
            return false
        }
        val pendingTurnId = pendingTurnCompletions[threadId] ?: return false
        if (!turnId.isNullOrBlank() && pendingTurnId != turnId) {
            return false
        }
        return true
    }

    private fun isThreadStatusTerminal(status: JSONObject?): Boolean {
        val type = status?.optString("type")?.trim().orEmpty()
        return type == "idle" || type == "systemError"
    }

    private fun emitMcpNotification(method: String, params: JSONObject) {
        sendHostMessage(
            JSONObject()
                .put("type", "mcp-notification")
                .put("hostId", hostId)
                .put("method", method)
                .put("params", params)
                .put(
                    "notification",
                    JSONObject()
                        .put("method", method)
                        .put("params", params),
                ),
        )
    }

    private fun emitSyntheticTurnCompletion(threadId: String, turn: JSONObject) {
        val turnId = turn.optString("id").trim()
        if (turnId.isEmpty()) {
            return
        }
        if (emittedTurnCompletions[threadId] == turnId) {
            return
        }

        emitMcpNotification(
            "turn/started",
            JSONObject()
                .put("threadId", threadId)
                .put("turn", deepCopyJsonObject(turn)),
        )

        val items = turn.optJSONArray("items")
        if (items != null) {
            for (index in 0 until items.length()) {
                val item = items.optJSONObject(index) ?: continue
                if (item.optString("type") == "userMessage") {
                    continue
                }
                val itemPayload =
                    JSONObject()
                        .put("threadId", threadId)
                        .put("turnId", turnId)
                        .put("item", deepCopyJsonObject(item))
                emitMcpNotification("item/started", itemPayload)
                emitMcpNotification("item/completed", itemPayload)
            }
        }

        emitMcpNotification(
            "turn/completed",
            JSONObject()
                .put("threadId", threadId)
                .put("turn", deepCopyJsonObject(turn)),
        )
        emittedTurnCompletions[threadId] = turnId
        clearPendingTurnCompletion(threadId, turnId)
    }

    private fun reconcileCompletedThreadState(
        threadId: String,
        loadTarget: BridgeLoadTarget,
        authToken: String?,
    ) {
        if (!activeTurnReconciliations.add(threadId)) {
            return
        }
        thread(name = "codex-turn-reconcile-$threadId") {
            try {
                repeat(90) { attempt ->
                    val result =
                        BridgeApi.performDirectRpcByBaseUrl(
                            baseUrl = loadTarget.baseUrl,
                            method = "thread/read",
                            params = JSONObject().put("threadId", threadId).put("includeTurns", true),
                            authToken = authToken,
                        )
                    integrateDirectRpcResult("thread/read", result)
                    val threadObject = result.optJSONObject("thread")
                    val statusObject = threadObject?.optJSONObject("status")
                    if (threadObject != null && statusObject != null && isThreadStatusTerminal(statusObject)) {
                        val turns = threadObject.optJSONArray("turns")
                        val latestTurn = turns?.optJSONObject(turns.length() - 1)
                        val latestTurnId = latestTurn?.optString("id")?.trim().orEmpty()
                        val latestTurnStatus = latestTurn?.optString("status")?.trim().orEmpty()
                        if (
                            latestTurn != null &&
                            latestTurnId.isNotEmpty() &&
                            latestTurnStatus.isNotEmpty() &&
                            latestTurnStatus != "inProgress"
                        ) {
                            emitSyntheticTurnCompletion(threadId, latestTurn)
                        }
                        emitMcpNotification(
                            "thread/status/changed",
                            JSONObject()
                                .put("threadId", threadId)
                                .put("status", deepCopyJsonObject(statusObject)),
                        )
                        return@thread
                    }
                    if (attempt < 89) {
                        Thread.sleep(if (attempt < 5) 350L else 800L)
                    }
                }
            } catch (error: Exception) {
                logWarning("failed to reconcile thread completion for $threadId", error)
            } finally {
                activeTurnReconciliations.remove(threadId)
            }
        }
    }
}
