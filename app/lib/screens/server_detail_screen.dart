import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import 'package:guarch/app.dart';
import 'package:guarch/models/server_config.dart';
import 'package:guarch/providers/app_provider.dart';
import 'package:guarch/screens/add_server_screen.dart';
import 'package:guarch/screens/export_screen.dart';

class ServerDetailScreen extends StatefulWidget {
  final ServerConfig server;
  const ServerDetailScreen({super.key, required this.server});

  @override
  State<ServerDetailScreen> createState() => _ServerDetailScreenState();
}

class _ServerDetailScreenState extends State<ServerDetailScreen> {
  bool _showPsk = false;

  @override
  Widget build(BuildContext context) {
    final server = widget.server;

    return Scaffold(
      appBar: AppBar(
        title: Text(server.name),
        actions: [
          IconButton(icon: const Icon(Icons.edit), onPressed: () => Navigator.push(context, MaterialPageRoute(builder: (_) => AddServerScreen(server: server)))),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.all(24),
        children: [
          Center(child: Text(server.pingEmoji, style: const TextStyle(fontSize: 64))),
          const SizedBox(height: 16),
          Center(child: Text(server.name, style: TextStyle(fontSize: 24, fontWeight: FontWeight.bold, color: textSecondary(context)))),
          const SizedBox(height: 32),

          _sectionTitle(context, '🎯 Connection'),
          _infoTile(context, 'Address', server.address, Icons.dns),
          _infoTile(context, 'Port', server.port.toString(), Icons.numbers),
          _infoTile(context, 'SOCKS5 Port', server.listenPort.toString(), Icons.settings_ethernet),
          _infoTile(context, 'Ping', server.pingText, Icons.speed),

          const SizedBox(height: 24),
          _sectionTitle(context, '🔐 Security'),
          Card(
            child: ListTile(
              leading: Icon(Icons.key, size: 20, color: accentColor(context)),
              title: Text('PSK', style: TextStyle(fontSize: 13, color: textMuted(context))),
              subtitle: Text(
                _showPsk ? (server.psk.isEmpty ? 'Not set ⚠️' : server.psk) : (server.psk.isEmpty ? 'Not set ⚠️' : '••••••••••••'),
                style: TextStyle(fontFamily: 'monospace', fontSize: 12, color: server.psk.isEmpty ? Colors.red : textSecondary(context)),
              ),
              trailing: Row(mainAxisSize: MainAxisSize.min, children: [
                IconButton(icon: Icon(_showPsk ? Icons.visibility_off : Icons.visibility, size: 18, color: textMuted(context)), onPressed: () => setState(() => _showPsk = !_showPsk)),
                if (server.psk.isNotEmpty)
                  IconButton(icon: Icon(Icons.copy, size: 18, color: textMuted(context)), onPressed: () {
                    Clipboard.setData(ClipboardData(text: server.psk));
                    ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('PSK copied')));
                  }),
              ]),
            ),
          ),
          Card(
            child: ListTile(
              leading: Icon(Icons.verified_user, size: 20, color: accentColor(context)),
              title: Text('Certificate PIN', style: TextStyle(fontSize: 13, color: textMuted(context))),
              subtitle: Text(
                server.certPin != null && server.certPin!.isNotEmpty ? '${server.certPin!.substring(0, 16)}...' : 'Not set (less secure)',
                style: TextStyle(fontFamily: 'monospace', fontSize: 12, color: server.certPin != null && server.certPin!.isNotEmpty ? textSecondary(context) : Colors.orange),
              ),
            ),
          ),

          const SizedBox(height: 24),
          _sectionTitle(context, '🎭 Cover Traffic'),
          _infoTile(context, 'Status', server.coverEnabled ? 'Enabled' : 'Disabled', Icons.masks),
          _infoTile(context, 'Pattern', server.shapingPattern, Icons.pattern),
          _infoTile(context, 'Created', server.createdAt.toString().substring(0, 16), Icons.calendar_today),

          if (server.coverEnabled) ...[
            const SizedBox(height: 12),
            ...server.coverDomains.map((d) => Card(
              child: ListTile(
                leading: Icon(Icons.public, size: 20, color: accentColor(context)),
                title: Text(d.domain, style: TextStyle(color: textSecondary(context))),
                trailing: Text('${d.weight}%', style: TextStyle(color: textMuted(context))),
              ),
            )),
          ],

          const SizedBox(height: 32),
          Row(children: [
            Expanded(child: OutlinedButton.icon(
              onPressed: () async {
                final provider = context.read<AppProvider>();
                final ping = await provider.pingServer(server);
                if (context.mounted) ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(ping > 0 ? 'Ping: ${ping}ms' : 'Ping: timeout')));
              },
              icon: const Icon(Icons.speed), label: const Text('Ping'),
            )),
            const SizedBox(width: 12),
            Expanded(child: FilledButton.icon(
              onPressed: () => Navigator.push(context, MaterialPageRoute(builder: (_) => ExportScreen(server: server))),
              icon: const Icon(Icons.share), label: const Text('Export'),
            )),
          ]),
          const SizedBox(height: 12),
          OutlinedButton.icon(
            onPressed: () {
              final provider = context.read<AppProvider>();
              provider.setActiveServer(server.id);
              ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('${server.name} set as active')));
            },
            icon: const Icon(Icons.check_circle_outline), label: const Text('Set as Active Server'),
          ),

          if (server.psk.isEmpty) ...[
            const SizedBox(height: 24),
            Card(
              color: Colors.red.withOpacity(0.1),
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Row(children: [
                  const Icon(Icons.warning, color: Colors.red),
                  const SizedBox(width: 12),
                  Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                    const Text('Security Warning', style: TextStyle(fontWeight: FontWeight.w600, color: Colors.red)),
                    const SizedBox(height: 4),
                    Text('PSK is not set. Edit this server and add a PSK.', style: TextStyle(color: Colors.red.withOpacity(0.7), fontSize: 12)),
                  ])),
                ]),
              ),
            ),
          ],
          const SizedBox(height: 32),
        ],
      ),
    );
  }

  Widget _sectionTitle(BuildContext context, String title) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: Text(title, style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600, color: textPrimary(context))),
    );
  }

  Widget _infoTile(BuildContext context, String label, String value, IconData icon) {
    return Card(
      child: ListTile(
        leading: Icon(icon, size: 20, color: accentColor(context)),
        title: Text(label, style: TextStyle(fontSize: 13, color: textMuted(context))),
        trailing: Text(value, style: TextStyle(fontWeight: FontWeight.w600, color: textSecondary(context))),
      ),
    );
  }
}
