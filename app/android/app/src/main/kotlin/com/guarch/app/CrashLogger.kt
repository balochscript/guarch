package com.guarch.app

import android.util.Log
import java.text.SimpleDateFormat
import java.util.*

object CrashLogger {
    private val logs = mutableListOf<String>()
    private val dateFormat = SimpleDateFormat("HH:mm:ss.SSS", Locale.US)

    fun clear() { logs.clear() }

    fun d(tag: String, msg: String) {
        val entry = "[${dateFormat.format(Date())}] D/$tag: $msg"
        synchronized(logs) { logs.add(entry) }
        Log.d(tag, msg)
    }

    fun e(tag: String, msg: String, t: Throwable? = null) {
        val stack = t?.let {
            "\n  → ${it.javaClass.simpleName}: ${it.message}" +
            (it.stackTrace.take(5).joinToString("") { s -> "\n    at $s" })
        } ?: ""
        val entry = "[${dateFormat.format(Date())}] E/$tag: $msg$stack"
        synchronized(logs) { logs.add(entry) }
        Log.e(tag, msg, t)
    }

    fun w(tag: String, msg: String) {
        val entry = "[${dateFormat.format(Date())}] W/$tag: $msg"
        synchronized(logs) { logs.add(entry) }
        Log.w(tag, msg)
    }

    fun getAllLogs(): String = synchronized(logs) {
        if (logs.isEmpty()) "No logs yet" else logs.joinToString("\n")
    }
}
