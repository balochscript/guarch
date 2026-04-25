import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:guarch/app.dart';
import 'package:guarch/services/guarch_engine.dart';
import 'package:share_plus/share_plus.dart';

class LogViewerScreen extends StatefulWidget {
  const LogViewerScreen({super.key});

  @override
  State<LogViewerScreen> createState() => _LogViewerScreenState();
}

class _LogViewerScreenState extends State<LogViewerScreen>
    with SingleTickerProviderStateMixin {
  late TabController _tabController;
  String _flutterLogs = '';
  String _goLogs = '';
  String _nativeLogs = '';
  bool _isLoading = true;
  bool _autoScroll = true;

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 3, vsync: this);
    _loadLogs();
  }

  @override
  void dispose() {
    _tabController.dispose();
    super.dispose();
  }

  Future<void> _loadLogs() async {
    setState(() => _isLoading = true);

    try {
      // Flutter logs (from FlutterLog)
      _flutterLogs = FlutterLog.getAll();

      // Go engine logs (from mobile/mobile.go)
      try {
        _goLogs = await GuarchEngine().readGoLog();
      } catch (e) {
        _goLogs = 'Error loading Go logs: $e';
      }

      // Native Android logs (from CrashLogger.kt)
      try {
        final logChannel = const MethodChannel('com.guarch.app/logs');
        _nativeLogs = await logChannel.invokeMethod<String>('getLogs') ?? '';
      } catch (e) {
        _nativeLogs = 'Error loading native logs: $e';
      }
    } catch (e) {
      _flutterLogs = 'Error: $e';
    }

    setState(() => _isLoading = false);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('🐛 Debug Logs'),
        bottom: TabBar(
          controller: _tabController,
          tabs: const [
            Tab(text: 'Flutter'),
            Tab(text: 'Go Engine'),
            Tab(text: 'Native'),
          ],
        ),
        actions: [
          // Auto-scroll toggle
          IconButton(
            icon: Icon(
              _autoScroll ? Icons.sync : Icons.sync_disabled,
              color: _autoScroll ? accentColor(context) : null,
            ),
            tooltip: 'Auto-scroll',
            onPressed: () => setState(() => _autoScroll = !_autoScroll),
          ),
          // Refresh
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: 'Refresh',
            onPressed: _loadLogs,
          ),
          // More menu
          PopupMenuButton<String>(
            onSelected: (value) {
              switch (value) {
                case 'copy_all':
                  _copyAllLogs();
                  break;
                case 'share':
                  _shareLogs();
                  break;
                case 'clear':
                  _clearLogs();
                  break;
              }
            },
            itemBuilder: (context) => [
              const PopupMenuItem(
                value: 'copy_all',
                child: Row(
                  children: [
                    Icon(Icons.copy, size: 18),
                    SizedBox(width: 8),
                    Text('Copy All Logs'),
                  ],
                ),
              ),
              const PopupMenuItem(
                value: 'share',
                child: Row(
                  children: [
                    Icon(Icons.share, size: 18),
                    SizedBox(width: 8),
                    Text('Share Logs'),
                  ],
                ),
              ),
              const PopupMenuItem(
                value: 'clear',
                child: Row(
                  children: [
                    Icon(Icons.delete_outline, size: 18),
                    SizedBox(width: 8),
                    Text('Clear Logs'),
                  ],
                ),
              ),
            ],
          ),
        ],
      ),
      body: _isLoading
          ? const Center(child: CircularProgressIndicator())
          : TabBarView(
              controller: _tabController,
              children: [
                _buildLogView('Flutter', _flutterLogs),
                _buildLogView('Go Engine', _goLogs),
                _buildLogView('Native', _nativeLogs),
              ],
            ),
    );
  }

  Widget _buildLogView(String title, String content) {
    final lines = content.split('\n');
    final lineCount = lines.length;

    return Column(
      children: [
        // Stats bar
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
          color: accentColor(context).withOpacity(0.05),
          child: Row(
            children: [
              Icon(
                Icons.article,
                size: 16,
                color: textMuted(context),
              ),
              const SizedBox(width: 8),
              Text(
                '$lineCount lines',
                style: TextStyle(
                  color: textMuted(context),
                  fontSize: 12,
                ),
              ),
              const Spacer(),
              IconButton(
                icon: Icon(
                  Icons.copy,
                  size: 18,
                  color: textMuted(context),
                ),
                onPressed: () => _copyLog(title, content),
                tooltip: 'Copy $title log',
              ),
            ],
          ),
        ),

        // Log content
        Expanded(
          child: content.isEmpty
              ? Center(
                  child: Column(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      Icon(
                        Icons.article_outlined,
                        size: 64,
                        color: accentColor(context).withOpacity(0.2),
                      ),
                      const SizedBox(height: 16),
                      Text(
                        'No $title logs',
                        style: TextStyle(color: textMuted(context)),
                      ),
                    ],
                  ),
                )
              : ListView.builder(
                  padding: const EdgeInsets.all(16),
                  itemCount: lineCount,
                  reverse: _autoScroll,
                  itemBuilder: (context, index) {
                    final line = _autoScroll
                        ? lines[lineCount - 1 - index]
                        : lines[index];

                    final lineNumber = _autoScroll
                        ? lineCount - index
                        : index + 1;

                    return Padding(
                      padding: const EdgeInsets.only(bottom: 2),
                      child: Row(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          // Line number
                          SizedBox(
                            width: 50,
                            child: Text(
                              '$lineNumber',
                              style: TextStyle(
                                fontFamily: 'monospace',
                                fontSize: 10,
                                color: textMuted(context).withOpacity(0.5),
                              ),
                              textAlign: TextAlign.right,
                            ),
                          ),
                          const SizedBox(width: 8),
                          // Log line
                          Expanded(
                            child: Text(
                              line,
                              style: TextStyle(
                                fontFamily: 'monospace',
                                fontSize: 11,
                                color: _getLogColor(line),
                                height: 1.4,
                              ),
                            ),
                          ),
                        ],
                      ),
                    );
                  },
                ),
        ),
      ],
    );
  }

  Color _getLogColor(String line) {
    final lower = line.toLowerCase();

    if (lower.contains('error') ||
        lower.contains('failed') ||
        lower.contains('crash') ||
        lower.contains('panic')) {
      return Colors.red;
    } else if (lower.contains('warning') ||
        lower.contains('warn') ||
        lower.contains('⚠️')) {
      return Colors.orange;
    } else if (lower.contains('success') ||
        lower.contains('connected') ||
        lower.contains('✅')) {
      return Colors.green;
    } else if (lower.contains('debug') || lower.contains('[d]')) {
      return Colors.blue;
    } else {
      return textMuted(context);
    }
  }

  void _copyLog(String title, String content) {
    Clipboard.setData(ClipboardData(text: content));
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text('$title log copied')),
    );
  }

  void _copyAllLogs() {
    final allLogs = '''
=== Flutter Logs ===
$_flutterLogs

=== Go Engine Logs ===
$_goLogs

=== Native Logs ===
$_nativeLogs
''';

    Clipboard.setData(ClipboardData(text: allLogs));
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('All logs copied')),
    );
  }

  void _shareLogs() {
    final allLogs = '''
Guarch Debug Logs v1.0.1
Generated: ${DateTime.now().toString()}

=== Flutter Logs ===
$_flutterLogs

=== Go Engine Logs ===
$_goLogs

=== Native Logs ===
$_nativeLogs
''';

    Share.share(allLogs, subject: 'Guarch Debug Logs');
  }

  void _clearLogs() {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Clear All Logs'),
        content: const Text(
          'This will clear all debug logs (Flutter, Go, and Native).\n\n'
          'Are you sure?',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancel'),
          ),
          TextButton(
            onPressed: () async {
              // Clear Flutter logs
              FlutterLog.clear();

              // Clear Go logs
              try {
                await GuarchEngine().clearGoLog();
              } catch (_) {}

              // Clear Native logs
              try {
                const logChannel = MethodChannel('com.guarch.app/logs');
                await logChannel.invokeMethod('clearLogs');
              } catch (_) {}

              Navigator.pop(ctx);
              _loadLogs();

              if (context.mounted) {
                ScaffoldMessenger.of(context).showSnackBar(
                  const SnackBar(
                    content: Text('All logs cleared'),
                    backgroundColor: Colors.green,
                  ),
                );
              }
            },
            style: TextButton.styleFrom(foregroundColor: Colors.red),
            child: const Text('Clear'),
          ),
        ],
      ),
    );
  }
}
