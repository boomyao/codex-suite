package com.boomyao.codexmobile.nativehost

import android.content.Context
import android.text.format.DateUtils
import com.boomyao.codexmobile.R
import com.boomyao.codexmobile.shared.SessionUi

object NativeHostSessionUi {
    fun profileAvatar(profile: BridgeProfile): String = SessionUi.profileAvatar(profile)

    fun displayProfileLabel(profile: BridgeProfile): String = SessionUi.displayProfileLabel(profile)

    fun displayProfileDetail(profile: BridgeProfile): String = SessionUi.displayProfileDetail(profile)

    fun describeProfileStatus(context: Context, profile: BridgeProfile, isCurrent: Boolean): String {
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

    fun describeConnectingStatus(
        context: Context,
        profile: BridgeProfile,
        statusMessage: String,
        stage: NativeHostConnectionStage?,
    ): String {
        return statusMessage.ifBlank {
            when (stage) {
                NativeHostConnectionStage.PAYLOAD_RECEIVED -> context.getString(R.string.native_host_status_payload_received)
                NativeHostConnectionStage.PAIRING_DEVICE -> context.getString(R.string.native_host_status_pairing_device)
                NativeHostConnectionStage.OPENING_WORKSPACE, null -> context.getString(R.string.native_host_status_opening_workspace)
            }
        }.ifBlank { displayProfileDetail(profile) }
    }

    fun summarizeWorkspaceError(
        context: Context,
        message: String,
        isAutoReconnectPending: Boolean = false,
    ): String {
        val normalized = message.trim()
        if (isAutoReconnectPending) {
            return context.getString(R.string.native_host_workspace_reconnecting_body)
        }
        return when {
            normalized.contains("invalid key", ignoreCase = true) ||
                normalized.contains("not valid", ignoreCase = true) ||
                normalized.contains("fresh enrollment qr", ignoreCase = true) ||
                normalized.contains("re-enrolled", ignoreCase = true) ->
                context.getString(R.string.native_host_error_secure_link_expired)
            normalized.contains("expired", ignoreCase = true) ->
                context.getString(R.string.native_host_error_code_expired)
            normalized.isBlank() ->
                context.getString(R.string.native_host_workspace_error_body)
            else -> SessionUi.compactLabel(normalized, maxLength = 96)
        }
    }

    fun workspaceErrorTitle(
        context: Context,
        message: String,
        isAutoReconnectPending: Boolean = false,
    ): String {
        val normalized = message.trim()
        return when {
            isAutoReconnectPending -> context.getString(R.string.native_host_workspace_reconnecting_title)
            normalized.contains("expired", ignoreCase = true) -> context.getString(R.string.native_host_workspace_expired_title)
            else -> context.getString(R.string.native_host_workspace_error)
        }
    }
}
