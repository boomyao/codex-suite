package com.boomyao.codexmobile.shared

object BridgeEndpoint {
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
}
