package com.guarch.app

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Intent
import android.net.VpnService
import android.os.Build
import android.os.Handler
import android.os.Looper
import android.os.ParcelFileDescriptor
import androidx.core.app.NotificationCompat
import java.util.concurrent.atomic.AtomicBoolean
import java.util.concurrent.atomic.AtomicInteger

class GuarchService : VpnService() {

    companion object {
        const val CHANNEL_ID = "guarch_vpn_service"
        const val NOTIFICATION_ID = 1001
        const val ACTION_START = "com.guarch.app.START"
        const val ACTION_STOP = "com.guarch.app.STOP"
        const val ACTION_UPDATE_NOTIFICATION = "com.guarch.app.UPDATE_NOTIFICATION"

        private val _isRunning = AtomicBoolean(false)
        private val _tunFd = AtomicInteger(-1)

        @Volatile
        private var _instance: GuarchService? = null

        @Volatile
        private var statusCallback: ((String) -> Unit)? = null

        val isRunning: Boolean
            get() = _isRunning.get()

        val tunFd: Int
            get() = _tunFd.get()

        val instance: GuarchService?
            get() = _instance

        fun updateNotification(text: String) {
            _instance?.updateNotificationText(text)
        }

        fun setStatusCallback(callback: (String) -> Unit) {
            statusCallback = callback
            CrashLogger.d("Service", "Status callback registered")
        }

        fun clearStatusCallback() {
            statusCallback = null
            CrashLogger.d("Service", "Status callback cleared")
        }
    }

    private var vpnInterface: ParcelFileDescriptor? = null
    private var lastNotificationText = "Initializing..."
    private var bytesUploaded = 0L
    private var bytesDownloaded = 0L
    private var connectedTime = 0L
    private var lastUploadSpeed = 0L
    private var lastDownloadSpeed = 0L

    override fun onCreate() {
        super.onCreate()
        CrashLogger.d("Service", "=== onCreate (v1.1.1) ===")

        _instance = this
        MainActivity.setVpnServiceReference(this)

        createNotificationChannel()
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        val action = intent?.action ?: ""
        CrashLogger.d("Service", "=== onStartCommand === action=$action")

        try {
            when (action) {
                ACTION_STOP -> {
                    notifyStatus("vpn_stopping")
                    stopVpn()
                    return START_NOT_STICKY
                }

                ACTION_START -> {
                    val socksPort = intent?.getIntExtra("socks_port", 7070) ?: 7070
                    val enableIPv6 = intent?.getBooleanExtra("enable_ipv6", false) ?: false
                    val allowedApps = intent?.getStringArrayListExtra("allowed_apps") ?: ArrayList()
                    val disallowedApps = intent?.getStringArrayListExtra("disallowed_apps") ?: ArrayList()
                    CrashLogger.d("Service", "  socksPort=$socksPort preferIPv6=$enableIPv6 isRunning=$isRunning")

                    if (isRunning) {
                        CrashLogger.d("Service", "  Already running — restarting...")
                        cleanupTun()
                    }

                    startVpn(socksPort, enableIPv6, allowedApps, disallowedApps)
                }

                ACTION_UPDATE_NOTIFICATION -> {
                    val text = intent?.getStringExtra("text") ?: "Connected"
                    updateNotificationText(text)
                }

                else -> {
                    CrashLogger.w("Service", "  Unknown action: $action")
                }
            }
        } catch (e: Throwable) {
            CrashLogger.e("Service", "onStartCommand CRASHED", e)
            notifyStatus("vpn_establish_failed")
        }

        return START_STICKY
    }

