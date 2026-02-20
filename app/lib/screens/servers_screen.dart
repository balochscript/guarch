import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import 'package:guarch/providers/app_provider.dart';
import 'package:guarch/models/server_config.dart';
import 'package:guarch/screens/add_server_screen.dart';
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
              IconButton(
                icon: const Icon(Icons.speed),
                tooltip: 'Ping all',
                onPressed: () => provider.pingAllServers(),
              ),
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
            backgroundColor: const Color(0xFF6C5CE7),
            child: const Icon(Icons.add, color: Colors.white),
          ),
        );
      },
    );
  }

  Widget _buildEmpty(BuildContext context) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(Icons.dns_outlined, size: 80, color: Colors.grey.shade600),
          const SizedBox(height: 16),
          Text(
            'No servers yet',
            style: Theme.of(context).textTheme.titleMedium?.copyWith(
                  color: Colors.grey,
                ),
          ),
          const SizedBox(height: 8),
          const Text(
            'Add a server or import a config',
            style: TextStyle(color: Colors.grey),
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
                ? const BorderSide(color: Color(0xFF6C5CE7), width: 2)
                : BorderSide.none,
          ),
          child: InkWell(
            borderRadius: BorderRadius.circular(16),
            onTap: () => provider.setActiveServer(server.id),
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                children: [
                  Row(
                    children: [
                      Text(server.pingEmoji,
                          style: const TextStyle(fontSize: 28)),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Row(
                              children: [
                                Text(
                                  server.name,
                                  style: const TextStyle(
                                    fontWeight: FontWeight.w600,
                                    fontSize: 16,
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
                                      color: const Color(0xFF6C5CE7)
                                          .withOpacity(0.2),
                                      borderRadius: BorderRadius.circular(8),
                                    ),
                                    child: const Text(
                                      'Active',
                                      style: TextStyle(
                                        fontSize: 10,
                                        color: Color(0xFF6C5CE7),
                                        fontWeight: FontWeight.w600,
                                      ),
                                    ),
                                  ),
                                ],
                              ],
                            ),
                            const SizedBox(height: 4),
                            Text(
                              server.fullAddress,
                              style: const TextStyle(
                                color: Colors.grey,
                                fontSize: 13,
                              ),
                            ),
                          ],
                        ),
                      ),
                      Column(
                        children: [
                          Text(
                            server.pingText,
                            style: TextStyle(
                              fontWeight: FontWeight.w600,
                              color: _pingColor(server.ping),
                            ),
                          ),
                          if (server.coverEnabled)
                            const Text(
                              '🎭 Cover',
                              style: TextStyle(fontSize: 10),
                            ),
                        ],
                      ),
                    ],
                  ),
                  const SizedBox(height: 8),
                  Row(
                    mainAxisAlignment: MainAxisAlignment.end,
                    children: [
                      _actionButton(
                        Icons.speed,
                        'Ping',
                        () async {
                          final ping = await provider.pingServer(server);
                          provider.updateServer(server.copyWith(ping: ping));
                        },
                      ),
                      _actionButton(
                        Icons.share,
                        'Share',
                        () {
                          final config = provider.exportConfig(server);
                          Share.share(config);
                        },
                      ),
                      _actionButton(
                        Icons.copy,
                        'Copy',
                        () {
                          final config = provider.exportConfigJson(server);
                          Clipboard.setData(ClipboardData(text: config));
                          ScaffoldMessenger.of(context).showSnackBar(
                            const SnackBar(content: Text('Config copied')),
                          );
                        },
                      ),
                      _actionButton(
                        Icons.edit,
                        'Edit',
                        () => _openEditServer(context, server),
                      ),
                      _actionButton(
                        Icons.delete_outline,
                        'Delete',
                        () => _confirmDelete(context, provider, server),
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

  Widget _actionButton(IconData icon, String tooltip, VoidCallback onTap) {
    return IconButton(
      icon: Icon(icon, size: 18),
      tooltip: tooltip,
      onPressed: onTap,
      visualDensity: VisualDensity.compact,
    );
  }

  Color _pingColor(int? ping) {
    if (ping == null) return Colors.grey;
    if (ping < 0) return Colors.red;
    if (ping < 100) return Colors.green;
    if (ping < 300) return Colors.orange;
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

  void _importFromClipboard(BuildContext context, AppProvider provider) async {
    final data = await Clipboard.getData(Clipboard.kTextPlain);
    if (data?.text != null && data!.text!.isNotEmpty) {
      provider.importConfig(data.text!);
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Config imported')),
        );
      }
    }
  }

  void _confirmDelete(
      BuildContext context, AppProvider provider, ServerConfig server) {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Delete Server'),
        content: Text('Delete "${server.name}"?'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancel'),
          ),
          TextButton(
            onPressed: () {
              provider.removeServer(server.id);
              Navigator.pop(ctx);
            },
            style: TextButton.styleFrom(foregroundColor: Colors.red),
            child: const Text('Delete'),
          ),
        ],
      ),
    );
  }
}
