package com.guarch.app

import android.app.Activity
import android.content.Intent
import android.net.VpnService
import android.os.Build
import androidx.core.content.FileProvider
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.EventChannel
import io.flutter.plugin.common.MethodChannel
import java.io.File

class MainActivity : FlutterActivity() {

    companion object {
        const val ENGINE_CHANNEL = "com.guarch.app/engine"
        const val EVENT_CHANNEL = "com.guarch.app/events"
        const val LOG_CHANNEL = "com.guarch.app/logs"
        const val VPN_REQUEST_CODE = 1001
        const val TAG = "Guarch"
    }

    private var vpnPermissionResult: MethodChannel.Result? = null
    private var methodChannel: MethodChannel? = null
    private var pendingSocksPort: Int = 1080
    private var goEngine: Any? = null

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)

        // ← اول از همه لاگر رو init کن
        CrashLogger.init(this)
        CrashLogger.d(TAG, "====== APP STARTED ======")
        CrashLogger.d(TAG, "SDK: ${Build.VERSION.SDK_INT} | Device: ${Build.MANUFACTURER} ${Build.MODEL}")

        tryInitGoEngine()

        // Engine channel
        methodChannel = MethodChannel(flutterEngine.dartExecutor.binaryMessenger, ENGINE_CHANNEL)
        methodChannel?.setMethodCallHandler { call, result ->
            CrashLogger.d(TAG, ">> Method: ${call.method}")
            try {
                when (call.method) {
                    "connect" -> handleConnect(call.arguments, result)
                    "disconnect" -> handleDisconnect(result)
                    "getStatus" -> handleGetStatus(result)
                    "getStats" -> handleGetStats(result)
                    "requestVpnPermission" -> requestVpnAndStart(result)
                    else -> {
                        CrashLogger.w(TAG, "Unknown method: ${call.method}")
                        result.notImplemented()
                    }
                }
            } catch (e: Throwable) {
                CrashLogger.e(TAG, "CRASH in ${call.method}", e)
                try {
                    result.error("CRASH", "${e.javaClass.simpleName}: ${e.message}", null)
                } catch (_: Exception) {}
            }
        }

        // Log channel
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, LOG_CHANNEL)
            .setMethodCallHandler { call, result ->
                when (call.method) {
                    "getLogs" -> result.success(CrashLogger.getCurrentLog(this))
                    "getCrashLog" -> result.success(CrashLogger.getPreviousCrashLog(this))
                    "clearLogs" -> {
                        CrashLogger.init(this)
                        result.success(true)
                    }
                    "shareLogs" -> {
                        shareLogs()
                        result.success(true)
                    }
                    "writeFlutterLog" -> {
                        val msg = call.arguments as? String ?: ""
                        CrashLogger.d("Flutter", msg)
                        result.success(true)
                    }
                    else -> result.notImplemented()
                }
            }

        // Event channel
        EventChannel(flutterEngine.dartExecutor.binaryMessenger, EVENT_CHANNEL)
            .setStreamHandler(object : EventChannel.StreamHandler {
                override fun onListen(arguments: Any?, events: EventChannel.EventSink?) {
                    CrashLogger.d(TAG, "EventChannel: onListen")
                }
                override fun onCancel(arguments: Any?) {}
            })

        CrashLogger.d(TAG, "configureFlutterEngine done")
    }

    // ===== Share logs =====
    private fun shareLogs() {
        try {
            val logFile = File(filesDir, "guarch_debug.log")
            if (!logFile.exists()) return

            // کپی به cache برای share
            val shareFile = File(cacheDir, "guarch_log.txt")
            logFile.copyTo(shareFile, overwrite = true)

            val uri = FileProvider.getUriForFile(this, "$packageName.fileprovider", shareFile)
            val intent = Intent(Intent.ACTION_SEND).apply {
                type = "text/plain"
                putExtra(Intent.EXTRA_STREAM, uri)
                putExtra(Intent.EXTRA_SUBJECT, "Guarch Debug Log")
                addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
            }
            startActivity(Intent.createChooser(intent, "Share Guarch Log"))
        } catch (e: Exception) {
            CrashLogger.e(TAG, "Share failed", e)
        }
    }

    // ===== Connect =====
    private fun handleConnect(arguments: Any?, result: MethodChannel.Result) {
        CrashLogger.d(TAG, "=== handleConnect ===")
        CrashLogger.d(TAG, "  args type: ${arguments?.javaClass?.simpleName ?: "NULL"}")
        CrashLogger.d(TAG, "  args: ${arguments?.toString()?.take(200) ?: "NULL"}")
        CrashLogger.d(TAG, "  goEngine: ${if (goEngine != null) "LOADED" else "NULL"}")

        val config = arguments as? String
        if (config == null) {
            CrashLogger.e(TAG, "  Config is NULL!")
            result.error("NULL_CONFIG", "Config is null", null)
            return
        }

        if (goEngine == null) {
            CrashLogger.w(TAG, "  No Go engine")
            result.error("NO_ENGINE", "Native engine not available", null)
            return
        }

        try {
            CrashLogger.d(TAG, "  Calling goEngine.connect()...")
            val connectMethod = goEngine!!.javaClass.getMethod("connect", String::class.java)
            CrashLogger.d(TAG, "  Method found, invoking...")
            val success = connectMethod.invoke(goEngine, config) as Boolean
            CrashLogger.d(TAG, "  connect result: $success")

            if (success) {
                pendingSocksPort = 1080
                requestVpnAndStart(result)
            } else {
                result.success(false)
            }
        } catch (e: Throwable) {
            CrashLogger.e(TAG, "  goEngine.connect CRASHED", e)
            result.success(false)
        }
    }

    // ===== Disconnect =====
    private fun handleDisconnect(result: MethodChannel.Result) {
        CrashLogger.d(TAG, "=== handleDisconnect ===")
        try {
            if (goEngine != null) {
                try {
                    goEngine!!.javaClass.getMethod("stopTun").invoke(goEngine)
                } catch (e: Throwable) { CrashLogger.e(TAG, "stopTun err", e) }
                try {
                    goEngine!!.javaClass.getMethod("disconnect").invoke(goEngine)
                } catch (e: Throwable) { CrashLogger.e(TAG, "disconnect err", e) }
            }
        } catch (e: Throwable) {
            CrashLogger.e(TAG, "Disconnect error", e)
        }
        stopVpnService()
        result.success(true)
    }

    private fun handleGetStatus(result: MethodChannel.Result) {
        try {
            if (goEngine != null) {
                result.success(goEngine!!.javaClass.getMethod("getStatus").invoke(goEngine) as String)
            } else {
                result.success(if (GuarchService.isRunning) "connected" else "disconnected")
            }
        } catch (e: Throwable) {
            result.success("disconnected")
        }
    }

    private fun handleGetStats(result: MethodChannel.Result) {
        try {
            if (goEngine != null) {
                result.success(goEngine!!.javaClass.getMethod("getStats").invoke(goEngine) as String)
            } else {
                result.success("{}")
            }
        } catch (_: Throwable) {
            result.success("{}")
        }
    }

    // ===== Go Engine =====
    private fun tryInitGoEngine() {
        CrashLogger.d(TAG, "--- tryInitGoEngine ---")
        try {
            val cls = Class.forName("mobile.Mobile")
            goEngine = cls.getMethod("new_").invoke(null)
            CrashLogger.d(TAG, "  Go engine LOADED")
            val methods = goEngine!!.javaClass.methods.map { it.name }.distinct().sorted()
            CrashLogger.d(TAG, "  Methods: ${methods.joinToString(", ")}")
        } catch (e: ClassNotFoundException) {
            CrashLogger.w(TAG, "  mobile.Mobile NOT FOUND")
            goEngine = null
        } catch (e: Throwable) {
            CrashLogger.e(TAG, "  Go engine init FAILED", e)
            goEngine = null
        }
    }

    // ===== VPN =====
    private fun requestVpnAndStart(result: MethodChannel.Result) {
        CrashLogger.d(TAG, "--- requestVpnAndStart ---")
        try {
            val intent = VpnService.prepare(this)
            CrashLogger.d(TAG, "  prepare: ${if (intent != null) "needs perm" else "granted"}")
            if (intent != null) {
                vpnPermissionResult = result
                startActivityForResult(intent, VPN_REQUEST_CODE)
            } else {
                startVpnAndTun(result)
            }
        } catch (e: Throwable) {
            CrashLogger.e(TAG, "  requestVpn CRASHED", e)
            try { result.error("VPN_ERROR", e.message, null) } catch (_: Exception) {}
        }
    }

    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        CrashLogger.d(TAG, "onActivityResult: req=$requestCode res=$resultCode")
        super.onActivityResult(requestCode, resultCode, data)
        if (requestCode == VPN_REQUEST_CODE) {
            if (resultCode == Activity.RESULT_OK) {
                vpnPermissionResult?.let { startVpnAndTun(it) }
            } else {
                vpnPermissionResult?.success(false)
            }
            vpnPermissionResult = null
        }
    }

    private fun startVpnAndTun(result: MethodChannel.Result) {
        CrashLogger.d(TAG, "=== startVpnAndTun ===")
        try {
            val intent = Intent(this, GuarchService::class.java).apply {
                action = GuarchService.ACTION_START
                putExtra("socks_port", pendingSocksPort)
            }

            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                CrashLogger.d(TAG, "  startForegroundService")
                startForegroundService(intent)
            } else {
                startService(intent)
            }
            CrashLogger.d(TAG, "  Service started")

            Thread {
                try {
                    CrashLogger.d(TAG, "  Waiting for TUN fd...")
                    var attempts = 0
                    while (GuarchService.tunFd < 0 && attempts < 50) {
                        Thread.sleep(100)
                        attempts++
                        if (attempts % 10 == 0) {
                            CrashLogger.d(TAG, "  wait... attempt=$attempts fd=${GuarchService.tunFd}")
                        }
                    }

                    val fd = GuarchService.tunFd
                    CrashLogger.d(TAG, "  fd=$fd attempts=$attempts")

                    if (fd >= 0 && goEngine != null) {
                        try {
                            CrashLogger.d(TAG, "  startTun(fd=$fd, port=$pendingSocksPort)")
                            goEngine!!.javaClass.getMethod("startTun", Int::class.java, Int::class.java)
                                .invoke(goEngine, fd, pendingSocksPort)
                            CrashLogger.d(TAG, "  startTun done ✅")
                        } catch (e: Throwable) {
                            CrashLogger.e(TAG, "  startTun FAILED", e)
                        }
                    }

                    runOnUiThread {
                        try { result.success(fd >= 0) } catch (e: Throwable) {
                            CrashLogger.e(TAG, "  Result send failed", e)
                        }
                    }
                } catch (e: Throwable) {
                    CrashLogger.e(TAG, "  Thread CRASHED", e)
                    runOnUiThread {
                        try { result.error("THREAD", e.message, null) } catch (_: Exception) {}
                    }
                }
            }.start()
        } catch (e: Throwable) {
            CrashLogger.e(TAG, "  startVpnAndTun CRASHED", e)
            try { result.error("CRASH", e.message, null) } catch (_: Exception) {}
        }
    }

    private fun stopVpnService() {
        CrashLogger.d(TAG, "--- stopVpnService ---")
        try {
            startService(Intent(this, GuarchService::class.java).apply {
                action = GuarchService.ACTION_STOP
            })
        } catch (e: Throwable) {
            CrashLogger.e(TAG, "stop CRASHED", e)
        }
    }

    override fun onDestroy() {
        CrashLogger.d(TAG, "=== Activity onDestroy ===")
        CrashLogger.close()
        super.onDestroy()
    }
}
