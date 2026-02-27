import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'dart:async';

class LogViewerScreen extends StatefulWidget {
  const LogViewerScreen({super.key});

  @override
  State<LogViewerScreen> createState() => _LogViewerScreenState();
}

class _LogViewerScreenState extends State<LogViewerScreen> {
  static const _channel = MethodChannel('com.guarch.app/logs');
  String _logs = 'Loading...';
  bool _showCrashLog = false;
  Timer? _timer;
  final _scroll = ScrollController();

  @override
  void initState() {
    super.initState();
    _refresh();
    _timer = Timer.periodic(const Duration(seconds: 3), (_) {
      if (!_showCrashLog) _refresh();
    });
  }

  @override
  void dispose() {
    _timer?.cancel();
    _scroll.dispose();
    super.dispose();
  }

  Future<void> _refresh() async {
    try {
      final logs = await _channel.invokeMethod<String>('getLogs') ?? 'No logs';
      if (mounted) setState(() => _logs = logs);
    } catch (e) {
      if (mounted) setState(() => _logs = 'Error: $e');
    }
  }

  Future<void> _showPrevCrash() async {
    try {
      final crash = await _channel.invokeMethod<String>('getCrashLog') ?? 'No crash log';
      if (mounted) {
        setState(() {
          _logs = '══════ PREVIOUS CRASH LOG ══════\n\n$crash';
          _showCrashLog = true;
        });
      }
    } catch (e) {
      if (mounted) setState(() => _logs = 'Error: $e');
    }
  }

  void _share() {
    try {
      _channel.invokeMethod('shareLogs');
    } catch (_) {}
  }

  void _copy() {
    Clipboard.setData(ClipboardData(text: _logs));
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('کپی شد ✅'), duration: Duration(seconds: 1)),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: const Color(0xFF0A0A0A),
      appBar: AppBar(
        title: Text(_showCrashLog ? '💥 Crash Log' : '🔍 Live Log'),
        backgroundColor: const Color(0xFF1A1A1A),
        actions: [
          // دکمه تغییر بین Live و Crash
          IconButton(
            icon: Icon(
              _showCrashLog ? Icons.play_arrow : Icons.warning_amber,
              color: _showCrashLog ? Colors.green : Colors.orange,
            ),
            onPressed: () {
              if (_showCrashLog) {
                _showCrashLog = false;
                _refresh();
              } else {
                _showPrevCrash();
              }
            },
            tooltip: _showCrashLog ? 'Live Log' : 'Crash Log',
          ),
          // Share
          IconButton(
            icon: const Icon(Icons.share, color: Colors.white70),
            onPressed: _share,
            tooltip: 'Share',
          ),
          // Copy
          IconButton(
            icon: const Icon(Icons.copy, color: Colors.white70),
            onPressed: _copy,
            tooltip: 'Copy',
          ),
          // Refresh
          IconButton(
            icon: const Icon(Icons.refresh, color: Colors.white70),
            onPressed: _refresh,
          ),
        ],
      ),
      body: SingleChildScrollView(
        controller: _scroll,
        padding: const EdgeInsets.all(8),
        child: SelectableText(
          _logs,
          style: const TextStyle(
            fontFamily: 'monospace',
            fontSize: 10,
            color: Colors.greenAccent,
            height: 1.5,
          ),
        ),
      ),
      floatingActionButton: FloatingActionButton.small(
        onPressed: () {
          if (_scroll.hasClients) {
            _scroll.animateTo(
              _scroll.position.maxScrollExtent,
              duration: const Duration(milliseconds: 200),
              curve: Curves.easeOut,
            );
          }
        },
        backgroundColor: Colors.grey[800],
        child: const Icon(Icons.arrow_downward, size: 18),
      ),
    );
  }
}
