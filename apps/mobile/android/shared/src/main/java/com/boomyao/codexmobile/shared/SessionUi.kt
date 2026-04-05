package com.boomyao.codexmobile.shared

import java.net.URI

object SessionUi {
    fun profileAvatar(profile: BridgeProfile): String {
        val label = displayProfileLabel(profile)
        val tokens =
            label
                .split('-', '_', '.', ' ')
                .map(String::trim)
                .filter(String::isNotEmpty)
        val preferredToken = tokens.lastOrNull() ?: label
        return preferredToken.firstOrNull()?.uppercaseChar()?.toString() ?: "C"
    }

    fun displayProfileLabel(profile: BridgeProfile): String {
        val normalizedEndpoint = BridgeEndpoint.normalizeEndpoint(profile.serverEndpoint)
        val rawName = profile.name.trim()
        if (rawName.isNotEmpty() && rawName != normalizedEndpoint && !rawName.startsWith("http")) {
            val candidate =
                if ('.' in rawName && ' ' !in rawName) {
                    rawName.substringBefore('.')
                } else {
                    rawName
                }
            return compactLabel(candidate)
        }
        val host = runCatching { URI(normalizedEndpoint).host.orEmpty() }.getOrDefault("").trim()
        val candidate =
            host.substringBefore('.').ifBlank {
                normalizedEndpoint
                    .removePrefix("https://")
                    .removePrefix("http://")
                    .removePrefix("ws://")
                    .removePrefix("wss://")
            }
        return compactLabel(candidate)
    }

    fun displayProfileDetail(profile: BridgeProfile): String {
        val normalizedEndpoint = BridgeEndpoint.normalizeEndpoint(profile.serverEndpoint)
        val uri = runCatching { URI(normalizedEndpoint) }.getOrNull()
        val host = (uri?.host ?: "").trim()
        val port = uri?.port ?: -1
        val portSuffix = if (port > 0) ":$port" else ""
        if (host.isNotEmpty()) {
            val remainder = host.substringAfter('.', "")
            val candidate = if (remainder.isNotBlank()) remainder else host
            return compactLabel(candidate + portSuffix, maxLength = 28)
        }
        return compactLabel(
            normalizedEndpoint
                .removePrefix("https://")
                .removePrefix("http://")
                .removePrefix("ws://")
                .removePrefix("wss://"),
            maxLength = 28,
        )
    }

    fun compactLabel(value: String, maxLength: Int = 28): String {
        val trimmed = value.trim()
        return if (trimmed.length <= maxLength) {
            trimmed
        } else {
            trimmed.take(maxLength - 1).trimEnd() + "\u2026"
        }
    }
}
