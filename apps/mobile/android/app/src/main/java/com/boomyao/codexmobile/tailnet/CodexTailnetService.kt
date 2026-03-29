package com.boomyao.codexmobile.tailnet

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.Context
import android.content.Intent
import android.net.ConnectivityManager
import android.net.LinkProperties
import android.net.Network
import android.net.NetworkCapabilities
import android.net.VpnService
import android.os.Build
import com.boomyao.codexmobile.R
import org.json.JSONArray
import org.json.JSONObject
import java.net.NetworkInterface
import java.util.Collections
import kotlin.text.StringBuilder

class CodexTailnetService : VpnService() {
    private var connectivityManager: ConnectivityManager? = null
    private var defaultNetworkCallback: ConnectivityManager.NetworkCallback? = null

    override fun onCreate() {
        super.onCreate()
        android.util.Log.i("CodexMobile", "CodexTailnetService.onCreate")
        ensureNotificationChannel()
        CodexTailnetBridge.installVpnService(this)
        connectivityManager = getSystemService(ConnectivityManager::class.java)
        registerDefaultNetworkCallback()
        updateAndroidNetworkSnapshot()
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        android.util.Log.i(
            "CodexMobile",
            "CodexTailnetService.onStartCommand action=${intent?.action} startId=$startId",
        )
        when (intent?.action) {
            ACTION_STOP -> {
                android.util.Log.i("CodexMobile", "CodexTailnetService stopping runtime")
                CodexTailnetBridge.stop(applicationContext)
                stopForeground(STOP_FOREGROUND_REMOVE)
                stopSelf()
                return START_NOT_STICKY
            }

            ACTION_START -> {
                val rawPayload = intent.getStringExtra(EXTRA_ENROLLMENT_PAYLOAD).orEmpty()
                return try {
                    val payload = parseTailnetEnrollmentPayload(rawPayload)
                    startForeground(NOTIFICATION_ID, buildNotification(getString(R.string.tailnet_notification_starting)))
                    val snapshot = startVpnRuntime(payload)
                    android.util.Log.i(
                        "CodexMobile",
                        "CodexTailnetService.startVpnRuntime state=${snapshot.state} message=${snapshot.message}",
                    )
                    updateNotification(snapshot.message)
                    if (snapshot.state == "running") START_STICKY else START_NOT_STICKY
                } catch (error: Exception) {
                    android.util.Log.w("CodexMobile", "CodexTailnetService failed to start", error)
                    val snapshot = TailnetStatusSnapshot(
                        state = "error",
                        mode = "native-shell",
                        message = error.message ?: "Failed to stage Android tailnet shell.",
                        bridgeName = null,
                        bridgeServerEndpoint = null,
                        localProxyUrl = null,
                        rawEnrollmentType = "codex-mobile-enrollment",
                        auth = null,
                    )
                    EnrollmentStore(applicationContext).writeStatus(snapshot)
                    stopForeground(STOP_FOREGROUND_REMOVE)
                    stopSelf()
                    START_NOT_STICKY
                }
            }
        }

        return START_NOT_STICKY
    }

    override fun onDestroy() {
        unregisterDefaultNetworkCallback()
        CodexTailnetBridge.clearVpnService()
        super.onDestroy()
    }

