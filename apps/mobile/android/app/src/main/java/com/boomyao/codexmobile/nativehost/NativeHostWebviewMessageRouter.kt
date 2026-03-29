package com.boomyao.codexmobile.nativehost

import android.util.Log
import org.json.JSONArray
import org.json.JSONObject

class NativeHostWebviewMessageRouter(
    private val activeProfileProvider: () -> BridgeProfile?,
    private val setStatus: (String) -> Unit,
    private val sendPersistedAtomSync: () -> Unit,
    private val sendHostMessage: (JSONObject) -> Unit,
    private val sendBridgeResponse: (String, JSONObject?) -> Unit,
    private val sendWorkerMessage: (String, JSONObject) -> Unit,
    private val openUrl: (String) -> Unit,
    private val broadcastSharedObjectUpdate: (String, Any?) -> Unit,
    private val sharedObjects: MutableMap<String, Any?>,
    private val sharedObjectSubscribers: MutableMap<String, Int>,
    private val persistedAtomStateProvider: () -> JSONObject,
    private val setPersistedAtomState: (JSONObject) -> Unit,
    private val createDefaultPersistedAtomState: () -> JSONObject,
    private val toJsonCompatible: (Any?) -> Any,
    private val updateWorkspaceRoots: (List<String>) -> Unit,
    private val setActiveWorkspaceRoot: (String) -> Unit,
    private val renameWorkspaceRoot: (String, String) -> Boolean,
    private val onFetchMessage: (JSONObject) -> Unit,
    private val onMcpRequestMessage: (JSONObject) -> Unit,
) {
    fun handleEnvelope(rawMessage: String) {
        try {
            val envelope = JSONObject(rawMessage)
            if (!envelope.optBoolean("__codexMobile", false)) {
                return
            }
            when (envelope.optString("kind")) {
                "preload-ready" -> {
                    Log.d("CodexMobile", "preload ready ${envelope.opt("payload")}")
                    activeProfileProvider()?.let { profile ->
                        setStatus("Connected to ${profile.serverEndpoint}")
                    }
                }
                "console" -> Log.d("CodexMobile", "renderer ${envelope.opt("payload")}")
                "bridge-send-message" -> {
                    val payload = envelope.optJSONObject("payload") ?: return
                    Log.d(
                        "CodexMobile",
                        "bridge-send-message ${payload.optString("type")} ${payload.optJSONObject("request")?.optString("method")}",
                    )
                    handleRendererMessage(payload)
                }
                "bridge-send-worker-message" -> {
                    val payload = envelope.optJSONObject("payload") ?: return
                    Log.d("CodexMobile", "bridge-send-worker-message ${payload.optString("workerId")}")
                    handleWorkerBridgeMessage(payload)
                }
                "bridge-show-context-menu", "bridge-show-application-menu" -> {
                    val payload = envelope.optJSONObject("payload") ?: return
                    val requestId = payload.optString("requestId").trim()
                    if (requestId.isNotEmpty()) {
                        sendBridgeResponse(requestId, null)
                    }
                }
            }
        } catch (error: Exception) {
            Log.w("CodexMobile", "failed to parse webview envelope", error)
        }
    }

    private fun handleRendererReady() {
        Log.d("CodexMobile", "renderer ready")
        sendPersistedAtomSync()
        sendHostMessage(JSONObject().put("type", "custom-prompts-updated").put("prompts", JSONArray()))
        sendHostMessage(JSONObject().put("type", "app-update-ready-changed").put("isUpdateReady", false))
        sendHostMessage(JSONObject().put("type", "electron-window-focus-changed").put("isFocused", true))
        sendHostMessage(JSONObject().put("type", "workspace-root-options-updated"))
        sendHostMessage(JSONObject().put("type", "active-workspace-roots-updated"))
        sharedObjects.forEach { (key, value) ->
            broadcastSharedObjectUpdate(key, value)
        }
    }

    private fun handleRendererMessage(message: JSONObject) {
        Log.d("CodexMobile", "renderer message ${message.optString("type")}")
        when (message.optString("type")) {
            "ready" -> handleRendererReady()
            "persisted-atom-sync-request" -> sendPersistedAtomSync()
            "persisted-atom-update" -> {
                val key = message.optString("key").trim()
                if (key.isEmpty()) {
                    return
                }
                val deleted = message.optBoolean("deleted", false)
                val persistedAtomState = persistedAtomStateProvider()
                if (deleted) {
                    persistedAtomState.remove(key)
                } else {
                    persistedAtomState.put(key, toJsonCompatible(message.opt("value")))
                }
                sendHostMessage(
                    JSONObject()
                        .put("type", "persisted-atom-updated")
                        .put("key", key)
                        .put("value", if (deleted) JSONObject.NULL else toJsonCompatible(message.opt("value")))
                        .put("deleted", deleted),
                )
            }
            "persisted-atom-reset" -> {
                setPersistedAtomState(createDefaultPersistedAtomState())
                sendPersistedAtomSync()
            }
            "shared-object-subscribe" -> {
                val key = message.optString("key").trim()
                if (key.isEmpty()) {
                    return
                }
                sharedObjectSubscribers[key] = (sharedObjectSubscribers[key] ?: 0) + 1
                broadcastSharedObjectUpdate(key, sharedObjects[key])
            }
            "shared-object-unsubscribe" -> {
                val key = message.optString("key").trim()
                if (key.isEmpty()) {
                    return
                }
                val count = sharedObjectSubscribers[key] ?: 0
                if (count <= 1) {
                    sharedObjectSubscribers.remove(key)
                } else {
                    sharedObjectSubscribers[key] = count - 1
                }
            }
            "shared-object-set" -> {
                val key = message.optString("key").trim()
                if (key.isEmpty()) {
                    return
                }
                sharedObjects[key] = message.opt("value")
                broadcastSharedObjectUpdate(key, sharedObjects[key])
            }
            "electron-window-focus-request" -> sendHostMessage(
                JSONObject().put("type", "electron-window-focus-changed").put("isFocused", true),
            )
            "open-in-browser" -> {
                val url = message.optString("url").trim()
                if (url.isNotEmpty()) {
                    runCatching {
                        openUrl(url)
                    }
                }
            }
            "terminal-create", "terminal-attach" -> {
                val sessionId = message.optString("sessionId").trim()
                if (sessionId.isNotEmpty()) {
                    sendHostMessage(
                        JSONObject()
                            .put("type", "terminal-attached")
                            .put("sessionId", sessionId)
                            .put("cwd", message.optString("cwd"))
                            .put("shell", "zsh"),
                    )
                    sendHostMessage(
                        JSONObject()
                            .put("type", "terminal-init-log")
                            .put("sessionId", sessionId)
                            .put("log", ""),
                    )
                }
            }
            "terminal-close" -> {
                val sessionId = message.optString("sessionId").trim()
                if (sessionId.isNotEmpty()) {
                    sendHostMessage(
                        JSONObject()
                            .put("type", "terminal-exit")
                            .put("sessionId", sessionId)
                            .put("code", 0)
                            .put("signal", JSONObject.NULL),
                    )
                }
            }
            "workspace-root-option-picked" -> {
                val root = message.optString("root").trim()
                if (root.isNotEmpty()) {
                    setActiveWorkspaceRoot(root)
                }
            }
            "electron-update-workspace-root-options" -> {
                val roots = mutableListOf<String>()
                val values = message.optJSONArray("roots")
                if (values != null) {
                    for (index in 0 until values.length()) {
                        roots.add(values.optString(index))
                    }
                }
                updateWorkspaceRoots(roots)
            }
            "electron-rename-workspace-root-option" -> {
                if (renameWorkspaceRoot(message.optString("root"), message.optString("label").trim())) {
                    sendHostMessage(JSONObject().put("type", "workspace-root-options-updated"))
                }
            }
            "electron-set-active-workspace-root" -> {
                val root = message.optString("root").trim()
                if (root.isNotEmpty()) {
                    setActiveWorkspaceRoot(root)
                }
            }
            "fetch" -> onFetchMessage(message)
            "cancel-fetch" -> Unit
            "fetch-stream" -> {
                val requestId = message.optString("requestId").trim()
                if (requestId.isNotEmpty()) {
                    sendHostMessage(
                        JSONObject()
                            .put("type", "fetch-stream-error")
                            .put("requestId", requestId)
                            .put("error", "Streaming fetch is not supported in Codex Mobile yet."),
                    )
                }
            }
            "cancel-fetch-stream", "bridge-unimplemented", "view-focused", "power-save-blocker-set",
            "electron-set-window-mode", "electron-request-microphone-permission",
            "electron-set-badge-count", "desktop-notification-hide", "desktop-notification-show",
            "install-app-update", "open-debug-window", "open-thread-overlay",
            "thread-stream-state-changed", "set-telemetry-user", "toggle-trace-recording",
            "hotkey-window-enabled-changed", "electron-desktop-features-changed" -> Unit
            "log-message" -> {
                Log.d("CodexMobile", "renderer log-message ${message.opt("message")}")
            }
            "mcp-request" -> onMcpRequestMessage(message)
        }
    }

    private fun handleWorkerBridgeMessage(payload: JSONObject) {
        val workerId = payload.optString("workerId").trim()
        val workerPayload = payload.optJSONObject("payload") ?: return
        if (workerId.isEmpty() || workerPayload.optString("type") != "worker-request") {
            return
        }
        val request = workerPayload.optJSONObject("request") ?: return
        val requestId = request.optString("id").trim()
        val method = request.optString("method").trim()
        if (requestId.isEmpty() || method.isEmpty()) {
            return
        }
        val response = JSONObject()
            .put("type", "worker-response")
            .put("workerId", workerId)
            .put(
                "response",
                JSONObject()
                    .put("id", requestId)
                    .put("method", method)
                    .put(
                        "result",
                        if (workerId == "git" && method == "stable-metadata") {
                            JSONObject()
                                .put("type", "ok")
                                .put(
                                    "value",
                                    JSONObject()
                                        .put("cwd", "")
                                        .put("root", "")
                                        .put("commonDir", "")
                                        .put("gitDir", JSONObject.NULL)
                                        .put("branch", JSONObject.NULL)
                                        .put("upstreamBranch", JSONObject.NULL)
                                        .put("headSha", JSONObject.NULL)
                                        .put("originUrl", JSONObject.NULL)
                                        .put("isRepository", false)
                                        .put("isWorktree", false)
                                        .put("worktreeRoot", ""),
                                )
                        } else {
                            JSONObject()
                                .put("type", "error")
                                .put(
                                    "error",
                                    JSONObject().put("message", "Unsupported worker request: $workerId/$method"),
                                )
                        },
                    ),
            )
        sendWorkerMessage(workerId, response)
    }

}
