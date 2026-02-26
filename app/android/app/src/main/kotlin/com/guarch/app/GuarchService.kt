package com.guarch.app

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Intent
import android.net.VpnService
import android.os.Build
import android.os.IBinder
import android.os.ParcelFileDescriptor
import androidx.core.app.NotificationCompat
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel

class GuarchService : VpnService() {

    companion object {
        const val CHANNEL_ID = "guarch_service"
        const val NOTIFICATION_ID = 1
        const val ACTION_START = "START"
        const val ACTION_STOP = "STOP"
        
        var isRunning = false
            private set
    }

    private var vpnInterface: ParcelFileDescriptor? = null
    private var tun2socksThread: Thread? = null

    override fun onCreate() {
        super.onCreate()
        createNotificationChannel()
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            ACTION_STOP -> {
                stopVpn()
                return START_NOT_STICKY
            }
            ACTION_START -> {
                val socksPort = intent.getIntExtra("socks_port", 1080)
                startVpn(socksPort)
            }
        }
        return START_STICKY
    }

    private fun startVpn(socksPort: Int) {
        if (isRunning) return

        val notification = createNotification()
        startForeground(NOTIFICATION_ID, notification)

        // TUN interface بساز
        vpnInterface = Builder()
            .setSession("Guarch")
            .addAddress("10.10.10.2", 32)
            .addRoute("0.0.0.0", 0)
            .addDnsServer("8.8.8.8")
            .addDnsServer("1.1.1.1")
            .setMtu(1500)
            .setBlocking(false)
            .establish()

        if (vpnInterface == null) {
            android.util.Log.e("GuarchService", "Failed to establish VPN interface")
            stopSelf()
            return
        }

        val fd = vpnInterface!!.fd

        // tun2socks رو اجرا کن
        tun2socksThread = Thread {
            try {
                val cmd = arrayOf(
                    "${applicationInfo.nativeLibraryDir}/libtun2socks.so",
                    "--netif-ipaddr", "10.10.10.2",
                    "--netif-netmask", "255.255.255.0",
                    "--socks-server-addr", "127.0.0.1:$socksPort",
                    "--tunfd", fd.toString(),
                    "--tunmtu", "1500",
                    "--loglevel", "warning"
                )
                
                val processBuilder = ProcessBuilder(*cmd)
                processBuilder.redirectErrorStream(true)
                val process = processBuilder.start()
                
                // لاگ بخون
                val reader = process.inputStream.bufferedReader()
                reader.forEachLine { line ->
                    android.util.Log.d("tun2socks", line)
                }
                
                process.waitFor()
            } catch (e: Exception) {
                android.util.Log.e("GuarchService", "tun2socks error: ${e.message}")
            }
        }
        tun2socksThread?.start()
        
        isRunning = true
        android.util.Log.i("GuarchService", "VPN started on fd=$fd, socks=127.0.0.1:$socksPort")
    }

    private fun stopVpn() {
        isRunning = false
        
        tun2socksThread?.interrupt()
        tun2socksThread = null
        
        vpnInterface?.close()
        vpnInterface = null
        
        stopForeground(STOP_FOREGROUND_REMOVE)
        stopSelf()
        
        android.util.Log.i("GuarchService", "VPN stopped")
    }

    override fun onDestroy() {
        stopVpn()
        super.onDestroy()
    }

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onRevoke() {
        stopVpn()
        super.onRevoke()
    }

    private fun createNotificationChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                CHANNEL_ID,
                "Guarch Connection",
                NotificationManager.IMPORTANCE_LOW
            ).apply {
                description = "Guarch tunnel is active"
                setShowBadge(false)
            }
            val manager = getSystemService(NotificationManager::class.java)
            manager.createNotificationChannel(channel)
        }
    }

    private fun createNotification(): Notification {
        val intent = Intent(this, MainActivity::class.java)
        val pendingIntent = PendingIntent.getActivity(
            this, 0, intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        return NotificationCompat.Builder(this, CHANNEL_ID)
            .setContentTitle("Guarch Active")
            .setContentText("🏹 Hidden like a Balochi hunter")
            .setSmallIcon(android.R.drawable.ic_lock_lock)
            .setContentIntent(pendingIntent)
            .setOngoing(true)
            .build()
    }
}
