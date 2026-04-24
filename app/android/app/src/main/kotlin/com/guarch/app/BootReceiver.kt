package com.guarch.app

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.os.Build

class BootReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action == Intent.ACTION_BOOT_COMPLETED) {
            CrashLogger.d("BootReceiver", "Device booted — checking auto-reconnect...")

            // TODO: Check SharedPreferences for auto-reconnect setting
            // If enabled, start VPN service automatically
            
            // Example:
            // val prefs = context.getSharedPreferences("guarch_prefs", Context.MODE_PRIVATE)
            // val autoConnect = prefs.getBoolean("auto_reconnect", false)
            // if (autoConnect && lastConnectedServer != null) {
            //     startVpnService(context)
            // }
        }
    }
}
