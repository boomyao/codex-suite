package com.boomyao.codexmobile.nativehost

import android.content.Context
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
import com.google.android.material.progressindicator.CircularProgressIndicator

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
            val detailView = card.findViewById<TextView>(R.id.nativeHostConnectionDetail)
            val badgeView = card.findViewById<TextView>(R.id.nativeHostConnectionBadge)
            val chevronView = card.findViewById<ImageView>(R.id.nativeHostConnectionChevron)
            val progressView = card.findViewById<CircularProgressIndicator>(R.id.nativeHostConnectionProgress)

            avatarView.text = NativeHostSessionUi.profileAvatar(profile)
            nameView.text = NativeHostSessionUi.displayProfileLabel(profile)
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
            detailView.text =
                when {
                    isConnecting ->
                        NativeHostSessionUi.describeConnectingStatus(
                            context = context,
                            profile = profile,
                            statusMessage = currentStatusMessageProvider(),
                            stage = currentConnectionStageProvider(),
                        )
                    isErrored ->
                        NativeHostSessionUi.summarizeWorkspaceError(
                            context = context,
                            message = currentStatusMessageProvider(),
                        )
                    else -> NativeHostSessionUi.describeProfileStatus(context, profile, isCurrent)
                }
            progressView.visibility = if (isConnecting) View.VISIBLE else View.GONE
            chevronView.visibility = if (isConnecting || isConnected) View.GONE else View.VISIBLE
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

    fun displayProfileLabel(profile: BridgeProfile): String {
        return NativeHostSessionUi.displayProfileLabel(profile)
    }

    fun displayProfileDetail(profile: BridgeProfile): String {
        return NativeHostSessionUi.displayProfileDetail(profile)
    }

    fun describeProfileStatus(profile: BridgeProfile, isCurrent: Boolean): String {
        return NativeHostSessionUi.describeProfileStatus(context, profile, isCurrent)
    }

    fun describeConnectingStatus(profile: BridgeProfile): String {
        return NativeHostSessionUi.describeConnectingStatus(
            context = context,
            profile = profile,
            statusMessage = currentStatusMessageProvider(),
            stage = currentConnectionStageProvider(),
        )
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

    private fun summarizeWorkspaceError(message: String): String {
        return NativeHostSessionUi.summarizeWorkspaceError(context, message)
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
