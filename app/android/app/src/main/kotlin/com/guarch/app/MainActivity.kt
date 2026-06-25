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

        @Volatile
        private var vpnServiceRef: GuarchService? = null

        fun setVpnServiceReference(service: GuarchService?) {
            vpnServiceRef = service
            CrashLogger.d("MainActivity", "VPN service reference: ${service != null}")
        }
    }

    private var vpnPermissionResult: MethodChannel.Result? = null
    private var pendingConfig: String? = null
    private var pendingSocksPort: Int = 7070
    private var pendingIPv6Enabled: Boolean = false
    private var currentVpnMode: Boolean = true
    private var methodChannel: MethodChannel? = null
    private var eventSink: EventChannel.EventSink? = null
    private var goEngine: Any? = null
    private var batteryReceiver: BroadcastReceiver? = null

    private var pendingAllowedApps: List<String> = emptyList()
    private var pendingDisallowedApps: List<String> = emptyList()

    private var vpnAndTunStarted = false
    private var proxyOnlyStarted = false
    private var currentBatteryLevel = 100

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)

        CrashLogger.init(this)
        CrashLogger.d(TAG, "====== APP STARTED (v1.1.1) ======")
        CrashLogger.d(TAG, "SDK: ${Build.VERSION.SDK_INT} | Device: ${Build.MANUFACTURER} ${Build.MODEL}")

        tryInitGoEngine()
        setupProtectFunc()
        setupBatteryMonitoring()
        setupEventChannel(flutterEngine)
        setupMethodChannel(flutterEngine)
        setupLogChannel(flutterEngine)
        setupServiceCallback()

        CrashLogger.d(TAG, "configureFlutterEngine done")
    }

    private fun setupEventChannel(flutterEngine: FlutterEngine) {
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
    }

    private fun setupMethodChannel(flutterEngine: FlutterEngine) {
        methodChannel = MethodChannel(flutterEngine.dartExecutor.binaryMessenger, ENGINE_CHANNEL)
        methodChannel?.setMethodCallHandler { call, result ->
            CrashLogger.d(TAG, ">> Method: ${call.method}")
            try {
                when (call.method) {
                    "getVersion" -> result.success("Guarch Android v1.1.1")
                    "setUserSettings" -> handleSetUserSettings(call.arguments, result)
                    "connect" -> handleConnectLegacy(call.arguments, result)
                    "connectWithConfig" -> handleConnectWithConfig(call.arguments, result)
                    "disconnect" -> handleDisconnect(call.arguments, result)
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
                    "readGoLog" -> handleReadGoLog(result)
                    else -> result.notImplemented()
                }
            } catch (e: Throwable) {
                CrashLogger.e(TAG, "CRASH in ${call.method}", e)
                try {
                    result.error("CRASH", e.message ?: "Unknown error", null)
                } catch (_: Exception) {}
            }
        }
    }

    private fun setupLogChannel(flutterEngine: FlutterEngine) {
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, LOG_CHANNEL)
            .setMethodCallHandler { call, result ->
                when (call.method) {
                    "getLogs" -> result.success(CrashLogger.getCurrentLog(this))
                    "getCrashLog" -> result.success(CrashLogger.getPreviousCrashLog(this))
                    "getGoLog" -> {
                        try {
                            val logFile = File(filesDir, "go_engine.log")
                            result.success(if (logFile.exists()) logFile.readText() else "No Go log yet")
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
    }

    private fun setupServiceCallback() {
        GuarchService.setStatusCallback { status ->
            runOnUiThread {
                CrashLogger.d(TAG, "Service status callback: $status")
                sendEvent("vpn_service_event", status)
            }
        }
    }

    private fun setupProtectFunc() {
        if (goEngine == null) {
            CrashLogger.w(TAG, "⚠️ Cannot setup protect: Go engine not loaded")
            return
        }

        try {
            val protectClass = Class.forName("com.guarch.mobile.mobile.ProtectFunc")

            val protectImpl = java.lang.reflect.Proxy.newProxyInstance(
                protectClass.classLoader,
                arrayOf(protectClass)
            ) { _, method, args ->
                if (method.name == "protectFd") {
                    val fd = args?.get(0) as? Long ?: return@newProxyInstance false
                    val service = vpnServiceRef

                    if (service == null) {
                        CrashLogger.w(TAG, "⚠️ Protect called but VPN service not available (fd=$fd)")
                        false
                    } else {
                        try {
                            val success = service.protect(fd.toInt())
                            CrashLogger.d(TAG, "Protect fd=$fd -> $success")
                            success
                        } catch (e: Exception) {
                            CrashLogger.e(TAG, "❌ Protect error for fd=$fd", e)
                            false
                        }
                    }
                } else {
                    null
                }
            }

            val mobileClass = Class.forName("com.guarch.mobile.mobile.Mobile")
            mobileClass.getMethod("setProtectFunc", protectClass)
                .invoke(null, protectImpl)

            CrashLogger.d(TAG, "✅ Protect function registered")
        } catch (e: ClassNotFoundException) {
            CrashLogger.e(TAG, "❌ Mobile class not found - rebuild AAR", e)
        } catch (e: NoSuchMethodException) {
            CrashLogger.e(TAG, "❌ setProtectFunc method not found - rebuild AAR", e)
        } catch (e: Throwable) {
            CrashLogger.w(TAG, "⚠️ Protect setup failed", e)
        }
    }

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
                        updateBatteryLevel(batteryPct)
                    }
                }
            }
        }

        val filter = IntentFilter(Intent.ACTION_BATTERY_CHANGED)
        registerReceiver(batteryReceiver, filter)
    }

    private fun updateBatteryLevel(level: Int) {
        if (goEngine != null && (vpnAndTunStarted || proxyOnlyStarted)) {
            try {
                goEngine!!.javaClass.getMethod("setBatteryLevel", Int::class.java)
                    .invoke(goEngine, level)
            } catch (_: Throwable) {}
        }
    }

    private fun handleSetUserSettings(arguments: Any?, result: MethodChannel.Result) {
        CrashLogger.d(TAG, "=== handleSetUserSettings ===")

        val json = arguments as? String
        if (json == null) {
            CrashLogger.w(TAG, "  Settings JSON is null")
            result.success(false)
            return
        }

        if (goEngine == null) {
            CrashLogger.w(TAG, "  Go engine not available - storing for later")
            result.success(true)
            return
        }

        try {
            CrashLogger.d(TAG, "  Settings: $json")

            val method = goEngine!!.javaClass.getMethod("setUserSettings", String::class.java)
            val success = method.invoke(goEngine, json) as? Boolean ?: false

            CrashLogger.d(TAG, "  setUserSettings result: $success")
            result.success(success)

        } catch (e: Throwable) {
            val real = unwrapException(e)
            CrashLogger.e(TAG, "  setUserSettings failed", real)
            result.success(true)
        }
    }

    private fun handleReadGoLog(result: MethodChannel.Result) {
        CrashLogger.d(TAG, ">> Method: readGoLog")
        try {
            val logFile = File(filesDir, "go_engine.log")
            if (logFile.exists()) {
                result.success(logFile.readText())
            } else {
                result.success("No Go log file")
            }
        } catch (e: Throwable) {
            result.success("Error: ${e.message}")
        }
    }

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
        currentVpnMode = true
        startVpnAndConnect(result)
    }

    private fun handleConnectWithConfig(arguments: Any?, result: MethodChannel.Result) {
        CrashLogger.d(TAG, "=== handleConnectWithConfig (v1.1.1) ===")

        @Suppress("UNCHECKED_CAST")
        val params = arguments as? Map<String, Any>
        if (params == null) {
            result.error("NULL_PARAMS", "Parameters are null", null)
            return
        }

        val configJson = params["config"] as? String
        val vpnMode = params["vpnMode"] as? Boolean ?: true
        val preferIPv6 = params["preferIPv6"] as? Boolean ?: false
        pendingAllowedApps = params["allowedApps"] as? List<String> ?: emptyList()
        pendingDisallowedApps = params["disallowedApps"] as? List<String> ?: emptyList()

        if (configJson == null) {
            result.error("NULL_CONFIG", "Config JSON is null", null)
            return
        }

        if (goEngine == null) {
            CrashLogger.e(TAG, "  Go engine is NULL!")
            result.error("NO_ENGINE", "Native engine not available", null)
            return
        }

        currentVpnMode = vpnMode
        val mode = if (vpnMode) "VPN" else "Proxy"

        val socksPort = try {
            val json = org.json.JSONObject(configJson)
            json.optInt("socks_port", 7070)
        } catch (e: Exception) {
            CrashLogger.w(TAG, "  Failed to parse socks_port, using default 7070")
            7070
        }

        CrashLogger.d(TAG, "  Mode: $mode")
        CrashLogger.d(TAG, "  Config: ${configJson.take(200)}...")
        CrashLogger.d(TAG, "  SOCKS5 Port: $socksPort")
        CrashLogger.d(TAG, "  Prefer IPv6: $preferIPv6")

        pendingConfig = configJson
        pendingSocksPort = socksPort
        pendingIPv6Enabled = preferIPv6

        if (vpnMode) {
            if (vpnAndTunStarted && GuarchService.isRunning) {
                CrashLogger.d(TAG, "  VPN/TUN already running — reconnecting...")
                reconnectGoEngine(result)
            } else {
                CrashLogger.d(TAG, "  Starting VPN + TUN + Go engine...")
                startVpnAndConnect(result)
            }
        } else {
            if (proxyOnlyStarted) {
                CrashLogger.d(TAG, "  Proxy already running — reconnecting...")
                reconnectGoEngine(result)
            } else {
                CrashLogger.d(TAG, "  Starting Proxy-only mode (SOCKS5 on :$socksPort)...")
                startProxyOnly(result)
            }
        }
    }

    private fun startProxyOnly(result: MethodChannel.Result) {
        CrashLogger.d(TAG, "=== startProxyOnly ===")

        Thread {
            try {
                val config = pendingConfig
                if (config != null && goEngine != null) {
                    CrashLogger.d(TAG, "  Loading config to Go engine...")
                    try {
                        goEngine!!.javaClass.getMethod("loadConfigJSON", String::class.java)
                            .invoke(goEngine, config)
                        CrashLogger.d(TAG, "  Config loaded ✅")
                    } catch (e: Throwable) {
                        CrashLogger.e(TAG, "  Config load failed", unwrapException(e))
                        runOnUiThread {
                            sendEvent("error", "Config load failed")
                            result.success(false)
                        }
                        return@Thread
                    }

                    CrashLogger.d(TAG, "  Starting SOCKS5 proxy on port $pendingSocksPort...")
                    try {
                        val startProxyMethod = goEngine!!.javaClass.getMethod(
                            "startProxyOnly",
                            Int::class.java
                        )
                        val started = startProxyMethod.invoke(goEngine, pendingSocksPort) as? Boolean ?: false

                        if (!started) {
                            CrashLogger.e(TAG, "  startProxyOnly() returned false")
                            runOnUiThread {
                                sendEvent("error", "Proxy start failed")
                                result.success(false)
                            }
                            return@Thread
                        }

                        CrashLogger.d(TAG, "  Proxy started successfully ✅")
                        proxyOnlyStarted = true

                        runOnUiThread {
                            sendEvent("status", "connected")
                            result.success(true)
                        }

                    } catch (e: Throwable) {
                        val real = unwrapException(e)
                        CrashLogger.e(TAG, "  Proxy start failed", real)
                        runOnUiThread {
                            sendEvent("error", "Proxy error: ${real.message}")
                            result.success(false)
                        }
                    }
                } else {
                    runOnUiThread {
                        result.success(false)
                    }
                }
            } catch (e: Throwable) {
                CrashLogger.e(TAG, "  startProxyOnly crashed", e)
                runOnUiThread {
                    sendEvent("error", "Setup failed: ${e.message}")
                    result.success(false)
                }
            }
        }.start()
    }

    private fun reconnectGoEngine(result: MethodChannel.Result) {
        Thread {
            try {
                val config = pendingConfig ?: return@Thread

                CrashLogger.d(TAG, "  Loading new config...")
                try {
                    goEngine!!.javaClass.getMethod("loadConfigJSON", String::class.java)
                        .invoke(goEngine, config)
                    CrashLogger.d(TAG, "  Config loaded ✅")
                } catch (e: Throwable) {
                    CrashLogger.e(TAG, "  loadConfigJSON failed", unwrapException(e))
                }

                if (currentVpnMode) {
                    CrashLogger.d(TAG, "  Calling connect() (VPN mode)...")
                    val connectMethod = goEngine!!.javaClass.getMethod("connect")
                    val success = connectMethod.invoke(goEngine) as? Boolean ?: false

                    CrashLogger.d(TAG, "  Go reconnect: $success")

                    runOnUiThread {
                        if (success) {
                            sendEvent("status", "connected")
                        }
                        result.success(success)
                    }
                } else {
                    CrashLogger.d(TAG, "  Restarting proxy...")
                    val startProxyMethod = goEngine!!.javaClass.getMethod(
                        "startProxyOnly",
                        Int::class.java
                    )
                    val success = startProxyMethod.invoke(goEngine, pendingSocksPort) as? Boolean ?: false

                    runOnUiThread {
                        if (success) {
                            sendEvent("status", "connected")
                        }
                        result.success(success)
                    }
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
                CrashLogger.d(TAG, "  VPN permission granted")
                vpnPermissionResult?.let { startVpnService(it) }
                sendEvent("vpn_permission_granted", true)
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
            val serviceIntent = Intent(this, GuarchService::class.java).apply {
                action = GuarchService.ACTION_START
                putExtra("socks_port", pendingSocksPort)
                putExtra("enable_ipv6", pendingIPv6Enabled)
                putStringArrayListExtra("allowed_apps", ArrayList(pendingAllowedApps))
                putStringArrayListExtra("disallowed_apps", ArrayList(pendingDisallowedApps))
            }

            CrashLogger.d(TAG, "  Starting service with SOCKS port: $pendingSocksPort, Prefer IPv6: $pendingIPv6Enabled")

            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                startForegroundService(serviceIntent)
            } else {
                startService(serviceIntent)
            }

            CrashLogger.d(TAG, "  VPN service started")
            sendEvent("status", "connecting")

            Thread {
                try {
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

                    runOnUiThread {
                        result.success(true)
                    }

                    val config = pendingConfig
                    if (config != null && goEngine != null) {
                        CrashLogger.d(TAG, "  Loading config to Go engine...")
                        try {
                            goEngine!!.javaClass.getMethod("loadConfigJSON", String::class.java)
                                .invoke(goEngine, config)
                            CrashLogger.d(TAG, "  Config loaded ✅")
                        } catch (e: Throwable) {
                            CrashLogger.e(TAG, "  Config load failed", unwrapException(e))
                            runOnUiThread {
                                sendEvent("error", "Config load failed")
                            }
                            return@Thread
                        }

                        CrashLogger.d(TAG, "  Calling Go connect()...")
                        try {
                            val connectMethod = goEngine!!.javaClass.getMethod("connect")
                            val started = connectMethod.invoke(goEngine) as? Boolean ?: false

                            if (!started) {
                                CrashLogger.e(TAG, "  Connect() returned false")
                                runOnUiThread {
                                    sendEvent("error", "Connection start failed")
                                }
                                return@Thread
                            }

                            CrashLogger.d(TAG, "  Waiting for connection status...")

                            val getStatusMethod = goEngine!!.javaClass.getMethod("getStatus")
                            var statusAttempts = 0
                            val maxStatusAttempts = 300
                            var connected = false

                            while (statusAttempts < maxStatusAttempts) {
                                try {
                                    val status = getStatusMethod.invoke(goEngine) as? String ?: "disconnected"

                                    when (status) {
                                        "connected" -> {
                                            connected = true
                                            CrashLogger.d(TAG, "  Connected! (attempts: $statusAttempts)")
                                            break
                                        }
                                        "disconnected" -> {
                                            if (statusAttempts > 10) {
                                                CrashLogger.e(TAG, "  Connection failed (status: disconnected)")
                                                runOnUiThread {
                                                    sendEvent("error", "Connection failed")
                                                }
                                                return@Thread
                                            }
                                        }
                                        "connecting" -> {
                                        }
                                    }

                                    Thread.sleep(100)
                                    statusAttempts++
                                } catch (e: Throwable) {
                                    CrashLogger.e(TAG, "  Status check error", unwrapException(e))
                                    break
                                }
                            }

                            if (!connected) {
                                CrashLogger.e(TAG, "  Connection timeout (30s)")
                                runOnUiThread {
                                    sendEvent("error", "Connection timeout")
                                }

                                try {
                                    goEngine!!.javaClass.getMethod("disconnect").invoke(goEngine)
                                } catch (_: Throwable) {}

                                return@Thread
                            }

                            if (!vpnAndTunStarted) {
                                try {
                                    CrashLogger.d(TAG, "  Starting TUN (fd=$fd, port=$pendingSocksPort)...")

                                    val startTunMethod = goEngine!!.javaClass.getMethod(
                                        "startTun",
                                        Int::class.java,
                                        Int::class.java
                                    )
                                    startTunMethod.invoke(goEngine, fd, pendingSocksPort)

                                    vpnAndTunStarted = true
                                    CrashLogger.d(TAG, "  TUN started ✅")
                                } catch (e: Throwable) {
                                    CrashLogger.e(TAG, "  TUN start failed", unwrapException(e))
                                    runOnUiThread {
                                        sendEvent("error", "TUN start failed: ${e.message}")
                                    }

                                    try {
                                        goEngine!!.javaClass.getMethod("disconnect").invoke(goEngine)
                                    } catch (_: Throwable) {}

                                    return@Thread
                                }
                            }

                            CrashLogger.d(TAG, "=== Setup complete ✅ ===")

                        } catch (e: Throwable) {
                            val real = unwrapException(e)
                            CrashLogger.e(TAG, "  Go connect failed", real)
                            runOnUiThread {
                                sendEvent("error", "Connection error: ${real.message}")
                            }
                        }
                    }

                } catch (e: Throwable) {
                    CrashLogger.e(TAG, "  Background thread crashed", e)
                    runOnUiThread {
                        sendEvent("error", "Setup failed: ${e.message}")
                    }
                }
            }.start()

        } catch (e: Throwable) {
            CrashLogger.e(TAG, "  startVpnService crashed", e)
            sendEvent("error", e.message ?: "VPN service failed")
            result.success(false)
        }
    }

    private fun stopVpnServiceDueToFailure() {
        CrashLogger.d(TAG, "=== stopVpnServiceDueToFailure ===")

        try {
            if (vpnAndTunStarted) {
                CrashLogger.d(TAG, "  Stopping VPN service due to connection failure...")

                val intent = Intent(this, GuarchService::class.java).apply {
                    action = GuarchService.ACTION_STOP
                }
                startService(intent)

                vpnAndTunStarted = false
                CrashLogger.d(TAG, "  VPN service stopped ✅")
            }
        } catch (e: Throwable) {
            CrashLogger.e(TAG, "  Failed to stop VPN service", e)
        }
    }

    private fun updateVpnNotification(statsJson: String) {
        try {
            val json = org.json.JSONObject(statsJson)
            val upload = json.getLong("upload")
            val download = json.getLong("download")
            val upSpeed = json.optLong("upSpeed", 0)
            val downSpeed = json.optLong("downSpeed", 0)

            GuarchService.instance?.updateStatsFromGo(upload, download, upSpeed, downSpeed)
        } catch (e: Exception) {
            CrashLogger.e(TAG, "Failed to parse notification stats", e)
        }
    }

    private fun handleDisconnect(arguments: Any?, result: MethodChannel.Result) {
        CrashLogger.d(TAG, "=== handleDisconnect ===")

        val vpnMode = arguments as? Boolean ?: currentVpnMode
        CrashLogger.d(TAG, "  VPN Mode: $vpnMode")

        Thread {
            try {
                if (goEngine != null) {
                    if (vpnMode || vpnAndTunStarted) {
                        try {
                            CrashLogger.d(TAG, "  Calling Go disconnect()...")
                            goEngine!!.javaClass.getMethod("disconnect").invoke(goEngine)
                            CrashLogger.d(TAG, "  Go disconnect ✅")
                        } catch (e: Throwable) {
                            CrashLogger.e(TAG, "  Go disconnect error", unwrapException(e))
                        }

                        try {
                            CrashLogger.d(TAG, "  Stopping TUN...")
                            goEngine!!.javaClass.getMethod("stopTun").invoke(goEngine)
                            CrashLogger.d(TAG, "  TUN stopped ✅")
                        } catch (e: Throwable) {
                            CrashLogger.e(TAG, "  TUN stop error", unwrapException(e))
                        }

                        try {
                            CrashLogger.d(TAG, "  Stopping VPN service...")
                            startService(Intent(this@MainActivity, GuarchService::class.java).apply {
                                action = GuarchService.ACTION_STOP
                            })
                            CrashLogger.d(TAG, "  VPN service stopped ✅")
                        } catch (e: Throwable) {
                            CrashLogger.e(TAG, "  VPN stop error", e)
                        }

                        vpnAndTunStarted = false
                    } else {
                        try {
                            CrashLogger.d(TAG, "  Stopping proxy-only mode...")
                            goEngine!!.javaClass.getMethod("stopProxyOnly").invoke(goEngine)
                            CrashLogger.d(TAG, "  Proxy stopped ✅")
                        } catch (e: Throwable) {
                            CrashLogger.e(TAG, "  Proxy stop error", unwrapException(e))
                        }

                        proxyOnlyStarted = false
                    }
                }

                sendEvent("status", "disconnected")

            } catch (e: Throwable) {
                CrashLogger.e(TAG, "  Disconnect CRASHED", e)
            }

            runOnUiThread {
                result.success(true)
            }
        }.start()
    }

    private fun handleLoadConfigJSON(arguments: Any?, result: MethodChannel.Result) {
        val json = arguments as? String
        if (json == null || goEngine == null) {
            result.success(false)
            return
        }

        try {
            goEngine!!.javaClass.getMethod("loadConfigJSON", String::class.java)
                .invoke(goEngine, json)
            CrashLogger.d(TAG, "loadConfigJSON success")
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
            CrashLogger.d(TAG, "loadConfigURI success")
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
            CrashLogger.d(TAG, "loadPreset success: $preset")
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
            CrashLogger.d(TAG, "Battery level set: $level%")
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
            CrashLogger.d(TAG, "Data saver mode: $enabled")
            result.success(true)
        } catch (e: Throwable) {
            CrashLogger.e(TAG, "setDataSaverMode failed", unwrapException(e))
            result.success(false)
        }
    }

    private fun handleSetSplitTunnelMode(arguments: Any?, result: MethodChannel.Result) {
        val mode = arguments as? String
        if (mode == null || goEngine == null) {
            result.success(false)
            return
        }

        try {
            goEngine!!.javaClass.getMethod("setSplitTunnelMode", String::class.java)
                .invoke(goEngine, mode)
            CrashLogger.d(TAG, "Split tunnel mode: $mode")
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
            CrashLogger.d(TAG, "Split tunnel domain added: $domain (whitelist=$isWhitelist)")
            result.success(true)
        } catch (e: Throwable) {
            CrashLogger.e(TAG, "addSplitTunnelDomain failed", unwrapException(e))
            result.success(false)
        }
    }

    private fun handleGetStatus(result: MethodChannel.Result) {
        try {
            if (goEngine != null) {
                val status = goEngine!!.javaClass.getMethod("getStatus")
                    .invoke(goEngine) as? String
                result.success(status ?: "disconnected")
            } else {
                val isRunning = if (currentVpnMode) GuarchService.isRunning else proxyOnlyStarted
                result.success(if (isRunning) "connected" else "disconnected")
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
            val stats = mutableMapOf<String, Any>(
                "running" to GuarchService.isRunning,
                "tun_fd" to GuarchService.tunFd
            )

            if (goEngine != null) {
                try {
                    val tunStats = goEngine!!.javaClass.getMethod("getTUNStats")
                        .invoke(goEngine) as? String
                    if (tunStats != null && tunStats != "{}") {
                        stats["go_stats"] = tunStats
                    }
                } catch (_: Throwable) {}
            }

            result.success(stats)
        } catch (_: Throwable) {
            result.success(mapOf(
                "running" to false,
                "tun_fd" to -1
            ))
        }
    }

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
            var testSuccess = false
            var previousConfig: String? = null

            try {
                CrashLogger.d(TAG, "  Extracting server info...")
                val serverAddress = try {
                    val json = org.json.JSONObject(configJson)
                    json.getJSONObject("server").getString("address")
                } catch (e: Exception) {
                    "unknown"
                }

                CrashLogger.d(TAG, "  Testing server: $serverAddress")

                try {
                    CrashLogger.d(TAG, "  Saving current config...")
                    previousConfig = goEngine!!.javaClass.getMethod("exportConfigJSON")
                        .invoke(goEngine) as? String
                    if (previousConfig != null) {
                        CrashLogger.d(TAG, "  Previous config saved (${previousConfig!!.length} chars)")
                    }
                } catch (e: Throwable) {
                    CrashLogger.w(TAG, "  Could not save previous config (OK if first run)")
                }

                CrashLogger.d(TAG, "  Loading test config...")
                goEngine!!.javaClass.getMethod("loadConfigJSON", String::class.java)
                    .invoke(goEngine, configJson)

                CrashLogger.d(TAG, "  Connecting to test server...")
                val startTime = System.currentTimeMillis()
                val connectMethod = goEngine!!.javaClass.getMethod("connect")
                val success = connectMethod.invoke(goEngine) as? Boolean ?: false
                val elapsed = System.currentTimeMillis() - startTime

                CrashLogger.d(TAG, "  Connection result: $success (${elapsed}ms)")

                if (success) {
                    testSuccess = true
                    Thread.sleep(100)

                    try {
                        CrashLogger.d(TAG, "  Disconnecting test connection...")
                        goEngine!!.javaClass.getMethod("disconnect").invoke(goEngine)
                        CrashLogger.d(TAG, "  Test connection disconnected")
                    } catch (e: Throwable) {
                        CrashLogger.e(TAG, "  Disconnect error", unwrapException(e))
                    }
                }

            } catch (e: Throwable) {
                val real = unwrapException(e)
                CrashLogger.e(TAG, "  Test failed: ${real.message}", real)
            } finally {
                if (previousConfig != null) {
                    try {
                        CrashLogger.d(TAG, "  Restoring previous config...")
                        goEngine!!.javaClass.getMethod("loadConfigJSON", String::class.java)
                            .invoke(goEngine, previousConfig)
                        CrashLogger.d(TAG, "  Previous config restored ✅")
                    } catch (e: Throwable) {
                        CrashLogger.e(TAG, "  Could not restore previous config", unwrapException(e))
                    }
                }
            }

            val engineStillAlive = goEngine != null
            CrashLogger.d(TAG, "  After test: engine=${if (engineStillAlive) "ALIVE ✅" else "NULL ❌"}")

            runOnUiThread {
                result.success(testSuccess)
            }
        }.start()
    }

    private fun handleTestConnection(arguments: Any?, result: MethodChannel.Result) {
        CrashLogger.d(TAG, "=== testConnection ===")

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
                goEngine!!.javaClass.getMethod("loadConfigJSON", String::class.java)
                    .invoke(goEngine, configJson)

                CrashLogger.d(TAG, "  Attempting connection...")
                val connectMethod = goEngine!!.javaClass.getMethod("connect")
                val success = connectMethod.invoke(goEngine) as? Boolean ?: false

                CrashLogger.d(TAG, "  Test result: $success")

                if (success) {
                    Thread.sleep(50)
                    try {
                        goEngine!!.javaClass.getMethod("disconnect").invoke(goEngine)
                        CrashLogger.d(TAG, "  Disconnected")
                    } catch (_: Throwable) {}
                }

                runOnUiThread {
                    result.success(success)
                }

            } catch (e: Throwable) {
                CrashLogger.e(TAG, "  Test connection failed", unwrapException(e))
                runOnUiThread {
                    result.success(false)
                }
            }
        }.start()
    }

    private fun requestVpnPermission(result: MethodChannel.Result) {
        currentVpnMode = true
        startVpnAndConnect(result)
    }

    private fun tryInitGoEngine() {
        CrashLogger.d(TAG, "--- tryInitGoEngine ---")
        try {
            CrashLogger.d(TAG, "  Loading Mobile class...")
            val cls = Class.forName("com.guarch.mobile.mobile.Mobile")

            CrashLogger.d(TAG, "  Creating new engine instance...")
            goEngine = cls.getMethod("new_").invoke(null)

            try {
                val logPath = File(filesDir, "go_engine.log").absolutePath
                CrashLogger.d(TAG, "  Initializing Go log: $logPath")
                val initLogMethod = cls.getMethod("initGoLog", String::class.java)
                initLogMethod.invoke(null, logPath)
                CrashLogger.d(TAG, "  Go log initialized ✅")
            } catch (e: Throwable) {
                CrashLogger.w(TAG, "  Go log init failed (method may not exist yet)")
            }

            try {
                CrashLogger.d(TAG, "  Setting up callback...")
                val callbackClass = Class.forName("com.guarch.mobile.mobile.Callback")
                val callback = java.lang.reflect.Proxy.newProxyInstance(
                    callbackClass.classLoader,
                    arrayOf(callbackClass)
                ) { _, method, args ->
                    when (method.name) {
                        "onStatusChanged" -> {
                            val status = args?.get(0) as? String
                            if (status != null) {
                                CrashLogger.d("GoCallback", "Status: $status")
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
                            val level = args?.get(0) as? String
                            val msg = args?.get(1) as? String
                            if (level != null && msg != null) {
                                if (level == "debug" && msg == "heartbeat") {
                                    runOnUiThread { sendEvent("heartbeat", null) }
                                } else if (level == "notification_update") {
                                    runOnUiThread {
                                        updateVpnNotification(msg)
                                    }
                                } else {
                                    CrashLogger.d("GoEngine", "[$level] $msg")
                                    runOnUiThread { sendEvent("log", msg) }
                                }
                            }
                        }
                        "onError" -> {
                            val error = args?.get(0) as? String
                            if (error != null) {
                                if (error == "connection_lost_stop_vpn") {
                                    CrashLogger.w("GoEngine", "Connection lost - stopping VPN service")
                                    runOnUiThread {
                                        stopVpnServiceDueToFailure()
                                        sendEvent("error", "Connection lost - VPN stopped")
                                    }
                                } else {
                                    CrashLogger.e("GoEngine", "Error: $error")
                                    runOnUiThread { sendEvent("error", error) }
                                }
                            }
                        }
                        "onSNIRotation" -> {
                            val sni = args?.get(0) as? String
                            if (sni != null) {
                                CrashLogger.d("GoEngine", "SNI rotated: $sni")
                                runOnUiThread { sendEvent("sni", sni) }
                            }
                        }
                        "onDNSFallback" -> {
                            val enabled = args?.get(0) as? Boolean
                            if (enabled != null) {
                                CrashLogger.d("GoEngine", "DNS fallback: $enabled")
                                runOnUiThread { sendEvent("dns_fallback", enabled) }
                            }
                        }
                    }
                    null
                }

                goEngine!!.javaClass.getMethod("setCallback", callbackClass)
                    .invoke(goEngine, callback)
                CrashLogger.d(TAG, "  Callback set ✅")
            } catch (e: Throwable) {
                CrashLogger.w(TAG, "  Callback setup failed", e)
            }

            CrashLogger.d(TAG, "  Go engine LOADED ✅")

            val methods = goEngine!!.javaClass.methods
                .map { it.name }
                .distinct()
                .sorted()
            CrashLogger.d(TAG, "  Available methods (${methods.size}): ${methods.joinToString(", ")}")
        } catch (e: ClassNotFoundException) {
            CrashLogger.e(TAG, "  com.guarch.mobile.mobile.Mobile NOT FOUND")
            CrashLogger.e(TAG, "  AAR file missing or not packaged correctly")
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
                CrashLogger.e(TAG, "sendEvent failed: $type", e)
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

        GuarchService.clearStatusCallback()

        try {
            batteryReceiver?.let {
                unregisterReceiver(it)
                CrashLogger.d(TAG, "  Battery receiver unregistered")
            }
        } catch (_: Throwable) {}

        CrashLogger.close()
        super.onDestroy()
    }
}
