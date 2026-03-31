package com.boomyao.codexmobile.nativehost

import android.content.Context
import androidx.appcompat.app.AppCompatDelegate

enum class ThemeMode(
    val storageValue: String,
    val appCompatMode: Int,
) {
    SYSTEM("system", AppCompatDelegate.MODE_NIGHT_FOLLOW_SYSTEM),
    LIGHT("light", AppCompatDelegate.MODE_NIGHT_NO),
    DARK("dark", AppCompatDelegate.MODE_NIGHT_YES),
    ;

    companion object {
        fun fromStorageValue(rawValue: String?): ThemeMode {
            return entries.firstOrNull { it.storageValue == rawValue } ?: SYSTEM
        }
    }
}

class NativeHostPreferences(context: Context) {
    private val preferences =
        context.getSharedPreferences(PREFERENCES_NAME, Context.MODE_PRIVATE)

    fun readThemeMode(): ThemeMode {
        return ThemeMode.fromStorageValue(preferences.getString(KEY_THEME_MODE, null))
    }

    fun writeThemeMode(themeMode: ThemeMode) {
        preferences.edit().putString(KEY_THEME_MODE, themeMode.storageValue).apply()
    }

    fun shouldAutoResumeActiveSession(): Boolean {
        return preferences.getBoolean(KEY_AUTO_RESUME_ACTIVE_SESSION, true)
    }

    fun setAutoResumeActiveSession(enabled: Boolean) {
        preferences.edit().putBoolean(KEY_AUTO_RESUME_ACTIVE_SESSION, enabled).apply()
    }

    fun applyThemeMode() {
        AppCompatDelegate.setDefaultNightMode(readThemeMode().appCompatMode)
    }

    companion object {
        private const val PREFERENCES_NAME = "codex_native_host_preferences"
        private const val KEY_THEME_MODE = "theme_mode"
        private const val KEY_AUTO_RESUME_ACTIVE_SESSION = "auto_resume_active_session"
    }
}
