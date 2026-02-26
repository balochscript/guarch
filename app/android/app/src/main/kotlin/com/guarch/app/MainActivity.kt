package com.guarch.app

import android.app.Activity
import android.content.Intent
import android.net.VpnService
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.EventChannel
import io.flutter.plugin.common.MethodChannel

class MainActivity : FlutterActivity() {

    companion object {
        const val ENGINE_CHANNEL = "com.guarch.app/engine"
        const val EVENT_CHANNEL = "com.guarch.app/events"
        const val VPN_REQUEST_CODE = 1001
    }

    private var vpnPermissionResult: MethodChannel.Result? = null
    private var methodChannel: MethodChannel? = null
    private var pendingSocksPort: Int = 1080
    private var goEngine: Any? = null

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)

        // Try to load Go engine (only works if mobile.aar is built)
        tryInitGoEngine()

        methodChannel = MethodChannel(flutterEngine.dartExecutor.binaryMessenger, ENGINE_CHANNEL)
        methodChannel?.setMethodCallHandler { call, result ->
            when (call.method) {
                "connect" -> {
                    val config = call.arguments as? String
                    if (config != null && goEngine != null) {
                        try {
                            val connectMethod = goEngine!!.javaClass.getMethod("connect", String::class.java)
                            val success = connectMethod.invoke(goEngine, config) as Boolean
                            if (success) {
                                pendingSocksPort = 1080
                                requestVpnAndStart(result)
                            } else {
                                result.success(false)
                            }
                        } catch (e: Exception) {
                            android.util.Log.e("Guarch", "Go engine connect failed: ${e.message}")
                            result.success(false)
                        }
                    } else {
                        // No Go engine — just start VPN service for testing
                        android.util.Log.w("Guarch", "Go engine not available — mobile.aar not built")
                        result.error("NO_ENGINE", "Native engine not available. Build mobile.aar first.", null)
                    }
                }
                "disconnect" -> {
                    try {
                        if (goEngine != null) {
                            val stopTunMethod = goEngine!!.javaClass.getMethod("stopTun")
                            stopTunMethod.invoke(goEngine)
                            val disconnectMethod = goEngine!!.javaClass.getMethod("disconnect")
                            disconnectMethod.invoke(goEngine)
                        }
                    } catch (e: Exception) {
                        android.util.Log.e("Guarch", "Disconnect error: ${e.message}")
                    }
                    stopVpnService()
                    result.success(true)
                }
                "getStatus" -> {
                    try {
                        if (goEngine != null) {
                            val method = goEngine!!.javaClass.getMethod("getStatus")
                            result.success(method.invoke(goEngine) as String)
                        } else {
                            result.success(if (GuarchService.isRunning) "connected" else "disconnected")
                        }
                    } catch (e: Exception) {
                        result.success("disconnected")
                    }
                }
                "getStats" -> {
                    try {
                        if (goEngine != null) {
                            val method = goEngine!!.javaClass.getMethod("getStats")
                            result.success(method.invoke(goEngine) as String)
                        } else {
                            result.success("{}")
                        }
                    } catch (e: Exception) {
                        result.success("{}")
                    }
                }
                "requestVpnPermission" -> {
                    requestVpnAndStart(result)
                }
                else -> result.notImplemented()
            }
        }

        EventChannel(flutterEngine.dartExecutor.binaryMessenger, EVENT_CHANNEL)
            .setStreamHandler(object : EventChannel.StreamHandler {
                override fun onListen(arguments: Any?, events: EventChannel.EventSink?) {}
                override fun onCancel(arguments: Any?) {}
            })
    }

    private fun tryInitGoEngine() {
        try {
            val mobileClass = Class.forName("mobile.Mobile")
            val newMethod = mobileClass.getMethod("new_")
            goEngine = newMethod.invoke(null)
            android.util.Log.i("Guarch", "Go engine loaded ✅")

            // Set callback
            try {
                val callbackClass = Class.forName("mobile.Callback")
                // For now, skip callback setup — stats will come from getStats polling
            } catch (e: Exception) {
                android.util.Log.w("Guarch", "Callback setup skipped: ${e.message}")
            }
        } catch (e: ClassNotFoundException) {
            android.util.Log.w("Guarch", "Go engine not found — mobile.aar not included")
            goEngine = null
        } catch (e: Exception) {
            android.util.Log.e("Guarch", "Go engine init failed: ${e.message}")
            goEngine = null
        }
    }

    private fun requestVpnAndStart(result: MethodChannel.Result) {
        val intent = VpnService.prepare(this)
        if (intent != null) {
            vpnPermissionResult = result
            startActivityForResult(intent, VPN_REQUEST_CODE)
        } else {
            startVpnAndTun(result)
        }
    }

    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
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
        val intent = Intent(this, GuarchService::class.java).apply {
            action = GuarchService.ACTION_START
            putExtra("socks_port", pendingSocksPort)
        }
        startService(intent)

        Thread {
            var attempts = 0
            while (GuarchService.tunFd < 0 && attempts < 50) {
                Thread.sleep(100)
                attempts++
            }

            val fd = GuarchService.tunFd
            if (fd >= 0 && goEngine != null) {
                try {
                    val startTunMethod = goEngine!!.javaClass.getMethod("startTun", Int::class.java, Int::class.java)
                    startTunMethod.invoke(goEngine, fd, pendingSocksPort)
                } catch (e: Exception) {
                    android.util.Log.e("Guarch", "StartTun failed: ${e.message}")
                }
            }

            runOnUiThread {
                result.success(fd >= 0)
            }
        }.start()
    }

    private fun stopVpnService() {
        val intent = Intent(this, GuarchService::class.java).apply {
            action = GuarchService.ACTION_STOP
        }
        startService(intent)
    }
}
