import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import 'package:guarch/app.dart';
import 'package:guarch/providers/app_provider.dart';

class LogsScreen extends StatefulWidget {
  const LogsScreen({super.key});

  @override
  State<LogsScreen> createState() => _LogsScreenState();
}

class _LogsScreenState extends State<LogsScreen> {
  String _filterType = 'all'; // all, info, warning, error, success
  final _searchController = TextEditingController();
  String _searchQuery = '';

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Consumer<AppProvider>(
      builder: (context, provider, _) {
        final filteredLogs = _filterLogs(provider.logs);

        return Scaffold(
          appBar: AppBar(
            title: const Text('Logs'),
            actions: [
              // Filter menu
              PopupMenuButton<String>(
                icon: Icon(
                  Icons.filter_list,
                  color: _filterType != 'all' ? accentColor(context) : null,
                ),
                tooltip: 'Filter logs',
                initialValue: _filterType,
                onSelected: (value) => setState(() => _filterType = value),
                itemBuilder: (context) => [
                  const PopupMenuItem(value: 'all', child: Text('🔍 All Logs')),
                  const PopupMenuItem(value: 'info', child: Text('ℹ️ Info')),
                  const PopupMenuItem(value: 'success', child: Text('✅ Success')),
                  const PopupMenuItem(value: 'warning', child: Text('⚠️ Warning')),
                  const PopupMenuItem(value: 'error', child: Text('❌ Error')),
                ],
              ),
              // Copy all
              if (provider.logs.isNotEmpty)
                IconButton(
                  icon: const Icon(Icons.copy),
                  tooltip: 'Copy all logs',
                  onPressed: () {
                    Clipboard.setData(
                      ClipboardData(text: provider.logs.join('\n')),
                    );
                    ScaffoldMessenger.of(context).showSnackBar(
                      SnackBar(
                        content: Text('${provider.logs.length} logs copied'),
                      ),
                    );
                  },
                ),
              // Clear logs
              if (provider.logs.isNotEmpty)
                IconButton(
                  icon: const Icon(Icons.delete_outline),
                  tooltip: 'Clear logs',
                  onPressed: () => _confirmClear(context, provider),
                ),
            ],
            bottom: PreferredSize(
              preferredSize: const Size.fromHeight(60),
              child: Padding(
                padding: const EdgeInsets.fromLTRB(16, 0, 16, 8),
                child: TextField(
                  controller: _searchController,
                  style: TextStyle(color: textSecondary(context)),
                  decoration: InputDecoration(
                    hintText: 'Search logs...',
                    prefixIcon: const Icon(Icons.search, size: 20),
                    suffixIcon: _searchQuery.isNotEmpty
                        ? IconButton(
                            icon: const Icon(Icons.clear, size: 20),
                            onPressed: () {
                              _searchController.clear();
                              setState(() => _searchQuery = '');
                            },
                          )
                        : null,
                    contentPadding: const EdgeInsets.symmetric(
                      horizontal: 16,
                      vertical: 8,
                    ),
                    border: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(12),
                    ),
                  ),
                  onChanged: (value) => setState(() => _searchQuery = value),
                ),
              ),
            ),
          ),
          body: provider.logs.isEmpty
              ? _buildEmpty(context)
              : Column(
                  children: [
                    // Stats bar
                    if (filteredLogs.isNotEmpty)
                      Container(
                        padding: const EdgeInsets.symmetric(
                          horizontal: 16,
                          vertical: 8,
                        ),
                        color: accentColor(context).withOpacity(0.05),
                        child: Row(
                          children: [
                            Text(
                              '${filteredLogs.length} of ${provider.logs.length} logs',
                              style: TextStyle(
                                color: textMuted(context),
                                fontSize: 12,
                              ),
                            ),
                            if (_searchQuery.isNotEmpty) ...[
                              const SizedBox(width: 8),
                              Container(
                                padding: const EdgeInsets.symmetric(
                                  horizontal: 8,
                                  vertical: 2,
                                ),
                                decoration: BoxDecoration(
                                  color: accentColor(context).withOpacity(0.1),
                                  borderRadius: BorderRadius.circular(4),
                                ),
                                child: Text(
                                  'Search: "$_searchQuery"',
                                  style: TextStyle(
                                    fontSize: 11,
                                    color: accentColor(context),
                                  ),
                                ),
                              ),
                            ],
                          ],
                        ),
                      ),
                    // Logs list
                    Expanded(
                      child: ListView.builder(
                        padding: const EdgeInsets.all(16),
                        itemCount: filteredLogs.length,
                        itemBuilder: (context, index) {
                          final log = filteredLogs[index];
                          final logType = _detectLogType(log);

                          return Padding(
                            padding: const EdgeInsets.only(bottom: 4),
                            child: Text(
                              log,
                              style: TextStyle(
                                fontFamily: 'monospace',
                                fontSize: 12,
                                color: _logColor(context, logType),
                                height: 1.4,
                              ),
                            ),
                          );
                        },
                      ),
                    ),
                  ],
                ),
          // Auto-scroll toggle
          floatingActionButton: provider.logs.isNotEmpty
              ? FloatingActionButton.small(
                  onPressed: () {
                    // Scroll to bottom
                    // TODO: Implement auto-scroll
                  },
                  child: const Icon(Icons.arrow_downward, size: 20),
                )
              : null,
        );
      },
    );
  }

  Widget _buildEmpty(BuildContext context) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(
            Icons.article_outlined,
            size: 80,
            color: accentColor(context).withOpacity(0.2),
          ),
          const SizedBox(height: 16),
          Text(
            'No logs yet',
            style: TextStyle(color: textMuted(context)),
          ),
          const SizedBox(height: 8),
          Text(
            'Logs will appear here when you connect',
            style: TextStyle(
              color: textMuted(context).withOpacity(0.6),
              fontSize: 12,
            ),
          ),
        ],
      ),
    );
  }

  List<String> _filterLogs(List<String> logs) {
    var filtered = logs;

    // Search filter
    if (_searchQuery.isNotEmpty) {
      filtered = filtered
          .where((log) =>
              log.toLowerCase().contains(_searchQuery.toLowerCase()))
          .toList();
    }

    // Type filter
    if (_filterType != 'all') {
      filtered = filtered.where((log) {
        final type = _detectLogType(log);
        return type == _filterType;
      }).toList();
    }

    return filtered;
  }

  String _detectLogType(String log) {
    final lower = log.toLowerCase();

    if (lower.contains('✅') ||
        lower.contains('connected') ||
        lower.contains('success') ||
        lower.contains('complete')) {
      return 'success';
    } else if (lower.contains('❌') ||
        lower.contains('failed') ||
        lower.contains('error') ||
        lower.contains('timeout') ||
        lower.contains('crash')) {
      return 'error';
    } else if (lower.contains('⚠️') ||
        lower.contains('warning') ||
        lower.contains('low battery') ||
        lower.contains('fallback')) {
      return 'warning';
    } else {
      return 'info';
    }
  }

  Color _logColor(BuildContext context, String type) {
    switch (type) {
      case 'success':
        return Colors.green;
      case 'error':
        return Colors.red;
      case 'warning':
        return Colors.orange;
      default:
        return textMuted(context);
    }
  }

  void _confirmClear(BuildContext context, AppProvider provider) {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Clear Logs'),
        content: Text(
          'Delete all ${provider.logs.length} log entries?',
          style: TextStyle(color: textSecondary(context)),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancel'),
          ),
          TextButton(
            onPressed: () {
              provider.clearLogs();
              Navigator.pop(ctx);
            },
            style: TextButton.styleFrom(foregroundColor: Colors.red),
            child: const Text('Clear'),
          ),
        ],
      ),
    );
  }
}
