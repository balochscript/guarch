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
  String _mode = 'live'; // live, crash, go
  Timer? _timer;
  final _scroll = ScrollController();

  @override
  void initState() {
    super.initState();
    _refresh();
    _timer = Timer.periodic(const Duration(seconds: 3), (_) {
      if (_mode == 'live') _refresh();
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
      String method;
      switch (_mode) {
        case 'crash':
          method = 'getCrashLog';
          break;
        case 'go':
          method = 'getGoLog';
          break;
        default:
          method = 'getLogs';
      }
      final logs = await _channel.invokeMethod<String>(method) ?? 'No logs';
      if (mounted) setState(() => _logs = logs);
    } catch (e) {
      if (mounted) setState(() => _logs = 'Error: $e');
    }
  }

  void _switchMode(String mode) {
    setState(() => _mode = mode);
    _refresh();
  }

  void _share() {
    try { _channel.invokeMethod('shareLogs'); } catch (_) {}
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
        title: Text(_modeTitle),
        backgroundColor: const Color(0xFF1A1A1A),
        actions: [
          IconButton(icon: const Icon(Icons.share, size: 20), onPressed: _share),
          IconButton(icon: const Icon(Icons.copy, size: 20), onPressed: _copy),
          IconButton(icon: const Icon(Icons.refresh, size: 20), onPressed: _refresh),
        ],
      ),
      body: Column(
        children: [
          // Tab bar
          Container(
            color: const Color(0xFF1A1A1A),
            child: Row(
              children: [
                _tab('Live', 'live', Icons.play_arrow, Colors.green),
                _tab('Crash', 'crash', Icons.warning, Colors.orange),
                _tab('Go', 'go', Icons.code, Colors.cyan),
              ],
            ),
          ),
          // Logs
          Expanded(
            child: SingleChildScrollView(
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
          ),
        ],
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

  String get _modeTitle {
    switch (_mode) {
      case 'crash': return '💥 Previous Crash';
      case 'go': return '🔧 Go Engine Log';
      default: return '🔍 Live Log';
    }
  }

  Widget _tab(String label, String mode, IconData icon, Color color) {
    final active = _mode == mode;
    return Expanded(
      child: InkWell(
        onTap: () => _switchMode(mode),
        child: Container(
          padding: const EdgeInsets.symmetric(vertical: 10),
          decoration: BoxDecoration(
            border: Border(
              bottom: BorderSide(
                color: active ? color : Colors.transparent,
                width: 2,
              ),
            ),
          ),
          child: Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Icon(icon, size: 16, color: active ? color : Colors.grey),
              const SizedBox(width: 4),
              Text(
                label,
                style: TextStyle(
                  color: active ? color : Colors.grey,
                  fontSize: 12,
                  fontWeight: active ? FontWeight.bold : FontWeight.normal,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
