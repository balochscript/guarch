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
    }

    private var vpnInterface: ParcelFileDescriptor? = null

    override fun onCreate() {
        super.onCreate()
        CrashLogger.d("Service", "=== onCreate ===")
        try {
            createNotificationChannel()
        } catch (e: Throwable) {
            CrashLogger.e("Service", "onCreate CRASHED", e)
        }
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        CrashLogger.d("Service", "=== onStartCommand === action=${intent?.action}")
        try {
            when (intent?.action) {
                ACTION_STOP -> {
                    CrashLogger.d("Service", "  Stopping...")
                    stopVpn()
                    return START_NOT_STICKY
                }
                ACTION_START -> {
                    val socksPort = intent.getIntExtra("socks_port", 1080)
                    CrashLogger.d("Service", "  Starting with port=$socksPort")
                    startVpn(socksPort)
                }
                else -> {
                    CrashLogger.w("Service", "  Unknown action: ${intent?.action}")
                }
            }
        } catch (e: Throwable) {
            CrashLogger.e("Service", "onStartCommand CRASHED", e)
        }
        return START_STICKY
    }

    private fun startVpn(socksPort: Int) {
        CrashLogger.d("Service", "--- startVpn ---")
        if (isRunning) {
            CrashLogger.d("Service", "  Already running, skip")
            return
        }

        // Step 1: Notification
        try {
            CrashLogger.d("Service", "  S1: Building notification...")
            val notification = createNotification()
            CrashLogger.d("Service", "  S1: Notification built")

            CrashLogger.d("Service", "  S1: startForeground...")
            startForeground(NOTIFICATION_ID, notification)
            CrashLogger.d("Service", "  S1: Foreground started")
        } catch (e: Throwable) {
            CrashLogger.e("Service", "  S1: Foreground FAILED", e)
            stopSelf()
            return
        }

        // Step 2: TUN interface
        try {
            CrashLogger.d("Service", "  S2: Building TUN interface...")
            
            val builder = Builder()
            CrashLogger.d("Service", "  S2: Builder created")
            
            builder.setSession("Guarch")
            CrashLogger.d("Service", "  S2: Session set")
            
            builder.addAddress("10.10.10.2", 32)
            CrashLogger.d("Service", "  S2: Address added")
            
            builder.addRoute("0.0.0.0", 0)
            CrashLogger.d("Service", "  S2: Route added")
            
            builder.addDnsServer("8.8.8.8")
            CrashLogger.d("Service", "  S2: DNS 1 added")
            
            builder.addDnsServer("1.1.1.1")
            CrashLogger.d("Service", "  S2: DNS 2 added")
            
            builder.setMtu(1500)
            CrashLogger.d("Service", "  S2: MTU set")

            try {
                builder.addDisallowedApplication(packageName)
                CrashLogger.d("Service", "  S2: Excluded: $packageName")
            } catch (e: Throwable) {
                CrashLogger.w("Service", "  S2: Exclude failed: ${e.message}")
            }

            builder.setBlocking(false)
            CrashLogger.d("Service", "  S2: Non-blocking set")

            CrashLogger.d("Service", "  S2: Calling establish()...")
            vpnInterface = builder.establish()

            if (vpnInterface == null) {
                CrashLogger.e("Service", "  S2: establish() returned NULL!")
                tunFd = -1
                isRunning = false
                stopSelf()
                return
            }

            CrashLogger.d("Service", "  S2: establish() OK, fd=${vpnInterface!!.fd}")

            // detachFd منتقل میکنه fd رو
            tunFd = vpnInterface!!.detachFd()
            isRunning = true
            CrashLogger.d("Service", "  S2: detachFd=$tunFd isRunning=true")
            
            // بعد از detach دیگه نباید vpnInterface رو close کنیم
            vpnInterface = null

        } catch (e: Throwable) {
            CrashLogger.e("Service", "  S2: TUN CRASHED", e)
            tunFd = -1
            isRunning = false
            stopSelf()
        }

        CrashLogger.d("Service", "--- startVpn DONE --- fd=$tunFd")
    }

    private fun stopVpn() {
        CrashLogger.d("Service", "--- stopVpn ---")
        isRunning = false
        tunFd = -1

        try {
            vpnInterface?.close()
            vpnInterface = null
            CrashLogger.d("Service", "  Interface closed")
        } catch (e: Throwable) {
            CrashLogger.e("Service", "  Close error", e)
        }

        try {
            stopForeground(STOP_FOREGROUND_REMOVE)
            stopSelf()
            CrashLogger.d("Service", "  Service stopped")
        } catch (e: Throwable) {
            CrashLogger.e("Service", "  Stop error", e)
        }
    }

    override fun onRevoke() {
        CrashLogger.d("Service", "=== onRevoke ===")
        stopVpn()
        super.onRevoke()
    }

    override fun onDestroy() {
        CrashLogger.d("Service", "=== onDestroy ===")
        stopVpn()
        super.onDestroy()
    }

    // *** حذف override onBind — بذار VpnService خودش هندل کنه ***

    private fun createNotificationChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            CrashLogger.d("Service", "  Creating notification channel...")
            val channel = NotificationChannel(
                CHANNEL_ID,
                "Guarch Connection",
                NotificationManager.IMPORTANCE_LOW
            ).apply {
                description = "Guarch tunnel is active"
                setShowBadge(false)
            }
            val manager = getSystemService(NotificationManager::class.java)
            manager?.createNotificationChannel(channel)
            CrashLogger.d("Service", "  Channel created")
        }
    }

    private fun createNotification(): Notification {
        CrashLogger.d("Service", "  Building notification...")
        val intent = Intent(this, MainActivity::class.java)
        val pendingIntent = PendingIntent.getActivity(
            this, 0, intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )
        return NotificationCompat.Builder(this, CHANNEL_ID)
            .setContentTitle("Guarch Active")
            .setContentText("Connected")
            .setSmallIcon(android.R.drawable.ic_lock_lock)
            .setContentIntent(pendingIntent)
            .setOngoing(true)
            .build()
    }
}
