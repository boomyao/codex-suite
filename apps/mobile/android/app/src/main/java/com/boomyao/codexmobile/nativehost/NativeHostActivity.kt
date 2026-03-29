package com.boomyao.codexmobile.nativehost

import android.annotation.SuppressLint
import android.content.Intent
import android.graphics.drawable.GradientDrawable
import android.net.Uri
import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.webkit.CookieManager
import android.webkit.JavascriptInterface
import android.webkit.ConsoleMessage
import android.webkit.WebChromeClient
import android.webkit.WebResourceError
import android.webkit.WebResourceRequest
import android.webkit.WebView
import android.webkit.WebViewClient
import android.widget.ImageView
import android.widget.LinearLayout
import android.widget.TextView
import androidx.appcompat.app.AppCompatActivity
import androidx.core.content.ContextCompat
import com.boomyao.codexmobile.R
import com.boomyao.codexmobile.tailnet.CodexTailnetBridge
import com.boomyao.codexmobile.tailnet.CodexTailnetService
import com.boomyao.codexmobile.tailnet.EnrollmentStore
import com.boomyao.codexmobile.tailnet.TailnetEnrollmentPayload
import com.boomyao.codexmobile.tailnet.parseTailnetEnrollmentPayload
import com.google.android.material.appbar.MaterialToolbar
import com.google.android.material.bottomsheet.BottomSheetDialog
import com.google.android.material.button.MaterialButton
import com.google.android.material.dialog.MaterialAlertDialogBuilder
import com.google.android.material.progressindicator.LinearProgressIndicator
import com.journeyapps.barcodescanner.ScanContract
import com.journeyapps.barcodescanner.ScanOptions
import org.json.JSONArray
import org.json.JSONObject
import java.io.BufferedReader
import java.io.InputStreamReader
import java.net.HttpURLConnection
import java.net.URL
import java.util.Locale
import java.util.concurrent.ConcurrentHashMap
import kotlin.concurrent.thread

class NativeHostActivity : AppCompatActivity() {
    private companion object {
        private const val LOCAL_HOST_ID = "local"
        private val DESKTOP_EXTENSION_INFO =
            JSONObject()
                .put("version", "26.323.20928")
                .put("buildFlavor", "prod")
                .put("buildNumber", "1173")
        private val LOCAL_HOST_CONFIG =
            JSONObject()
                .put("id", LOCAL_HOST_ID)
                .put("hostId", LOCAL_HOST_ID)
                .put("display_name", "Local")
                .put("displayName", "Local")
                .put("kind", "local")
        private val DEFAULT_PERSISTED_ATOM_STATE =
            JSONObject().put(
                "statsig_default_enable_features",
                JSONObject().put("fast_mode", true),
            )
        private val UNHANDLED_LOCAL_METHOD = Any()
    }

    private data class BridgeLoadTarget(
        val baseUrl: String,
        val usesLocalProxy: Boolean,
    )

    private data class HttpProxyResponse(
        val body: Any?,
        val status: Int,
        val headers: JSONObject,
    )

    private data class HostStateSnapshot(
        var account: Any? = null,
        var rateLimit: Any? = null,
        val workspaceRootOptions: MutableList<String> = mutableListOf(),
        val activeWorkspaceRoots: MutableList<String> = mutableListOf(),
        val workspaceRootLabels: MutableMap<String, String> = mutableMapOf(),
        val pinnedThreadIds: MutableList<String> = mutableListOf(),
    )

    private enum class ShellChromeState {
        DISCONNECTED,
        LOADING,
        CONNECTED,
        ERROR,
    }

    private enum class ConnectionStage {
        PAYLOAD_RECEIVED,
        STARTING_TAILNET,
        PAIRING_DEVICE,
        OPENING_WORKSPACE,
    }

    private enum class ChipTone {
        NEUTRAL,
        ACTIVE,
        SUCCESS,
        WARNING,
    }

