package com.boomyao.codexmobile.nativehost

import android.content.Context
import androidx.core.content.ContextCompat
import com.boomyao.codexmobile.R
import com.boomyao.codexmobile.tailnet.CodexTailnetBridge
import com.boomyao.codexmobile.tailnet.CodexTailnetService
import com.boomyao.codexmobile.tailnet.EnrollmentStore
import com.boomyao.codexmobile.tailnet.TailnetEnrollmentPayload
import com.boomyao.codexmobile.tailnet.TailnetStatusSnapshot
import com.boomyao.codexmobile.tailnet.parseTailnetEnrollmentPayload
import org.json.JSONObject
import kotlin.concurrent.thread

class NativeHostConnectionCoordinator(
    private val context: Context,
    private val profileStore: BridgeProfileStore,
    private val activeProfileProvider: () -> BridgeProfile?,
    private val openBridge: (BridgeProfile) -> Unit,
    private val renderEmptyState: (String) -> Unit,
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
        if (profile.tailnetEnrollmentPayload.isNullOrBlank()) {
            EnrollmentStore(context).clear()
            context.startService(CodexTailnetService.stopIntent(context))
        }
        openBridge(profile)
    }

    fun resetEnrollment(profile: BridgeProfile) {
        val nextProfile = profileStore.remove(profile.id)
        syncSavedConnectionsState()
        if (nextProfile == null) {
            EnrollmentStore(context).clear()
            context.startService(CodexTailnetService.stopIntent(context))
            renderEmptyState(context.getString(R.string.native_host_status_idle))
            return
        }
        if (nextProfile.tailnetEnrollmentPayload.isNullOrBlank()) {
            EnrollmentStore(context).clear()
            context.startService(CodexTailnetService.stopIntent(context))
        }
        openBridge(nextProfile)
    }

    fun restoreTailnetRuntimeIfNeeded() {
        val profile = activeProfileProvider() ?: return
        val enrollmentPayload = profile.tailnetEnrollmentPayload?.trim().orEmpty()
        if (enrollmentPayload.isEmpty()) {
            return
        }
        val enrollment = runCatching { parseTailnetEnrollmentPayload(enrollmentPayload) }.getOrNull() ?: return
        val snapshot = CodexTailnetBridge.status(context)
        if (snapshot.state == "running") {
            return
        }
        android.util.Log.i(
            "CodexMobile",
            "restoreTailnetRuntimeIfNeeded starting service state=${snapshot.state} message=${snapshot.message}",
        )
        ContextCompat.startForegroundService(
            context,
            CodexTailnetService.startIntent(context, enrollment.rawPayload),
        )
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
                        name = payload.name,
                        endpoint = payload.serverEndpoint,
                        pairingCode = payload.pairingCode.orEmpty(),
                        existingAuthToken = null,
                        tailnetEnrollmentPayload = null,
                    )
                }

                is EnrollmentPayload.Tailnet -> {
                    postConnectionProgress(
                        profileName = payload.bridgeName,
                        endpoint = payload.bridgeServerEndpoint,
                        stage = NativeHostConnectionStage.PAYLOAD_RECEIVED,
                        requiresPairing = !payload.pairingCode.isNullOrBlank(),
                    )
                    postStatus(context.getString(R.string.native_host_status_payload_received))
                    val stagedPayload = parseTailnetEnrollmentPayload(payload.rawJson)
                    val snapshot = CodexTailnetBridge.stage(context, stagedPayload)
                    postConnectionProgress(
                        profileName = payload.bridgeName,
                        endpoint = payload.bridgeServerEndpoint,
                        stage = NativeHostConnectionStage.STARTING_TAILNET,
                        requiresPairing = !payload.pairingCode.isNullOrBlank(),
                    )
                    postStatus(snapshot.message)
                    ContextCompat.startForegroundService(
                        context,
                        CodexTailnetService.startIntent(context, payload.rawJson),
                    )
                    if (payload.bridgeServerEndpoint.isNotBlank()) {
                        saveBridgeProfile(
                            name = payload.bridgeName,
                            endpoint = payload.bridgeServerEndpoint,
                            pairingCode = payload.pairingCode.orEmpty(),
                            existingAuthToken = null,
                            tailnetEnrollmentPayload = payload.rawJson,
                        )
                    }
                }
            }
        } catch (error: Exception) {
            postStatus(normalizeErrorMessage(error))
        }
    }

    fun resolveBridgeLoadTarget(profile: BridgeProfile): BridgeLoadTarget {
        return resolveBridgeLoadTarget(
            endpoint = profile.serverEndpoint,
            authToken = profile.authToken,
            tailnetEnrollmentPayload = profile.tailnetEnrollmentPayload,
        )
    }

    private fun saveBridgeProfile(
        name: String,
        endpoint: String,
        pairingCode: String,
        existingAuthToken: String?,
        tailnetEnrollmentPayload: String?,
    ) {
        postStatus(context.getString(R.string.native_host_status_preparing_connection))
        thread {
            try {
                val normalizedEndpoint = BridgeApi.normalizeEndpoint(endpoint)
                var authToken = existingAuthToken
                postConnectionProgress(
                    profileName = name,
                    endpoint = normalizedEndpoint,
                    stage =
                        when {
                            BridgeApi.isLikelyTailnetEndpoint(normalizedEndpoint) -> NativeHostConnectionStage.STARTING_TAILNET
                            pairingCode.isNotBlank() -> NativeHostConnectionStage.PAIRING_DEVICE
                            else -> NativeHostConnectionStage.OPENING_WORKSPACE
                        },
                    requiresPairing = pairingCode.isNotBlank(),
                )
                ensureTailnetRuntimeReady(normalizedEndpoint, tailnetEnrollmentPayload)
                var connectionTarget =
                    resolveBridgeLoadTarget(normalizedEndpoint, authToken, tailnetEnrollmentPayload)
                var connection = BridgeApi.fetchConnectionTargetByBaseUrl(
                    baseUrl = connectionTarget.baseUrl,
                    authToken = if (connectionTarget.usesLocalProxy) null else authToken,
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
                        authToken = if (connectionTarget.usesLocalProxy) null else authToken,
                    )
                    authToken = pairing.accessToken
                    connectionTarget = resolveBridgeLoadTarget(normalizedEndpoint, authToken, tailnetEnrollmentPayload)
                    connection = BridgeApi.fetchConnectionTargetByBaseUrl(
                        baseUrl = connectionTarget.baseUrl,
                        authToken = if (connectionTarget.usesLocalProxy) null else authToken,
                    )
                } else if (connection.authMode == "device-token" && authToken.isNullOrBlank()) {
                    throw IllegalStateException(
                        connection.localAuthPage?.let {
                            "This bridge needs a fresh enrollment QR. Open $it on the bridge host."
                        } ?: "This bridge needs to be re-enrolled from the bridge host.",
                    )
                }

                val recommendedEndpoint = BridgeApi.normalizeEndpoint(connection.recommendedServerEndpoint)
                val existingProfile =
                    profileStore.list().firstOrNull {
                        BridgeApi.normalizeEndpoint(it.serverEndpoint) == recommendedEndpoint &&
                            (
                                (!tailnetEnrollmentPayload.isNullOrBlank() && it.tailnetEnrollmentPayload == tailnetEnrollmentPayload) ||
                                    it.name == name
                                )
                    }
                val profile = BridgeProfile(
                    id = existingProfile?.id ?: profileStore.createProfileId(name, recommendedEndpoint),
                    name = name,
                    serverEndpoint = recommendedEndpoint,
                    authToken = authToken,
                    tailnetEnrollmentPayload = tailnetEnrollmentPayload,
                )
                profileStore.write(profile)
                syncSavedConnectionsState()
                if (profile.tailnetEnrollmentPayload.isNullOrBlank()) {
                    EnrollmentStore(context).clear()
                    context.startService(CodexTailnetService.stopIntent(context))
                }
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
                postStatus(normalizeErrorMessage(error))
            }
        }
    }

    private fun resolveBridgeLoadTarget(
        endpoint: String,
        authToken: String?,
        tailnetEnrollmentPayload: String?,
    ): BridgeLoadTarget {
        val directBaseUrl = BridgeApi.deriveServerHttpBaseUrl(endpoint)
        if (!BridgeApi.isLikelyTailnetEndpoint(endpoint)) {
            return BridgeLoadTarget(
                baseUrl = directBaseUrl,
                usesLocalProxy = false,
            )
        }
        val enrollment = tailnetEnrollmentForProfile(endpoint, tailnetEnrollmentPayload)
        val matchesEnrollment = enrollment != null && matchesTailnetEnrollment(enrollment, endpoint)
        ensureTailnetRuntimeReady(endpoint, tailnetEnrollmentPayload)
        if (matchesEnrollment) {
            android.util.Log.i("CodexMobile", "resolveBridgeLoadTarget ensuring tailnet service is running")
            ContextCompat.startForegroundService(
                context,
                CodexTailnetService.startIntent(context, enrollment.rawPayload),
            )
        }

        var lastSnapshot = CodexTailnetBridge.configureBridgeProxy(context, endpoint, authToken)
        var lastProxyError: String? = null
        repeat(if (matchesEnrollment) 60 else 1) { attempt ->
            lastSnapshot =
                if (attempt == 0) {
                    lastSnapshot
                } else {
                    Thread.sleep(250L)
                    CodexTailnetBridge.configureBridgeProxy(context, endpoint, authToken)
                }
            val localProxyUrl = lastSnapshot.localProxyUrl?.trim().orEmpty()
            if (localProxyUrl.isNotEmpty()) {
                val readyProbe = BridgeApi.probeBridgeReadyByBaseUrl(localProxyUrl, null)
                if (readyProbe.ready) {
                    return BridgeLoadTarget(
                        baseUrl = localProxyUrl,
                        usesLocalProxy = true,
                    )
                }
                lastProxyError = readyProbe.errorMessage
            }
            if (!matchesEnrollment && lastSnapshot.state == "error") {
                throw IllegalStateException(describeTailnetState(lastSnapshot))
            }
        }
        if (matchesEnrollment) {
            throw IllegalStateException(
                lastProxyError?.ifBlank { null } ?: describeTailnetState(lastSnapshot),
            )
        }
        return BridgeLoadTarget(
            baseUrl = directBaseUrl,
            usesLocalProxy = false,
        )
    }

    private fun ensureTailnetRuntimeReady(endpoint: String, tailnetEnrollmentPayload: String?) {
        if (!BridgeApi.isLikelyTailnetEndpoint(endpoint)) {
            return
        }
        val enrollment = tailnetEnrollmentForProfile(endpoint, tailnetEnrollmentPayload) ?: return
        if (!matchesTailnetEnrollment(enrollment, endpoint)) {
            return
        }

        var snapshot = CodexTailnetBridge.status(context)
        if (isTailnetRuntimeRunning(snapshot)) {
            return
        }

        ContextCompat.startForegroundService(
            context,
            CodexTailnetService.startIntent(context, enrollment.rawPayload),
        )

        repeat(60) {
            Thread.sleep(250L)
            snapshot = CodexTailnetBridge.status(context)
            if (isTailnetRuntimeRunning(snapshot)) {
                return
            }
            if (snapshot.state == "error") {
                throw IllegalStateException(describeTailnetState(snapshot))
            }
        }
        android.util.Log.w(
            "CodexMobile",
            "tailnet runtime did not report running before timeout; continuing to bridge proxy setup",
        )
    }

    private fun tailnetEnrollmentForProfile(
        endpoint: String,
        tailnetEnrollmentPayload: String?,
    ): TailnetEnrollmentPayload? {
        val explicitPayload = tailnetEnrollmentPayload?.trim().orEmpty()
        if (explicitPayload.isNotEmpty()) {
            val enrollment = parseTailnetEnrollmentPayload(explicitPayload)
            if (!matchesTailnetEnrollment(enrollment, endpoint)) {
                throw IllegalStateException("Saved tailnet enrollment does not match $endpoint.")
            }
            return enrollment
        }
        return EnrollmentStore(context).readEnrollment()?.takeIf {
            matchesTailnetEnrollment(it, endpoint)
        }
    }

    private fun matchesTailnetEnrollment(
        enrollment: TailnetEnrollmentPayload,
        endpoint: String,
    ): Boolean {
        val target = BridgeApi.normalizeEndpoint(endpoint)
        val enrolled = BridgeApi.normalizeEndpoint(enrollment.bridgeServerEndpoint)
        return target.isNotBlank() && enrolled.isNotBlank() && target == enrolled
    }

    private fun isTailnetRuntimeRunning(snapshot: TailnetStatusSnapshot): Boolean {
        return snapshot.state == "running"
    }

    private fun describeTailnetState(snapshot: TailnetStatusSnapshot): String {
        val auth = snapshot.auth
        val parts = mutableListOf<String>()
        val primary = snapshot.message.trim()
        if (primary.isNotEmpty()) {
            parts += primary
        }
        val backendState = auth?.backendState?.trim().orEmpty()
        when {
            auth == null -> parts += "Tailnet auth state is unavailable."
            auth.needsLogin -> parts += "Tailnet backend state: NeedsLogin."
            auth.needsMachineAuth -> parts += "Tailnet backend state: NeedsMachineAuth."
            backendState.isNotEmpty() && backendState != "Running" -> parts += "Tailnet backend state: $backendState."
        }
        auth?.tailnet?.takeIf { it.isNotBlank() }?.let { parts += "Tailnet: $it." }
        auth?.selfDnsName?.takeIf { it.isNotBlank() }?.let { parts += "Node: $it." }
        if (!auth?.tailscaleIps.isNullOrEmpty()) {
            parts += "IPs: ${auth?.tailscaleIps?.joinToString(", ")}."
        }
        auth?.authUrl?.takeIf { it.isNotBlank() }?.let { parts += "Auth URL: $it" }
        return parts.joinToString(" ").trim()
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
