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
    private var pendingConfig: String? = null
    private var methodChannel: MethodChannel? = null
    private var pendingSocksPort: Int = 1080
    private var goEngine: Any? = null

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)

        CrashLogger.init(this)
        CrashLogger.d(TAG, "====== APP STARTED ======")
        CrashLogger.d(TAG, "SDK: ${Build.VERSION.SDK_INT} | Device: ${Build.MANUFACTURER} ${Build.MODEL}")

        tryInitGoEngine()

        methodChannel = MethodChannel(flutterEngine.dartExecutor.binaryMessenger, ENGINE_CHANNEL)
        methodChannel?.setMethodCallHandler { call, result ->
            CrashLogger.d(TAG, ">> Method: ${call.method}")
            try {
                when (call.method) {
                    "connect" -> handleConnect(call.arguments, result)
                    "disconnect" -> handleDisconnect(result)
                    "getStatus" -> handleGetStatus(result)
                    "getStats" -> handleGetStats(result)
                    "requestVpnPermission" -> requestVpnPermission(result)
                    else -> result.notImplemented()
                }
            } catch (e: Throwable) {
                CrashLogger.e(TAG, "CRASH in ${call.method}", e)
                try { result.error("CRASH", e.message, null) } catch (_: Exception) {}
            }
        }

        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, LOG_CHANNEL)
            .setMethodCallHandler { call, result ->
                when (call.method) {
                    "getLogs" -> result.success(CrashLogger.getCurrentLog(this))
                    "getCrashLog" -> result.success(CrashLogger.getPreviousCrashLog(this))
                    "clearLogs" -> { CrashLogger.init(this); result.success(true) }
                    "shareLogs" -> { shareLogs(); result.success(true) }
                    "writeFlutterLog" -> {
                        CrashLogger.d("Flutter", call.arguments as? String ?: "")
                        result.success(true)
                    }
                    else -> result.notImplemented()
                }
            }

        EventChannel(flutterEngine.dartExecutor.binaryMessenger, EVENT_CHANNEL)
            .setStreamHandler(object : EventChannel.StreamHandler {
                override fun onListen(arguments: Any?, events: EventChannel.EventSink?) {
                    CrashLogger.d(TAG, "EventChannel: onListen")
                }
                override fun onCancel(arguments: Any?) {}
            })

        CrashLogger.d(TAG, "configureFlutterEngine done")
    }

    // ═══════════════════════════════
    // NPV-Style Connect
    // ═══════════════════════════════

    private fun handleConnect(arguments: Any?, result: MethodChannel.Result) {
        CrashLogger.d(TAG, "=== handleConnect (NPV-style) ===")

        val config = arguments as? String
        if (config == null) {
            CrashLogger.e(TAG, "  Config NULL!")
            result.error("NULL_CONFIG", "Config is null", null)
            return
        }
        CrashLogger.d(TAG, "  config: ${config.take(200)}")
        CrashLogger.d(TAG, "  goEngine: ${if (goEngine != null) "LOADED" else "NULL"}")

        if (goEngine == null) {
            result.error("NO_ENGINE", "Native engine not available", null)
            return
        }

        // NPV-style: اول VPN شروع کن، بعد result برگردون
        pendingConfig = config
        startVpnFirst(result)
    }

    private fun startVpnFirst(result: MethodChannel.Result) {
        CrashLogger.d(TAG, "--- startVpnFirst ---")
        try {
            val intent = VpnService.prepare(this)
            if (intent != null) {
                CrashLogger.d(TAG, "  Needs VPN permission")
                vpnPermissionResult = result
                startActivityForResult(intent, VPN_REQUEST_CODE)
            } else {
                CrashLogger.d(TAG, "  VPN permission granted")
                startVpnThenConnect(result)
            }
        } catch (e: Throwable) {
            CrashLogger.e(TAG, "  startVpnFirst CRASHED", e)
            result.success(false)
        }
    }

    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        CrashLogger.d(TAG, "onActivityResult: req=$requestCode res=$resultCode")
        super.onActivityResult(requestCode, resultCode, data)
        if (requestCode == VPN_REQUEST_CODE) {
            if (resultCode == Activity.RESULT_OK) {
                vpnPermissionResult?.let { startVpnThenConnect(it) }
            } else {
                CrashLogger.w(TAG, "  VPN permission denied")
                vpnPermissionResult?.success(false)
            }
            vpnPermissionResult = null
        }
    }

    private fun startVpnThenConnect(result: MethodChannel.Result) {
        CrashLogger.d(TAG, "=== startVpnThenConnect ===")
        try {
            // ۱. VPN Service شروع کن
            CrashLogger.d(TAG, "  S1: Starting VPN service...")
            val serviceIntent = Intent(this, GuarchService::class.java).apply {
                action = GuarchService.ACTION_START
                putExtra("socks_port", pendingSocksPort)
            }
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                startForegroundService(serviceIntent)
            } else {
                startService(serviceIntent)
            }
            CrashLogger.d(TAG, "  S1: Service started ✅")

            // ۲. صبر کن fd آماده بشه (سریعه)
            Thread {
                try {
                    CrashLogger.d(TAG, "  S2: Waiting for TUN fd...")
                    var attempts = 0
                    while (GuarchService.tunFd < 0 && attempts < 30) {
                        Thread.sleep(100)
                        attempts++
                    }
                    val fd = GuarchService.tunFd
                    CrashLogger.d(TAG, "  S2: fd=$fd attempts=$attempts")

                    if (fd < 0) {
                        CrashLogger.e(TAG, "  S2: Failed to get TUN fd!")
                        runOnUiThread { result.success(false) }
                        return@Thread
                    }

                    // ۳. فوراً result برگردون → Flutter نشون میده "connected"
                    CrashLogger.d(TAG, "  S3: Returning success to Flutter (NPV-style)")
                    runOnUiThread { result.success(true) }

                    // ۴. پشت صحنه: Go engine وصل کن
                    CrashLogger.d(TAG, "  S4: Starting Go engine...")
                    val config = pendingConfig
                    if (config != null && goEngine != null) {
                        try {
                            val connectMethod = goEngine!!.javaClass.getMethod("connect", String::class.java)
                            val connected = connectMethod.invoke(goEngine, config) as Boolean
                            CrashLogger.d(TAG, "  S4: Go connect=$connected")
                        } catch (e: Throwable) {
                            val cause = unwrapException(e)
                            CrashLogger.e(TAG, "  S4: Go connect FAILED: ${cause.message}", cause)
                        }
                    }

                    // ۵. پشت صحنه: TUN شروع کن (خودش منتظر SOCKS5 میمونه)
                    if (goEngine != null && fd >= 0) {
                        try {
                            CrashLogger.d(TAG, "  S5: Starting TUN (fd=$fd port=$pendingSocksPort)...")
                            val startTunMethod = goEngine!!.javaClass.getMethod(
                                "startTun", Int::class.java, Int::class.java
                            )
                            startTunMethod.invoke(goEngine, fd, pendingSocksPort)
                            CrashLogger.d(TAG, "  S5: TUN started ✅")
                        } catch (e: Throwable) {
                            val cause = unwrapException(e)
                            CrashLogger.e(TAG, "  S5: TUN FAILED: ${cause.message}", cause)
                        }
                    }

                    CrashLogger.d(TAG, "=== Background setup complete ===")

                } catch (e: Throwable) {
                    CrashLogger.e(TAG, "  Background thread CRASHED", e)
                }
            }.start()

        } catch (e: Throwable) {
            CrashLogger.e(TAG, "  startVpnThenConnect CRASHED", e)
            result.success(false)
        }
    }

    // ═══════════════════════════════
    // Disconnect
    // ═══════════════════════════════

    private fun handleDisconnect(result: MethodChannel.Result) {
        CrashLogger.d(TAG, "=== handleDisconnect ===")
        Thread {
            try {
                if (goEngine != null) {
                    try {
                        goEngine!!.javaClass.getMethod("stopTun").invoke(goEngine)
                        CrashLogger.d(TAG, "  stopTun ok")
                    } catch (e: Throwable) {
                        CrashLogger.e(TAG, "  stopTun err", unwrapException(e))
                    }
                    try {
                        goEngine!!.javaClass.getMethod("disconnect").invoke(goEngine)
                        CrashLogger.d(TAG, "  disconnect ok")
                    } catch (e: Throwable) {
                        CrashLogger.e(TAG, "  disconnect err", unwrapException(e))
                    }
                }
            } catch (e: Throwable) {
                CrashLogger.e(TAG, "  Disconnect error", e)
            }

            stopVpnService()
            runOnUiThread { result.success(true) }
        }.start()
    }

    // ═══════════════════════════════
    // Helpers
    // ═══════════════════════════════

    private fun handleGetStatus(result: MethodChannel.Result) {
        try {
            if (goEngine != null) {
                result.success(goEngine!!.javaClass.getMethod("getStatus").invoke(goEngine) as String)
            } else {
                result.success(if (GuarchService.isRunning) "connected" else "disconnected")
            }
        } catch (_: Throwable) { result.success("disconnected") }
    }

    private fun handleGetStats(result: MethodChannel.Result) {
        try {
            if (goEngine != null) {
                result.success(goEngine!!.javaClass.getMethod("getStats").invoke(goEngine) as String)
            } else { result.success("{}") }
        } catch (_: Throwable) { result.success("{}") }
    }

    private fun requestVpnPermission(result: MethodChannel.Result) {
        startVpnFirst(result)
    }

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

    // InvocationTargetException رو باز کن تا خطای واقعی رو ببینیم
    private fun unwrapException(e: Throwable): Throwable {
        if (e is java.lang.reflect.InvocationTargetException && e.cause != null) {
            return e.cause!!
        }
        return e
    }

    private fun stopVpnService() {
        CrashLogger.d(TAG, "--- stopVpnService ---")
        try {
            startService(Intent(this, GuarchService::class.java).apply {
                action = GuarchService.ACTION_STOP
            })
        } catch (e: Throwable) {
            CrashLogger.e(TAG, "stop err", e)
        }
    }

    private fun shareLogs() {
        try {
            val logFile = File(filesDir, "guarch_debug.log")
            if (!logFile.exists()) return
            val shareFile = File(cacheDir, "guarch_log.txt")
            logFile.copyTo(shareFile, overwrite = true)
            val uri = FileProvider.getUriForFile(this, "$packageName.fileprovider", shareFile)
            startActivity(Intent.createChooser(
                Intent(Intent.ACTION_SEND).apply {
                    type = "text/plain"
                    putExtra(Intent.EXTRA_STREAM, uri)
                    putExtra(Intent.EXTRA_SUBJECT, "Guarch Debug Log")
                    addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
                }, "Share Log"
            ))
        } catch (e: Exception) {
            CrashLogger.e(TAG, "Share failed", e)
        }
    }

    override fun onDestroy() {
        CrashLogger.d(TAG, "=== Activity onDestroy ===")
        CrashLogger.close()
        super.onDestroy()
    }
}
