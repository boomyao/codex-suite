package com.boomyao.codexmobile.nativehost

import android.content.Context
import com.boomyao.codexmobile.R
import kotlin.concurrent.thread

class NativeHostConnectionCoordinator(
    private val context: Context,
    private val profileStore: BridgeProfileStore,
    private val activeProfileProvider: () -> BridgeProfile?,
    private val openBridge: (BridgeProfile) -> Unit,
    private val renderEmptyState: (String) -> Unit,
    private val renderConnectionFailure: (String, String, String) -> Unit,
    private val setStatus: (String) -> Unit,
    private val updateConnectionProgress: (String, String, NativeHostConnectionStage, Boolean) -> Unit,
    private val syncSavedConnectionsState: () -> Unit,
    private val normalizeErrorMessage: (Throwable) -> String,
) {
    fun reloadActiveBridge() {
        val profile = activeProfileProvider()
        if (profile == null) {
            renderEmptyState(context.getString(R.string.native_host_status_idle))
            return
        }
        openBridge(profile)
    }

    fun activateProfile(profile: BridgeProfile) {
        profileStore.setActive(profile.id)
        openBridge(profile)
    }

    fun resetEnrollment(profile: BridgeProfile) {
        val nextProfile = profileStore.remove(profile.id)
        syncSavedConnectionsState()
        if (nextProfile == null) {
            renderEmptyState(context.getString(R.string.native_host_status_idle))
            return
        }
        openBridge(nextProfile)
    }

    fun importEnrollment(rawJson: String) {
        try {
            when (val payload = EnrollmentParser.parse(rawJson)) {
                is EnrollmentPayload.Bridge -> {
                    postConnectionProgress(
                        profileName = payload.name,
                        endpoint = payload.serverEndpoint,
                        stage = NativeHostConnectionStage.PAYLOAD_RECEIVED,
                        requiresPairing = !payload.pairingCode.isNullOrBlank(),
                    )
                    postStatus(context.getString(R.string.native_host_status_payload_received))
                    saveBridgeProfile(
                        bridgeId = payload.bridgeId,
                        name = payload.name,
                        endpoint = payload.serverEndpoint,
                        pairingCode = payload.pairingCode.orEmpty(),
                        existingAuthToken = null,
                        libp2pPeerId = payload.libp2pPeerId,
                    )
                }
            }
        } catch (error: Exception) {
            val message = normalizeErrorMessage(error)
            postStatus(message)
            postToMain {
                renderConnectionFailure("", "", message)
            }
        }
    }

    fun resolveBridgeLoadTarget(profile: BridgeProfile): BridgeLoadTarget {
        // When a libp2p peer ID is available, the mobile app should start the
        // mobileproxy library (built with gomobile) to create a local tunnel.
        // The proxy provides a 127.0.0.1 base URL that this app can use
        // with the existing OkHttp / WebSocket client — no protocol changes.
        //
        // Usage:
        //   val proxy = Mobileproxy.startProxy(profile.libp2pPeerId, "")
        //   return BridgeLoadTarget(baseUrl = proxy.httpBaseURL(), usesLocalProxy = true)
        //
        // For now, fall back to direct HTTP if the proxy library is not linked.
        return resolveBridgeLoadTarget(profile.serverEndpoint)
    }

    private fun saveBridgeProfile(
        bridgeId: String?,
        name: String,
        endpoint: String,
        pairingCode: String,
        existingAuthToken: String?,
        libp2pPeerId: String? = null,
    ) {
        postStatus(context.getString(R.string.native_host_status_preparing_connection))
        thread {
            try {
                val normalizedEndpoint = BridgeApi.normalizeEndpoint(endpoint)
                var authToken = existingAuthToken
                postConnectionProgress(
                    profileName = name,
                    endpoint = normalizedEndpoint,
                    stage = if (pairingCode.isNotBlank()) {
                        NativeHostConnectionStage.PAIRING_DEVICE
                    } else {
                        NativeHostConnectionStage.OPENING_WORKSPACE
                    },
                    requiresPairing = pairingCode.isNotBlank(),
                )

                var connectionTarget = resolveBridgeLoadTarget(normalizedEndpoint)
                var connection = BridgeApi.fetchConnectionTargetByBaseUrl(
                    baseUrl = connectionTarget.baseUrl,
                    authToken = authToken,
                )
                if (pairingCode.isNotBlank()) {
                    postConnectionProgress(
                        profileName = name,
                        endpoint = normalizedEndpoint,
                        stage = NativeHostConnectionStage.PAIRING_DEVICE,
                        requiresPairing = true,
                    )
                    postStatus(context.getString(R.string.native_host_status_pairing_device))
                    val pairing = BridgeApi.completeDevicePairingByBaseUrl(
                        baseUrl = connectionTarget.baseUrl,
                        pairingCode = pairingCode,
                        authToken = authToken,
                    )
                    authToken = pairing.accessToken
                    connectionTarget = resolveBridgeLoadTarget(normalizedEndpoint)
                    connection = BridgeApi.fetchConnectionTargetByBaseUrl(
                        baseUrl = connectionTarget.baseUrl,
                        authToken = authToken,
                    )
                } else if (connection.authMode == "device-token" && authToken.isNullOrBlank()) {
                    throw IllegalStateException(
                        connection.localAuthPage?.let {
                            "This bridge needs a fresh enrollment QR. Open $it on the bridge host."
                        } ?: "This bridge needs to be re-enrolled from the bridge host.",
                    )
                }

                val resolvedBridgeId = connection.bridgeId ?: bridgeId
                val recommendedEndpoint = BridgeApi.normalizeEndpoint(connection.recommendedServerEndpoint)
                val existingProfile =
                    profileStore.list().firstOrNull {
                        matchesBridgeIdentity(it, resolvedBridgeId, recommendedEndpoint) ||
                            it.name == name ||
                            BridgeApi.normalizeEndpoint(it.serverEndpoint) == recommendedEndpoint
                    }
                val profile = BridgeProfile(
                    id = existingProfile?.id ?: profileStore.createProfileId(name, recommendedEndpoint, resolvedBridgeId),
                    bridgeId = resolvedBridgeId,
                    name = name,
                    serverEndpoint = recommendedEndpoint,
                    authToken = authToken,
                    libp2pPeerId = libp2pPeerId,
                )
                profileStore.write(profile)
                syncSavedConnectionsState()
                postConnectionProgress(
                    profileName = profile.name,
                    endpoint = profile.serverEndpoint,
                    stage = NativeHostConnectionStage.OPENING_WORKSPACE,
                    requiresPairing = pairingCode.isNotBlank(),
                )
                postStatus(context.getString(R.string.native_host_status_opening_workspace))
                postToMain {
                    openBridge(profile)
                    setStatus("Connected to ${profile.serverEndpoint}")
                }
            } catch (error: Exception) {
                android.util.Log.w("CodexMobile", "failed to save/open bridge profile", error)
                val message = normalizeErrorMessage(error)
                postStatus(message)
                postToMain {
                    renderConnectionFailure(name, endpoint, message)
                }
            }
        }
    }

    private fun resolveBridgeLoadTarget(endpoint: String): BridgeLoadTarget {
        return BridgeLoadTarget(
            baseUrl = BridgeApi.deriveServerHttpBaseUrl(endpoint),
            usesLocalProxy = false,
        )
    }

    private fun matchesBridgeIdentity(
        profile: BridgeProfile,
        bridgeId: String?,
        endpoint: String,
    ): Boolean {
        val normalizedBridgeId = bridgeId?.trim().orEmpty()
        if (normalizedBridgeId.isNotEmpty()) {
            return profile.bridgeId?.trim() == normalizedBridgeId
        }
        return BridgeApi.normalizeEndpoint(profile.serverEndpoint) == BridgeApi.normalizeEndpoint(endpoint)
    }

    private fun postStatus(message: String) {
        postToMain {
            setStatus(message)
        }
    }

    private fun postConnectionProgress(
        profileName: String,
        endpoint: String,
        stage: NativeHostConnectionStage,
        requiresPairing: Boolean,
    ) {
        postToMain {
            updateConnectionProgress(profileName, endpoint, stage, requiresPairing)
        }
    }

    private fun postToMain(block: () -> Unit) {
        if (android.os.Looper.myLooper() == android.os.Looper.getMainLooper()) {
            block()
        } else {
            android.os.Handler(android.os.Looper.getMainLooper()).post(block)
        }
    }
}
