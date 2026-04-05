package com.boomyao.codexmobile.shared

data class BridgeProfile(
    val id: String,
    val bridgeId: String?,
    val name: String,
    val serverEndpoint: String,
    val authToken: String?,
    val lastUsedAtMillis: Long? = null,
)

data class ConnectionTargetResponse(
    val bridgeId: String?,
    val recommendedServerEndpoint: String,
    val authMode: String,
    val localAuthPage: String?,
)

data class PairingResponse(
    val accessToken: String,
    val approved: Boolean,
)

data class BridgeLoadTarget(
    val baseUrl: String,
    val usesLocalProxy: Boolean,
)

sealed class EnrollmentPayload {
    data class Bridge(
        val bridgeId: String?,
        val name: String,
        val serverEndpoint: String,
        val pairingCode: String?,
    ) : EnrollmentPayload()
}
