package com.guarch.app

import android.app.Activity
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.net.VpnService
import android.os.BatteryManager
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
    private var eventSink: EventChannel.EventSink? = null
    private var goEngine: Any? = null
    private var batteryReceiver: BroadcastReceiver? = null

    // VPN + TUN lifecycle
    private var vpnAndTunStarted = false
    private var currentBatteryLevel = 100

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)

        CrashLogger.init(this)
        CrashLogger.d(TAG, "====== APP STARTED (v1.0.1) ======")
        CrashLogger.d(TAG, "SDK: ${Build.VERSION.SDK_INT} | Device: ${Build.MANUFACTURER} ${Build.MODEL}")

        tryInitGoEngine()
        setupBatteryMonitoring()

        // ═══════════════════════════════════════════════════════════════
        // Method Channel
        // ═══════════════════════════════════════════════════════════════
        methodChannel = MethodChannel(flutterEngine.dartExecutor.binaryMessenger, ENGINE_CHANNEL)
        methodChannel?.setMethodCallHandler { call, result ->
            CrashLogger.d(TAG, ">> Method: ${call.method}")
            try {
                when (call.method) {
                    "getVersion" -> result.success("Guarch Android v1.0.1")
                    "connect" -> handleConnectLegacy(call.arguments, result)
                    "connectWithConfig" -> handleConnectWithConfig(call.arguments, result)
                    "disconnect" -> handleDisconnect(result)
                    "getStatus" -> handleGetStatus(result)
                    "getStats" -> handleGetStats(result)
                    "setBatteryLevel" -> handleSetBatteryLevel(call.arguments, result)
                    "setDataSaverMode" -> handleSetDataSaverMode(call.arguments, result)
                    "loadConfigJSON" -> handleLoadConfigJSON(call.arguments, result)
                    "loadConfigURI" -> handleLoadConfigURI(call.arguments, result)
                    "loadPreset" -> handleLoadPreset(call.arguments, result)
                    "exportConfigURI" -> handleExportConfigURI(result)
                    "exportConfigJSON" -> handleExportConfigJSON(result)
                    "setSplitTunnelMode" -> handleSetSplitTunnelMode(call.arguments, result)
                    "addSplitTunnelDomain" -> handleAddSplitTunnelDomain(call.arguments, result)
                    "getTUNStats" -> handleGetTUNStats(result)
                    "requestVpnPermission" -> requestVpnPermission(result)
                    "testRealDelay" -> handleTestRealDelay(call.arguments, result)
                    "testConnection" -> handleTestConnection(call.arguments, result)
                    else -> result.notImplemented()
                }
            } catch (e: Throwable) {
                CrashLogger.e(TAG, "CRASH in ${call.method}", e)
                try {
                    result.error("CRASH", e.message ?: "Unknown error", null)
                } catch (_: Exception) {}
            }
        }

        // ═══════════════════════════════════════════════════════════════
        // Log Channel
        // ═══════════════════════════════════════════════════════════════
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, LOG_CHANNEL)
            .setMethodCallHandler { call, result ->
                when (call.method) {
                    "getLogs" -> result.success(CrashLogger.getCurrentLog(this))
                    "getCrashLog" -> result.success(CrashLogger.getPreviousCrashLog(this))
                    "getGoLog" -> {
                        try {
                            val logFile = File(filesDir, "go_debug.log")
                            result.success(if (logFile.exists()) logFile.readText() else "No Go log")
                        } catch (e: Throwable) {
                            result.success("Error reading Go log: ${e.message}")
                        }
                    }
                    "clearLogs" -> {
                        CrashLogger.init(this)
                        result.success(true)
                    }
                    "shareLogs" -> {
                        shareLogs()
                        result.success(true)
                    }
                    "writeFlutterLog" -> {
                        CrashLogger.d("Flutter", call.arguments as? String ?: "")
                        result.success(true)
                    }
                    else -> result.notImplemented()
                }
            }

        // ═══════════════════════════════════════════════════════════════
        // Event Channel
        // ═══════════════════════════════════════════════════════════════
        EventChannel(flutterEngine.dartExecutor.binaryMessenger, EVENT_CHANNEL)
            .setStreamHandler(object : EventChannel.StreamHandler {
                override fun onListen(arguments: Any?, events: EventChannel.EventSink?) {
                    CrashLogger.d(TAG, "EventChannel: onListen")
                    eventSink = events
                }

                override fun onCancel(arguments: Any?) {
                    CrashLogger.d(TAG, "EventChannel: onCancel")
                    eventSink = null
                }
            })

        CrashLogger.d(TAG, "configureFlutterEngine done")
    }

    // ═══════════════════════════════════════════════════════════════
    // Battery Monitoring
    // ═══════════════════════════════════════════════════════════════

    private fun setupBatteryMonitoring() {
        batteryReceiver = object : BroadcastReceiver() {
            override fun onReceive(context: Context?, intent: Intent?) {
                val level = intent?.getIntExtra(BatteryManager.EXTRA_LEVEL, -1) ?: -1
                val scale = intent?.getIntExtra(BatteryManager.EXTRA_SCALE, -1) ?: -1
                
                if (level >= 0 && scale > 0) {
                    val batteryPct = (level * 100 / scale)
                    if (batteryPct != currentBatteryLevel) {
                        currentBatteryLevel = batteryPct
                        CrashLogger.d(TAG, "Battery: $batteryPct%")
                        
                        // Update Go engine
                        updateBatteryLevel(batteryPct)
                    }
                }
            }
        }

        val filter = IntentFilter(Intent.ACTION_BATTERY_CHANGED)
        registerReceiver(batteryReceiver, filter)
    }

    private fun updateBatteryLevel(level: Int) {
        if (goEngine != null && vpnAndTunStarted) {
            try {
                goEngine!!.javaClass.getMethod("setBatteryLevel", Int::class.java)
                    .invoke(goEngine, level)
            } catch (_: Throwable) {}
        }
    }

    // ═══════════════════════════════════════════════════════════════
    // Connection Methods
    // ═══════════════════════════════════════════════════════════════

    private fun handleConnectLegacy(arguments: Any?, result: MethodChannel.Result) {
        CrashLogger.d(TAG, "=== handleConnectLegacy (deprecated) ===")
        
        val config = arguments as? String
        if (config == null) {
            result.error("NULL_CONFIG", "Config is null", null)
            return
        }

        if (goEngine == null) {
            result.error("NO_ENGINE", "Native engine not available", null)
            return
        }

        pendingConfig = config
        startVpnAndConnect(result)
    }

    private fun handleConnectWithConfig(arguments: Any?, result: MethodChannel.Result) {
        CrashLogger.d(TAG, "=== handleConnectWithConfig (v1.0.1) ===")

        val configJson = arguments as? String
        if (configJson == null) {
            result.error("NULL_CONFIG", "Config JSON is null", null)
            return
        }

        if (goEngine == null) {
            result.error("NO_ENGINE", "Native engine not available", null)
            return
        }

        CrashLogger.d(TAG, "  Config: ${configJson.take(200)}...")
        pendingConfig = configJson

        if (vpnAndTunStarted && GuarchService.isRunning) {
            CrashLogger.d(TAG, "  VPN/TUN already running — reconnecting...")
            reconnectGoEngine(result)
        } else {
            CrashLogger.d(TAG, "  Starting VPN + TUN + Go engine...")
            startVpnAndConnect(result)
        }
    }

    private fun reconnectGoEngine(result: MethodChannel.Result) {
        Thread {
            try {
                // Load config first
                val config = pendingConfig ?: return@Thread
                
                try {
                    goEngine!!.javaClass.getMethod("loadConfigJSON", String::class.java)
                        .invoke(goEngine, config)
                    CrashLogger.d(TAG, "  Config loaded ✅")
                } catch (e: Throwable) {
                    CrashLogger.e(TAG, "  loadConfigJSON failed", unwrapException(e))
                }

                // Connect
                val connectMethod = goEngine!!.javaClass.getMethod("connect")
                val success = connectMethod.invoke(goEngine) as? Boolean ?: false
                
                CrashLogger.d(TAG, "  Go reconnect: $success")
                
                runOnUiThread {
                    if (success) {
                        sendEvent("status", "connected")
                    }
                    result.success(success)
                }
            } catch (e: Throwable) {
                val real = unwrapException(e)
                CrashLogger.e(TAG, "  Reconnect FAILED", real)
                runOnUiThread {
                    sendEvent("error", real.message ?: "Reconnect failed")
                    result.success(false)
                }
            }
        }.start()
    }

    private fun startVpnAndConnect(result: MethodChannel.Result) {
        CrashLogger.d(TAG, "--- startVpnAndConnect ---")
        
        try {
            val intent = VpnService.prepare(this)
            if (intent != null) {
                CrashLogger.d(TAG, "  Requesting VPN permission...")
                vpnPermissionResult = result
                startActivityForResult(intent, VPN_REQUEST_CODE)
            } else {
                CrashLogger.d(TAG, "  VPN permission granted")
                startVpnService(result)
            }
        } catch (e: Throwable) {
            CrashLogger.e(TAG, "  startVpnAndConnect CRASHED", e)
            sendEvent("error", e.message ?: "VPN start failed")
            result.success(false)
        }
    }

    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        super.onActivityResult(requestCode, resultCode, data)
        
        CrashLogger.d(TAG, "onActivityResult: req=$requestCode res=$resultCode")
        
        if (requestCode == VPN_REQUEST_CODE) {
            if (resultCode == Activity.RESULT_OK) {
                vpnPermissionResult?.let { startVpnService(it) }
            } else {
                CrashLogger.w(TAG, "  VPN permission DENIED")
                vpnPermissionResult?.success(false)
                sendEvent("error", "VPN permission denied")
            }
            vpnPermissionResult = null
        }
    }

    private fun startVpnService(result: MethodChannel.Result) {
        CrashLogger.d(TAG, "=== startVpnService ===")
        
        try {
            // Start VPN service
            val serviceIntent = Intent(this, GuarchService::class.java).apply {
                action = GuarchService.ACTION_START
                putExtra("socks_port", 1080)
            }

            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                startForegroundService(serviceIntent)
            } else {
                startService(serviceIntent)
            }

            CrashLogger.d(TAG, "  VPN service started")
            sendEvent("status", "connecting")

            // Background thread: Wait for fd + Connect Go + Start TUN
            Thread {
                try {
                    // Wait for TUN fd
                    CrashLogger.d(TAG, "  Waiting for TUN fd...")
                    var attempts = 0
                    while (GuarchService.tunFd < 0 && attempts < 50) {
                        Thread.sleep(100)
                        attempts++
                    }

                    val fd = GuarchService.tunFd
                    CrashLogger.d(TAG, "  TUN fd: $fd (attempts: $attempts)")

                    if (fd < 0) {
                        CrashLogger.e(TAG, "  No TUN fd!")
                        runOnUiThread {
                            sendEvent("error", "Failed to create TUN interface")
                            result.success(false)
                        }
                        return@Thread
                    }

                    // Return success immediately (NPV-style)
                    runOnUiThread {
                        result.success(true)
                    }

                    // Load config
                    val config = pendingConfig
                    if (config != null && goEngine != null) {
                        try {
                            goEngine!!.javaClass.getMethod("loadConfigJSON", String::class.java)
                                .invoke(goEngine, config)
                            CrashLogger.d(TAG, "  Config loaded")
                        } catch (e: Throwable) {
                            CrashLogger.e(TAG, "  Config load failed", unwrapException(e))
                        }

                        // Connect Go engine
                        try {
                            val connectMethod = goEngine!!.javaClass.getMethod("connect")
                            val success = connectMethod.invoke(goEngine) as? Boolean ?: false
                            CrashLogger.d(TAG, "  Go connect: $success")

                            if (!success) {
                                sendEvent("error", "Go engine connection failed")
                            }
                        } catch (e: Throwable) {
                            CrashLogger.e(TAG, "  Go connect FAILED", unwrapException(e))
                            sendEvent("error", "Go engine error: ${e.message}")
                        }
                    }

                    // Start TUN (only once)
                    if (goEngine != null && !vpnAndTunStarted) {
                        try {
                            CrashLogger.d(TAG, "  Starting TUN (fd=$fd, port=1080)...")
                            val startTunMethod = goEngine!!.javaClass.getMethod(
                                "startTun",
                                Int::class.java,
                                Int::class.java
                            )
                            startTunMethod.invoke(goEngine, fd, 1080)
                            
                            vpnAndTunStarted = true
                            CrashLogger.d(TAG, "  TUN started ✅")
                        } catch (e: Throwable) {
                            CrashLogger.e(TAG, "  TUN start FAILED", unwrapException(e))
                            sendEvent("error", "TUN start failed: ${e.message}")
                        }
                    }

                    CrashLogger.d(TAG, "=== Setup complete ===")

                } catch (e: Throwable) {
                    CrashLogger.e(TAG, "  Background thread CRASHED", e)
                    sendEvent("error", "Setup failed: ${e.message}")
                }
            }.start()

        } catch (e: Throwable) {
            CrashLogger.e(TAG, "  startVpnService CRASHED", e)
            sendEvent("error", e.message ?: "VPN service failed")
            result.success(false)
        }
    }

    // ═══════════════════════════════════════════════════════════════
    // Disconnect
    // ═══════════════════════════════════════════════════════════════

    private fun handleDisconnect(result: MethodChannel.Result) {
        CrashLogger.d(TAG, "=== handleDisconnect ===")
        
        Thread {
            try {
                // Disconnect Go engine
                if (goEngine != null) {
                    try {
                        goEngine!!.javaClass.getMethod("disconnect").invoke(goEngine)
                        CrashLogger.d(TAG, "  Go disconnect ✅")
                    } catch (e: Throwable) {
                        CrashLogger.e(TAG, "  Go disconnect error", unwrapException(e))
                    }

                    // Stop TUN
                    try {
                        goEngine!!.javaClass.getMethod("stopTun").invoke(goEngine)
                        CrashLogger.d(TAG, "  TUN stopped ✅")
                    } catch (e: Throwable) {
                        CrashLogger.e(TAG, "  TUN stop error", unwrapException(e))
                    }
                }

                // Stop VPN service
                try {
                    startService(Intent(this@MainActivity, GuarchService::class.java).apply {
                        action = GuarchService.ACTION_STOP
                    })
                    CrashLogger.d(TAG, "  VPN service stopped ✅")
                } catch (e: Throwable) {
                    CrashLogger.e(TAG, "  VPN stop error", e)
                }

                vpnAndTunStarted = false
                sendEvent("status", "disconnected")

            } catch (e: Throwable) {
                CrashLogger.e(TAG, "  Disconnect CRASHED", e)
            }

            runOnUiThread {
                result.success(true)
            }
        }.start()
    }

    // ═══════════════════════════════════════════════════════════════
    // Config Methods (v1.0.1)
    // ═══════════════════════════════════════════════════════════════

    private fun handleLoadConfigJSON(arguments: Any?, result: MethodChannel.Result) {
        val json = arguments as? String
        if (json == null || goEngine == null) {
            result.success(false)
            return
        }

        try {
            goEngine!!.javaClass.getMethod("loadConfigJSON", String::class.java)
                .invoke(goEngine, json)
            result.success(true)
        } catch (e: Throwable) {
            CrashLogger.e(TAG, "loadConfigJSON failed", unwrapException(e))
            result.success(false)
        }
    }

    private fun handleLoadConfigURI(arguments: Any?, result: MethodChannel.Result) {
        val uri = arguments as? String
        if (uri == null || goEngine == null) {
            result.success(false)
            return
        }

        try {
            goEngine!!.javaClass.getMethod("loadConfigURI", String::class.java)
                .invoke(goEngine, uri)
            result.success(true)
        } catch (e: Throwable) {
            CrashLogger.e(TAG, "loadConfigURI failed", unwrapException(e))
            result.success(false)
        }
    }

    private fun handleLoadPreset(arguments: Any?, result: MethodChannel.Result) {
        val preset = arguments as? String
        if (preset == null || goEngine == null) {
            result.success(false)
            return
        }

        try {
            goEngine!!.javaClass.getMethod("loadPreset", String::class.java)
                .invoke(goEngine, preset)
            result.success(true)
        } catch (e: Throwable) {
            CrashLogger.e(TAG, "loadPreset failed", unwrapException(e))
            result.success(false)
        }
    }

    private fun handleExportConfigURI(result: MethodChannel.Result) {
        if (goEngine == null) {
            result.success(null)
            return
        }

        try {
            val uri = goEngine!!.javaClass.getMethod("exportConfigURI")
                .invoke(goEngine) as? String
            result.success(uri)
        } catch (e: Throwable) {
            CrashLogger.e(TAG, "exportConfigURI failed", unwrapException(e))
            result.success(null)
        }
    }

    private fun handleExportConfigJSON(result: MethodChannel.Result) {
        if (goEngine == null) {
            result.success(null)
            return
        }

        try {
            val json = goEngine!!.javaClass.getMethod("exportConfigJSON")
                .invoke(goEngine) as? String
            result.success(json)
        } catch (e: Throwable) {
            CrashLogger.e(TAG, "exportConfigJSON failed", unwrapException(e))
            result.success(null)
        }
    }

    // ═══════════════════════════════════════════════════════════════
    // Battery & Data Saver
    // ═══════════════════════════════════════════════════════════════

    private fun handleSetBatteryLevel(arguments: Any?, result: MethodChannel.Result) {
        val level = arguments as? Int
        if (level == null || goEngine == null) {
            result.success(false)
            return
        }

        try {
            goEngine!!.javaClass.getMethod("setBatteryLevel", Int::class.java)
                .invoke(goEngine, level)
            currentBatteryLevel = level
            result.success(true)
        } catch (e: Throwable) {
            CrashLogger.e(TAG, "setBatteryLevel failed", unwrapException(e))
            result.success(false)
        }
    }

    private fun handleSetDataSaverMode(arguments: Any?, result: MethodChannel.Result) {
        val enabled = arguments as? Boolean
        if (enabled == null || goEngine == null) {
            result.success(false)
            return
        }

        try {
            goEngine!!.javaClass.getMethod("setDataSaverMode", Boolean::class.java)
                .invoke(goEngine, enabled)
            result.success(true)
        } catch (e: Throwable) {
            CrashLogger.e(TAG, "setDataSaverMode failed", unwrapException(e))
            result.success(false)
        }
    }

    // ═══════════════════════════════════════════════════════════════
    // Split Tunneling
    // ═══════════════════════════════════════════════════════════════

    private fun handleSetSplitTunnelMode(arguments: Any?, result: MethodChannel.Result) {
        val mode = arguments as? String
        if (mode == null || goEngine == null) {
            result.success(false)
            return
        }

        try {
            goEngine!!.javaClass.getMethod("setSplitTunnelMode", String::class.java)
                .invoke(goEngine, mode)
            result.success(true)
        } catch (e: Throwable) {
            CrashLogger.e(TAG, "setSplitTunnelMode failed", unwrapException(e))
            result.success(false)
        }
    }

    private fun handleAddSplitTunnelDomain(arguments: Any?, result: MethodChannel.Result) {
        @Suppress("UNCHECKED_CAST")
        val params = arguments as? Map<String, Any>
        if (params == null || goEngine == null) {
            result.success(false)
            return
        }

        val domain = params["domain"] as? String
        val isWhitelist = params["isWhitelist"] as? Boolean

        if (domain == null || isWhitelist == null) {
            result.success(false)
            return
        }

        try {
            goEngine!!.javaClass.getMethod(
                "addSplitTunnelDomain",
                String::class.java,
                Boolean::class.java
            ).invoke(goEngine, domain, isWhitelist)
            result.success(true)
        } catch (e: Throwable) {
            CrashLogger.e(TAG, "addSplitTunnelDomain failed", unwrapException(e))
            result.success(false)
        }
    }

    // ═══════════════════════════════════════════════════════════════
    // Stats & Status
    // ═══════════════════════════════════════════════════════════════

    private fun handleGetStatus(result: MethodChannel.Result) {
        try {
            if (goEngine != null) {
                val status = goEngine!!.javaClass.getMethod("getStatus")
                    .invoke(goEngine) as? String
                result.success(status ?: "disconnected")
            } else {
                result.success(if (GuarchService.isRunning) "connected" else "disconnected")
            }
        } catch (_: Throwable) {
            result.success("disconnected")
        }
    }

    private fun handleGetStats(result: MethodChannel.Result) {
        try {
            if (goEngine != null) {
                val stats = goEngine!!.javaClass.getMethod("getStats")
                    .invoke(goEngine) as? String
                result.success(stats ?: "{}")
            } else {
                result.success("{}")
            }
        } catch (_: Throwable) {
            result.success("{}")
        }
    }

    private fun handleGetTUNStats(result: MethodChannel.Result) {
        try {
            if (goEngine != null) {
                val stats = goEngine!!.javaClass.getMethod("getTUNStats")
                    .invoke(goEngine) as? String
                result.success(stats ?: "{}")
            } else {
                result.success("{}")
            }
        } catch (_: Throwable) {
            result.success("{}")
        }
    }

    // ═══════════════════════════════════════════════════════════════
    // Ping & Delay Testing (v1.0.1)
    // ═══════════════════════════════════════════════════════════════

    private fun handleTestRealDelay(arguments: Any?, result: MethodChannel.Result) {
        CrashLogger.d(TAG, "=== testRealDelay ===")
        
        val configJson = arguments as? String
        if (configJson == null) {
            CrashLogger.w(TAG, "  Config is null")
            result.success(false)
            return
        }

        if (goEngine == null) {
            CrashLogger.w(TAG, "  Go engine not available")
            result.success(false)
            return
        }

        Thread {
            try {
                CrashLogger.d(TAG, "  Loading test config...")
                
                // Load config
                goEngine!!.javaClass.getMethod("loadConfigJSON", String::class.java)
                    .invoke(goEngine, configJson)
                
                CrashLogger.d(TAG, "  Config loaded, connecting...")
                
                // Connect (without starting VPN service)
                val connectMethod = goEngine!!.javaClass.getMethod("connect")
                val success = connectMethod.invoke(goEngine) as? Boolean ?: false
                
                CrashLogger.d(TAG, "  Connection result: $success")
                
                // Disconnect immediately after handshake
                if (success) {
                    Thread.sleep(100) // Wait for handshake to complete
                    
                    try {
                        goEngine!!.javaClass.getMethod("disconnect").invoke(goEngine)
                        CrashLogger.d(TAG, "  Test connection disconnected")
                    } catch (e: Throwable) {
                        CrashLogger.e(TAG, "  Disconnect error", unwrapException(e))
                    }
                }
                
                runOnUiThread {
                    result.success(success)
                }
                
            } catch (e: Throwable) {
                val real = unwrapException(e)
                CrashLogger.e(TAG, "  Real delay test failed", real)
                runOnUiThread {
                    result.success(false)
                }
            }
        }.start()
    }

    private fun handleTestConnection(arguments: Any?, result: MethodChannel.Result) {
        CrashLogger.d(TAG, "=== testConnection ===")
        
        val configJson = arguments as? String
        if (configJson == null) {
            result.success(false)
            return
        }

        if (goEngine == null) {
            result.success(false)
            return
        }

        Thread {
            try {
                // Load minimal config
                goEngine!!.javaClass.getMethod("loadConfigJSON", String::class.java)
                    .invoke(goEngine, configJson)
                
                // Try to connect
                val connectMethod = goEngine!!.javaClass.getMethod("connect")
                val success = connectMethod.invoke(goEngine) as? Boolean ?: false
                
                // Disconnect
                if (success) {
                    Thread.sleep(50)
                    try {
                        goEngine!!.javaClass.getMethod("disconnect").invoke(goEngine)
                    } catch (_: Throwable) {}
                }
                
                runOnUiThread {
                    result.success(success)
                }
                
            } catch (e: Throwable) {
                CrashLogger.e(TAG, "testConnection failed", unwrapException(e))
                runOnUiThread {
                    result.success(false)
                }
            }
        }.start()
    }

    // ═══════════════════════════════════════════════════════════════
    // Helpers
    // ═══════════════════════════════════════════════════════════════

    private fun requestVpnPermission(result: MethodChannel.Result) {
        startVpnAndConnect(result)
    }

    private fun tryInitGoEngine() {
        CrashLogger.d(TAG, "--- tryInitGoEngine ---")
        try {
            val cls = Class.forName("com.guarch.mobile.Mobile")
            goEngine = cls.getMethod("new_").invoke(null)
            
            // Set callback for Go engine events
            try {
                val callbackClass = Class.forName("com.guarch.mobile.Callback")
                val callback = java.lang.reflect.Proxy.newProxyInstance(
                    callbackClass.classLoader,
                    arrayOf(callbackClass)
                ) { _, method, args ->
                    when (method.name) {
                        "onStatusChanged" -> {
                            val status = args?.get(0) as? String
                            if (status != null) {
                                runOnUiThread { sendEvent("status", status) }
                            }
                        }
                        "onStatsUpdate" -> {
                            val stats = args?.get(0) as? String
                            if (stats != null) {
                                runOnUiThread { sendEvent("stats", stats) }
                            }
                        }
                        "onLog" -> {
                            val log = args?.get(0) as? String
                            if (log != null) {
                                CrashLogger.d("GoEngine", log)
                                runOnUiThread { sendEvent("log", log) }
                            }
                        }
                        "onError" -> {
                            val error = args?.get(0) as? String
                            if (error != null) {
                                runOnUiThread { sendEvent("error", error) }
                            }
                        }
                        "onSNIRotation" -> {
                            val sni = args?.get(0) as? String
                            if (sni != null) {
                                runOnUiThread { sendEvent("sni", sni) }
                            }
                        }
                        "onDNSFallback" -> {
                            val enabled = args?.get(0) as? Boolean
                            if (enabled != null) {
                                runOnUiThread { sendEvent("dns_fallback", enabled) }
                            }
                        }
                    }
                    null
                }

                goEngine!!.javaClass.getMethod("setCallback", callbackClass)
                    .invoke(goEngine, callback)
                CrashLogger.d(TAG, "  Go callback set ✅")
            } catch (e: Throwable) {
                CrashLogger.w(TAG, "  Callback setup failed (optional)", e)
            }

            CrashLogger.d(TAG, "  Go engine LOADED ✅")
            
            val methods = goEngine!!.javaClass.methods
                .map { it.name }
                .distinct()
                .sorted()
            CrashLogger.d(TAG, "  Methods: ${methods.joinToString(", ")}")
        } catch (e: ClassNotFoundException) {
            CrashLogger.w(TAG, "  mobile.Mobile NOT FOUND (gomobile binding missing)")
            goEngine = null
        } catch (e: Throwable) {
            CrashLogger.e(TAG, "  Go engine init FAILED", e)
            goEngine = null
        }
    }

    private fun sendEvent(type: String, data: Any?) {
        runOnUiThread {
            try {
                eventSink?.success(mapOf("type" to type, "data" to data))
            } catch (e: Throwable) {
                CrashLogger.e(TAG, "sendEvent failed", e)
            }
        }
    }

    private fun unwrapException(e: Throwable): Throwable {
        return if (e is java.lang.reflect.InvocationTargetException && e.cause != null) {
            e.cause!!
        } else {
            e
        }
    }

    private fun shareLogs() {
        try {
            val logFile = File(filesDir, "guarch_debug.log")
            if (!logFile.exists()) return

            val shareFile = File(cacheDir, "guarch_log_${System.currentTimeMillis()}.txt")
            logFile.copyTo(shareFile, overwrite = true)

            val uri = FileProvider.getUriForFile(
                this,
                "$packageName.fileprovider",
                shareFile
            )

            startActivity(Intent.createChooser(
                Intent(Intent.ACTION_SEND).apply {
                    type = "text/plain"
                    putExtra(Intent.EXTRA_STREAM, uri)
                    addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
                },
                "Share Guarch Logs"
            ))
        } catch (e: Exception) {
            CrashLogger.e(TAG, "Share logs failed", e)
        }
    }

    override fun onDestroy() {
        CrashLogger.d(TAG, "=== Activity onDestroy ===")
        
        try {
            batteryReceiver?.let { unregisterReceiver(it) }
        } catch (_: Throwable) {}
        
        CrashLogger.close()
        super.onDestroy()
    }
}
