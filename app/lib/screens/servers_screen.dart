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
              IconButton(icon: const Icon(Icons.speed), tooltip: 'Ping all', onPressed: () => provider.pingAllServers()),
              IconButton(icon: const Icon(Icons.content_paste), tooltip: 'Import from clipboard', onPressed: () => _importFromClipboard(context, provider)),
            ],
          ),
          body: provider.servers.isEmpty ? _buildEmpty(context) : _buildList(context, provider),
          floatingActionButton: FloatingActionButton(onPressed: () => _openAddServer(context), child: const Icon(Icons.add)),
        );
      },
    );
  }

  Widget _buildEmpty(BuildContext context) {
    return Center(
      child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
        Icon(Icons.dns_outlined, size: 80, color: kGold.withOpacity(0.3)),
        const SizedBox(height: 16),
        Text('No servers yet', style: TextStyle(color: kGold.withOpacity(0.5), fontSize: 18)),
        const SizedBox(height: 8),
        Text('Add a server or import a config', style: TextStyle(color: kGold.withOpacity(0.3))),
        const SizedBox(height: 24),
        FilledButton.icon(onPressed: () => _openAddServer(context), icon: const Icon(Icons.add), label: const Text('Add Server')),
      ]),
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
            side: isActive ? const BorderSide(color: kGold, width: 2) : BorderSide.none,
          ),
          child: InkWell(
            borderRadius: BorderRadius.circular(16),
            onTap: () => provider.setActiveServer(server.id),
            onLongPress: () => Navigator.push(context, MaterialPageRoute(builder: (_) => ServerDetailScreen(server: server))),
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(children: [
                Row(children: [
                  Text(server.pingEmoji, style: const TextStyle(fontSize: 28)),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                      Row(children: [
                        Text(server.name, style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 16, color: kGoldLight)),
                        if (isActive) ...[
                          const SizedBox(width: 8),
                          Container(
                            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                            decoration: BoxDecoration(color: kGold.withOpacity(0.2), borderRadius: BorderRadius.circular(8)),
                            child: const Text('Active', style: TextStyle(fontSize: 10, color: kGold, fontWeight: FontWeight.w600)),
                          ),
                        ],
                      ]),
                      const SizedBox(height: 4),
                      Row(children: [
                        Text(server.protocolEmoji, style: const TextStyle(fontSize: 12)),
                        const SizedBox(width: 4),
                        Text(server.protocolLabel, style: TextStyle(color: kGold.withOpacity(0.5), fontSize: 11, fontWeight: FontWeight.w500)),
                        const SizedBox(width: 8),
                        Text(server.fullAddress, style: TextStyle(color: kGold.withOpacity(0.4), fontSize: 13)),
                      ]),
                    ]),
                  ),
                  Column(children: [
                    Text(server.pingText, style: TextStyle(fontWeight: FontWeight.w600, color: _pingColor(server.ping))),
                    if (server.coverEnabled)
                      Text('🎭 ${server.coverDomains.length} sites', style: TextStyle(fontSize: 10, color: kGold.withOpacity(0.4))),
                  ]),
                ]),
                const SizedBox(height: 8),
                Row(mainAxisAlignment: MainAxisAlignment.end, children: [
                  _actionButton(Icons.speed, 'Ping', () async {
                    final ping = await provider.pingServer(server);
                    provider.updateServer(server.copyWith(ping: ping));
                  }),
                  _actionButton(Icons.share, 'Share', () => Share.share(provider.exportConfig(server))),
                  _actionButton(Icons.copy, 'Copy', () {
                    Clipboard.setData(ClipboardData(text: provider.exportConfigJson(server)));
                    ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('Config copied')));
                  }),
                  _actionButton(Icons.edit, 'Edit', () => _openEditServer(context, server)),
                  _actionButton(Icons.delete_outline, 'Delete', () => _confirmDelete(context, provider, server)),
                ]),
              ]),
            ),
          ),
        );
      },
    );
  }

  Widget _actionButton(IconData icon, String tooltip, VoidCallback onTap) {
    return IconButton(icon: Icon(icon, size: 18, color: kGold.withOpacity(0.6)), tooltip: tooltip, onPressed: onTap, visualDensity: VisualDensity.compact);
  }

  Color _pingColor(int? ping) {
    if (ping == null) return kGold.withOpacity(0.4);
    if (ping < 0) return Colors.red;
    if (ping < 100) return Colors.green;
    if (ping < 300) return kGold;
    return Colors.red;
  }

  void _openAddServer(BuildContext context) => Navigator.push(context, MaterialPageRoute(builder: (_) => const AddServerScreen()));
  void _openEditServer(BuildContext context, ServerConfig server) => Navigator.push(context, MaterialPageRoute(builder: (_) => AddServerScreen(server: server)));

  void _importFromClipboard(BuildContext context, AppProvider provider) async {
    final data = await Clipboard.getData(Clipboard.kTextPlain);
    if (data?.text != null && data!.text!.isNotEmpty) {
      provider.importConfig(data.text!);
      if (context.mounted) ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('Config imported')));
    }
  }

  void _confirmDelete(BuildContext context, AppProvider provider, ServerConfig server) {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text('Delete Server', style: TextStyle(color: kGold)),
        content: Text('Delete "${server.name}"?', style: TextStyle(color: kGoldLight)),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: Text('Cancel', style: TextStyle(color: kGold.withOpacity(0.5)))),
          TextButton(onPressed: () { provider.removeServer(server.id); Navigator.pop(ctx); }, style: TextButton.styleFrom(foregroundColor: Colors.red), child: const Text('Delete')),
        ],
      ),
    );
  }
}
