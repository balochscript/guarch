package com.guarch.app

import android.app.Activity
import android.content.Intent
import android.net.VpnService
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.EventChannel
import io.flutter.plugin.common.MethodChannel
import mobile.Mobile
import mobile.Engine
import mobile.Callback

class MainActivity : FlutterActivity() {

    companion object {
        const val ENGINE_CHANNEL = "com.guarch.app/engine"
        const val EVENT_CHANNEL = "com.guarch.app/events"
        const val VPN_REQUEST_CODE = 1001
    }

    private var engine: Engine? = null
    private var vpnPermissionResult: MethodChannel.Result? = null
    private var methodChannel: MethodChannel? = null
    private var pendingSocksPort: Int = 1080

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)

        // Go Engine
        engine = Mobile.new_()
        engine?.setCallback(object : Callback {
            override fun onStatusChanged(status: String) {
                runOnUiThread {
                    methodChannel?.invokeMethod("onStatusChanged", status)
                }
            }
            override fun onStatsUpdate(jsonData: String) {
                runOnUiThread {
                    methodChannel?.invokeMethod("onStatsUpdate", jsonData)
                }
            }
            override fun onLog(message: String) {
                runOnUiThread {
                    methodChannel?.invokeMethod("onLog", message)
                }
            }
        })

        // Method Channel
        methodChannel = MethodChannel(flutterEngine.dartExecutor.binaryMessenger, ENGINE_CHANNEL)
        methodChannel?.setMethodCallHandler { call, result ->
            when (call.method) {
                "connect" -> {
                    val config = call.arguments as String
                    val success = engine?.connect(config) ?: false
                    if (success) {
                        pendingSocksPort = 1080
                        requestVpnAndStart(result)
                    } else {
                        result.success(false)
                    }
                }
                "disconnect" -> {
                    engine?.stopTun()
                    engine?.disconnect()
                    stopVpnService()
                    result.success(true)
                }
                "getStatus" -> {
                    result.success(engine?.getStatus() ?: "disconnected")
                }
                "getStats" -> {
                    result.success(engine?.getStats() ?: "{}")
                }
                "requestVpnPermission" -> {
                    requestVpnAndStart(result)
                }
                else -> result.notImplemented()
            }
        }

        // Event Channel
        EventChannel(flutterEngine.dartExecutor.binaryMessenger, EVENT_CHANNEL)
            .setStreamHandler(object : EventChannel.StreamHandler {
                override fun onListen(arguments: Any?, events: EventChannel.EventSink?) {}
                override fun onCancel(arguments: Any?) {}
            })
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
            if (fd >= 0) {
                engine?.startTun(fd.toLong(), pendingSocksPort.toLong())
                runOnUiThread { result.success(true) }
            } else {
                runOnUiThread { result.success(false) }
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
