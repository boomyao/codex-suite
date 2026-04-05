package com.boomyao.codexmobile.shared

data class WorkspaceSelectionChange(
    val optionsChanged: Boolean,
    val activeChanged: Boolean,
)

class NativeHostSessionState {
    val workspaceRootOptions: MutableList<String> = mutableListOf()
    val activeWorkspaceRoots: MutableList<String> = mutableListOf()
    val workspaceRootLabels: MutableMap<String, String> = mutableMapOf()
    val pinnedThreadIds: MutableList<String> = mutableListOf()

    fun updateWorkspaceRoots(nextRoots: List<String>, preferredRoot: String? = null): WorkspaceSelectionChange? {
        val normalizedRoots = uniqueTrimmedStrings(nextRoots.mapNotNull(::normalizeWorkspaceRootCandidate))
        val preferred = normalizeWorkspaceRootCandidate(preferredRoot)
        val nextActiveRoots =
            when {
                preferred != null && normalizedRoots.contains(preferred) -> mutableListOf(preferred)
                activeWorkspaceRoots.firstOrNull() in normalizedRoots -> mutableListOf(activeWorkspaceRoots.first())
                normalizedRoots.isNotEmpty() -> mutableListOf(normalizedRoots.first())
                else -> mutableListOf()
            }

        val optionsChanged = workspaceRootOptions != normalizedRoots
        val activeChanged = activeWorkspaceRoots != nextActiveRoots
        if (!optionsChanged && !activeChanged) {
            return null
        }

        workspaceRootOptions.clear()
        workspaceRootOptions.addAll(normalizedRoots)
        activeWorkspaceRoots.clear()
        activeWorkspaceRoots.addAll(nextActiveRoots)
        return WorkspaceSelectionChange(
            optionsChanged = optionsChanged,
            activeChanged = activeChanged,
        )
    }

    fun mergeWorkspaceRoots(nextRoots: List<String>, preferredRoot: String? = null): WorkspaceSelectionChange? {
        return updateWorkspaceRoots(workspaceRootOptions + nextRoots, preferredRoot)
    }

    fun setActiveWorkspaceRoot(root: String): WorkspaceSelectionChange? {
        val normalizedRoot = normalizeWorkspaceRootCandidate(root) ?: return null
        val nextOptions =
            if (workspaceRootOptions.contains(normalizedRoot)) {
                workspaceRootOptions.toMutableList()
            } else {
                mutableListOf(normalizedRoot).apply { addAll(workspaceRootOptions) }
            }
        val nextActiveRoots = mutableListOf(normalizedRoot)
        val optionsChanged = workspaceRootOptions != nextOptions
        val activeChanged = activeWorkspaceRoots != nextActiveRoots
        if (!optionsChanged && !activeChanged) {
            return null
        }

        workspaceRootOptions.clear()
        workspaceRootOptions.addAll(nextOptions)
        activeWorkspaceRoots.clear()
        activeWorkspaceRoots.addAll(nextActiveRoots)
        return WorkspaceSelectionChange(
            optionsChanged = optionsChanged,
            activeChanged = activeChanged,
        )
    }

    fun replaceWorkspaceRootLabels(labels: Map<String, String>) {
        workspaceRootLabels.clear()
        workspaceRootLabels.putAll(labels)
    }

    fun renameWorkspaceRoot(root: String, label: String): Boolean {
        val normalizedRoot = normalizeWorkspaceRootCandidate(root) ?: return false
        if (!workspaceRootOptions.contains(normalizedRoot)) {
            return false
        }
        if (label.isEmpty()) {
            workspaceRootLabels.remove(normalizedRoot)
        } else {
            workspaceRootLabels[normalizedRoot] = label
        }
        return true
    }

    fun setThreadPinned(threadId: String, pinned: Boolean): Boolean {
        if (threadId.isBlank()) {
            return false
        }
        if (pinned) {
            val nextThreadIds = uniqueTrimmedStrings(pinnedThreadIds + threadId)
            pinnedThreadIds.clear()
            pinnedThreadIds.addAll(nextThreadIds)
        } else {
            pinnedThreadIds.removeAll { it == threadId }
        }
        return true
    }

    fun replacePinnedThreadIds(threadIds: List<String>) {
        pinnedThreadIds.clear()
        pinnedThreadIds.addAll(uniqueTrimmedStrings(threadIds))
    }

    companion object {
        fun normalizeWorkspaceRootCandidate(value: String?): String? {
            val normalized = value?.trim()?.replace('\\', '/')?.replace(Regex("/+"), "/").orEmpty()
            return normalized.ifBlank { null }
        }
    }
}
