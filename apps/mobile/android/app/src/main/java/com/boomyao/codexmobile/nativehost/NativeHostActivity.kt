package com.boomyao.codexmobile.nativehost

import android.annotation.SuppressLint
import android.content.Intent
import android.graphics.drawable.GradientDrawable
import android.net.Uri
import android.os.Bundle
import android.content.res.Configuration
import android.content.res.ColorStateList
import android.view.Gravity
import android.view.View
import android.view.ViewGroup
import android.view.ViewOutlineProvider
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
import androidx.activity.OnBackPressedCallback
import androidx.appcompat.app.AppCompatActivity
import androidx.core.content.ContextCompat
import androidx.core.view.ViewCompat
import androidx.core.view.WindowInsetsCompat
import androidx.core.view.updatePadding
import com.boomyao.codexmobile.R
import com.google.android.material.appbar.MaterialToolbar
import com.google.android.material.button.MaterialButton
import com.google.android.material.dialog.MaterialAlertDialogBuilder
import com.google.android.material.progressindicator.LinearProgressIndicator
import com.journeyapps.barcodescanner.ScanContract
import com.journeyapps.barcodescanner.ScanOptions
import org.json.JSONArray
import org.json.JSONObject
import java.io.InterruptedIOException
import java.net.ConnectException
import java.net.NoRouteToHostException
import java.net.SocketException
import java.net.SocketTimeoutException
import java.net.UnknownHostException
import java.util.Locale
import kotlin.concurrent.thread

class NativeHostActivity : AppCompatActivity() {
    private companion object {
        private const val LOCAL_HOST_ID = "local"
        private const val HOME_SESSION_LIMIT = 2
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

    enum class ShellChromeState {
        DISCONNECTED,
        LOADING,
        CONNECTED,
        ERROR,
    }

    private enum class ChipTone {
        NEUTRAL,
        ACTIVE,
        SUCCESS,
        WARNING,
    }

    private lateinit var rootContainer: ViewGroup
    private lateinit var toolbar: MaterialToolbar
    private lateinit var contentContainer: View
    private lateinit var sessionBarView: View
    private lateinit var sessionNameView: TextView
    private lateinit var sessionDetailView: TextView
    private lateinit var sessionStateChipView: TextView
    private lateinit var sessionActionButton: MaterialButton
    private lateinit var progressIndicator: LinearProgressIndicator
    private lateinit var workspaceFrameView: ViewGroup
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
    private lateinit var settingsButton: MaterialButton
    private lateinit var connectionButton: MaterialButton
    private lateinit var workspaceMessageGroupView: View
    private lateinit var workspaceMessageEyebrowView: TextView
    private lateinit var workspaceIllustrationView: ImageView
    private lateinit var welcomeSceneCardView: View
    private lateinit var workspaceEmptyTitleView: TextView
    private lateinit var workspaceEmptyBodyView: TextView
    private lateinit var welcomePrimaryActionButton: MaterialButton
    private lateinit var emptyStateContentView: LinearLayout
    private lateinit var welcomeGuideCardView: View
    private lateinit var welcomeGuideTitleView: TextView
    private lateinit var welcomeGuideBodyView: TextView
    private lateinit var recentSessionsSectionView: View
    private lateinit var recentSessionsHeaderView: View
    private lateinit var recentSessionsTitleView: TextView
    private lateinit var recentSessionsBodyView: TextView
    private lateinit var recentSessionsListView: LinearLayout
    private lateinit var profileStore: BridgeProfileStore
    private lateinit var preferencesStore: NativeHostPreferences
    private lateinit var connectionSheetController: NativeHostConnectionSheetController
    private var activeProfile: BridgeProfile? = null
    private var activeLoadTarget: BridgeLoadTarget? = null
    private var chromeState = ShellChromeState.DISCONNECTED
    private var currentConnectionStage: NativeHostConnectionStage? = null
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
    private val sessionState = NativeHostSessionState()
    private lateinit var webViewMessageRouter: NativeHostWebviewMessageRouter
    private lateinit var backendRequestRouter: NativeHostBackendRequestRouter
    private lateinit var connectionCoordinator: NativeHostConnectionCoordinator
    private lateinit var appServerCoordinator: NativeHostAppServerCoordinator
    private val scanLauncher = registerForActivityResult(ScanContract()) { result ->
        val contents = result.contents?.trim().orEmpty()
        if (contents.isEmpty()) {
            setStatus("QR scan canceled.")
            return@registerForActivityResult
        }
        importEnrollment(contents)
    }
    private var bridgeLoadGeneration = 0
    private var autoReconnectProfileId: String? = null
    private var autoReconnectAttemptCount = 0
    private var autoReconnectRunnable: Runnable? = null
    private var workspaceFrameMarginStart = 0
    private var workspaceFrameMarginTop = 0
    private var workspaceFrameMarginEnd = 0
    private var workspaceFrameMarginBottom = 0

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_native_host)