    private fun ensureNotificationChannel() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) {
            return
        }
        val manager = getSystemService(NotificationManager::class.java) ?: return
        val channel = NotificationChannel(
            CHANNEL_ID,
            getString(R.string.tailnet_notification_channel_name),
            NotificationManager.IMPORTANCE_LOW,
        )
        manager.createNotificationChannel(channel)
    }

    private fun buildNotification(text: String): Notification {
        val builder =
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                Notification.Builder(this, CHANNEL_ID)
            } else {
                @Suppress("DEPRECATION")
                Notification.Builder(this)
            }
        return builder
            .setSmallIcon(android.R.drawable.stat_sys_download_done)
            .setContentTitle(getString(R.string.tailnet_notification_title))
            .setContentText(text)
            .setOngoing(true)
            .build()
    }

    private fun updateNotification(text: String) {
        val manager = getSystemService(NotificationManager::class.java) ?: return
        manager.notify(NOTIFICATION_ID, buildNotification(text))
    }

    private fun startVpnRuntime(payload: TailnetEnrollmentPayload): TailnetStatusSnapshot {
        try {
            updateAndroidNetworkSnapshot()
            return CodexTailnetBridge.start(applicationContext, payload, -1)
        } catch (error: Exception) {
            return TailnetStatusSnapshot(
                state = "error",
                mode = "native-bridge",
                message = error.message ?: "Failed to establish Android VPN interface.",
                bridgeName = payload.bridgeName,
                bridgeServerEndpoint = payload.bridgeServerEndpoint,
                localProxyUrl = null,
                rawEnrollmentType = "codex-mobile-enrollment",
                auth = null,
            )
        }
    }

    private fun registerDefaultNetworkCallback() {
        if (connectivityManager == null || defaultNetworkCallback != null) {
            return
        }
        defaultNetworkCallback = object : ConnectivityManager.NetworkCallback() {
            override fun onAvailable(network: Network) {
                updateAndroidNetworkSnapshot()
            }

            override fun onCapabilitiesChanged(network: Network, networkCapabilities: NetworkCapabilities) {
                updateAndroidNetworkSnapshot()
            }

            override fun onLost(network: Network) {
                updateAndroidNetworkSnapshot()
            }

            override fun onLinkPropertiesChanged(network: Network, linkProperties: LinkProperties) {
                updateAndroidNetworkSnapshot()
            }
        }
        try {
            connectivityManager?.registerDefaultNetworkCallback(defaultNetworkCallback!!)
        } catch (_: Exception) {
            defaultNetworkCallback = null
        }
    }

    private fun unregisterDefaultNetworkCallback() {
        val callback = defaultNetworkCallback ?: return
        try {
            connectivityManager?.unregisterNetworkCallback(callback)
        } catch (_: Exception) {
        } finally {
            defaultNetworkCallback = null
        }
    }

    private fun updateAndroidNetworkSnapshot() {
        val snapshot = JSONArray()
        try {
            val interfaces = Collections.list(NetworkInterface.getNetworkInterfaces())
            for (networkInterface in interfaces) {
                val item = JSONObject()
                item.put("name", networkInterface.name)
                item.put("index", networkInterface.index)
                item.put("mtu", networkInterface.mtu)
                item.put("flags", encodeFlags(networkInterface))

                val hardwareAddress = networkInterface.hardwareAddress
                if (hardwareAddress != null && hardwareAddress.isNotEmpty()) {
                    item.put("hardware_addr", hexBytes(hardwareAddress))
                }

                val addrs = JSONArray()
                for (interfaceAddress in networkInterface.interfaceAddresses) {
                    val address = interfaceAddress?.address ?: continue
                    val prefixLength = interfaceAddress.networkPrefixLength
                    if (prefixLength < 0) {
                        continue
                    }
                    addrs.put("${address.hostAddress}/${prefixLength}")
                }
                item.put("addrs", addrs)
                snapshot.put(item)
            }
        } catch (_: Exception) {
        }
        CodexTailnetBridge.setInterfaceSnapshot(snapshot.toString())

        var interfaceName: String? = null
        try {
            val network = connectivityManager?.activeNetwork
            val linkProperties = if (network != null) connectivityManager?.getLinkProperties(network) else null
            interfaceName = linkProperties?.interfaceName
        } catch (_: Exception) {
            interfaceName = null
        }
        CodexTailnetBridge.setDefaultRouteInterface(interfaceName)
    }

    private fun encodeFlags(networkInterface: NetworkInterface): Int {
        var flags = 0
        try {
            if (networkInterface.isUp) {
                flags = flags or 1
            }
            if (networkInterface.isLoopback) {
                flags = flags or (1 shl 2)
            }
            if (networkInterface.isPointToPoint) {
                flags = flags or (1 shl 3)
            }
            if (networkInterface.supportsMulticast()) {
                flags = flags or (1 shl 4)
            }
        } catch (_: Exception) {
        }
        return flags
    }

    private fun hexBytes(bytes: ByteArray): String {
        val builder = StringBuilder(bytes.size * 3)
        for (index in bytes.indices) {
            if (index > 0) {
                builder.append(':')
            }
            builder.append(String.format("%02x", bytes[index].toInt() and 0xff))
        }
        return builder.toString()
    }

    companion object {
        private const val CHANNEL_ID = "codex-tailnet"
        private const val NOTIFICATION_ID = 3201
        private const val EXTRA_ENROLLMENT_PAYLOAD = "enrollment_payload"
        const val ACTION_START = "com.boomyao.codexmobile.tailnet.action.START"
        const val ACTION_STOP = "com.boomyao.codexmobile.tailnet.action.STOP"

        fun startIntent(context: Context, enrollmentPayload: String): Intent {
            return Intent(context, CodexTailnetService::class.java).apply {
                action = ACTION_START
                putExtra(EXTRA_ENROLLMENT_PAYLOAD, enrollmentPayload)
            }
        }

        fun stopIntent(context: Context): Intent {
            return Intent(context, CodexTailnetService::class.java).apply {
                action = ACTION_STOP
            }
        }
    }
}