    private fun startVpn(socksPort: Int, enableIPv6: Boolean, allowedApps: List<String>, disallowedApps: List<String>) {
        CrashLogger.d("Service", "--- startVpn ---")

        try {
            CrashLogger.d("Service", "  S1: Starting foreground service...")
            startForeground(NOTIFICATION_ID, createNotification("Connecting..."))
            CrashLogger.d("Service", "  S1: Foreground started ✅")
        } catch (e: Throwable) {
            CrashLogger.e("Service", "  S1: Foreground FAILED", e)
            notifyStatus("vpn_establish_failed")
            stopSelf()
            return
        }

        try {
            CrashLogger.d("Service", "  S2: Building VPN interface...")

            val builder = Builder()
                .setSession("Guarch VPN v1.1.1")
                .addAddress("10.10.10.2", 32)
                .setMtu(1500)
                .setBlocking(false)

            builder.addDnsServer("8.8.8.8")
            builder.addDnsServer("1.1.1.1")

            if (allowedApps.isNotEmpty()) {
                for (app in allowedApps) {
                    try {
                        builder.addAllowedApplication(app)
                        CrashLogger.d("Service", "  S2: Added allowed app: $app")
                    } catch (e: Exception) {
                        CrashLogger.w("Service", "  S2: Failed to add allowed app $app: ${e.message}")
                    }
                }
            } else {
                if (disallowedApps.isNotEmpty()) {
                    for (app in disallowedApps) {
                        try {
                            builder.addDisallowedApplication(app)
                            CrashLogger.d("Service", "  S2: Excluded app: $app")
                        } catch (e: Exception) {
                            CrashLogger.w("Service", "  S2: Failed to exclude app $app: ${e.message}")
                        }
                    }
                }

                try {
                    builder.addDisallowedApplication(packageName)
                    CrashLogger.d("Service", "  S2: Excluded self from VPN ✅")
                } catch (e: Throwable) {
                    CrashLogger.w("Service", "  S2: Could not exclude self (OK on older Android)")
                }
            }

            CrashLogger.d("Service", "  S2: Configuring full tunnel...")
            builder.addRoute("0.0.0.0", 0)
            CrashLogger.d("Service", "  S2: ✅ Full tunneling configured (0.0.0.0/0)")

            if (enableIPv6) {
                try {
                    builder.addAddress("fd00::2", 64)
                    builder.addRoute("::", 0)
                    CrashLogger.d("Service", "  S2: IPv6 enabled ✅")
                } catch (e: Throwable) {
                    CrashLogger.w("Service", "  S2: IPv6 setup failed: ${e.message}")
                }
            } else {
                try {
                    builder.addRoute("2000::", 3)
                    CrashLogger.d("Service", "  S2: IPv6 disabled (blackhole route)")
                } catch (e: Throwable) {
                }
            }

            CrashLogger.d("Service", "  S2: Calling establish()...")
            vpnInterface = builder.establish()

            if (vpnInterface == null) {
                CrashLogger.e("Service", "  S2: establish() returned NULL!")
                _tunFd.set(-1)
                _isRunning.set(false)
                notifyStatus("vpn_establish_failed")
                stopSelf()
                return
            }

            val fd = vpnInterface!!.fd
            _tunFd.set(fd)
            _isRunning.set(true)
            connectedTime = System.currentTimeMillis()

            CrashLogger.d("Service", "  S2: VPN interface established ✅")
            CrashLogger.d("Service", "  S2: TUN fd = $fd")
            CrashLogger.d("Service", "  S2: Address = 10.10.10.2/32")
            CrashLogger.d("Service", "  S2: Route = 0.0.0.0/0 (full tunnel)")
            CrashLogger.d("Service", "  S2: DNS = 8.8.8.8, 1.1.1.1")
            CrashLogger.d("Service", "  S2: MTU = 1500")
            CrashLogger.d("Service", "  S2: IPv6 = ${if (enableIPv6) "preferred" else "not preferred (IPv4 first)"}")

            updateNotificationText("Connected ✅")
            notifyStatus("vpn_started")

            CrashLogger.d("Service", "=== VPN STARTED ===")

        } catch (e: Throwable) {
            CrashLogger.e("Service", "  S2: VPN setup CRASHED", e)
            _tunFd.set(-1)
            _isRunning.set(false)
            notifyStatus("vpn_establish_failed")
            stopSelf()
        }
    }

    private fun cleanupTun() {
        CrashLogger.d("Service", "  cleanupTun")
        _tunFd.set(-1)

        try {
            vpnInterface?.close()
            vpnInterface = null
        } catch (e: Throwable) {
            CrashLogger.e("Service", "  cleanupTun error", e)
        }
    }

    private fun stopVpn() {
        CrashLogger.d("Service", "--- stopVpn ---")

        _isRunning.set(false)
        _tunFd.set(-1)
        _instance = null

        try {
            vpnInterface?.close()
            vpnInterface = null
        } catch (e: Throwable) {
            CrashLogger.e("Service", "  vpnInterface.close() error", e)
        }

        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.N) {
            stopForeground(STOP_FOREGROUND_REMOVE)
        } else {
            @Suppress("DEPRECATION")
            stopForeground(true)
        }