        rootContainer = findViewById(R.id.nativeHostRoot)
        profileStore = BridgeProfileStore(this)
        preferencesStore = NativeHostPreferences(this)
        contentContainer = findViewById(R.id.nativeHostContent)
        toolbar = findViewById(R.id.nativeHostToolbar)
        sessionBarView = findViewById(R.id.nativeHostSessionBar)
        sessionNameView = findViewById(R.id.nativeHostSessionName)
        sessionDetailView = findViewById(R.id.nativeHostSessionDetail)
        sessionStateChipView = findViewById(R.id.nativeHostSessionStateChip)
        sessionActionButton = findViewById(R.id.nativeHostSessionAction)
        progressIndicator = findViewById(R.id.nativeHostProgress)
        workspaceFrameView = findViewById(R.id.nativeHostWorkspaceFrame)
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
        settingsButton = findViewById(R.id.nativeHostSettingsButton)
        connectionButton = findViewById(R.id.nativeHostConnectionButton)
        emptyStateContentView = findViewById(R.id.nativeHostEmptyStateContent)
        workspaceMessageGroupView = findViewById(R.id.nativeHostWorkspaceMessageGroup)
        workspaceMessageEyebrowView = findViewById(R.id.nativeHostWorkspaceMessageEyebrow)
        workspaceIllustrationView = findViewById(R.id.nativeHostWorkspaceIllustration)
        welcomeSceneCardView = findViewById(R.id.nativeHostWelcomeSceneCard)
        workspaceEmptyTitleView = findViewById(R.id.nativeHostWorkspaceEmptyTitle)
        workspaceEmptyBodyView = findViewById(R.id.nativeHostWorkspaceEmptyBody)
        welcomePrimaryActionButton = findViewById(R.id.nativeHostWelcomePrimaryAction)
        welcomeGuideCardView = findViewById(R.id.nativeHostWelcomeGuideCard)
        welcomeGuideTitleView = findViewById(R.id.nativeHostWelcomeGuideTitle)
        welcomeGuideBodyView = findViewById(R.id.nativeHostWelcomeGuideBody)
        recentSessionsSectionView = findViewById(R.id.nativeHostRecentSessionsSection)
        recentSessionsHeaderView = findViewById(R.id.nativeHostRecentSessionsHeader)
        recentSessionsTitleView = findViewById(R.id.nativeHostRecentSessionsTitle)
        recentSessionsBodyView = findViewById(R.id.nativeHostRecentSessionsBody)
        recentSessionsListView = findViewById(R.id.nativeHostRecentSessionsList)
        webViewMessageRouter =
            NativeHostWebviewMessageRouter(
                activeProfileProvider = { activeProfile },
                setStatus = ::setStatus,
                sendPersistedAtomSync = ::sendPersistedAtomSync,
                sendHostMessage = ::sendHostMessage,
                sendBridgeResponse = ::sendBridgeResponse,
                sendWorkerMessage = ::sendWorkerMessage,
                openUrl = { url ->
                    runCatching {
                        startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
                    }
                },
                broadcastSharedObjectUpdate = ::broadcastSharedObjectUpdate,
                sharedObjects = sharedObjects,
                sharedObjectSubscribers = sharedObjectSubscribers,
                persistedAtomStateProvider = { persistedAtomState },
                setPersistedAtomState = { persistedAtomState = it },
                createDefaultPersistedAtomState = { deepCopyJsonObject(DEFAULT_PERSISTED_ATOM_STATE) },
                toJsonCompatible = ::toJsonCompatible,
                updateWorkspaceRoots = ::updateWorkspaceRoots,
                setActiveWorkspaceRoot = ::setActiveWorkspaceRoot,
                renameWorkspaceRoot = { root, label -> sessionState.renameWorkspaceRoot(root, label) },
                onFetchMessage = { message -> backendRequestRouter.handleFetchMessage(message) },
                onMcpRequestMessage = { message -> backendRequestRouter.handleMcpRequestMessage(message) },
            )
        appServerCoordinator =
            NativeHostAppServerCoordinator(
                hostId = LOCAL_HOST_ID,
                sendHostMessage = ::sendHostMessage,
                integrateDirectRpcResult = ::integrateDirectRpcResult,
                logWarning = { message, error ->
                    android.util.Log.w("CodexMobile", message, error)
                },
                onConnectionLost = ::handleBridgeConnectionLost,
            )
        connectionSheetController =
            NativeHostConnectionSheetController(
                context = this,
                profileStore = profileStore,
                activeProfileProvider = { activeProfile },
                isConnectedProvider = { chromeState == ShellChromeState.CONNECTED },
                isLoadingProvider = { chromeState == ShellChromeState.LOADING || isAutoReconnectPending() },
                isErrorProvider = { chromeState == ShellChromeState.ERROR && !isAutoReconnectPending() },
                currentStatusMessageProvider = { currentStatusMessage },
                currentConnectionStageProvider = { currentConnectionStage },
                activateProfile = ::activateProfile,
                reloadActiveBridge = ::reloadActiveBridge,
                openScanner = ::openScanner,
                resetEnrollment = { profile -> connectionCoordinator.resetEnrollment(profile) },
            )
        backendRequestRouter =
            NativeHostBackendRequestRouter(
                hostId = LOCAL_HOST_ID,
                activeProfileProvider = { activeProfile },
                activeLoadTargetProvider = { activeLoadTarget },
                resolveBridgeLoadTarget = ::resolveBridgeLoadTarget,
                resolveFetchMethodPayload = ::resolveFetchMethodPayload,
                unhandledLocalMethod = UNHANDLED_LOCAL_METHOD,
                sendHostMessage = ::sendHostMessage,
                performAppServerMcpRequest = { loadTarget, authToken, method, params ->
                    appServerCoordinator.performRequest(loadTarget, authToken, method, params)
                },
                resetAppServerWebSocketClient = { appServerCoordinator.reset() },
                integrateDirectRpcResult = ::integrateDirectRpcResult,
                rememberPendingTurnCompletion = { threadId, turnId ->
                    appServerCoordinator.rememberPendingTurnCompletion(threadId, turnId)
                },
                scheduleTurnCompletionFallback = { threadId, turnId, loadTarget, authToken ->
                    appServerCoordinator.scheduleTurnCompletionFallback(threadId, turnId, loadTarget, authToken)
                },
                reconcileThreadSnapshotIfNeeded = { method, result, loadTarget, authToken ->
                    appServerCoordinator.reconcileThreadSnapshotIfNeeded(method, result, loadTarget, authToken)
                },
                onBridgeConnectionIssue = ::handleBridgeConnectionLost,
                normalizeErrorMessage = ::normalizeErrorMessage,
            )
        connectionCoordinator =
            NativeHostConnectionCoordinator(
                context = applicationContext,
                profileStore = profileStore,
                activeProfileProvider = { activeProfile },
                openBridge = { profile ->
                    resetAutoReconnectState()
                    openBridge(profile)
                },
                renderEmptyState = ::renderEmptyState,
                renderConnectionFailure = ::renderConnectionFailure,
                setStatus = ::setStatus,
                updateConnectionProgress = ::updateConnectionProgress,
                syncSavedConnectionsState = ::syncSavedConnectionsState,
                normalizeErrorMessage = ::normalizeErrorMessage,
            )
        workspaceFrameView.clipToOutline = true
        (workspaceFrameView.layoutParams as? ViewGroup.MarginLayoutParams)?.let { params ->
            workspaceFrameMarginStart = params.marginStart
            workspaceFrameMarginTop = params.topMargin
            workspaceFrameMarginEnd = params.marginEnd
            workspaceFrameMarginBottom = params.bottomMargin
        }

