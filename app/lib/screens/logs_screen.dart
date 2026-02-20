import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import 'package:guarch/providers/app_provider.dart';

class LogsScreen extends StatelessWidget {
  const LogsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Consumer<AppProvider>(
      builder: (context, provider, _) {
        return Scaffold(
          appBar: AppBar(
            title: const Text('Logs'),
            actions: [
              if (provider.logs.isNotEmpty) ...[
                IconButton(
                  icon: const Icon(Icons.copy),
                  tooltip: 'Copy all logs',
                  onPressed: () {
                    Clipboard.setData(
                      ClipboardData(text: provider.logs.join('\n')),
                    );
                    ScaffoldMessenger.of(context).showSnackBar(
                      const SnackBar(content: Text('Logs copied')),
                    );
                  },
                ),
                IconButton(
                  icon: const Icon(Icons.delete_outline),
                  tooltip: 'Clear logs',
                  onPressed: () => provider.clearLogs(),
                ),
              ],
            ],
          ),
          body: provider.logs.isEmpty
              ? Center(
                  child: Column(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      Icon(Icons.article_outlined,
                          size: 80, color: Colors.grey.shade600),
                      const SizedBox(height: 16),
                      const Text(
                        'No logs yet',
                        style: TextStyle(color: Colors.grey),
                      ),
                    ],
                  ),
                )
              : ListView.builder(
                  padding: const EdgeInsets.all(16),
                  itemCount: provider.logs.length,
                  itemBuilder: (context, index) {
                    final log = provider.logs[index];
                    Color textColor = Colors.grey.shade300;

                    if (log.contains('Connected')) {
                      textColor = Colors.green;
                    } else if (log.contains('failed') ||
                        log.contains('error') ||
                        log.contains('timeout')) {
                      textColor = Colors.red;
                    } else if (log.contains('Connecting') ||
                        log.contains('Disconnecting')) {
                      textColor = Colors.orange;
                    } else if (log.contains('Ping')) {
                      textColor = Colors.cyan;
                    } else if (log.contains('Cover') ||
                        log.contains('cover')) {
                      textColor = const Color(0xFF6C5CE7);
                    }

                    return Padding(
                      padding: const EdgeInsets.only(bottom: 4),
                      child: Text(
                        log,
                        style: TextStyle(
                          fontFamily: 'monospace',
                          fontSize: 12,
                          color: textColor,
                        ),
                      ),
                    );
                  },
                ),
        );
      },
    );
  }
}
