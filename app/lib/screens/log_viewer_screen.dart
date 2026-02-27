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
  String _flutterLogs = '';
  Timer? _timer;
  final _scroll = ScrollController();

  // لاگ‌های Flutter رو هم اینجا نگه میداریم
  static final List<String> flutterLogEntries = [];

  static void addFlutterLog(String msg) {
    final time = DateTime.now().toString().substring(11, 23);
    flutterLogEntries.add('[$time] FLUTTER: $msg');
    if (flutterLogEntries.length > 500) {
      flutterLogEntries.removeAt(0);
    }
  }

  @override
  void initState() {
    super.initState();
    _loadLogs();
    _timer = Timer.periodic(const Duration(seconds: 2), (_) => _loadLogs());
  }

  @override
  void dispose() {
    _timer?.cancel();
    _scroll.dispose();
    super.dispose();
  }

  Future<void> _loadLogs() async {
    String nativeLogs = 'Could not load native logs';
    try {
      nativeLogs = await _channel.invokeMethod<String>('getLogs') ?? 'No native logs';
    } catch (e) {
      nativeLogs = 'Native log error: $e';
    }

    final flutter = flutterLogEntries.join('\n');

    if (mounted) {
      setState(() {
        _logs = '══════ FLUTTER LOGS ══════\n$flutter\n\n══════ NATIVE LOGS ══════\n$nativeLogs';
      });
    }
  }

  void _copyLogs() {
    Clipboard.setData(ClipboardData(text: _logs));
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('لاگ‌ها کپی شدند ✅')),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.black,
      appBar: AppBar(
        title: const Text('🔍 Debug Logs'),
        backgroundColor: Colors.grey[900],
        actions: [
          IconButton(icon: const Icon(Icons.copy), onPressed: _copyLogs),
          IconButton(icon: const Icon(Icons.refresh), onPressed: _loadLogs),
          IconButton(
            icon: const Icon(Icons.delete),
            onPressed: () {
              flutterLogEntries.clear();
              _loadLogs();
            },
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
            color: Colors.green,
            height: 1.5,
          ),
        ),
      ),
      floatingActionButton: FloatingActionButton.small(
        onPressed: () => _scroll.animateTo(
          _scroll.position.maxScrollExtent,
          duration: const Duration(milliseconds: 300),
          curve: Curves.easeOut,
        ),
        backgroundColor: Colors.grey[800],
        child: const Icon(Icons.arrow_downward, size: 18),
      ),
    );
  }
}
