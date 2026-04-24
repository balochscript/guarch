import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import 'package:guarch/app.dart';
import 'package:guarch/providers/app_provider.dart';
import 'package:guarch/models/server_config.dart';
import 'package:guarch/screens/add_server_screen.dart';
import 'package:guarch/screens/server_detail_screen.dart';
import 'package:share_plus/share_plus.dart';

class ServersScreen extends StatelessWidget {
  const ServersScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Consumer<AppProvider>(
      builder: (context, provider, _) {
        return Scaffold(
          appBar: AppBar(
            title: const Text('Servers'),
            actions: [
              // Ping Options Menu
              PopupMenuButton<String>(
                icon: const Icon(Icons.speed),
                tooltip: 'Ping options',
                onSelected: (value) {
                  if (value == 'tcping') {
                    provider.pingAllServers(includeRealDelay: false);
                  } else if (value == 'real') {
                    provider.pingAllServers(includeRealDelay: true);
                    ScaffoldMessenger.of(context).showSnackBar(
                      const SnackBar(
                        content: Text('Real delay test may take longer...'),
                        duration: Duration(seconds: 2),
                      ),
                    );
                  }
                },
                itemBuilder: (context) => [
                  const PopupMenuItem(
                    value: 'tcping',
                    child: Row(
                      children: [
                        Icon(Icons.bolt, size: 18),
                        SizedBox(width: 8),
                        Text('TCPing (Fast)'),
                      ],
                    ),
                  ),
                  const PopupMenuItem(
                    value: 'real',
                    child: Row(
                      children: [
                        Icon(Icons.auto_graph, size: 18),
                        SizedBox(width: 8),
                        Text('Real Delay (Accurate)'),
                      ],
                    ),
                  ),
                ],
              ),
              // Import from clipboard
              IconButton(
                icon: const Icon(Icons.content_paste),
                tooltip: 'Import from clipboard',
                onPressed: () => _importFromClipboard(context, provider),
              ),
            ],
          ),
          body: provider.servers.isEmpty
              ? _buildEmpty(context)
              : _buildList(context, provider),
          floatingActionButton: FloatingActionButton(
            onPressed: () => _openAddServer(context),
            child: const Icon(Icons.add),
          ),
        );
      },
    );
  }

  // ═══════════════════════════════════════════════════════════════
  // Empty State
  // ═══════════════════════════════════════════════════════════════

  Widget _buildEmpty(BuildContext context) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(
            Icons.dns_outlined,
            size: 80,
            color: accentColor(context).withOpacity(0.3),
          ),
          const SizedBox(height: 16),
          Text(
            'No servers yet',
            style: TextStyle(
              color: textMuted(context),
              fontSize: 18,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            'Add a server or import a config',
            style: TextStyle(
              color: textMuted(context).withOpacity(0.6),
            ),
          ),
          const SizedBox(height: 24),
          FilledButton.icon(
            onPressed: () => _openAddServer(context),
            icon: const Icon(Icons.add),
            label: const Text('Add Server'),
          ),
        ],
      ),
    );
  }

  // ═══════════════════════════════════════════════════════════════
  // Server List
  // ═══════════════════════════════════════════════════════════════

  Widget _buildList(BuildContext context, AppProvider provider) {
    return ListView.builder(
      padding: const EdgeInsets.all(16),
      itemCount: provider.servers.length,
      itemBuilder: (context, index) {
        final server = provider.servers[index];
        final isActive = provider.activeServer?.id == server.id;

        return Card(
          margin: const EdgeInsets.only(bottom: 12),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(16),
            side: isActive
                ? BorderSide(color: accentColor(context), width: 2)
                : BorderSide.none,
          ),
          child: InkWell(
            borderRadius: BorderRadius.circular(16),
            onTap: () => provider.setActiveServer(server.id),
            onLongPress: () => Navigator.push(
              context,
              MaterialPageRoute(
                builder: (_) => ServerDetailScreen(server: server),
              ),
            ),
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                children: [
                  // ─── Header Row ───
                  Row(
                    children: [
                      // Protocol Emoji / Ping Emoji
                      Text(
                        server.pingEmoji,
                        style: const TextStyle(fontSize: 28),
                      ),
                      const SizedBox(width: 12),

                      // Server Info
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            // Name + Active badge
                            Row(
                              children: [
                                Flexible(
                                  child: Text(
                                    server.name,
                                    style: TextStyle(
                                      fontWeight: FontWeight.w600,
                                      fontSize: 16,
                                      color: textSecondary(context),
                                    ),
                                    overflow: TextOverflow.ellipsis,
                                  ),
                                ),
                                if (isActive) ...[
                                  const SizedBox(width: 8),
                                  Container(
                                    padding: const EdgeInsets.symmetric(
                                      horizontal: 8,
                                      vertical: 2,
                                    ),
                                    decoration: BoxDecoration(
                                      color: accentColor(context).withOpacity(0.2),
                                      borderRadius: BorderRadius.circular(8),
                                    ),
                                    child: Text(
                                      'Active',
                                      style: TextStyle(
                                        fontSize: 10,
                                        color: accentColor(context),
                                        fontWeight: FontWeight.w600,
                                      ),
                                    ),
                                  ),
                                ],
                              ],
                            ),
                            const SizedBox(height: 4),

                            // Protocol + Address
                            Row(
                              children: [
                                Text(
                                  server.protocolEmoji,
                                  style: const TextStyle(fontSize: 12),
                                ),
                                const SizedBox(width: 4),
                                Text(
                                  server.protocolLabel,
                                  style: TextStyle(
                                    color: textMuted(context),
                                    fontSize: 11,
                                    fontWeight: FontWeight.w500,
                                  ),
                                ),
                                const SizedBox(width: 8),
                                Expanded(
                                  child: Text(
                                    server.fullAddress,
                                    style: TextStyle(
                                      color: textMuted(context).withOpacity(0.7),
                                      fontSize: 13,
                                    ),
                                    overflow: TextOverflow.ellipsis,
                                  ),
                                ),
                              ],
                            ),
                          ],
                        ),
                      ),

                      // Ping & Stats Column
                      Column(
                        crossAxisAlignment: CrossAxisAlignment.end,
                        children: [
                          // TCP Ping
                          Row(
                            children: [
                              const Icon(
                                Icons.bolt,
                                size: 12,
                                color: Colors.grey,
                              ),
                              const SizedBox(width: 4),
                              Text(
                                server.pingText,
                                style: TextStyle(
                                  fontWeight: FontWeight.w600,
                                  fontSize: 13,
                                  color: _pingColor(context, server.ping),
                                ),
                              ),
                            ],
                          ),

                          // Real Delay (if tested)
                          if (server.realDelay != null) ...[
                            const SizedBox(height: 2),
                            Row(
                              children: [
                                const Icon(
                                  Icons.auto_graph,
                                  size: 12,
                                  color: Colors.grey,
                                ),
                                const SizedBox(width: 4),
                                Text(
                                  server.realDelayText,
                                  style: TextStyle(
                                    fontWeight: FontWeight.w600,
                                    fontSize: 11,
                                    color: _pingColor(context, server.realDelay),
                                  ),
                                ),
                              ],
                            ),
                          ],

                          // Cover Traffic Info
                          if (server.coverEnabled) ...[
                            const SizedBox(height: 2),
                            Text(
                              '🎭 ${server.coverDomains.length} sites',
                              style: TextStyle(
                                fontSize: 10,
                                color: textMuted(context),
                              ),
                            ),
                          ],

                          // Last Tested
                          if (server.lastTested != null) ...[
                            const SizedBox(height: 2),
                            Text(
                              server.lastTestedText,
                              style: TextStyle(
                                fontSize: 9,
                                color: textMuted(context).withOpacity(0.5),
                              ),
                            ),
                          ],
                        ],
                      ),
                    ],
                  ),

                  const SizedBox(height: 8),
                  const Divider(height: 1, thickness: 0.5),
                  const SizedBox(height: 4),

                  // ─── Action Buttons ───
                  Row(
                    mainAxisAlignment: MainAxisAlignment.end,
                    children: [
                      // TCP Ping
                      _actionButton(
                        context,
                        Icons.bolt,
                        'TCPing',
                        () async {
                          final ping = await provider.pingServer(server);
                          provider.updateServer(server.copyWith(
                            ping: ping,
                            lastTested: DateTime.now(),
                          ));
                        },
                      ),

                      // Real Delay Test
                      _actionButton(
                        context,
                        Icons.auto_graph,
                        'Real Delay',
                        () async {
                          final delay = await provider.testRealDelay(server);
                          provider.updateServer(server.copyWith(
                            realDelay: delay,
                            lastTested: DateTime.now(),
                          ));
                        },
                      ),

                      // Share
                      _actionButton(
                        context,
                        Icons.share,
                        'Share',
                        () => Share.share(provider.exportConfig(server)),
                      ),

                      // Copy JSON
                      _actionButton(
                        context,
                        Icons.copy,
                        'Copy Config',
                        () {
                          Clipboard.setData(ClipboardData(
                            text: provider.exportConfigJson(server),
                          ));
                          ScaffoldMessenger.of(context).showSnackBar(
                            const SnackBar(content: Text('Config copied')),
                          );
                        },
                      ),

                      // Edit
                      _actionButton(
                        context,
                        Icons.edit,
                        'Edit',
                        () => _openEditServer(context, server),
                      ),

                      // Delete
                      _actionButton(
                        context,
                        Icons.delete_outline,
                        'Delete',
                        () => _confirmDelete(context, provider, server),
                        color: Colors.red.withOpacity(0.7),
                      ),
                    ],
                  ),
                ],
              ),
            ),
          ),
        );
      },
    );
  }

  // ═══════════════════════════════════════════════════════════════
  // Helper Widgets
  // ═══════════════════════════════════════════════════════════════

  Widget _actionButton(
    BuildContext context,
    IconData icon,
    String tooltip,
    VoidCallback onTap, {
    Color? color,
  }) {
    return IconButton(
      icon: Icon(
        icon,
        size: 18,
        color: color ?? textMuted(context),
      ),
      tooltip: tooltip,
      onPressed: onTap,
      visualDensity: VisualDensity.compact,
    );
  }

  // ═══════════════════════════════════════════════════════════════
  // Helper Methods
  // ═══════════════════════════════════════════════════════════════

  Color _pingColor(BuildContext context, int? ping) {
    if (ping == null) return textMuted(context);
    if (ping < 0) return Colors.red;
    if (ping < 100) return Colors.green;
    if (ping < 300) return accentColor(context);
    return Colors.red;
  }

  void _openAddServer(BuildContext context) {
    Navigator.push(
      context,
      MaterialPageRoute(builder: (_) => const AddServerScreen()),
    );
  }

  void _openEditServer(BuildContext context, ServerConfig server) {
    Navigator.push(
      context,
      MaterialPageRoute(builder: (_) => AddServerScreen(server: server)),
    );
  }

  Future<void> _importFromClipboard(
    BuildContext context,
    AppProvider provider,
  ) async {
    final data = await Clipboard.getData(Clipboard.kTextPlain);
    if (data?.text != null && data!.text!.isNotEmpty) {
      provider.importConfig(data.text!);
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Config imported')),
        );
      }
    } else {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Clipboard is empty')),
        );
      }
    }
  }

  void _confirmDelete(
    BuildContext context,
    AppProvider provider,
    ServerConfig server,
  ) {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(
          'Delete Server',
          style: TextStyle(color: textPrimary(context)),
        ),
        content: Text(
          'Delete "${server.name}"?\nThis action cannot be undone.',
          style: TextStyle(color: textSecondary(context)),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: Text(
              'Cancel',
              style: TextStyle(color: textMuted(context)),
            ),
          ),
          TextButton(
            onPressed: () {
              provider.removeServer(server.id);
              Navigator.pop(ctx);
              ScaffoldMessenger.of(context).showSnackBar(
                SnackBar(
                  content: Text('${server.name} deleted'),
                  action: SnackBarAction(
                    label: 'Undo',
                    onPressed: () => provider.addServer(server),
                  ),
                ),
              );
            },
            style: TextButton.styleFrom(foregroundColor: Colors.red),
            child: const Text('Delete'),
          ),
        ],
      ),
    );
  }
}
