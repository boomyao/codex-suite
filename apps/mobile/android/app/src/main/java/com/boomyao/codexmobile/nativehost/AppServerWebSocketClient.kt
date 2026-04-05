package com.boomyao.codexmobile.nativehost

import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import org.json.JSONObject
import java.io.IOException
import java.util.concurrent.CompletableFuture
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicInteger

class AppServerWebSocketClient(
    private val webSocketUrl: String,
    private val headers: Map<String, String>,
    private val onNotification: (String, JSONObject) -> Unit,
    private val onLog: (String, Throwable?) -> Unit,
    private val onDisconnected: (Throwable?) -> Unit,
) : WebSocketListener() {
    private val httpClient =
        OkHttpClient.Builder()
            .pingInterval(15, TimeUnit.SECONDS)
            .readTimeout(0, TimeUnit.MILLISECONDS)
            .build()
    private val nextRequestId = AtomicInteger(2)
    private val pendingRequests = ConcurrentHashMap<Int, CompletableFuture<JSONObject>>()
    private val readyFuture = CompletableFuture<Unit>()

    @Volatile
    private var socket: WebSocket? = null

    @Volatile
    private var connectStarted = false

    @Volatile
    private var closeRequested = false

    @Volatile
    private var disconnectNotified = false

    fun matches(url: String, extraHeaders: Map<String, String>): Boolean {
        return webSocketUrl == url && headers == extraHeaders
    }

    fun request(method: String, params: JSONObject): JSONObject {
        ensureConnected()
        readyFuture.get(15, TimeUnit.SECONDS)

        val requestId = nextRequestId.getAndIncrement()
        val resultFuture = CompletableFuture<JSONObject>()
        pendingRequests[requestId] = resultFuture

        val payload =
            JSONObject()
                .put("jsonrpc", "2.0")
                .put("id", requestId)
                .put("method", method)
                .put("params", JSONObject(params.toString()))

        val currentSocket = socket ?: throw IOException("App server websocket is not connected.")
        if (!currentSocket.send(payload.toString())) {
            pendingRequests.remove(requestId)
            val error = IOException("Failed to write app server websocket request.")
            notifyDisconnected(error)
            throw error
        }

        return resultFuture.get(10, TimeUnit.MINUTES)
    }

    fun close() {
        closeRequested = true
        failAll(IOException("App server websocket closed."))
        socket?.close(1000, "closing")
        socket = null
        httpClient.dispatcher.executorService.shutdown()
        httpClient.connectionPool.evictAll()
    }

    override fun onOpen(webSocket: WebSocket, response: Response) {
        socket = webSocket
        val initialize =
            JSONObject()
                .put("jsonrpc", "2.0")
                .put("id", 1)
                .put("method", "initialize")
                .put(
                    "params",
                    JSONObject()
                        .put(
                            "clientInfo",
                            JSONObject()
                                .put("name", "codex_mobile_host")
                                .put("title", "Codex Mobile Host")
                                .put("version", "0.1.0"),
                        )
                        .put("capabilities", JSONObject().put("experimentalApi", true)),
                )
        webSocket.send(initialize.toString())
    }

    override fun onMessage(webSocket: WebSocket, text: String) {
        val payload = JSONObject(text)
        if (payload.has("id")) {
            handleResponse(webSocket, payload)
            return
        }

        val method = payload.optString("method").trim()
        if (method.isEmpty()) {
            return
        }
        val params = payload.optJSONObject("params") ?: JSONObject()
        onNotification(method, JSONObject(params.toString()))
    }

    override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
        val error = IOException("App server websocket closed: $code $reason")
        failAll(error)
        socket = null
        notifyDisconnected(error)
    }

    override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
        onLog("app server websocket failure", t)
        failAll(t)
        socket = null
        notifyDisconnected(t)
    }

    private fun ensureConnected() {
        if (socket != null || connectStarted) {
            return
        }

        synchronized(this) {
            if (socket != null || connectStarted) {
                return
            }
            connectStarted = true
            val requestBuilder = Request.Builder().url(webSocketUrl)
            headers.forEach { (key, value) ->
                requestBuilder.addHeader(key, value)
            }
            httpClient.newWebSocket(requestBuilder.build(), this)
        }
    }

    private fun handleResponse(webSocket: WebSocket, payload: JSONObject) {
        val id = payload.optInt("id", -1)
        if (id == 1) {
            if (payload.has("error")) {
                val message =
                    payload.optJSONObject("error")
                        ?.optString("message")
                        ?.trim()
                        .orEmpty()
                        .ifEmpty { "Failed to initialize app server websocket." }
                readyFuture.completeExceptionally(IOException(message))
                return
            }
            webSocket.send(
                JSONObject()
                    .put("jsonrpc", "2.0")
                    .put("method", "initialized")
                    .put("params", JSONObject())
                    .toString(),
            )
            readyFuture.complete(Unit)
            return
        }

        val future = pendingRequests.remove(id) ?: return
        if (payload.has("error")) {
            val message =
                payload.optJSONObject("error")
                    ?.optString("message")
                    ?.trim()
                    .orEmpty()
                    .ifEmpty { "App server request failed." }
            future.completeExceptionally(IOException(message))
            return
        }

        val result = payload.optJSONObject("result") ?: JSONObject()
        future.complete(JSONObject(result.toString()))
    }

    private fun failAll(error: Throwable) {
        if (!readyFuture.isDone) {
            readyFuture.completeExceptionally(error)
        }
        val iterator = pendingRequests.entries.iterator()
        while (iterator.hasNext()) {
            val entry = iterator.next()
            iterator.remove()
            entry.value.completeExceptionally(error)
        }
    }

    private fun notifyDisconnected(error: Throwable?) {
        if (closeRequested || disconnectNotified) {
            return
        }

        synchronized(this) {
            if (closeRequested || disconnectNotified) {
                return
            }
            disconnectNotified = true
        }
        onDisconnected(error)
    }
}
