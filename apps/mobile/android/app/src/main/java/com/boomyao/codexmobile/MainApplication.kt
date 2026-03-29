package com.boomyao.codexmobile

import android.app.Application
import com.boomyao.codexmobile.nativehost.NativeHostPreferences

class MainApplication : Application() {
    override fun onCreate() {
        super.onCreate()
        NativeHostPreferences(this).applyThemeMode()
    }
}
