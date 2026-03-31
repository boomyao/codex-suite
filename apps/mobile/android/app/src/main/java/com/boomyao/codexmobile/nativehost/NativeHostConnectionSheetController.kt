package com.boomyao.codexmobile.nativehost

import android.content.Context
import android.net.Uri
import android.text.format.DateUtils
import android.view.LayoutInflater
import android.view.View
import android.widget.FrameLayout
import android.widget.ImageView
import android.widget.LinearLayout
import android.widget.TextView
import androidx.core.content.ContextCompat
import com.boomyao.codexmobile.R
import com.google.android.material.bottomsheet.BottomSheetDialog
import com.google.android.material.button.MaterialButton
import com.google.android.material.card.MaterialCardView
import com.google.android.material.dialog.MaterialAlertDialogBuilder
import com.google.android.material.progressindicator.LinearProgressIndicator

class NativeHostConnectionSheetController(
    private val context: Context,
    private val profileStore: BridgeProfileStore,
    private val activeProfileProvider: () -> BridgeProfile?,
    private val isConnectedProvider: () -> Boolean,
    private val isLoadingProvider: () -> Boolean,
    private val isErrorProvider: () -> Boolean,
    private val currentStatusMessageProvider: () -> String,
    private val currentConnectionStageProvider: () -> NativeHostConnectionStage?,
    private val activateProfile: (BridgeProfile) -> Unit,
    private val reloadActiveBridge: () -> Unit,
    private val openScanner: () -> Unit,
    private val resetEnrollment: (BridgeProfile) -> Unit,
) {
    private var activeDialog: BottomSheetDialog? = null

    private enum class ChipTone {
        ACTIVE,
        SUCCESS,
        WARNING,
    }

    fun openConnectionSheet() {
        activeDialog?.dismiss()
        val dialog = BottomSheetDialog(context)
        activeDialog = dialog
        val sheetRoot = FrameLayout(context)
        val content =
            LayoutInflater.from(context).inflate(
                R.layout.sheet_native_host_actions,
                sheetRoot,
                false,
            )
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

        val profile = activeProfileProvider() ?: profileStore.readActive()
        val savedProfiles = profileStore.list()
        if (profile == null) {
            titleView.text = context.getString(R.string.native_host_sheet_connect_title)
            bodyView.text = context.getString(R.string.native_host_sheet_connect_body)
            metadataView.visibility = View.GONE
            renderConnectionCards(
                container = savedConnectionsList,
                profiles = savedProfiles,
                activeProfileId = null,
            ) { nextProfile ->
                dismissOpenSheet()
                activateProfile(nextProfile)
            }
            savedConnectionsSection.visibility = if (savedProfiles.isEmpty()) View.GONE else View.VISIBLE
            primaryButton.text = context.getString(R.string.native_host_sheet_action_scan)
            primaryButton.setOnClickListener {
                dismissOpenSheet()
                openScanner()
            }
            secondaryButton.visibility = View.GONE
            tertiaryButton.visibility = View.GONE
        } else {
            titleView.text = context.getString(R.string.native_host_sheet_connected_title)
            bodyView.text = context.getString(R.string.native_host_sheet_connected_body)
            metadataView.visibility = View.VISIBLE
            metadataTitleView.text = context.getString(R.string.native_host_sheet_current_bridge)
            metadataValueView.text = connectionMetadataText(profile)
            renderConnectionCards(
                container = savedConnectionsList,
                profiles = savedProfiles,
                activeProfileId = profile.id,
            ) { nextProfile ->
                dismissOpenSheet()
                activateProfile(nextProfile)
            }
            savedConnectionsSection.visibility = if (savedProfiles.isEmpty()) View.GONE else View.VISIBLE
            primaryButton.text = context.getString(R.string.native_host_sheet_action_reload)
            primaryButton.setOnClickListener {
                dismissOpenSheet()
                reloadActiveBridge()
            }
            secondaryButton.visibility = View.VISIBLE
            secondaryButton.text = context.getString(R.string.native_host_sheet_action_rescan)
            secondaryButton.setOnClickListener {
                dismissOpenSheet()
                openScanner()
            }
            tertiaryButton.visibility = View.VISIBLE
            tertiaryButton.text = context.getString(R.string.native_host_sheet_action_reset)
            tertiaryButton.setOnClickListener {
                dismissOpenSheet()
                confirmReset(profile)
            }
        }

        dialog.setContentView(content)
        dialog.setOnDismissListener {
            if (activeDialog === dialog) {
                activeDialog = null
            }
        }
        dialog.show()
    }

    fun dismissOpenSheet() {
        activeDialog?.dismiss()
        activeDialog = null
    }

    fun renderConnectionCards(
        container: LinearLayout,
        profiles: List<BridgeProfile>,
        activeProfileId: String?,
        onActivate: (BridgeProfile) -> Unit,
    ) {
        container.removeAllViews()
        profiles.forEachIndexed { index, profile ->
            val card = LayoutInflater.from(context).inflate(R.layout.item_native_host_connection, container, false)
            val layoutParams =
                (card.layoutParams as? LinearLayout.LayoutParams)
                    ?: LinearLayout.LayoutParams(
                        LinearLayout.LayoutParams.MATCH_PARENT,
                        LinearLayout.LayoutParams.WRAP_CONTENT,
                    )
            layoutParams.topMargin =
                if (index == 0) 0 else (10 * context.resources.displayMetrics.density).toInt()
            card.layoutParams = layoutParams

            val cardView = card as MaterialCardView
            val avatarView = card.findViewById<TextView>(R.id.nativeHostConnectionAvatar)
            val nameView = card.findViewById<TextView>(R.id.nativeHostConnectionName)
            val endpointView = card.findViewById<TextView>(R.id.nativeHostConnectionEndpoint)
            val badgeView = card.findViewById<TextView>(R.id.nativeHostConnectionBadge)
            val modeView = card.findViewById<TextView>(R.id.nativeHostConnectionMode)
            val chevronView = card.findViewById<ImageView>(R.id.nativeHostConnectionChevron)
            val progressView = card.findViewById<LinearProgressIndicator>(R.id.nativeHostConnectionProgress)

            avatarView.text = profileAvatar(profile)
            nameView.text = displayProfileLabel(profile)
            endpointView.text = displayProfileDetail(profile)
            val isCurrent = profile.id == activeProfileId
            val isConnected = isCurrent && isConnectedProvider()
            val isConnecting = isCurrent && isLoadingProvider()
            val isErrored = isCurrent && isErrorProvider()
            val badgeTextResId =
                when {
                    isConnecting -> R.string.native_host_sheet_connection_connecting_badge
                    isErrored -> R.string.native_host_sheet_connection_error_badge
                    isConnected -> R.string.native_host_sheet_connection_current_badge
                    else -> null
                }
            val badgeTone =
                when {
                    isConnecting -> ChipTone.ACTIVE
                    isErrored -> ChipTone.WARNING
                    isConnected -> ChipTone.SUCCESS
                    else -> null
                }
            if (badgeTextResId != null && badgeTone != null) {
                badgeView.visibility = View.VISIBLE
                badgeView.text = context.getString(badgeTextResId)
                applyChipTone(badgeView, badgeTone)
            } else {
                badgeView.visibility = View.GONE
            }
            modeView.text =
                when {
                    isConnecting -> describeConnectingStatus(profile)
                    isErrored -> summarizeWorkspaceError(currentStatusMessageProvider())
                    else -> describeProfileStatus(profile, isConnected)
                }
            modeView.setCompoundDrawablesRelativeWithIntrinsicBounds(
                if (isConnecting || isErrored) 0 else R.drawable.ic_native_host_clock,
                0,
                0,
                0,
            )
            progressView.visibility = if (isConnecting) View.VISIBLE else View.GONE
            chevronView.visibility = if (isConnecting) View.GONE else View.VISIBLE
            cardView.strokeWidth =
                when {
                    isConnecting -> (2 * context.resources.displayMetrics.density).toInt()
                    isErrored -> (2 * context.resources.displayMetrics.density).toInt()
                    else -> (1 * context.resources.displayMetrics.density).toInt()
                }
            cardView.strokeColor =
                ContextCompat.getColor(
                    context,
                    when {
                        isConnecting -> R.color.nativeHostAccent
                        isErrored -> R.color.nativeHostWarning
                        else -> R.color.nativeHostDivider
                    },
                )

            val disableAction = isConnected || isConnecting
            cardView.isEnabled = !disableAction
            card.isClickable = !disableAction
            card.isFocusable = !disableAction
            card.setOnClickListener(
                if (disableAction) {
                    null
                } else {
                    View.OnClickListener { onActivate(profile) }
                },
            )

            container.addView(card)
        }
    }

    fun savedConnectionsSummary(): String {
        val count = profileStore.list().size
        return if (count == 0) {
            context.getString(R.string.native_host_saved_none)
        } else {
            context.resources.getQuantityString(R.plurals.native_host_saved_count, count, count)
        }
    }

    fun profileAvatar(profile: BridgeProfile): String {
        val tokens =
            displayProfileLabel(profile)
                .split('-', '_', '.', ' ')
                .map(String::trim)
                .filter(String::isNotEmpty)
        val preferredToken = tokens.lastOrNull() ?: displayProfileLabel(profile)
        return preferredToken.firstOrNull()?.uppercaseChar()?.toString() ?: "C"
    }

    fun displayProfileLabel(profile: BridgeProfile): String {
        val normalizedEndpoint = BridgeApi.normalizeEndpoint(profile.serverEndpoint)
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
        val host = runCatching { Uri.parse(normalizedEndpoint).host.orEmpty() }.getOrDefault("").trim()
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
        val normalizedEndpoint = BridgeApi.normalizeEndpoint(profile.serverEndpoint)
        val uri = runCatching { Uri.parse(normalizedEndpoint) }.getOrNull()
        val host = uri?.host.orEmpty().trim()
        val portSuffix = if ((uri?.port ?: -1) > 0) ":${uri?.port}" else ""
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

    fun describeProfileStatus(profile: BridgeProfile, isCurrent: Boolean): String {
        val lastUsedAt = profile.lastUsedAtMillis
        if (lastUsedAt != null) {
            val relativeTime =
                DateUtils.getRelativeTimeSpanString(
                    lastUsedAt,
                    System.currentTimeMillis(),
                    DateUtils.MINUTE_IN_MILLIS,
                    DateUtils.FORMAT_ABBREV_RELATIVE,
                ).toString()
            return context.getString(R.string.native_host_sheet_connection_last_used, relativeTime)
        }
        return context.getString(
            if (isCurrent) {
                R.string.native_host_sheet_connection_ready_now
            } else {
                R.string.native_host_sheet_connection_saved_status
            },
        )
    }

    fun describeConnectingStatus(profile: BridgeProfile): String {
        return currentStatusMessageProvider().ifBlank {
            when (currentConnectionStageProvider()) {
                NativeHostConnectionStage.PAYLOAD_RECEIVED -> context.getString(R.string.native_host_status_payload_received)
                NativeHostConnectionStage.STARTING_TAILNET -> context.getString(R.string.native_host_status_preparing_connection)
                NativeHostConnectionStage.PAIRING_DEVICE -> context.getString(R.string.native_host_status_pairing_device)
                NativeHostConnectionStage.OPENING_WORKSPACE, null -> context.getString(R.string.native_host_status_opening_workspace)
            }
        }.ifBlank { displayProfileDetail(profile) }
    }

    private fun connectionMetadataText(profile: BridgeProfile): String {
        return buildString {
            append(displayProfileLabel(profile))
            append('\n')
            append(profile.serverEndpoint)
            append("\n\n")
            append(context.getString(R.string.native_host_sheet_status))
            append(": ")
            append(currentStatusMessageProvider())
        }
    }

    private fun confirmReset(profile: BridgeProfile) {
        MaterialAlertDialogBuilder(context)
            .setTitle(R.string.native_host_reset_title)
            .setMessage(context.getString(R.string.native_host_reset_body, profile.name))
            .setNegativeButton("Cancel", null)
            .setPositiveButton("Reset") { _, _ ->
                resetEnrollment(profile)
            }
            .show()
    }

    private fun compactLabel(value: String, maxLength: Int = 28): String {
        val trimmed = value.trim()
        return if (trimmed.length <= maxLength) {
            trimmed
        } else {
            trimmed.take(maxLength - 1).trimEnd() + "…"
        }
    }

    private fun summarizeWorkspaceError(message: String): String {
        val normalized = message.trim()
        return when {
            normalized.contains("tailnet runtime is not running", ignoreCase = true) ->
                "The secure link on this phone has stopped."
            normalized.contains("invalid key", ignoreCase = true) ||
                normalized.contains("not valid", ignoreCase = true) ||
                normalized.contains("fresh enrollment qr", ignoreCase = true) ||
                normalized.contains("re-enrolled", ignoreCase = true) ->
                context.getString(R.string.native_host_error_secure_link_expired)
            normalized.contains("expired", ignoreCase = true) ->
                "This setup code expired before the workspace opened."
            normalized.isBlank() ->
                context.getString(R.string.native_host_workspace_error_body)
            else -> compactLabel(normalized, maxLength = 90)
        }
    }

    private fun applyChipTone(view: TextView, tone: ChipTone) {
        val (backgroundColorRes, textColorRes) =
            when (tone) {
                ChipTone.ACTIVE -> R.color.nativeHostChipActiveBg to R.color.nativeHostChipActiveText
                ChipTone.SUCCESS -> R.color.nativeHostChipSuccessBg to R.color.nativeHostChipSuccessText
                ChipTone.WARNING -> R.color.nativeHostChipWarningBg to R.color.nativeHostChipWarningText
            }
        (view.background?.mutate() as? android.graphics.drawable.GradientDrawable)?.setColor(
            ContextCompat.getColor(context, backgroundColorRes),
        )
        view.setTextColor(ContextCompat.getColor(context, textColorRes))
    }
}