    private lateinit var toolbar: MaterialToolbar
    private lateinit var progressIndicator: LinearProgressIndicator
    private lateinit var webView: WebView
    private lateinit var heroCardView: View
    private lateinit var heroEyebrowView: TextView
    private lateinit var stateChipView: TextView
    private lateinit var statusView: TextView
    private lateinit var emptyStateContainer: View
    private lateinit var stateIconView: ImageView
    private lateinit var emptyTitleView: TextView
    private lateinit var emptyBodyView: TextView
    private lateinit var runtimeDetailsView: View
    private lateinit var runtimeLabelView: TextView
    private lateinit var runtimeValueView: TextView
    private lateinit var hintCardView: View
    private lateinit var hintTitleView: TextView
    private lateinit var hintBodyView: TextView
    private lateinit var primaryActionButton: MaterialButton
    private lateinit var secondaryActionButton: MaterialButton
    private lateinit var connectionButton: MaterialButton
    private lateinit var workspaceEmptyTitleView: TextView
    private lateinit var workspaceEmptyBodyView: TextView
    private lateinit var profileStore: BridgeProfileStore
    private var activeProfile: BridgeProfile? = null
    private var activeLoadTarget: BridgeLoadTarget? = null
    private var chromeState = ShellChromeState.DISCONNECTED
    private var currentConnectionStage: ConnectionStage? = null
    private var currentConnectionRequiresPairing = false
    private var currentStatusMessage = ""
    private var persistedAtomState = deepCopyJsonObject(DEFAULT_PERSISTED_ATOM_STATE)
    private val globalState = linkedMapOf<String, Any?>()
    private val configurationState = linkedMapOf<String, Any?>()
    private val sharedObjects = linkedMapOf<String, Any?>(
        "pending_worktrees" to JSONArray(),
        "remote_connections" to JSONArray(),
        "host_config" to deepCopyJsonObject(LOCAL_HOST_CONFIG),
    )
    private val sharedObjectSubscribers = linkedMapOf<String, Int>()
    private val hostState = HostStateSnapshot()
    private val pendingTurnCompletions = ConcurrentHashMap<String, String>()
    private val activeTurnReconciliations = ConcurrentHashMap.newKeySet<String>()
    private val emittedTurnCompletions = ConcurrentHashMap<String, String>()
    private var appServerWebSocketClient: AppServerWebSocketClient? = null
    private val scanLauncher = registerForActivityResult(ScanContract()) { result ->
        val contents = result.contents?.trim().orEmpty()
        if (contents.isEmpty()) {
            setStatus("QR scan canceled.")
            return@registerForActivityResult
        }
        importEnrollment(contents)
    }
    private var bridgeLoadGeneration = 0

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_native_host)

        profileStore = BridgeProfileStore(this)
        toolbar = findViewById(R.id.nativeHostToolbar)
        progressIndicator = findViewById(R.id.nativeHostProgress)
        webView = findViewById(R.id.nativeHostWebView)
        heroCardView = findViewById(R.id.nativeHostHeroCard)
        heroEyebrowView = findViewById(R.id.nativeHostHeroEyebrow)
        stateChipView = findViewById(R.id.nativeHostStateChip)
        statusView = findViewById(R.id.nativeHostStatus)
        emptyStateContainer = findViewById(R.id.nativeHostEmptyState)
        stateIconView = findViewById(R.id.nativeHostStateIcon)
        emptyTitleView = findViewById(R.id.nativeHostEmptyTitle)
        emptyBodyView = findViewById(R.id.nativeHostEmptyBody)
        runtimeDetailsView = findViewById(R.id.nativeHostRuntimeDetails)
        runtimeLabelView = findViewById(R.id.nativeHostRuntimeLabel)
        runtimeValueView = findViewById(R.id.nativeHostRuntimeValue)
        hintCardView = findViewById(R.id.nativeHostHintCard)
        hintTitleView = findViewById(R.id.nativeHostHintTitle)
        hintBodyView = findViewById(R.id.nativeHostHintBody)
        primaryActionButton = findViewById(R.id.nativeHostPrimaryAction)
        secondaryActionButton = findViewById(R.id.nativeHostSecondaryAction)
        connectionButton = findViewById(R.id.nativeHostConnectionButton)
        workspaceEmptyTitleView = findViewById(R.id.nativeHostWorkspaceEmptyTitle)
        workspaceEmptyBodyView = findViewById(R.id.nativeHostWorkspaceEmptyBody)

        primaryActionButton.setOnClickListener { handlePrimaryAction() }
        secondaryActionButton.setOnClickListener { handleSecondaryAction() }
        connectionButton.setOnClickListener { openConnectionSheet() }

        configureWebView()
        renderDisconnectedChrome(getString(R.string.native_host_status_idle))
        restoreTailnetRuntimeIfNeeded()
        loadSavedProfile()
    }

    override fun onDestroy() {
        resetAppServerWebSocketClient()
        super.onDestroy()
    }

    @SuppressLint("SetJavaScriptEnabled")
    private fun configureWebView() {
        webView.settings.javaScriptEnabled = true
        webView.settings.domStorageEnabled = true
        webView.settings.allowFileAccess = false
        webView.addJavascriptInterface(NativeHostJavascriptBridge(), "CodexMobileNativeBridge")
        webView.webChromeClient = object : WebChromeClient() {
            override fun onConsoleMessage(consoleMessage: ConsoleMessage?): Boolean {
                if (consoleMessage != null) {
                    android.util.Log.d(
                        "CodexMobile",
                        "console ${consoleMessage.messageLevel()}: ${consoleMessage.message()}",
                    )
                }
                return true
            }
        }
        webView.webViewClient = object : WebViewClient() {
            override fun onPageStarted(view: WebView?, url: String?, favicon: android.graphics.Bitmap?) {
                activeProfile?.let(::renderLoadingChrome)
            }

            override fun onPageFinished(view: WebView?, url: String?) {
                val profile = activeProfile ?: return
                if (url.isNullOrBlank() || url == "about:blank") {
                    return
                }
                renderConnectedChrome(profile)
                setStatus("Connected to ${profile.serverEndpoint}")
            }

            override fun onReceivedError(
                view: WebView?,
                request: WebResourceRequest?,
                error: WebResourceError?,
            ) {
                if (request?.isForMainFrame == true) {
                    val message = error?.description?.toString() ?: "Failed to load bridge."
                    setStatus(message)
                    activeProfile?.let { profile ->
                        renderWorkspaceErrorChrome(profile, message)
                    }
                }
            }
        }
    }

    private fun loadSavedProfile() {
        val profile = profileStore.readActive()
        if (profile == null) {
            renderEmptyState(getString(R.string.native_host_status_idle))
            return
        }
        activeProfile = profile
        syncSavedConnectionsState()
        openBridge(profile)
    }

    private fun restoreTailnetRuntimeIfNeeded() {
        val profile = profileStore.readActive() ?: return
        val enrollmentPayload = profile.tailnetEnrollmentPayload?.trim().orEmpty()
        if (enrollmentPayload.isEmpty()) {
            return
        }
        val enrollment = runCatching { parseTailnetEnrollmentPayload(enrollmentPayload) }.getOrNull() ?: return
        val snapshot = CodexTailnetBridge.status(applicationContext)
        if (snapshot.state == "running") {
            return
        }
        android.util.Log.i(
            "CodexMobile",
            "restoreTailnetRuntimeIfNeeded starting service state=${snapshot.state} message=${snapshot.message}",
        )
        ContextCompat.startForegroundService(
            this,
            CodexTailnetService.startIntent(this, enrollment.rawPayload),
        )
    }

    private fun reloadActiveBridge() {
        val profile = activeProfile
        if (profile == null) {
            renderEmptyState(getString(R.string.native_host_status_idle))
            return
        }
        openBridge(profile)
    }

    private fun openBridge(profile: BridgeProfile) {
        resetAppServerWebSocketClient()
        activeProfile = profile
        activeLoadTarget = null
        syncSavedConnectionsState()
        if (currentConnectionStage == null) {
            updateConnectionProgress(
                profileName = profile.name,
                endpoint = profile.serverEndpoint,
                stage = ConnectionStage.OPENING_WORKSPACE,
                requiresPairing = false,
            )
        }
        renderLoadingChrome(profile)
        setStatus(getString(R.string.native_host_status_opening_workspace))
        val generation = ++bridgeLoadGeneration
        thread {
            try {
                val loadTarget = resolveBridgeLoadTarget(profile)
                val bootstrapState =
                    runCatching {
                        hydrateBridgeBootstrapState(
                            loadTarget = loadTarget,
                            authToken = if (loadTarget.usesLocalProxy) null else profile.authToken,
                        )
                    }.onFailure { error ->
                        android.util.Log.w("CodexMobile", "failed to hydrate bridge bootstrap state", error)
                    }.getOrNull()
                runOnUiThread {
                    if (generation != bridgeLoadGeneration || activeProfile != profile) {
                        return@runOnUiThread
                    }
                    if (bootstrapState != null) {
                        applyBridgeBootstrapState(bootstrapState)
                    }
                    activeLoadTarget = loadTarget
                    val url = BridgeApi.buildRemoteShellUrlFromBaseUrl(loadTarget.baseUrl)
                    configureBridgeCookies(loadTarget.baseUrl, profile.authToken, loadTarget.usesLocalProxy)
                    val headers = mutableMapOf<String, String>()
                    if (!loadTarget.usesLocalProxy && !profile.authToken.isNullOrBlank()) {
                        headers["Authorization"] = "Bearer ${profile.authToken}"
                    }
                    webView.loadUrl(url, headers)
                }
            } catch (error: Exception) {
                android.util.Log.w("CodexMobile", "failed to open bridge", error)
                runOnUiThread {
                    if (generation != bridgeLoadGeneration || activeProfile != profile) {
                        return@runOnUiThread
                    }
                    activeLoadTarget = null
                    val message = normalizeErrorMessage(error)
                    setStatus(message)
                    renderWorkspaceErrorChrome(profile, message)
                }
            }
        }
    }

    private fun renderEmptyState(message: String) {
        resetAppServerWebSocketClient()
        activeProfile = null
        activeLoadTarget = null
        syncSavedConnectionsState()
        renderDisconnectedChrome(message)
        setStatus(message)
    }

    private fun setStatus(message: String) {
        currentStatusMessage = message
        statusView.text = message
    }

    private fun savedConnectionsPayload(): JSONArray {
        val activeProfileId = activeProfile?.id ?: profileStore.readActive()?.id
        return JSONArray().apply {
            profileStore.list().forEach { profile ->
                put(
                    JSONObject()
                        .put("id", profile.id)
                        .put("name", profile.name)
                        .put("serverEndpoint", profile.serverEndpoint)
                        .put("active", profile.id == activeProfileId)
                        .put("tailnetManaged", !profile.tailnetEnrollmentPayload.isNullOrBlank()),
                )
            }
        }
    }

    private fun syncSavedConnectionsState() {
        sharedObjects["remote_connections"] = savedConnectionsPayload()
        if ((sharedObjectSubscribers["remote_connections"] ?: 0) > 0) {
            broadcastSharedObjectUpdate("remote_connections", sharedObjects["remote_connections"])
        }
    }

    private fun activateProfile(profile: BridgeProfile) {
        profileStore.setActive(profile.id)
        if (profile.tailnetEnrollmentPayload.isNullOrBlank()) {
            EnrollmentStore(applicationContext).clear()
            startService(CodexTailnetService.stopIntent(this))
        }
        openBridge(profile)
    }

    private fun savedConnectionsSummary(): String {
        val count = profileStore.list().size
        return if (count == 0) {
            getString(R.string.native_host_saved_none)
        } else {
            getString(R.string.native_host_saved_count, count)
        }
    }

    private fun updateHintCard(titleResId: Int, bodyResId: Int) {
        hintTitleView.text = getString(titleResId)
        hintBodyView.text = getString(bodyResId)
    }

    private fun updateWorkspacePreview(titleResId: Int, bodyResId: Int) {
        workspaceEmptyTitleView.text = getString(titleResId)
        workspaceEmptyBodyView.text = getString(bodyResId)
    }

    private fun applyChipTone(view: TextView, tone: ChipTone) {
        val (backgroundColorRes, textColorRes) =
            when (tone) {
                ChipTone.NEUTRAL -> R.color.nativeHostChipNeutralBg to R.color.nativeHostChipNeutralText
                ChipTone.ACTIVE -> R.color.nativeHostChipActiveBg to R.color.nativeHostChipActiveText
                ChipTone.SUCCESS -> R.color.nativeHostChipSuccessBg to R.color.nativeHostChipSuccessText
                ChipTone.WARNING -> R.color.nativeHostChipWarningBg to R.color.nativeHostChipWarningText
            }
        (view.background?.mutate() as? GradientDrawable)?.setColor(
            ContextCompat.getColor(this, backgroundColorRes),
        )
        view.setTextColor(ContextCompat.getColor(this, textColorRes))
    }

    private fun updateStateChip(labelResId: Int, tone: ChipTone) {
        stateChipView.text = getString(labelResId)
        applyChipTone(stateChipView, tone)
    }

    private fun updateRuntimeCard(labelResId: Int, value: String) {
        runtimeDetailsView.visibility = View.VISIBLE
        runtimeLabelView.text = getString(labelResId)
        runtimeValueView.text = value
    }

    private fun showDisconnectedSupportingViews() {
        stateIconView.visibility = View.VISIBLE
        updateRuntimeCard(R.string.native_host_runtime_label_saved, savedConnectionsSummary())
        hintCardView.visibility = View.GONE
    }

    private fun showRuntimeSupportingViews(labelResId: Int, value: String) {
        stateIconView.visibility = View.GONE
        updateRuntimeCard(labelResId, value)
        hintCardView.visibility = View.VISIBLE
    }

    private fun clearConnectionProgress() {
        currentConnectionStage = null
        currentConnectionRequiresPairing = false
    }

    private fun updateConnectionProgress(
        profileName: String,
        endpoint: String,
        stage: ConnectionStage,
        requiresPairing: Boolean,
    ) {
        currentConnectionStage = stage
        currentConnectionRequiresPairing = requiresPairing
        renderLoadingChrome(
            BridgeProfile(
                id = "pending",
                name = profileName.ifBlank { endpoint.ifBlank { getString(R.string.native_host_title) } },
                serverEndpoint = endpoint,
                authToken = null,
            ),
        )
    }

    private fun buildConnectionProgressSummary(
        stage: ConnectionStage,
        requiresPairing: Boolean,
        endpoint: String,
    ): String {
        val steps =
            buildList {
                add(ConnectionStage.PAYLOAD_RECEIVED to getString(R.string.native_host_progress_step_payload))
                add(ConnectionStage.STARTING_TAILNET to getString(R.string.native_host_progress_step_tailnet))
                if (requiresPairing) {
                    add(ConnectionStage.PAIRING_DEVICE to getString(R.string.native_host_progress_step_pairing))
                }
                add(ConnectionStage.OPENING_WORKSPACE to getString(R.string.native_host_progress_step_workspace))
            }
        val activeIndex = steps.indexOfFirst { it.first == stage }.coerceAtLeast(0)
        val timeline =
            steps.mapIndexed { index, (_, label) ->
                when {
                    index < activeIndex -> "[done] $label"
                    index == activeIndex -> "[now] $label"
                    else -> "[next] $label"
                }
            }.joinToString("\n")
        return buildString {
            append(timeline)
            if (endpoint.isNotBlank()) {
                append("\n\n")
                append(getString(R.string.native_host_runtime_label_endpoint))
                append(": ")
                append(endpoint)
            }
        }
    }

    private fun renderDisconnectedChrome(message: String) {
        clearConnectionProgress()
        chromeState = ShellChromeState.DISCONNECTED
        toolbar.subtitle = getString(R.string.native_host_subtitle_default)
        heroCardView.visibility = View.VISIBLE
        heroEyebrowView.text = getString(R.string.native_host_hero_eyebrow_idle)
        updateStateChip(R.string.native_host_state_ready, ChipTone.NEUTRAL)
        progressIndicator.visibility = View.GONE
        emptyStateContainer.visibility = View.VISIBLE
        showDisconnectedSupportingViews()
        emptyTitleView.text = getString(R.string.native_host_empty_title)
        emptyBodyView.text = getString(R.string.native_host_empty_body)
        updateWorkspacePreview(
            R.string.native_host_workspace_preview_title,
            R.string.native_host_workspace_preview_body,
        )
        primaryActionButton.visibility = View.VISIBLE
        primaryActionButton.text = getString(R.string.native_host_empty_action)
        secondaryActionButton.visibility = View.VISIBLE
        secondaryActionButton.text = getString(R.string.native_host_secondary_action_manage)
        webView.loadUrl("about:blank")
        webView.visibility = View.GONE
        connectionButton.visibility = View.VISIBLE
        connectionButton.text = getString(R.string.native_host_fab_connect)
        statusView.text = message
    }

    private fun renderLoadingChrome(profile: BridgeProfile) {
        chromeState = ShellChromeState.LOADING
        toolbar.subtitle = profile.name
        heroCardView.visibility = View.VISIBLE
        heroEyebrowView.text = getString(R.string.native_host_hero_eyebrow_loading)
        updateStateChip(R.string.native_host_state_connecting, ChipTone.ACTIVE)
        progressIndicator.visibility = View.VISIBLE
        emptyStateContainer.visibility = View.VISIBLE
        val stage = currentConnectionStage
        emptyTitleView.text =
            if (stage == null) {
                getString(R.string.native_host_workspace_loading)
            } else {
                getString(R.string.native_host_workspace_connecting)
            }
        emptyBodyView.text =
            when (stage) {
                ConnectionStage.PAYLOAD_RECEIVED -> getString(R.string.native_host_progress_body_payload)
                ConnectionStage.STARTING_TAILNET -> getString(R.string.native_host_progress_body_tailnet)
                ConnectionStage.PAIRING_DEVICE -> getString(R.string.native_host_progress_body_pairing)
                ConnectionStage.OPENING_WORKSPACE -> getString(R.string.native_host_progress_body_workspace)
                null -> getString(R.string.native_host_workspace_loading_body)
            }
        updateHintCard(
            R.string.native_host_hint_loading_title,
            R.string.native_host_hint_loading_body,
        )
        updateWorkspacePreview(
            R.string.native_host_workspace_loading,
            R.string.native_host_workspace_loading_body,
        )
        if (stage == null) {
            showRuntimeSupportingViews(R.string.native_host_runtime_label_endpoint, profile.serverEndpoint)
        } else {
            showRuntimeSupportingViews(
                R.string.native_host_runtime_label_progress,
                buildConnectionProgressSummary(stage, currentConnectionRequiresPairing, profile.serverEndpoint),
            )
        }
        primaryActionButton.visibility = View.GONE
        secondaryActionButton.visibility = View.GONE
        webView.visibility = View.GONE
        connectionButton.visibility = View.VISIBLE
        connectionButton.text = getString(R.string.native_host_fab_manage)
    }

    private fun renderConnectedChrome(profile: BridgeProfile) {
        clearConnectionProgress()
        chromeState = ShellChromeState.CONNECTED
        toolbar.subtitle = profile.name
        heroCardView.visibility = View.GONE
        updateStateChip(R.string.native_host_state_live, ChipTone.SUCCESS)
        progressIndicator.visibility = View.GONE
        emptyStateContainer.visibility = View.GONE
        runtimeDetailsView.visibility = View.GONE
        hintCardView.visibility = View.GONE
        primaryActionButton.visibility = View.GONE
        secondaryActionButton.visibility = View.GONE
        webView.visibility = View.VISIBLE
        connectionButton.visibility = View.VISIBLE
        connectionButton.text = getString(R.string.native_host_fab_manage)
    }

    private fun renderWorkspaceErrorChrome(profile: BridgeProfile, message: String) {
        clearConnectionProgress()
        chromeState = ShellChromeState.ERROR
        activeLoadTarget = null
        toolbar.subtitle = profile.name
        heroCardView.visibility = View.VISIBLE
        heroEyebrowView.text = getString(R.string.native_host_hero_eyebrow_error)
        updateStateChip(R.string.native_host_state_attention, ChipTone.WARNING)
        progressIndicator.visibility = View.GONE
        emptyStateContainer.visibility = View.VISIBLE
        emptyTitleView.text = getString(R.string.native_host_workspace_error)
        emptyBodyView.text = getString(R.string.native_host_workspace_error_body)
        updateHintCard(
            R.string.native_host_hint_error_title,
            R.string.native_host_hint_error_body,
        )
        updateWorkspacePreview(
            R.string.native_host_workspace_error,
            R.string.native_host_workspace_error_body,
        )
        showRuntimeSupportingViews(R.string.native_host_runtime_label_error, message)
        primaryActionButton.visibility = View.VISIBLE
        primaryActionButton.text = getString(R.string.native_host_retry_action)
        secondaryActionButton.visibility = View.VISIBLE
        secondaryActionButton.text = getString(R.string.native_host_secondary_action_manage)
        webView.visibility = View.GONE
        connectionButton.visibility = View.VISIBLE
        connectionButton.text = getString(R.string.native_host_fab_manage)
    }

    private fun handlePrimaryAction() {
        when (chromeState) {
            ShellChromeState.DISCONNECTED -> openScanner()
            ShellChromeState.ERROR -> reloadActiveBridge()
            ShellChromeState.LOADING -> Unit
            ShellChromeState.CONNECTED -> openConnectionSheet()
        }
    }

    private fun handleSecondaryAction() {
        openConnectionSheet()
    }

    private fun isTailnetConnected(snapshot: com.boomyao.codexmobile.tailnet.TailnetStatusSnapshot): Boolean {
        val backendState = snapshot.auth?.backendState?.trim()
        return snapshot.state == "running" && backendState == "Running"
    }

    private fun isTailnetRuntimeRunning(snapshot: com.boomyao.codexmobile.tailnet.TailnetStatusSnapshot): Boolean {
        return snapshot.state == "running"
    }

    private fun describeTailnetState(snapshot: com.boomyao.codexmobile.tailnet.TailnetStatusSnapshot): String {
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

    private fun configureBridgeCookies(
        baseUrl: String,
        authToken: String?,
        usesLocalProxy: Boolean,
    ) {
        val cookieManager = CookieManager.getInstance()
        cookieManager.setAcceptCookie(true)
        val normalizedBaseUrl = BridgeApi.normalizeEndpoint(baseUrl)
        cookieManager.setCookie(normalizedBaseUrl, "codex_mobile_host=android-native; Path=/")
        if (!usesLocalProxy && !authToken.isNullOrBlank()) {
            cookieManager.setCookie(
                normalizedBaseUrl,
                "codex_bridge_token=${authToken.trim()}; Path=/; HttpOnly",
            )
        } else {
            cookieManager.setCookie(
                normalizedBaseUrl,
                "codex_bridge_token=; Path=/; Max-Age=0",
            )
        }
        cookieManager.flush()
    }

    private fun sendPersistedAtomSync() {
        sendHostMessage(
            JSONObject()
                .put("type", "persisted-atom-sync")
                .put("state", deepCopyJsonObject(persistedAtomState)),
        )
    }

    private fun broadcastSharedObjectUpdate(key: String, value: Any?) {
        sendHostMessage(
            JSONObject()
                .put("type", "shared-object-updated")
                .put("key", key)
                .put("value", toJsonCompatible(value)),
        )
    }

    private fun updateWorkspaceRoots(nextRoots: List<String>, preferredRoot: String? = null) {
        val normalizedRoots = uniqueTrimmedStrings(nextRoots.mapNotNull { normalizeWorkspaceRootCandidate(it) })
        val preferred = normalizeWorkspaceRootCandidate(preferredRoot)
        val nextActiveRoots =
            when {
                preferred != null && normalizedRoots.contains(preferred) -> mutableListOf(preferred)
                hostState.activeWorkspaceRoots.firstOrNull() in normalizedRoots -> mutableListOf(hostState.activeWorkspaceRoots.first())
                normalizedRoots.isNotEmpty() -> mutableListOf(normalizedRoots.first())
                else -> mutableListOf()
            }

        if (hostState.workspaceRootOptions == normalizedRoots && hostState.activeWorkspaceRoots == nextActiveRoots) {
            return
        }

        hostState.workspaceRootOptions.clear()
        hostState.workspaceRootOptions.addAll(normalizedRoots)
        hostState.activeWorkspaceRoots.clear()
        hostState.activeWorkspaceRoots.addAll(nextActiveRoots)
        sendHostMessage(JSONObject().put("type", "workspace-root-options-updated"))
        sendHostMessage(JSONObject().put("type", "active-workspace-roots-updated"))
    }

    private fun mergeWorkspaceRoots(nextRoots: List<String>, preferredRoot: String? = null) {
        updateWorkspaceRoots(hostState.workspaceRootOptions + nextRoots, preferredRoot)
    }

    private fun setActiveWorkspaceRoot(root: String) {
        val normalizedRoot = normalizeWorkspaceRootCandidate(root) ?: return
        val nextOptions =
            if (hostState.workspaceRootOptions.contains(normalizedRoot)) {
                hostState.workspaceRootOptions.toMutableList()
            } else {
                mutableListOf(normalizedRoot).apply { addAll(hostState.workspaceRootOptions) }
            }
        val nextActiveRoots = mutableListOf(normalizedRoot)
        val optionsChanged = hostState.workspaceRootOptions != nextOptions
        val activeChanged = hostState.activeWorkspaceRoots != nextActiveRoots
        if (!optionsChanged && !activeChanged) {
            return
        }

        hostState.workspaceRootOptions.clear()
        hostState.workspaceRootOptions.addAll(nextOptions)
        hostState.activeWorkspaceRoots.clear()
        hostState.activeWorkspaceRoots.addAll(nextActiveRoots)
        if (optionsChanged) {
            sendHostMessage(JSONObject().put("type", "workspace-root-options-updated"))
        }
        if (activeChanged) {
            sendHostMessage(JSONObject().put("type", "active-workspace-roots-updated"))
        }
    }

    private fun normalizeWorkspaceRootCandidate(value: String?): String? {
        val normalized = value?.trim()?.replace('\\', '/')?.replace(Regex("/+"), "/").orEmpty()
        return normalized.ifBlank { null }
    }

    private fun hydrateBridgeBootstrapState(
        loadTarget: BridgeLoadTarget,
        authToken: String?,
    ): BridgeBootstrapState {
        val snapshot = BridgeApi.fetchGlobalStateSnapshotByBaseUrl(loadTarget.baseUrl, authToken)
        val state = snapshot.optJSONObject("state") ?: JSONObject()
        val workspacePayload = snapshot.optJSONObject("workspaceRootOptions")

        val persistedState = state.optJSONObject("electron-persisted-atom-state")
        val workspaceRoots =
            jsonArrayStrings(workspacePayload?.optJSONArray("roots")).ifEmpty {
                jsonArrayStrings(state.optJSONArray("project-order")) +
                    jsonArrayStrings(state.optJSONArray("electron-saved-workspace-roots")) +
                    jsonArrayStrings(state.optJSONArray("active-workspace-roots"))
            }
        val activeRoots =
            jsonArrayStrings(workspacePayload?.optJSONArray("activeRoots")).ifEmpty {
                jsonArrayStrings(state.optJSONArray("active-workspace-roots"))
            }
        val workspaceLabels =
            jsonObjectStringMap(workspacePayload?.optJSONObject("labels")).ifEmpty {
                jsonObjectStringMap(state.optJSONObject("workspace-root-labels"))
            }
        val pinnedThreadIds = jsonArrayStrings(state.optJSONArray("pinned-thread-ids"))

        val globalSnapshot = linkedMapOf<String, Any?>()
        val keys = state.keys()
        while (keys.hasNext()) {
            val key = keys.next()
            globalSnapshot[key] = deepCopyJsonValue(state.opt(key))
        }

        return BridgeBootstrapState(
            persistedAtomState = persistedState?.let(::deepCopyJsonObject),
            workspaceRootOptions = uniqueTrimmedStrings(workspaceRoots),
            activeWorkspaceRoots = uniqueTrimmedStrings(activeRoots),
            workspaceRootLabels = workspaceLabels,
            pinnedThreadIds = uniqueTrimmedStrings(pinnedThreadIds),
            globalState = globalSnapshot,
        )
    }

    private fun applyBridgeBootstrapState(state: BridgeBootstrapState) {
        persistedAtomState = deepCopyJsonObject(DEFAULT_PERSISTED_ATOM_STATE)
        state.persistedAtomState?.let { persisted ->
            val keys = persisted.keys()
            while (keys.hasNext()) {
                val key = keys.next()
                persistedAtomState.put(key, toJsonCompatible(persisted.opt(key)))
            }
        }

        globalState.clear()
        state.globalState.forEach { (key, value) ->
            globalState[key] = deepCopyJsonValue(value)
        }

        hostState.workspaceRootLabels.clear()
        hostState.workspaceRootLabels.putAll(state.workspaceRootLabels)
        hostState.pinnedThreadIds.clear()
        hostState.pinnedThreadIds.addAll(state.pinnedThreadIds)
        updateWorkspaceRoots(
            nextRoots = state.workspaceRootOptions,
            preferredRoot = state.activeWorkspaceRoots.firstOrNull(),
        )
    }

    private fun jsonArrayStrings(values: JSONArray?): List<String> {
        if (values == null) {
            return emptyList()
        }
        val result = mutableListOf<String>()
        for (index in 0 until values.length()) {
            val value = values.optString(index).trim()
            if (value.isNotEmpty()) {
                result.add(value)
            }
        }
        return result
    }

    private fun jsonObjectStringMap(value: JSONObject?): Map<String, String> {
        if (value == null) {
            return emptyMap()
        }
        val result = linkedMapOf<String, String>()
        val keys = value.keys()
        while (keys.hasNext()) {
            val key = keys.next()
            val text = value.optString(key).trim()
            if (text.isNotEmpty()) {
                result[key] = text
            }
        }
        return result
    }

    private fun mergeWorkspaceRootsFromResult(result: JSONObject) {
        val candidates = mutableListOf<String>()

        fun collectThreadRoots(value: JSONObject?) {
            val cwd = normalizeWorkspaceRootCandidate(value?.optString("cwd"))
            if (!cwd.isNullOrBlank()) {
                candidates.add(cwd)
            }
        }

        normalizeWorkspaceRootCandidate(result.optString("cwd"))?.let(candidates::add)
        collectThreadRoots(result.optJSONObject("thread"))

        val data = result.optJSONArray("data")
        if (data != null) {
            for (index in 0 until data.length()) {
                collectThreadRoots(data.optJSONObject(index))
            }
        }

        if (candidates.isNotEmpty()) {
            mergeWorkspaceRoots(candidates, candidates.firstOrNull())
        }
    }

    private fun uniqueTrimmedStrings(values: List<String>): MutableList<String> {
        val result = mutableListOf<String>()
        val seen = linkedSetOf<String>()
        values.forEach { value ->
            val normalized = value.trim()
            if (normalized.isNotEmpty() && seen.add(normalized)) {
                result.add(normalized)
            }
        }
        return result
    }

    private fun deepCopyJsonObject(value: JSONObject): JSONObject = JSONObject(value.toString())

    private fun deepCopyJsonValue(value: Any?): Any? =
        when (value) {
            null, JSONObject.NULL -> JSONObject.NULL
            is JSONObject -> JSONObject(value.toString())
            is JSONArray -> JSONArray(value.toString())
            is Map<*, *> -> {
                val objectValue = JSONObject()
                value.forEach { (key, entryValue) ->
                    if (key is String) {
                        objectValue.put(key, deepCopyJsonValue(entryValue))
                    }
                }
                objectValue
            }
            is Iterable<*> -> {
                val arrayValue = JSONArray()
                value.forEach { entryValue ->
                    arrayValue.put(deepCopyJsonValue(entryValue))
                }
                arrayValue
            }
            else -> value
        }

    private fun toJsonCompatible(value: Any?): Any =
        when (val copied = deepCopyJsonValue(value)) {
            null -> JSONObject.NULL
            else -> copied
        }

    private fun deriveLocaleInfo(): JSONObject {
        val locale = Locale.getDefault().toLanguageTag().ifBlank { "en-US" }
        return JSONObject()
            .put("ideLocale", locale)
            .put("systemLocale", locale)
    }

    private fun resolveFetchMethodPayload(method: String, params: JSONObject?): Any {
        return when (method) {
            "get-global-state" -> JSONObject().put(
                "value",
                toJsonCompatible(params?.optString("key")?.takeIf { it.isNotBlank() }?.let { globalState[it] }),
            )
            "set-global-state" -> {
                val key = params?.optString("key")?.trim().orEmpty()
                if (key.isEmpty()) {
                    JSONObject().put("success", false)
                } else {
                    if (params?.has("value") == true) {
                        globalState[key] = params.opt("value")
                    } else {
                        globalState.remove(key)
                    }
                    JSONObject().put("success", true)
                }
            }
            "get-configuration" -> JSONObject().put(
                "value",
                toJsonCompatible(params?.optString("key")?.takeIf { it.isNotBlank() }?.let { configurationState[it] }),
            )
            "set-configuration" -> {
                val key = params?.optString("key")?.trim().orEmpty()
                if (key.isEmpty()) {
                    JSONObject().put("success", false)
                } else {
                    if (params?.has("value") == true) {
                        configurationState[key] = params.opt("value")
                    } else {
                        configurationState.remove(key)
                    }
                    JSONObject().put("success", true)
                }
            }
            "list-pinned-threads" -> JSONObject().put("threadIds", JSONArray(hostState.pinnedThreadIds))
            "set-thread-pinned" -> {
                val threadId = params?.optString("threadId")?.trim().orEmpty()
                if (threadId.isEmpty()) {
                    JSONObject()
                        .put("success", false)
                        .put("threadIds", JSONArray(hostState.pinnedThreadIds))
                } else {
                    val pinned = params?.optBoolean("pinned", false) == true
                    if (pinned) {
                        val nextThreadIds = uniqueTrimmedStrings(hostState.pinnedThreadIds + threadId)
                        hostState.pinnedThreadIds.clear()
                        hostState.pinnedThreadIds.addAll(nextThreadIds)
                    } else {
                        hostState.pinnedThreadIds.removeAll { it == threadId }
                    }
                    JSONObject()
                        .put("success", true)
                        .put("threadIds", JSONArray(hostState.pinnedThreadIds))
                }
            }
            "set-pinned-threads-order" -> {
                val nextThreadIds =
                    if (params?.has("threadIds") == true) {
                        uniqueTrimmedStrings(
                            buildList {
                                val values = params.optJSONArray("threadIds")
                                if (values != null) {
                                    for (index in 0 until values.length()) {
                                        add(values.optString(index))
                                    }
                                }
                            },
                        )
                    } else {
                        hostState.pinnedThreadIds.toMutableList()
                    }
                hostState.pinnedThreadIds.clear()
                hostState.pinnedThreadIds.addAll(nextThreadIds)
                JSONObject()
                    .put("success", true)
                    .put("threadIds", JSONArray(hostState.pinnedThreadIds))
            }
            "experimentalFeature/list" -> JSONObject()
                .put("data", JSONArray())
                .put("nextCursor", JSONObject.NULL)
            "os-info" -> JSONObject()
                .put("platform", "android")
                .put("isMacOS", false)
                .put("isWindows", false)
                .put("isLinux", false)
            "locale-info" -> deriveLocaleInfo()
            "active-workspace-roots" -> JSONObject().put("roots", JSONArray(hostState.activeWorkspaceRoots))
            "workspace-root-options" -> JSONObject()
                .put("roots", JSONArray(hostState.workspaceRootOptions))
                .put("activeRoots", JSONArray(hostState.activeWorkspaceRoots))
                .put("labels", JSONObject(hostState.workspaceRootLabels.toMap()))
            "paths-exist" -> {
                val existingPaths = JSONArray()
                val paths = params?.optJSONArray("paths")
                if (paths != null) {
                    for (index in 0 until paths.length()) {
                        val path = normalizeWorkspaceRootCandidate(paths.optString(index))
                        if (!path.isNullOrBlank()) {
                            existingPaths.put(path)
                        }
                    }
                }
                JSONObject().put("existingPaths", existingPaths)
            }
            else -> UNHANDLED_LOCAL_METHOD
        }
    }

    private fun resetAppServerWebSocketClient() {
        appServerWebSocketClient?.close()
        appServerWebSocketClient = null
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
            onLog = { message, error ->
                android.util.Log.w("CodexMobile", message, error)
            },
        ).also {
            appServerWebSocketClient = it
        }
    }

    private fun performAppServerMcpRequest(
        loadTarget: BridgeLoadTarget,
        authToken: String?,
        method: String,
        params: JSONObject,
    ): JSONObject {
        return getAppServerWebSocketClient(loadTarget, authToken).request(method, params)
    }

    private fun integrateDirectRpcResult(method: String, result: JSONObject) {
        when (method) {
            "account/read" -> hostState.account = deepCopyJsonObject(result)
            "account/rateLimits/read" -> hostState.rateLimit = deepCopyJsonObject(result)
            "thread/list", "thread/read", "thread/start", "thread/resume" -> mergeWorkspaceRootsFromResult(result)
            "workspace-root-options" -> {
                val labels = jsonObjectStringMap(result.optJSONObject("labels"))
                hostState.workspaceRootLabels.clear()
                hostState.workspaceRootLabels.putAll(labels)
                updateWorkspaceRoots(
                    nextRoots = jsonArrayStrings(result.optJSONArray("roots")),
                    preferredRoot = jsonArrayStrings(result.optJSONArray("activeRoots")).firstOrNull(),
                )
            }
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
            "account/updated", "account/changed" -> {
                hostState.account = deepCopyJsonObject(params.optJSONObject("account") ?: JSONObject())
            }
            "account/rateLimits/updated", "rate-limit/updated", "rate-limit/changed",
            "rateLimit/updated", "rateLimit/changed" -> {
                hostState.rateLimit = deepCopyJsonObject(params.optJSONObject("rateLimit") ?: JSONObject())
            }
        }

        emitMcpNotification(method, deepCopyJsonObject(params))
    }

    private fun rememberPendingTurnCompletion(threadId: String?, turnId: String?) {
        if (threadId.isNullOrBlank() || turnId.isNullOrBlank()) {
            return
        }
        pendingTurnCompletions[threadId] = turnId
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

    private fun scheduleTurnCompletionFallback(
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

    private fun emitMcpNotification(method: String, params: JSONObject) {
        sendHostMessage(
            JSONObject()
                .put("type", "mcp-notification")
                .put("hostId", LOCAL_HOST_ID)
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

    private fun reconcileCompletedThreadState(threadId: String, loadTarget: BridgeLoadTarget, authToken: String?) {
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
                android.util.Log.w("CodexMobile", "failed to reconcile thread completion for $threadId", error)
            } finally {
                activeTurnReconciliations.remove(threadId)
            }
        }
    }

    private fun reconcileThreadSnapshotIfNeeded(
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

    private fun injectHostMessages(messages: JSONArray) {
        if (messages.length() == 0) {
            return
        }
        val script =
            """
            (function () {
              var host = window.__codexMobileHost;
              if (!host || typeof host.dispatchHostMessage !== "function") {
                return;
              }
              var messages = ${messages};
              for (var index = 0; index < messages.length; index += 1) {
                host.dispatchHostMessage(messages[index]);
              }
            })();
            """.trimIndent()
        webView.post {
            webView.evaluateJavascript(script, null)
        }
    }

    private fun sendHostMessage(message: JSONObject) {
        val messages = JSONArray()
        messages.put(message)
        injectHostMessages(messages)
    }

    private fun sendBridgeResponse(requestId: String, result: JSONObject?) {
        val serializedResult = result?.toString() ?: "null"
        val script =
            """
            (function () {
              var host = window.__codexMobileHost;
              if (host && typeof host.resolveBridgeRequest === "function") {
                host.resolveBridgeRequest(${JSONObject.quote(requestId)}, ${serializedResult});
              }
            })();
            """.trimIndent()
        webView.post {
            webView.evaluateJavascript(script, null)
        }
    }

    private fun sendWorkerMessage(workerId: String, payload: JSONObject) {
        val script =
            """
            (function () {
              var host = window.__codexMobileHost;
              if (host && typeof host.dispatchWorkerMessage === "function") {
                host.dispatchWorkerMessage(${JSONObject.quote(workerId)}, ${payload});
              }
            })();
            """.trimIndent()
        webView.post {
            webView.evaluateJavascript(script, null)
        }
    }

    private fun handleEnvelope(rawMessage: String) {
        try {
            val envelope = JSONObject(rawMessage)
            if (!envelope.optBoolean("__codexMobile", false)) {
                return
            }
            when (envelope.optString("kind")) {
                "preload-ready" -> {
                    android.util.Log.d("CodexMobile", "preload ready ${envelope.opt("payload")}")
                    activeProfile?.let { profile ->
                        setStatus("Connected to ${profile.serverEndpoint}")
                    }
                }
                "console" -> android.util.Log.d("CodexMobile", "renderer ${envelope.opt("payload")}")
                "bridge-send-message" -> {
                    val payload = envelope.optJSONObject("payload") ?: return
                    android.util.Log.d(
                        "CodexMobile",
                        "bridge-send-message ${payload.optString("type")} ${payload.optJSONObject("request")?.optString("method")}",
                    )
                    handleRendererMessage(payload)
                }
                "bridge-send-worker-message" -> {
                    val payload = envelope.optJSONObject("payload") ?: return
                    android.util.Log.d("CodexMobile", "bridge-send-worker-message ${payload.optString("workerId")}")
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
            android.util.Log.w("CodexMobile", "failed to parse webview envelope", error)
        }
    }

    private fun handleRendererReady() {
        android.util.Log.d("CodexMobile", "renderer ready")
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
        android.util.Log.d("CodexMobile", "renderer message ${message.optString("type")}")
        when (message.optString("type")) {
            "ready" -> handleRendererReady()
            "persisted-atom-sync-request" -> sendPersistedAtomSync()
            "persisted-atom-update" -> {
                val key = message.optString("key").trim()
                if (key.isEmpty()) {
                    return
                }
                val deleted = message.optBoolean("deleted", false)
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
                persistedAtomState = deepCopyJsonObject(DEFAULT_PERSISTED_ATOM_STATE)
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
                        startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
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
                val root = normalizeWorkspaceRootCandidate(message.optString("root")) ?: return
                if (!hostState.workspaceRootOptions.contains(root)) {
                    return
                }
                val label = message.optString("label").trim()
                if (label.isEmpty()) {
                    hostState.workspaceRootLabels.remove(root)
                } else {
                    hostState.workspaceRootLabels[root] = label
                }
                sendHostMessage(JSONObject().put("type", "workspace-root-options-updated"))
            }
            "electron-set-active-workspace-root" -> {
                val root = message.optString("root").trim()
                if (root.isNotEmpty()) {
                    setActiveWorkspaceRoot(root)
                }
            }
            "fetch" -> handleFetchMessage(message)
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
                android.util.Log.d("CodexMobile", "renderer log-message ${message.opt("message")}")
            }
            "mcp-request" -> handleMcpRequestMessage(message)
        }
    }

    private fun handleFetchMessage(message: JSONObject) {
        val requestId = message.optString("requestId").trim()
        val rawUrl = message.optString("url").trim()
        if (requestId.isEmpty() || rawUrl.isEmpty()) {
            return
        }
        val profile = activeProfile ?: return
        val loadTarget = activeLoadTarget ?: resolveBridgeLoadTarget(profile)
        thread {
            try {
                val method = resolveRequestMethodName(rawUrl)
                if (method != null) {
                    val localResult = resolveFetchMethodPayload(method, parseJsonBodyObject(message.optString("body")))
                    if (localResult !== UNHANDLED_LOCAL_METHOD) {
                        sendHostMessage(buildFetchSuccessResponse(requestId, localResult, 200, JSONObject()))
                        return@thread
                    }
                    val result = BridgeApi.performDirectRpcByBaseUrl(
                        baseUrl = loadTarget.baseUrl,
                        method = method,
                        params = parseJsonBodyObject(message.optString("body")) ?: JSONObject(),
                        authToken = if (loadTarget.usesLocalProxy) null else profile.authToken,
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
                    authToken = if (loadTarget.usesLocalProxy) null else profile.authToken,
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
                android.util.Log.w("CodexMobile", "fetch handler failed for $rawUrl", error)
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
        val parsedBody = parseJsonValue(responseText)
        return HttpProxyResponse(
            body = parsedBody,
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

    private fun normalizeErrorMessage(error: Throwable): String {
        val message = error.message?.trim().orEmpty()
        if (message.equals("Invalid or expired pairing code", ignoreCase = true)) {
            return getString(R.string.native_host_error_pairing_expired)
        }
        return if (message.isNotEmpty()) {
            message
        } else {
            error::class.java.simpleName
        }
    }

    private fun handleMcpRequestMessage(message: JSONObject) {
        val request = message.optJSONObject("request") ?: return
        val requestId = request.opt("id")?.toString()?.trim().orEmpty()
        val method = request.optString("method").trim()
        if (requestId.isEmpty() || method.isEmpty()) {
            return
        }
        android.util.Log.d("CodexMobile", "mcp request $method id=$requestId")
        val profile = activeProfile ?: return
        val loadTarget = activeLoadTarget ?: resolveBridgeLoadTarget(profile)
        thread {
            try {
                val localResult = resolveFetchMethodPayload(method, request.optJSONObject("params"))
                if (localResult !== UNHANDLED_LOCAL_METHOD) {
                    val localObject =
                        when (localResult) {
                            is JSONObject -> localResult
                            else -> JSONObject.wrap(localResult) as? JSONObject ?: JSONObject()
                        }
                    sendHostMessage(
                        JSONObject()
                            .put("type", "mcp-response")
                            .put("hostId", LOCAL_HOST_ID)
                            .put("id", request.opt("id"))
                            .put("result", localObject)
                            .put("message", JSONObject().put("id", request.opt("id")).put("result", localObject))
                            .put("response", JSONObject().put("id", request.opt("id")).put("result", localObject)),
                    )
                    return@thread
                }
                val requestParams = request.optJSONObject("params") ?: JSONObject()
                val requestAuthToken = if (loadTarget.usesLocalProxy) null else profile.authToken
                val result =
                    try {
                        performAppServerMcpRequest(
                            loadTarget = loadTarget,
                            authToken = requestAuthToken,
                            method = method,
                            params = requestParams,
                        )
                    } catch (socketError: Exception) {
                        android.util.Log.w(
                            "CodexMobile",
                            "app server websocket request failed; falling back to direct RPC for $method",
                            socketError,
                        )
                        resetAppServerWebSocketClient()
                        BridgeApi.performDirectRpcByBaseUrl(
                            baseUrl = loadTarget.baseUrl,
                            method = method,
                            params = requestParams,
                            authToken = requestAuthToken,
                        )
                    }
                integrateDirectRpcResult(method, result)
                android.util.Log.d("CodexMobile", "mcp response ok $method id=$requestId")
                sendHostMessage(
                    JSONObject()
                        .put("type", "mcp-response")
                        .put("hostId", LOCAL_HOST_ID)
                        .put("id", request.opt("id"))
                        .put(
                            "result",
                            result,
                        )
                        .put(
                            "message",
                            JSONObject().put("id", request.opt("id")).put("result", result),
                        )
                        .put(
                            "response",
                            JSONObject().put("id", request.opt("id")).put("result", result),
                        ),
                )
                if (method == "turn/start") {
                    val requestThreadId = request.optJSONObject("params")?.optString("threadId")?.trim().orEmpty()
                    val responseTurnId = result.optJSONObject("turn")?.optString("id")?.trim().orEmpty()
                    rememberPendingTurnCompletion(
                        threadId = requestThreadId.ifEmpty { null },
                        turnId = responseTurnId.ifEmpty { null },
                    )
                    if (requestThreadId.isNotEmpty()) {
                        android.util.Log.d(
                            "CodexMobile",
                            "schedule turn fallback thread=$requestThreadId turn=$responseTurnId",
                        )
                        scheduleTurnCompletionFallback(
                            threadId = requestThreadId,
                            turnId = responseTurnId.ifEmpty { null },
                            loadTarget = loadTarget,
                            authToken = requestAuthToken,
                        )
                    }
                } else {
                    reconcileThreadSnapshotIfNeeded(
                        method = method,
                        result = result,
                        loadTarget = loadTarget,
                        authToken = requestAuthToken,
                    )
                }
            } catch (error: Exception) {
                android.util.Log.w("CodexMobile", "mcp response error $method id=$requestId", error)
                val errorPayload = JSONObject().put("message", error.message ?: "MCP request failed.")
                sendHostMessage(
                    JSONObject()
                        .put("type", "mcp-response")
                        .put("hostId", LOCAL_HOST_ID)
                        .put("id", request.opt("id"))
                        .put("error", errorPayload)
                        .put(
                            "message",
                            JSONObject().put("id", request.opt("id")).put("error", errorPayload),
                        )
                        .put(
                            "response",
                            JSONObject().put("id", request.opt("id")).put("error", errorPayload),
                        ),
                )
            }
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

    private inner class NativeHostJavascriptBridge {
        @JavascriptInterface
        fun postMessage(message: String) {
            android.util.Log.d(
                "CodexMobile",
                "native bridge raw ${message.take(180)}",
            )
            runOnUiThread {
                handleEnvelope(message)
            }
        }
    }

    private fun openConnectionSheet() {
        val dialog = BottomSheetDialog(this)
        val content = LayoutInflater.from(this).inflate(R.layout.sheet_native_host_actions, null)
        val titleView = content.findViewById<TextView>(R.id.nativeHostSheetTitle)
        val bodyView = content.findViewById<TextView>(R.id.nativeHostSheetBody)
        val metadataView = content.findViewById<View>(R.id.nativeHostSheetMetadata)
        val metadataTitleView = content.findViewById<TextView>(R.id.nativeHostSheetMetadataTitle)
        val metadataValueView = content.findViewById<TextView>(R.id.nativeHostSheetMetadataValue)
        val primaryButton = content.findViewById<MaterialButton>(R.id.nativeHostSheetPrimaryAction)
        val secondaryButton = content.findViewById<MaterialButton>(R.id.nativeHostSheetSecondaryAction)
        val tertiaryButton = content.findViewById<MaterialButton>(R.id.nativeHostSheetTertiaryAction)
        val savedConnectionsSection = content.findViewById<View>(R.id.nativeHostSheetConnectionsSection)
        val savedConnectionsList = content.findViewById<LinearLayout>(R.id.nativeHostSheetConnectionsList)

        val profile = activeProfile ?: profileStore.readActive()
        val savedProfiles = profileStore.list()
        if (profile == null) {
            titleView.text = getString(R.string.native_host_sheet_connect_title)
            bodyView.text = getString(R.string.native_host_sheet_connect_body)
            metadataView.visibility = View.GONE
            renderSavedConnectionCards(
                container = savedConnectionsList,
                profiles = savedProfiles,
                activeProfileId = null,
                dialog = dialog,
            )
            savedConnectionsSection.visibility =
                if (savedProfiles.isEmpty()) {
                    View.GONE
                } else {
                    View.VISIBLE
                }
            primaryButton.text = getString(R.string.native_host_sheet_action_scan)
            primaryButton.setOnClickListener {
                dialog.dismiss()
                openScanner()
            }
            secondaryButton.visibility =
                if (savedProfiles.isEmpty()) {
                    View.GONE
                } else {
                    View.VISIBLE
                }
            secondaryButton.text = getString(R.string.native_host_sheet_action_open_saved)
            secondaryButton.setOnClickListener {
                val nextProfile = profileStore.readActive() ?: savedProfiles.firstOrNull() ?: return@setOnClickListener
                dialog.dismiss()
                activateProfile(nextProfile)
            }
            tertiaryButton.visibility = View.GONE
        } else {
            titleView.text = getString(R.string.native_host_sheet_connected_title)
            bodyView.text = getString(R.string.native_host_sheet_connected_body)
            metadataView.visibility = View.VISIBLE
            metadataTitleView.text = getString(R.string.native_host_sheet_current_bridge)
            metadataValueView.text =
                buildString {
                    append(profile.name)
                    append('\n')
                    append(profile.serverEndpoint)
                    append("\n\n")
                    append(getString(R.string.native_host_sheet_status))
                    append(": ")
                    append(currentStatusMessage)
                }
            renderSavedConnectionCards(
                container = savedConnectionsList,
                profiles = savedProfiles,
                activeProfileId = profile.id,
                dialog = dialog,
            )
            savedConnectionsSection.visibility =
                if (savedProfiles.isEmpty()) {
                    View.GONE
                } else {
                    View.VISIBLE
                }
            primaryButton.text = getString(R.string.native_host_sheet_action_reload)
            primaryButton.setOnClickListener {
                dialog.dismiss()
                reloadActiveBridge()
            }
            secondaryButton.visibility = View.VISIBLE
            secondaryButton.text = getString(R.string.native_host_sheet_action_rescan)
            secondaryButton.setOnClickListener {
                dialog.dismiss()
                openScanner()
            }
            tertiaryButton.visibility = View.VISIBLE
            tertiaryButton.text = getString(R.string.native_host_sheet_action_reset)
            tertiaryButton.setOnClickListener {
                dialog.dismiss()
                confirmReset(profile)
            }
        }

        dialog.setContentView(content)
        dialog.show()
    }

    private fun renderSavedConnectionCards(
        container: LinearLayout,
        profiles: List<BridgeProfile>,
        activeProfileId: String?,
        dialog: BottomSheetDialog,
    ) {
        container.removeAllViews()
        profiles.forEachIndexed { index, profile ->
            val card = LayoutInflater.from(this).inflate(R.layout.item_native_host_connection, container, false)
            val layoutParams =
                (card.layoutParams as? LinearLayout.LayoutParams)
                    ?: LinearLayout.LayoutParams(
                        LinearLayout.LayoutParams.MATCH_PARENT,
                        LinearLayout.LayoutParams.WRAP_CONTENT,
                    )
            if (index == 0) {
                layoutParams.topMargin = 0
            }
            card.layoutParams = layoutParams

            val nameView = card.findViewById<TextView>(R.id.nativeHostConnectionName)
            val endpointView = card.findViewById<TextView>(R.id.nativeHostConnectionEndpoint)
            val badgeView = card.findViewById<TextView>(R.id.nativeHostConnectionBadge)
            val modeView = card.findViewById<TextView>(R.id.nativeHostConnectionMode)
            val actionButton = card.findViewById<MaterialButton>(R.id.nativeHostConnectionAction)

            nameView.text = profile.name
            endpointView.text = profile.serverEndpoint
            val isCurrent = profile.id == activeProfileId
            badgeView.text =
                getString(
                    if (isCurrent) {
                        R.string.native_host_sheet_connection_current_badge
                    } else {
                        R.string.native_host_sheet_connection_saved_badge
                    },
                )
            applyChipTone(
                badgeView,
                if (isCurrent) {
                    ChipTone.SUCCESS
                } else {
                    ChipTone.NEUTRAL
                },
            )
            modeView.text =
                getString(
                    if (profile.tailnetEnrollmentPayload.isNullOrBlank()) {
                        R.string.native_host_sheet_connection_direct
                    } else {
                        R.string.native_host_sheet_connection_tailnet
                    },
                )

            if (isCurrent) {
                actionButton.isEnabled = false
                actionButton.text = getString(R.string.native_host_sheet_connection_active_action)
            } else {
                actionButton.isEnabled = true
                actionButton.text = getString(R.string.native_host_sheet_connection_switch)
                actionButton.setOnClickListener {
                    dialog.dismiss()
                    activateProfile(profile)
                }
            }

            container.addView(card)
        }
    }

    private fun confirmReset(profile: BridgeProfile) {
        MaterialAlertDialogBuilder(this)
            .setTitle(R.string.native_host_reset_title)
            .setMessage(getString(R.string.native_host_reset_body, profile.name))
            .setNegativeButton("Cancel", null)
            .setPositiveButton("Reset") { _, _ ->
                resetEnrollment(profile)
            }
            .show()
    }

    private fun resetEnrollment(profile: BridgeProfile) {
        val nextProfile = profileStore.remove(profile.id)
        syncSavedConnectionsState()
        if (nextProfile == null) {
            EnrollmentStore(applicationContext).clear()
            startService(CodexTailnetService.stopIntent(this))
            renderEmptyState(getString(R.string.native_host_status_idle))
            return
        }
        if (nextProfile.tailnetEnrollmentPayload.isNullOrBlank()) {
            EnrollmentStore(applicationContext).clear()
            startService(CodexTailnetService.stopIntent(this))
        }
        openBridge(nextProfile)
    }

    private fun openScanner() {
        val options = ScanOptions()
            .setDesiredBarcodeFormats(ScanOptions.QR_CODE)
            .setPrompt("Scan Codex Mobile enrollment QR")
            .setBeepEnabled(false)
            .setOrientationLocked(true)
        scanLauncher.launch(options)
    }

    private fun importEnrollment(rawJson: String) {
        try {
            when (val payload = EnrollmentParser.parse(rawJson)) {
                is EnrollmentPayload.Bridge -> {
                    updateConnectionProgress(
                        profileName = payload.name,
                        endpoint = payload.serverEndpoint,
                        stage = ConnectionStage.PAYLOAD_RECEIVED,
                        requiresPairing = !payload.pairingCode.isNullOrBlank(),
                    )
                    setStatus(getString(R.string.native_host_status_payload_received))
                    saveBridgeProfile(
                        name = payload.name,
                        endpoint = payload.serverEndpoint,
                        pairingCode = payload.pairingCode.orEmpty(),
                        existingAuthToken = null,
                        tailnetEnrollmentPayload = null,
                    )
                }

                is EnrollmentPayload.Tailnet -> {
                    updateConnectionProgress(
                        profileName = payload.bridgeName,
                        endpoint = payload.bridgeServerEndpoint,
                        stage = ConnectionStage.PAYLOAD_RECEIVED,
                        requiresPairing = !payload.pairingCode.isNullOrBlank(),
                    )
                    setStatus(getString(R.string.native_host_status_payload_received))
                    val stagedPayload = parseTailnetEnrollmentPayload(payload.rawJson)
                    val snapshot = CodexTailnetBridge.stage(applicationContext, stagedPayload)
                    updateConnectionProgress(
                        profileName = payload.bridgeName,
                        endpoint = payload.bridgeServerEndpoint,
                        stage = ConnectionStage.STARTING_TAILNET,
                        requiresPairing = !payload.pairingCode.isNullOrBlank(),
                    )
                    setStatus(snapshot.message)
                    ContextCompat.startForegroundService(
                        this,
                        CodexTailnetService.startIntent(this, payload.rawJson),
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
            setStatus(normalizeErrorMessage(error))
        }
    }

    private fun saveBridgeProfile(
        name: String,
        endpoint: String,
        pairingCode: String,
        existingAuthToken: String?,
        tailnetEnrollmentPayload: String?,
    ) {
        setStatus(getString(R.string.native_host_status_preparing_connection))
        thread {
            try {
                val normalizedEndpoint = BridgeApi.normalizeEndpoint(endpoint)
                var authToken = existingAuthToken
                runOnUiThread {
                    updateConnectionProgress(
                        profileName = name,
                        endpoint = normalizedEndpoint,
                        stage =
                            if (BridgeApi.isLikelyTailnetEndpoint(normalizedEndpoint)) {
                                ConnectionStage.STARTING_TAILNET
                            } else if (pairingCode.isNotBlank()) {
                                ConnectionStage.PAIRING_DEVICE
                            } else {
                                ConnectionStage.OPENING_WORKSPACE
                            },
                        requiresPairing = pairingCode.isNotBlank(),
                    )
                }
                ensureTailnetRuntimeReady(normalizedEndpoint, tailnetEnrollmentPayload)
                var connectionTarget = resolveBridgeLoadTarget(normalizedEndpoint, authToken, tailnetEnrollmentPayload)
                var connection = BridgeApi.fetchConnectionTargetByBaseUrl(
                    baseUrl = connectionTarget.baseUrl,
                    authToken = if (connectionTarget.usesLocalProxy) null else authToken,
                )
                if (pairingCode.isNotBlank()) {
                    runOnUiThread {
                        updateConnectionProgress(
                            profileName = name,
                            endpoint = normalizedEndpoint,
                            stage = ConnectionStage.PAIRING_DEVICE,
                            requiresPairing = true,
                        )
                        setStatus(getString(R.string.native_host_status_pairing_device))
                    }
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
                    EnrollmentStore(applicationContext).clear()
                    startService(CodexTailnetService.stopIntent(this))
                }
                runOnUiThread {
                    updateConnectionProgress(
                        profileName = profile.name,
                        endpoint = profile.serverEndpoint,
                        stage = ConnectionStage.OPENING_WORKSPACE,
                        requiresPairing = pairingCode.isNotBlank(),
                    )
                    setStatus(getString(R.string.native_host_status_opening_workspace))
                    openBridge(profile)
                    setStatus("Connected to ${profile.serverEndpoint}")
                }
            } catch (error: Exception) {
                android.util.Log.w("CodexMobile", "failed to save/open bridge profile", error)
                runOnUiThread {
                    setStatus(normalizeErrorMessage(error))
                }
            }
        }
    }

    private fun resolveBridgeLoadTarget(profile: BridgeProfile): BridgeLoadTarget {
        return resolveBridgeLoadTarget(
            endpoint = profile.serverEndpoint,
            authToken = profile.authToken,
            tailnetEnrollmentPayload = profile.tailnetEnrollmentPayload,
        )
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
                this,
                CodexTailnetService.startIntent(this, enrollment.rawPayload),
            )
        }

        var lastSnapshot = CodexTailnetBridge.configureBridgeProxy(applicationContext, endpoint, authToken)
        repeat(if (matchesEnrollment) 60 else 1) { attempt ->
            lastSnapshot =
                if (attempt == 0) {
                    lastSnapshot
                } else {
                    Thread.sleep(250L)
                    CodexTailnetBridge.configureBridgeProxy(applicationContext, endpoint, authToken)
                }
            val localProxyUrl = lastSnapshot.localProxyUrl?.trim().orEmpty()
            if (localProxyUrl.isNotEmpty() && BridgeApi.isBridgeReadyByBaseUrl(localProxyUrl, null)) {
                return BridgeLoadTarget(
                    baseUrl = localProxyUrl,
                    usesLocalProxy = true,
                )
            }
            if (!matchesEnrollment && lastSnapshot.state == "error") {
                throw IllegalStateException(describeTailnetState(lastSnapshot))
            }
        }
        if (matchesEnrollment) {
            throw IllegalStateException(describeTailnetState(lastSnapshot))
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

        var snapshot = CodexTailnetBridge.status(applicationContext)
        if (isTailnetRuntimeRunning(snapshot)) {
            return
        }

        ContextCompat.startForegroundService(
            this,
            CodexTailnetService.startIntent(this, enrollment.rawPayload),
        )

        repeat(60) {
            Thread.sleep(250L)
            snapshot = CodexTailnetBridge.status(applicationContext)
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
        return EnrollmentStore(applicationContext).readEnrollment()?.takeIf {
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
}
