package com.guarch.app

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Intent
import android.net.VpnService
import android.os.Build
import android.os.ParcelFileDescriptor
import androidx.core.app.NotificationCompat

class GuarchService : VpnService() {

    companion object {
        const val CHANNEL_ID = "guarch_service"
        const val NOTIFICATION_ID = 1
        const val ACTION_START = "START"
        const val ACTION_STOP = "STOP"

        @Volatile var isRunning = false
            private set
        @Volatile var tunFd: Int = -1
            private set

        // ← reference به instance فعلی
        @Volatile var instance: GuarchService? = null
            private set
    }

    private var vpnInterface: ParcelFileDescriptor? = null

    override fun onCreate() {
        super.onCreate()
        CrashLogger.d("Service", "=== onCreate ===")
        instance = this
        createNotificationChannel()
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        CrashLogger.d("Service", "=== onStartCommand === action=${intent?.action}")
        try {
            when (intent?.action) {
                ACTION_STOP -> { stopVpn(); return START_NOT_STICKY }
                ACTION_START -> {
                    val port = intent.getIntExtra("socks_port", 1080)
                    CrashLogger.d("Service", "  port=$port isRunning=$isRunning")

                    if (isRunning) {
                        CrashLogger.d("Service", "  Restarting...")
                        cleanupTun()
                    }
                    startVpn(port)
                }
            }
        } catch (e: Throwable) {
            CrashLogger.e("Service", "onStartCommand CRASHED", e)
        }
        return START_STICKY
    }

    private fun startVpn(socksPort: Int) {
        try {
            CrashLogger.d("Service", "  S1: Foreground...")
            startForeground(NOTIFICATION_ID, createNotification("Connecting..."))
            CrashLogger.d("Service", "  S1: Done ✅")
        } catch (e: Throwable) {
            CrashLogger.e("Service", "  S1: FAILED", e)
            stopSelf()
            return
        }

        try {
            CrashLogger.d("Service", "  S2: Building TUN...")
            val builder = Builder()
                .setSession("Guarch VPN")
                .addAddress("10.10.10.2", 32)
                .addRoute("0.0.0.0", 0)
                .addDnsServer("8.8.8.8")
                .addDnsServer("1.1.1.1")
                .setMtu(1500)
                .setBlocking(false)

            try { builder.addDisallowedApplication(packageName) }
            catch (_: Throwable) {}

            CrashLogger.d("Service", "  S2: establish()...")
            vpnInterface = builder.establish()

            if (vpnInterface == null) {
                CrashLogger.e("Service", "  S2: establish NULL!")
                tunFd = -1; isRunning = false; stopSelf()
                return
            }

            // ← fd رو بگیر بدون detach — VPN key میمونه
            tunFd = vpnInterface!!.fd
            isRunning = true
            CrashLogger.d("Service", "  S2: Done fd=$tunFd ✅")

            // notification آپدیت
            updateNotification("Connected ✅")

        } catch (e: Throwable) {
            CrashLogger.e("Service", "  S2: CRASHED", e)
            tunFd = -1; isRunning = false; stopSelf()
        }
    }

    // فقط TUN رو پاک کن بدون stop سرویس
    private fun cleanupTun() {
        CrashLogger.d("Service", "  cleanupTun")
        tunFd = -1
        try {
            vpnInterface?.close()
            vpnInterface = null
        } catch (_: Throwable) {}
    }

    private fun stopVpn() {
        CrashLogger.d("Service", "--- stopVpn ---")
        isRunning = false
        tunFd = -1
        instance = null
        try {
            vpnInterface?.close()
            vpnInterface = null
        } catch (_: Throwable) {}
        stopForeground(STOP_FOREGROUND_REMOVE)
        stopSelf()
    }

    fun updateNotification(text: String) {
        try {
            val manager = getSystemService(NotificationManager::class.java)
            manager?.notify(NOTIFICATION_ID, createNotification(text))
        } catch (_: Throwable) {}
    }

    override fun onRevoke() {
        CrashLogger.d("Service", "onRevoke")
        stopVpn()
        super.onRevoke()
    }

    override fun onDestroy() {
        CrashLogger.d("Service", "onDestroy")
        isRunning = false
        tunFd = -1
        instance = null
        try { vpnInterface?.close() } catch (_: Throwable) {}
        vpnInterface = null
        super.onDestroy()
    }

    private fun createNotificationChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                CHANNEL_ID, "Guarch VPN", NotificationManager.IMPORTANCE_LOW
            ).apply { setShowBadge(false) }
            getSystemService(NotificationManager::class.java)?.createNotificationChannel(channel)
        }
    }

    private fun createNotification(text: String): Notification {
        val pi = PendingIntent.getActivity(this, 0,
            Intent(this, MainActivity::class.java),
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE)
        return NotificationCompat.Builder(this, CHANNEL_ID)
            .setContentTitle("🏹 Guarch")
            .setContentText(text)
            .setSmallIcon(android.R.drawable.ic_lock_lock)
            .setContentIntent(pi)
            .setOngoing(true)
            .build()
    }
}