        primaryActionButton.setOnClickListener { handlePrimaryAction() }
        secondaryActionButton.setOnClickListener { handleSecondaryAction() }
        settingsButton.setOnClickListener { openThemeSettings() }
        welcomePrimaryActionButton.setOnClickListener { openScanner() }
        connectionButton.setOnClickListener {
            if (chromeState == ShellChromeState.DISCONNECTED) {
                openScanner()
            } else {
                openConnectionSheet()
            }
        }
        sessionActionButton.setOnClickListener { openConnectionSheet() }
        onBackPressedDispatcher.addCallback(
            this,
            object : OnBackPressedCallback(true) {
                override fun handleOnBackPressed() {
                    if (chromeState == ShellChromeState.CONNECTED ||
                        chromeState == ShellChromeState.LOADING ||
                        chromeState == ShellChromeState.ERROR
                    ) {
                        preferencesStore.setAutoResumeActiveSession(false)
                        resetAutoReconnectState()
                        renderEmptyState(getString(R.string.native_host_status_idle))
                    } else {
                        finish()
                    }
                }
            },
        )

        applyWindowInsets()
        configureWebView()
        renderDisconnectedChrome(getString(R.string.native_host_status_idle))
        loadSavedProfile()
    }

    override fun onStart() {
        super.onStart()
        activeProfile?.let(::maybeScheduleAutoReconnect)
    }

    override fun onDestroy() {
        appServerCoordinator.reset()
        super.onDestroy()
    }

    private fun applyWindowInsets() {
        val contentPaddingTop = contentContainer.paddingTop
        val buttonLayoutParams = connectionButton.layoutParams as ViewGroup.MarginLayoutParams
        val buttonBottomMargin = buttonLayoutParams.bottomMargin
        ViewCompat.setOnApplyWindowInsetsListener(contentContainer) { view, insets ->
            val systemBars = insets.getInsets(WindowInsetsCompat.Type.systemBars())
            view.updatePadding(top = contentPaddingTop + systemBars.top)
            (connectionButton.layoutParams as? ViewGroup.MarginLayoutParams)?.let { params ->
                params.bottomMargin = buttonBottomMargin + systemBars.bottom
                connectionButton.layoutParams = params
            }
            insets
        }
        ViewCompat.requestApplyInsets(contentContainer)
    }

    private fun applyWindowBackgroundMode(connected: Boolean) {
        if (connected) {
            val backgroundColor =
                ContextCompat.getColor(
                    this,
                    if (resolveThemeVariant() == "dark") {
                        R.color.nativeHostWebSurface
                    } else {
                        android.R.color.white
                    },
                )
            rootContainer.setBackgroundColor(backgroundColor)
        } else {
            rootContainer.setBackgroundResource(R.drawable.bg_native_host_window)
        }
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
                if (chromeState != ShellChromeState.CONNECTED) {
                    activeProfile?.let(::renderLoadingChrome)
                }
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
        syncSavedConnectionsState()
        renderDisconnectedChrome(getString(R.string.native_host_status_idle))
    }

    private fun maybeScheduleAutoReconnect(profile: BridgeProfile) {
        if (!preferencesStore.shouldAutoResumeActiveSession()) {
            return
        }
        if (chromeState != ShellChromeState.ERROR) {
            return
        }
        if (!shouldAutoReconnect(profile, currentStatusMessage)) {
            return
        }
        if (autoReconnectRunnable != null) {
            return
        }

        val nextAttempt =
            if (autoReconnectProfileId == profile.id) {
                autoReconnectAttemptCount + 1
            } else {
                1
            }
        autoReconnectProfileId = profile.id
        autoReconnectAttemptCount = nextAttempt
        val reconnectRunnable =
            Runnable {
                autoReconnectRunnable = null
                val activeSession = activeProfile
                if (activeSession?.id != profile.id || chromeState != ShellChromeState.ERROR) {
                    return@Runnable
                }
                openBridge(
                    profile,
                    initialStatusMessage = getString(R.string.native_host_status_reconnecting_session),
                )
            }
        autoReconnectRunnable = reconnectRunnable
        rootContainer.postDelayed(reconnectRunnable, reconnectDelayMs(nextAttempt))
    }

    private fun resetAutoReconnectState() {
        autoReconnectRunnable?.let(rootContainer::removeCallbacks)
        autoReconnectRunnable = null
        autoReconnectProfileId = null
        autoReconnectAttemptCount = 0
    }

    private fun reloadActiveBridge() {
        connectionCoordinator.reloadActiveBridge()
    }

    private fun resolveBridgeLoadTarget(profile: BridgeProfile): BridgeLoadTarget {
        return connectionCoordinator.resolveBridgeLoadTarget(profile)
    }

    private fun openBridge(
        profile: BridgeProfile,
        initialStatusMessage: String = getString(R.string.native_host_status_opening_workspace),
    ) {
        preferencesStore.setAutoResumeActiveSession(true)
        appServerCoordinator.reset()
        activeProfile = profile
        activeLoadTarget = null
        syncSavedConnectionsState()
        if (currentConnectionStage == null) {
            updateConnectionProgress(
                profileName = profile.name,
                endpoint = profile.serverEndpoint,
                stage = NativeHostConnectionStage.OPENING_WORKSPACE,
                requiresPairing = false,
            )
        }
        renderLoadingChrome(profile)
        setStatus(initialStatusMessage)
        val generation = ++bridgeLoadGeneration
        thread {
            try {
                val loadTarget = connectionCoordinator.resolveBridgeLoadTarget(profile)
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
                    val url = themedRemoteShellUrl(loadTarget.baseUrl)
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
        appServerCoordinator.reset()
        activeProfile = null
        activeLoadTarget = null
        if (profileStore.readActive() == null) {
            preferencesStore.setAutoResumeActiveSession(false)
        }
        syncSavedConnectionsState()
        renderDisconnectedChrome(message)
        setStatus(message)
    }

    private fun renderConnectionFailure(profileName: String, endpoint: String, message: String) {
        appServerCoordinator.reset()
        activeLoadTarget = null
        preferencesStore.setAutoResumeActiveSession(false)
        val activeOrPendingProfile =
            activeProfile ?: BridgeProfile(
                id = "pending-error",
                bridgeId = null,
                name = profileName.ifBlank { endpoint.ifBlank { getString(R.string.native_host_title) } },
                serverEndpoint = endpoint.ifBlank { getString(R.string.native_host_title) },
                authToken = null,
            )
        activeProfile = activeOrPendingProfile
        syncSavedConnectionsState()
        renderWorkspaceErrorChrome(activeOrPendingProfile, message)
        setStatus(message)
    }

    private fun setStatus(message: String) {
        currentStatusMessage = message
        statusView.text = message
        if (sessionBarView.visibility == View.VISIBLE) {
            sessionDetailView.text = message
        }
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
                        .put("tailnetManaged", false),
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
        connectionCoordinator.activateProfile(profile)
    }

    private fun reconnectDelayMs(attempt: Int): Long {
        return when (attempt) {
            1 -> 1200L
            2 -> 2500L
            3 -> 5000L
            4 -> 10_000L
            else -> 15_000L
        }
    }

    private fun isAutoReconnectPending(): Boolean {
        return chromeState == ShellChromeState.ERROR && autoReconnectRunnable != null
    }

    private fun shouldAutoReconnect(profile: BridgeProfile, message: String): Boolean {
        if (profile.id == "pending-error") {
            return false
        }

        val normalized = message.trim()
        if (normalized.isEmpty()) {
            return true
        }

        return !(
            normalized == getString(R.string.native_host_error_pairing_expired) ||
                normalized == getString(R.string.native_host_error_secure_link_expired) ||
                normalized.contains("expired", ignoreCase = true) ||
                normalized.contains("fresh enrollment qr", ignoreCase = true) ||
                normalized.contains("re-enrolled", ignoreCase = true) ||
                normalized.contains("no longer valid", ignoreCase = true) ||
                normalized.contains("invalid key", ignoreCase = true)
            )
    }

    private fun isBridgeConnectionIssue(error: Throwable): Boolean {
        if (
            error is UnknownHostException ||
            error is ConnectException ||
            error is NoRouteToHostException ||
            error is SocketTimeoutException ||
            error is SocketException ||
            error is InterruptedIOException
        ) {
            return true
        }

        val message = error.message?.trim().orEmpty()
        return message.contains("timed out", ignoreCase = true) ||
            message.contains("not connected", ignoreCase = true) ||
            message.contains("connection refused", ignoreCase = true) ||
            message.contains("connection reset", ignoreCase = true) ||
            message.contains("network is unreachable", ignoreCase = true) ||
            message.contains("broken pipe", ignoreCase = true) ||
            message.contains("failed to connect", ignoreCase = true) ||
            message.contains("websocket closed", ignoreCase = true)
    }

    private fun handleBridgeConnectionLost(error: Throwable?) {
        val failure = error ?: IllegalStateException(getString(R.string.native_host_status_bridge_disconnected))
        if (!isBridgeConnectionIssue(failure)) {
            return
        }

        runOnUiThread {
            val profile = activeProfile ?: return@runOnUiThread
            if (chromeState == ShellChromeState.DISCONNECTED || chromeState == ShellChromeState.ERROR) {
                return@runOnUiThread
            }

            appServerCoordinator.reset()
            activeLoadTarget = null
            renderWorkspaceErrorChrome(
                profile,
                getString(R.string.native_host_status_bridge_disconnected),
            )
        }
    }

    private fun compactLabel(value: String, maxLength: Int = 28): String {
        val trimmed = value.trim()
        return if (trimmed.length <= maxLength) {
            trimmed
        } else {
            trimmed.take(maxLength - 1).trimEnd() + "…"
        }
    }

    private fun resolveThemeVariant(): String {
        return when (preferencesStore.readThemeMode()) {
            ThemeMode.LIGHT -> "light"
            ThemeMode.DARK -> "dark"
            ThemeMode.SYSTEM -> {
                val mask = resources.configuration.uiMode and Configuration.UI_MODE_NIGHT_MASK
                if (mask == Configuration.UI_MODE_NIGHT_YES) "dark" else "light"
            }
        }
    }

    private fun themedRemoteShellUrl(baseUrl: String): String {
        val rawUrl = BridgeApi.buildRemoteShellUrlFromBaseUrl(baseUrl)
        val uri = Uri.parse(rawUrl)
        return uri.buildUpon()
            .clearQuery()
            .apply {
                uri.queryParameterNames.forEach { key ->
                    uri.getQueryParameters(key).forEach { value ->
                        appendQueryParameter(key, value)
                    }
                }
                appendQueryParameter("codexTheme", resolveThemeVariant())
            }
            .build()
            .toString()
    }

    private fun openThemeSettings() {
        val modes = arrayOf(ThemeMode.SYSTEM, ThemeMode.LIGHT, ThemeMode.DARK)
        val labels =
            arrayOf(
                getString(R.string.native_host_settings_theme_system),
                getString(R.string.native_host_settings_theme_light),
                getString(R.string.native_host_settings_theme_dark),
            )
        val currentMode = preferencesStore.readThemeMode()
        val checkedIndex = modes.indexOf(currentMode).coerceAtLeast(0)
        MaterialAlertDialogBuilder(this)
            .setTitle(R.string.native_host_settings_theme_title)
            .setSingleChoiceItems(labels, checkedIndex) { dialog, which ->
                val nextMode = modes.getOrElse(which) { ThemeMode.SYSTEM }
                if (nextMode != currentMode) {
                    preferencesStore.writeThemeMode(nextMode)
                    preferencesStore.applyThemeMode()
                    recreate()
                }
                dialog.dismiss()
            }
            .setNegativeButton("Cancel", null)
            .show()
    }

    private fun summarizeWorkspaceError(message: String): String {
        val normalized = message.trim()
        return when {
            normalized.contains("expired", ignoreCase = true) ->
                "This setup code expired before the workspace opened."
            normalized.isBlank() ->
                getString(R.string.native_host_workspace_error_body)
            else -> compactLabel(normalized, maxLength = 90)
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

    private fun showSessionBar(
        profile: BridgeProfile,
        stateLabelResId: Int,
        tone: ChipTone,
        detail: String = currentStatusMessage.ifBlank { connectionSheetController.displayProfileDetail(profile) },
        showAction: Boolean = true,
    ) {
        sessionBarView.visibility = View.VISIBLE
        sessionNameView.text = connectionSheetController.displayProfileLabel(profile)
        sessionDetailView.text = detail.ifBlank { connectionSheetController.displayProfileDetail(profile) }
        sessionStateChipView.text = getString(stateLabelResId)
        applyChipTone(sessionStateChipView, tone)
        sessionActionButton.visibility = if (showAction) View.VISIBLE else View.GONE
    }

    private fun hideSessionBar() {
        sessionBarView.visibility = View.GONE
        sessionActionButton.visibility = View.VISIBLE
    }

    private fun updateWorkspaceMessage(
        eyebrowResId: Int,
        titleResId: Int,
        bodyResId: Int,
        showIllustration: Boolean = true,
    ) {
        workspaceMessageGroupView.visibility = View.VISIBLE
        workspaceMessageEyebrowView.visibility = View.VISIBLE
        workspaceMessageEyebrowView.text = getString(eyebrowResId)
        workspaceEmptyTitleView.text = getString(titleResId)
        workspaceEmptyBodyView.text = getString(bodyResId)
        workspaceIllustrationView.visibility = if (showIllustration) View.VISIBLE else View.GONE
    }

    private fun configureWelcomeState(enabled: Boolean) {
        emptyStateContentView.gravity =
            if (enabled) {
                Gravity.CENTER_HORIZONTAL or Gravity.CENTER_VERTICAL
            } else {
                Gravity.NO_GRAVITY
            }
        welcomeGuideCardView.visibility = if (enabled) View.VISIBLE else View.GONE
        welcomeGuideTitleView.text = getString(R.string.native_host_welcome_guide_title)
        welcomeGuideBodyView.text = getString(R.string.native_host_welcome_guide_body)
        welcomeSceneCardView.visibility = if (enabled) View.VISIBLE else View.GONE
        welcomePrimaryActionButton.visibility = if (enabled) View.VISIBLE else View.GONE

        if (enabled) {
            welcomePrimaryActionButton.text = getString(R.string.native_host_empty_action_start)
        } else {
            connectionButton.text = getString(R.string.native_host_empty_action)
            connectionButton.setTextColor(ContextCompat.getColor(this, R.color.nativeHostTextPrimary))
            connectionButton.backgroundTintList =
                ColorStateList.valueOf(ContextCompat.getColor(this, R.color.nativeHostSurface))
            connectionButton.strokeWidth = (resources.displayMetrics.density).toInt()
            connectionButton.strokeColor =
                ColorStateList.valueOf(ContextCompat.getColor(this, R.color.nativeHostDividerStrong))
            connectionButton.iconTint = ColorStateList.valueOf(ContextCompat.getColor(this, R.color.nativeHostAccent))
        }
    }

    private fun renderRecentSessionsSection(
        activeProfileId: String?,
        bodyResId: Int,
        showBody: Boolean = true,
        showTitle: Boolean = true,
        transientProfile: BridgeProfile? = null,
    ): Boolean {
        val savedProfiles = profileStore.list()
        val profiles =
            buildList {
                transientProfile?.let { pendingProfile ->
                    if (savedProfiles.none { it.id == pendingProfile.id }) {
                        add(pendingProfile)
                    }
                }
                addAll(savedProfiles)
            }
        recentSessionsHeaderView.visibility = if (showTitle && profiles.isNotEmpty()) View.VISIBLE else View.GONE
        recentSessionsBodyView.visibility = if (showBody && profiles.isNotEmpty()) View.VISIBLE else View.GONE
        recentSessionsBodyView.text = getString(bodyResId)
        (recentSessionsListView.layoutParams as? ViewGroup.MarginLayoutParams)?.let { params ->
            params.topMargin =
                when {
                    showBody && profiles.isNotEmpty() -> (14 * resources.displayMetrics.density).toInt()
                    showTitle && profiles.isNotEmpty() -> (10 * resources.displayMetrics.density).toInt()
                    else -> 0
                }
            recentSessionsListView.layoutParams = params
        }
        recentSessionsListView.removeAllViews()

        if (profiles.isEmpty()) {
            recentSessionsSectionView.visibility = View.GONE
            return false
        }

        recentSessionsSectionView.visibility = View.VISIBLE
        connectionSheetController.renderConnectionCards(
            container = recentSessionsListView,
            profiles = profiles.take(HOME_SESSION_LIMIT),
            activeProfileId = activeProfileId,
        ) { profile ->
            activateProfile(profile)
        }
        return true
    }

    private fun renderRecentSessions(
        activeProfileId: String?,
        bodyResId: Int,
        showMessageWhenEmpty: Boolean,
        showBody: Boolean = true,
        showTitle: Boolean = true,
    ) {
        val hasProfiles = renderRecentSessionsSection(activeProfileId, bodyResId, showBody, showTitle)
        workspaceMessageGroupView.visibility =
            when {
                hasProfiles -> View.GONE
                showMessageWhenEmpty -> View.VISIBLE
                else -> View.GONE
            }
    }

    private fun showDisconnectedSupportingViews() {
        stateIconView.visibility = View.VISIBLE
        updateRuntimeCard(R.string.native_host_runtime_label_saved, connectionSheetController.savedConnectionsSummary())
        hintCardView.visibility = View.VISIBLE
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

    private fun setWorkspaceFrameMode(fullScreen: Boolean) {
        (workspaceFrameView.layoutParams as? ViewGroup.MarginLayoutParams)?.let { params ->
            if (fullScreen) {
                params.marginStart = 0
                params.topMargin = 0
                params.marginEnd = 0
                params.bottomMargin = 0
            } else {
                params.marginStart = workspaceFrameMarginStart
                params.topMargin = workspaceFrameMarginTop
                params.marginEnd = workspaceFrameMarginEnd
                params.bottomMargin = workspaceFrameMarginBottom
            }
            workspaceFrameView.layoutParams = params
        }
        if (fullScreen) {
            workspaceFrameView.background = null
            workspaceFrameView.elevation = 0f
            workspaceFrameView.clipToOutline = false
            workspaceFrameView.outlineProvider = null
        } else {
            workspaceFrameView.setBackgroundResource(R.drawable.bg_native_host_surface)
            workspaceFrameView.elevation = 6f * resources.displayMetrics.density
            workspaceFrameView.clipToOutline = true
            workspaceFrameView.outlineProvider = ViewOutlineProvider.BACKGROUND
        }
    }

    private fun updateConnectionProgress(
        profileName: String,
        endpoint: String,
        stage: NativeHostConnectionStage,
        requiresPairing: Boolean,
    ) {
        currentConnectionStage = stage
        currentConnectionRequiresPairing = requiresPairing
        renderLoadingChrome(
            BridgeProfile(
                id = "pending",
                bridgeId = null,
                name = profileName.ifBlank { endpoint.ifBlank { getString(R.string.native_host_title) } },
                serverEndpoint = endpoint,
                authToken = null,
            ),
        )
    }

    private fun buildConnectionProgressSummary(
        stage: NativeHostConnectionStage,
        requiresPairing: Boolean,
    ): String {
        val steps =
            buildList {
                add(NativeHostConnectionStage.PAYLOAD_RECEIVED to getString(R.string.native_host_progress_step_payload))
                if (requiresPairing) {
                    add(NativeHostConnectionStage.PAIRING_DEVICE to getString(R.string.native_host_progress_step_pairing))
                }
                add(NativeHostConnectionStage.OPENING_WORKSPACE to getString(R.string.native_host_progress_step_workspace))
            }
        val activeIndex = steps.indexOfFirst { it.first == stage }.coerceAtLeast(0)
        val timeline =
            steps.mapIndexed { index, (_, label) ->
                when {
                    index < activeIndex -> "Done  $label"
                    index == activeIndex -> "Now   $label"
                    else -> "Next  $label"
                }
            }.joinToString("\n")
        return timeline
    }

    private fun renderDisconnectedChrome(message: String) {
        resetAutoReconnectState()
        clearConnectionProgress()
        chromeState = ShellChromeState.DISCONNECTED
        workspaceFrameView.visibility = View.VISIBLE
        setWorkspaceFrameMode(fullScreen = false)
        applyWindowBackgroundMode(connected = false)
        toolbar.visibility = View.GONE
        settingsButton.visibility = View.VISIBLE
        hideSessionBar()
        activeLoadTarget = null
        val hasSavedProfiles = profileStore.list().isNotEmpty()
        heroCardView.visibility = View.GONE
        progressIndicator.visibility = View.GONE
        emptyStateContainer.visibility = View.VISIBLE
        stateIconView.visibility = if (hasSavedProfiles) View.GONE else View.VISIBLE
        configureWelcomeState(enabled = !hasSavedProfiles)
        if (hasSavedProfiles) {
            updateWorkspaceMessage(
                R.string.native_host_workspace_preview_eyebrow,
                R.string.native_host_workspace_preview_title,
                R.string.native_host_workspace_preview_body,
                showIllustration = false,
            )
        } else {
            updateWorkspaceMessage(
                R.string.native_host_workspace_welcome_eyebrow,
                R.string.native_host_workspace_welcome_title,
                R.string.native_host_workspace_welcome_body,
                showIllustration = false,
            )
            workspaceMessageEyebrowView.visibility = View.GONE
        }
        renderRecentSessions(
            activeProfileId = profileStore.readActive()?.id,
            bodyResId =
                if (hasSavedProfiles) {
                    R.string.native_host_recent_sessions_saved_body
                } else {
                    R.string.native_host_recent_sessions_body
                },
            showMessageWhenEmpty = true,
            showBody = !hasSavedProfiles,
        )
        runtimeDetailsView.visibility = View.GONE
        hintCardView.visibility = View.GONE
        primaryActionButton.visibility = View.GONE
        secondaryActionButton.visibility = View.GONE
        webView.loadUrl("about:blank")
        webView.visibility = View.GONE
        connectionButton.visibility = View.VISIBLE
        statusView.text = message
    }

    private fun renderLoadingChrome(profile: BridgeProfile) {
        chromeState = ShellChromeState.LOADING
        workspaceFrameView.visibility = View.VISIBLE
        setWorkspaceFrameMode(fullScreen = false)
        applyWindowBackgroundMode(connected = false)
        toolbar.visibility = View.GONE
        settingsButton.visibility = View.VISIBLE
        hideSessionBar()
        heroCardView.visibility = View.GONE
        progressIndicator.visibility = View.GONE
        emptyStateContainer.visibility = View.VISIBLE
        configureWelcomeState(enabled = false)
        updateWorkspaceMessage(
            R.string.native_host_loading_eyebrow,
            R.string.native_host_loading_title,
            R.string.native_host_loading_body,
            showIllustration = false,
        )
        welcomeSceneCardView.visibility = View.GONE
        welcomePrimaryActionButton.visibility = View.GONE
        welcomeGuideCardView.visibility = View.VISIBLE
        welcomeGuideTitleView.text = getString(R.string.native_host_loading_progress_title)
        welcomeGuideBodyView.text =
            buildConnectionProgressSummary(
                currentConnectionStage ?: NativeHostConnectionStage.OPENING_WORKSPACE,
                currentConnectionRequiresPairing,
            )
        runtimeDetailsView.visibility = View.GONE
        hintCardView.visibility = View.GONE
        renderRecentSessionsSection(
            activeProfileId = profile.id,
            bodyResId = R.string.native_host_recent_sessions_loading_body,
            showBody = false,
            showTitle = false,
            transientProfile = profile,
        )
        primaryActionButton.visibility = View.GONE
        secondaryActionButton.visibility = View.GONE
        webView.visibility = View.GONE
        connectionButton.visibility = View.GONE
    }

    private fun renderConnectedChrome(profile: BridgeProfile) {
        resetAutoReconnectState()
        clearConnectionProgress()
        chromeState = ShellChromeState.CONNECTED
        workspaceFrameView.visibility = View.VISIBLE
        setWorkspaceFrameMode(fullScreen = true)
        applyWindowBackgroundMode(connected = true)
        toolbar.visibility = View.GONE
        settingsButton.visibility = View.GONE
        hideSessionBar()
        heroCardView.visibility = View.GONE
        updateStateChip(R.string.native_host_state_live, ChipTone.SUCCESS)
        progressIndicator.visibility = View.GONE
        emptyStateContainer.visibility = View.GONE
        runtimeDetailsView.visibility = View.GONE
        hintCardView.visibility = View.GONE
        primaryActionButton.visibility = View.GONE
        secondaryActionButton.visibility = View.GONE
        webView.visibility = View.VISIBLE
        connectionButton.visibility = View.GONE
        toolbar.title = connectionSheetController.displayProfileLabel(profile)
    }

    private fun renderWorkspaceErrorChrome(profile: BridgeProfile, message: String) {
        clearConnectionProgress()
        chromeState = ShellChromeState.ERROR
        currentStatusMessage = message
        workspaceFrameView.visibility = View.GONE
        setWorkspaceFrameMode(fullScreen = false)
        applyWindowBackgroundMode(connected = false)
        activeLoadTarget = null
        toolbar.visibility = View.VISIBLE
        settingsButton.visibility = View.VISIBLE
        toolbar.title = connectionSheetController.displayProfileLabel(profile)
        toolbar.subtitle = null
        hideSessionBar()
        heroCardView.visibility = View.VISIBLE
        stateIconView.visibility = View.GONE
        val reconnecting = shouldAutoReconnect(profile, message)
        heroEyebrowView.text =
            getString(
                if (reconnecting) {
                    R.string.native_host_hero_eyebrow_reconnecting
                } else {
                    R.string.native_host_hero_eyebrow_error
                },
            )
        updateStateChip(
            if (reconnecting) {
                R.string.native_host_state_reconnecting
            } else {
                R.string.native_host_state_attention
            },
            if (reconnecting) ChipTone.ACTIVE else ChipTone.WARNING,
        )
        progressIndicator.visibility = View.GONE
        emptyStateContainer.visibility = View.GONE
        configureWelcomeState(enabled = false)
        emptyTitleView.text =
            getString(
                if (reconnecting) {
                    R.string.native_host_workspace_reconnecting
                } else {
                    R.string.native_host_workspace_error
                },
            )
        emptyBodyView.text =
            getString(
                if (reconnecting) {
                    R.string.native_host_workspace_reconnecting_body
                } else {
                    R.string.native_host_workspace_error_body
                },
            )
        statusView.text = summarizeWorkspaceError(message)
        runtimeDetailsView.visibility = View.GONE
        hintCardView.visibility = View.GONE
        primaryActionButton.visibility = View.VISIBLE
        primaryActionButton.text =
            getString(
                if (reconnecting) {
                    R.string.native_host_retry_now_action
                } else {
                    R.string.native_host_retry_action
                },
            )
        secondaryActionButton.visibility = View.VISIBLE
        secondaryActionButton.text = getString(R.string.native_host_secondary_action_manage)
        webView.visibility = View.GONE
        connectionButton.visibility = View.GONE
        maybeScheduleAutoReconnect(profile)
    }

    private fun handlePrimaryAction() {
        when (chromeState) {
            ShellChromeState.DISCONNECTED -> openScanner()
            ShellChromeState.ERROR ->
                if (activeProfile?.id == "pending-error") {
                    openScanner()
                } else {
                    reloadActiveBridge()
                }
            ShellChromeState.LOADING -> Unit
            ShellChromeState.CONNECTED -> openConnectionSheet()
        }
    }

    private fun handleSecondaryAction() {
        when (chromeState) {
            ShellChromeState.DISCONNECTED -> openConnectionSheet()
            ShellChromeState.LOADING,
            ShellChromeState.CONNECTED,
            ShellChromeState.ERROR,
            -> openConnectionSheet()
        }
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
        emitWorkspaceSelectionChange(
            sessionState.updateWorkspaceRoots(nextRoots, preferredRoot),
        )
    }

    private fun mergeWorkspaceRoots(nextRoots: List<String>, preferredRoot: String? = null) {
        emitWorkspaceSelectionChange(
            sessionState.mergeWorkspaceRoots(nextRoots, preferredRoot),
        )
    }

    private fun setActiveWorkspaceRoot(root: String) {
        emitWorkspaceSelectionChange(
            sessionState.setActiveWorkspaceRoot(root),
        )
    }

    private fun emitWorkspaceSelectionChange(change: WorkspaceSelectionChange?) {
        if (change == null) {
            return
        }
        if (change.optionsChanged) {
            sendHostMessage(JSONObject().put("type", "workspace-root-options-updated"))
        }
        if (change.activeChanged) {
            sendHostMessage(JSONObject().put("type", "active-workspace-roots-updated"))
        }
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

        sessionState.replaceWorkspaceRootLabels(state.workspaceRootLabels)
        sessionState.replacePinnedThreadIds(state.pinnedThreadIds)
        updateWorkspaceRoots(
            nextRoots = state.workspaceRootOptions,
            preferredRoot = state.activeWorkspaceRoots.firstOrNull(),
        )
    }

    private fun mergeWorkspaceRootsFromResult(result: JSONObject) {
        val candidates = mutableListOf<String>()

        fun collectThreadRoots(value: JSONObject?) {
            val cwd = NativeHostSessionState.normalizeWorkspaceRootCandidate(value?.optString("cwd"))
            if (!cwd.isNullOrBlank()) {
                candidates.add(cwd)
            }
        }

        NativeHostSessionState.normalizeWorkspaceRootCandidate(result.optString("cwd"))?.let(candidates::add)
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
            "list-pinned-threads" -> JSONObject().put("threadIds", JSONArray(sessionState.pinnedThreadIds))
            "set-thread-pinned" -> {
                val threadId = params?.optString("threadId")?.trim().orEmpty()
                if (threadId.isEmpty()) {
                    JSONObject()
                        .put("success", false)
                        .put("threadIds", JSONArray(sessionState.pinnedThreadIds))
                } else {
                    sessionState.setThreadPinned(
                        threadId = threadId,
                        pinned = params?.optBoolean("pinned", false) == true,
                    )
                    JSONObject()
                        .put("success", true)
                        .put("threadIds", JSONArray(sessionState.pinnedThreadIds))
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
                        sessionState.pinnedThreadIds.toMutableList()
                    }
                sessionState.replacePinnedThreadIds(nextThreadIds)
                JSONObject()
                    .put("success", true)
                    .put("threadIds", JSONArray(sessionState.pinnedThreadIds))
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
            "active-workspace-roots" -> JSONObject().put("roots", JSONArray(sessionState.activeWorkspaceRoots))
            "workspace-root-options" -> JSONObject()
                .put("roots", JSONArray(sessionState.workspaceRootOptions))
                .put("activeRoots", JSONArray(sessionState.activeWorkspaceRoots))
                .put("labels", JSONObject(sessionState.workspaceRootLabels.toMap()))
            "paths-exist" -> {
                val existingPaths = JSONArray()
                val paths = params?.optJSONArray("paths")
                if (paths != null) {
                    for (index in 0 until paths.length()) {
                        val path = NativeHostSessionState.normalizeWorkspaceRootCandidate(paths.optString(index))
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

    private fun integrateDirectRpcResult(method: String, result: JSONObject) {
        when (method) {
            "thread/list", "thread/read", "thread/start", "thread/resume" -> mergeWorkspaceRootsFromResult(result)
            "workspace-root-options" -> {
                sessionState.replaceWorkspaceRootLabels(jsonObjectStringMap(result.optJSONObject("labels")))
                updateWorkspaceRoots(
                    nextRoots = jsonArrayStrings(result.optJSONArray("roots")),
                    preferredRoot = jsonArrayStrings(result.optJSONArray("activeRoots")).firstOrNull(),
                )
            }
        }
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

    private fun normalizeErrorMessage(error: Throwable): String {
        val message = error.message?.trim().orEmpty()
        if (message.equals("Invalid or expired pairing code", ignoreCase = true)) {
            return getString(R.string.native_host_error_pairing_expired)
        }
        if (
            message.contains("invalid key", ignoreCase = true) ||
            message.contains("not valid", ignoreCase = true) ||
            message.contains("fresh enrollment qr", ignoreCase = true) ||
            message.contains("re-enrolled", ignoreCase = true)
        ) {
            return getString(R.string.native_host_error_secure_link_expired)
        }
        return if (message.isNotEmpty()) {
            message
        } else {
            error::class.java.simpleName
        }
    }

    private inner class NativeHostJavascriptBridge {
        @JavascriptInterface
        fun postMessage(message: String) {
            android.util.Log.d(
                "CodexMobile",
                "native bridge raw ${message.take(180)}",
            )
            runOnUiThread {
                webViewMessageRouter.handleEnvelope(message)
            }
        }
    }

    private fun openConnectionSheet() {
        connectionSheetController.openConnectionSheet()
    }

    private fun openScanner() {
        val options = ScanOptions()
            .setDesiredBarcodeFormats(ScanOptions.QR_CODE)
            .setPrompt("Scan Codex Mobile enrollment QR")
            .setBeepEnabled(false)
            .setOrientationLocked(true)
            .setCaptureActivity(PortraitCaptureActivity::class.java)
        scanLauncher.launch(options)
    }

    private fun importEnrollment(rawJson: String) {
        connectionSheetController.dismissOpenSheet()
        connectionCoordinator.importEnrollment(rawJson)
    }
}