        stopSelf()
        CrashLogger.d("Service", "  VPN stopped ✅")
    }

    private fun createNotificationChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                CHANNEL_ID,
                "Guarch VPN Service",
                NotificationManager.IMPORTANCE_LOW
            ).apply {
                description = "Persistent notification for Guarch VPN connection"
                setShowBadge(false)
                lockscreenVisibility = Notification.VISIBILITY_PUBLIC
            }

            val manager = getSystemService(NotificationManager::class.java)
            manager?.createNotificationChannel(channel)
            CrashLogger.d("Service", "Notification channel created")
        }
    }

    private fun createNotification(text: String): Notification {
        val pendingIntent = PendingIntent.getActivity(
            this,
            0,
            Intent(this, MainActivity::class.java).apply {
                flags = Intent.FLAG_ACTIVITY_SINGLE_TOP
            },
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M) {
                PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
            } else {
                PendingIntent.FLAG_UPDATE_CURRENT
            }
        )

        val disconnectIntent = PendingIntent.getService(
            this,
            1,
            Intent(this, GuarchService::class.java).apply {
                action = ACTION_STOP
            },
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M) {
                PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
            } else {
                PendingIntent.FLAG_UPDATE_CURRENT
            }
        )

        return NotificationCompat.Builder(this, CHANNEL_ID)
            .setContentTitle("🏹 Guarch VPN")
            .setContentText(text)
            .setSmallIcon(android.R.drawable.ic_lock_lock)
            .setContentIntent(pendingIntent)
            .setOngoing(true)
            .setPriority(NotificationCompat.PRIORITY_LOW)
            .setCategory(NotificationCompat.CATEGORY_SERVICE)
            .addAction(
                android.R.drawable.ic_menu_close_clear_cancel,
                "Disconnect",
                disconnectIntent
            )
            .build()
    }

    fun updateNotificationText(text: String) {
        if (text == lastNotificationText) return

        lastNotificationText = text

        try {
            val manager = getSystemService(NotificationManager::class.java)
            manager?.notify(NOTIFICATION_ID, createNotification(text))
        } catch (e: Throwable) {
            CrashLogger.e("Service", "updateNotification failed", e)
        }
    }

    fun updateStatsFromGo(upload: Long, download: Long, upSpeed: Long, downSpeed: Long) {
        bytesUploaded = upload
        bytesDownloaded = download
        lastUploadSpeed = upSpeed
        lastDownloadSpeed = downSpeed

        val uploadMB = upload / (1024.0 * 1024.0)
        val downloadMB = download / (1024.0 * 1024.0)

        val upSpeedKB = upSpeed / 1024.0
        val downSpeedKB = downSpeed / 1024.0

        val uptime = if (connectedTime > 0) {
            (System.currentTimeMillis() - connectedTime) / 1000
        } else {
            0
        }

        val statsText = if (upSpeedKB > 0 || downSpeedKB > 0) {
            String.format(
                "↑ %.1f KB/s ↓ %.1f KB/s • %.2f/%.2f MB",
                upSpeedKB,
                downSpeedKB,
                uploadMB,
                downloadMB
            )
        } else {
            String.format(
                "↑ %.2f MB ↓ %.2f MB • %dm%ds",
                uploadMB,
                downloadMB,
                uptime / 60,
                uptime % 60
            )
        }

        updateNotificationText(statsText)
    }

    fun updateStats(upload: Long, download: Long) {
        updateStatsFromGo(upload, download, 0, 0)
    }

    private fun notifyStatus(status: String) {
        CrashLogger.d("Service", "Notify status: $status")
        statusCallback?.let { callback ->
            Handler(Looper.getMainLooper()).post {
                try {
                    callback(status)
                    CrashLogger.d("Service", "Status callback executed: $status")
                } catch (e: Throwable) {
                    CrashLogger.e("Service", "Status callback failed", e)
                }
            }
        } ?: run {
            CrashLogger.w("Service", "Status callback is null, cannot notify: $status")
        }
    }

    override fun onRevoke() {
        CrashLogger.d("Service", "=== onRevoke (user revoked VPN permission) ===")
        notifyStatus("vpn_revoked")
        stopVpn()
        super.onRevoke()
    }

    override fun onDestroy() {
        CrashLogger.d("Service", "=== onDestroy ===")

        _isRunning.set(false)
        _tunFd.set(-1)

        notifyStatus("vpn_destroyed")

        _instance = null
        MainActivity.setVpnServiceReference(null)

        try {
            vpnInterface?.close()
        } catch (_: Throwable) {}

        vpnInterface = null

        super.onDestroy()
    }
}
