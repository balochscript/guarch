package com.guarch.app

import android.os.Bundle
import android.os.Handler
import android.os.Looper
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.EventChannel
import io.flutter.plugin.common.MethodChannel
import mobile.Mobile
import mobile.Callback

class MainActivity : FlutterActivity() {

    private val METHOD_CHANNEL = "com.guarch.app/engine"
    private val EVENT_CHANNEL = "com.guarch.app/events"

    private var engine: mobile.Engine? = null
    private var eventSink: EventChannel.EventSink? = null
    private val mainHandler = Handler(Looper.getMainLooper())

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)

        // Initialize Go engine
        engine = Mobile.new_()

        // Method Channel
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, METHOD_CHANNEL)
            .setMethodCallHandler { call, result ->
                when (call.method) {
                    "connect" -> {
                        val config = call.arguments as? String ?: ""
                        val success = engine?.connect(config) ?: false
                        result.success(success)
                    }
                    "disconnect" -> {
                        val success = engine?.disconnect() ?: false
                        result.success(success)
                    }
                    "getStatus" -> {
                        result.success(engine?.getStatus() ?: "disconnected")
                    }
                    "getStats" -> {
                        result.success(engine?.getStats() ?: "{}")
                    }
                    else -> result.notImplemented()
                }
            }

        // Event Channel
        EventChannel(flutterEngine.dartExecutor.binaryMessenger, EVENT_CHANNEL)
            .setStreamHandler(object : EventChannel.StreamHandler {
                override fun onListen(arguments: Any?, events: EventChannel.EventSink?) {
                    eventSink = events
                    setupCallback()
                }

                override fun onCancel(arguments: Any?) {
                    eventSink = null
                }
            })
    }

    private fun setupCallback() {
        engine?.setCallback(object : Callback {
            override fun onStatusChanged(status: String?) {
                mainHandler.post {
                    eventSink?.success(mapOf(
                        "type" to "status",
                        "data" to (status ?: "disconnected")
                    ))
                }
            }

            override fun onStatsUpdate(jsonData: String?) {
                mainHandler.post {
                    eventSink?.success(mapOf(
                        "type" to "stats",
                        "data" to (jsonData ?: "{}")
                    ))
                }
            }

            override fun onLog(message: String?) {
                mainHandler.post {
                    eventSink?.success(mapOf(
                        "type" to "log",
                        "data" to (message ?: "")
                    ))
                }
            }
        })
    }

    override fun onDestroy() {
        engine?.disconnect()
        super.onDestroy()
    }
}
