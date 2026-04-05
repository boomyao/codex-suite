package com.boomyao.codexmobile.nativehost

import org.json.JSONObject

// Core models are in com.boomyao.codexmobile.shared.
// Re-export them so existing imports keep working.
typealias BridgeProfile = com.boomyao.codexmobile.shared.BridgeProfile
typealias ConnectionTargetResponse = com.boomyao.codexmobile.shared.ConnectionTargetResponse
typealias PairingResponse = com.boomyao.codexmobile.shared.PairingResponse
typealias BridgeLoadTarget = com.boomyao.codexmobile.shared.BridgeLoadTarget

data class HttpProxyResponse(
    val body: Any?,
    val status: Int,
    val headers: JSONObject,
)

data class BridgeBootstrapState(
    val persistedAtomState: JSONObject?,
    val workspaceRootOptions: List<String>,
    val activeWorkspaceRoots: List<String>,
    val workspaceRootLabels: Map<String, String>,
    val pinnedThreadIds: List<String>,
    val globalState: Map<String, Any?>,
)
