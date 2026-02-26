package com.guarch.app

import android.app.Activity
import android.content.Intent
import android.net.VpnService
import android.os.Bundle
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel

class MainActivity : FlutterActivity() {
    
    companion object {
        const val CHANNEL = "com.guarch.app/engine"
        const val VPN_REQUEST_CODE = 1001
    }

    private var vpnPermissionResult: MethodChannel.Result? = null

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, CHANNEL)
            .setMethodCallHandler { call, result ->
                when (call.method) {
                    "requestVpnPermission" -> {
                        val intent = VpnService.prepare(this)
                        if (intent != null) {
                            vpnPermissionResult = result
                            startActivityForResult(intent, VPN_REQUEST_CODE)
                        } else {
                            result.success(true)
                        }
                    }
                    "connect" -> {
                        val config = call.arguments as String
                        startVpnService(1080)
                        result.success(true)
                    }
                    "disconnect" -> {
                        stopVpnService()
                        result.success(true)
                    }
                    else -> result.notImplemented()
                }
            }
    }

    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        super.onActivityResult(requestCode, resultCode, data)
        if (requestCode == VPN_REQUEST_CODE) {
            vpnPermissionResult?.success(resultCode == Activity.RESULT_OK)
            vpnPermissionResult = null
        }
    }

    private fun startVpnService(socksPort: Int) {
        val intent = Intent(this, GuarchService::class.java).apply {
            action = GuarchService.ACTION_START
            putExtra("socks_port", socksPort)
        }
        startService(intent)
    }

    private fun stopVpnService() {
        val intent = Intent(this, GuarchService::class.java).apply {
            action = GuarchService.ACTION_STOP
        }
        startService(intent)
    }
}
