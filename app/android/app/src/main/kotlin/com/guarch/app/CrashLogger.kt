package com.guarch.app

import android.content.Context
import android.util.Log
import java.io.File
import java.io.FileWriter
import java.text.SimpleDateFormat
import java.util.*

object CrashLogger {
    private val dateFormat = SimpleDateFormat("HH:mm:ss.SSS", Locale.US)
    private var logFile: File? = null
    private var writer: FileWriter? = null

    // اولین بار باید context بدی
    fun init(context: Context) {
        try {
            logFile = File(context.filesDir, "guarch_debug.log")

            // لاگ قبلی رو نگه دار بعنوان crash log
            if (logFile!!.exists() && logFile!!.length() > 0) {
                val prevFile = File(context.filesDir, "guarch_prev_crash.log")
                logFile!!.copyTo(prevFile, overwrite = true)
            }

            // فایل جدید شروع کن
            writer = FileWriter(logFile, false) // overwrite
            val header = "====== GUARCH LOG ${Date()} ======\n"
            writer?.write(header)
            writer?.flush()
        } catch (e: Exception) {
            Log.e("CrashLogger", "Init failed: ${e.message}")
        }
    }

    fun d(tag: String, msg: String) {
        val line = "[${dateFormat.format(Date())}] D/$tag: $msg"
        writeLine(line)
        Log.d(tag, msg)
    }

    fun e(tag: String, msg: String, t: Throwable? = null) {
        val stack = t?.let {
            "\n  >> ${it.javaClass.name}: ${it.message}" +
            it.stackTrace.take(10).joinToString("") { s -> "\n     at $s" }
        } ?: ""
        val line = "[${dateFormat.format(Date())}] E/$tag: $msg$stack"
        writeLine(line)
        Log.e(tag, msg, t)
    }

    fun w(tag: String, msg: String) {
        val line = "[${dateFormat.format(Date())}] W/$tag: $msg"
        writeLine(line)
        Log.w(tag, msg)
    }

    private fun writeLine(line: String) {
        try {
            writer?.write(line + "\n")
            writer?.flush()  // ← فوری بنویس، قبل از کرش
        } catch (_: Exception) {}
    }

    // لاگ این session
    fun getCurrentLog(context: Context): String {
        return try {
            File(context.filesDir, "guarch_debug.log").readText()
        } catch (_: Exception) {
            "No current log"
        }
    }

    // لاگ session قبلی (کرش)
    fun getPreviousCrashLog(context: Context): String {
        return try {
            val f = File(context.filesDir, "guarch_prev_crash.log")
            if (f.exists()) f.readText() else "No previous crash log"
        } catch (_: Exception) {
            "Could not read crash log"
        }
    }

    // مسیر فایل لاگ
    fun getLogFilePath(context: Context): String {
        return File(context.filesDir, "guarch_debug.log").absolutePath
    }

    fun close() {
        try { writer?.close() } catch (_: Exception) {}
    }
}
