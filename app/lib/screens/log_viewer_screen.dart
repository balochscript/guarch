import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'dart:async';
import 'package:guarch/services/guarch_engine.dart';

class LogViewerScreen extends StatefulWidget {
  const LogViewerScreen({super.key});

  @override
  State<LogViewerScreen> createState() => _LogViewerScreenState();
}

class _LogViewerScreenState extends State<LogViewerScreen> {
  static const _logChannel = MethodChannel('com.guarch.app/logs');
  String _allLogs = 'Loading...';
  Timer? _timer;
  final _scroll = ScrollController();
  bool _autoScroll = true;

  @override
  void initState() {
    super.initState();
    _refresh();
    _timer = Timer.periodic(const Duration(seconds: 2), (_) => _refresh());
  }

  @override
  void dispose() {
    _timer?.cancel();
    _scroll.dispose();
    super.dispose();
  }

  Future<void> _refresh() async {
    // لاگ‌های Flutter
    final flutterLogs = FlutterLog.getAll();

    // لاگ‌های Native
    String nativeLogs;
    try {
      nativeLogs = await _logChannel.invokeMethod<String>('getLogs') ?? 'No native logs';
    } catch (e) {
      nativeLogs = 'Could not get native logs: $e';
    }

    if (mounted) {
      setState(() {
        _allLogs = '════════ FLUTTER ════════\n'
            '$flutterLogs\n\n'
            '════════ NATIVE ════════\n'
            '$nativeLogs';
      });

      if (_autoScroll && _scroll.hasClients) {
        WidgetsBinding.instance.addPostFrameCallback((_) {
          if (_scroll.hasClients) {
            _scroll.jumpTo(_scroll.position.maxScrollExtent);
          }
        });
      }
    }
  }

  void _copy() {
    Clipboard.setData(ClipboardData(text: _allLogs));
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(
        content: Text('لاگ‌ها کپی شدند ✅'),
        duration: Duration(seconds: 1),
      ),
    );
  }

  void _clear() async {
    FlutterLog.entries.clear();
    try {
      await _logChannel.invokeMethod('clearLogs');
    } catch (_) {}
    _refresh();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: const Color(0xFF0A0A0A),
      appBar: AppBar(
        title: const Text('🔍 Debug Logs'),
        backgroundColor: const Color(0xFF1A1A1A),
        actions: [
          IconButton(
            icon: Icon(
              _autoScroll ? Icons.vertical_align_bottom : Icons.vertical_align_top,
              color: _autoScroll ? Colors.greenAccent : Colors.grey,
            ),
            onPressed: () => setState(() => _autoScroll = !_autoScroll),
            tooltip: 'Auto-scroll',
          ),
          IconButton(
            icon: const Icon(Icons.copy, color: Colors.white70),
            onPressed: _copy,
            tooltip: 'Copy All',
          ),
          IconButton(
            icon: const Icon(Icons.refresh, color: Colors.white70),
            onPressed: _refresh,
            tooltip: 'Refresh',
          ),
          IconButton(
            icon: const Icon(Icons.delete_outline, color: Colors.redAccent),
            onPressed: _clear,
            tooltip: 'Clear',
          ),
        ],
      ),
      body: SingleChildScrollView(
        controller: _scroll,
        padding: const EdgeInsets.all(8),
        child: SelectableText(
          _allLogs,
          style: const TextStyle(
            fontFamily: 'monospace',
            fontSize: 10,
            color: Colors.greenAccent,
            height: 1.5,
          ),
        ),
      ),
    );
  }
}
