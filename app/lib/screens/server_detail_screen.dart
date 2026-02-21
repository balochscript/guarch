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
          IconButton(
            icon: const Icon(Icons.edit),
            onPressed: () => Navigator.push(
              context,
              MaterialPageRoute(
                builder: (_) => AddServerScreen(server: server),
              ),
            ),
          ),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.all(24),
        children: [
          Center(
            child: Text(
              server.pingEmoji,
              style: const TextStyle(fontSize: 64),
            ),
          ),
          const SizedBox(height: 16),
          Center(
            child: Text(
              server.name,
              style: const TextStyle(
                fontSize: 24,
                fontWeight: FontWeight.bold,
                color: kGoldLight,
              ),
            ),
          ),
          const SizedBox(height: 32),

          // Connection Info
          _sectionTitle('🎯 Connection'),
          _infoTile('Address', server.address, Icons.dns),
          _infoTile('Port', server.port.toString(), Icons.numbers),
          _infoTile('SOCKS5 Port', server.listenPort.toString(), Icons.settings_ethernet),
          _infoTile('Ping', server.pingText, Icons.speed),

          const SizedBox(height: 24),

          // Security Info
          _sectionTitle('🔐 Security'),
          Card(
            child: ListTile(
              leading: const Icon(Icons.key, size: 20, color: kGold),
              title: const Text('PSK',
                  style: TextStyle(fontSize: 13, color: Colors.grey)),
              subtitle: Text(
                _showPsk
                    ? (server.psk.isEmpty ? 'Not set ⚠️' : server.psk)
                    : (server.psk.isEmpty ? 'Not set ⚠️' : '••••••••••••'),
                style: TextStyle(
                  fontFamily: 'monospace',
                  fontSize: 12,
                  color: server.psk.isEmpty ? Colors.red : kGoldLight,
                ),
              ),
              trailing: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  IconButton(
                    icon: Icon(
                      _showPsk ? Icons.visibility_off : Icons.visibility,
                      size: 18,
                      color: kGold.withOpacity(0.5),
                    ),
                    onPressed: () => setState(() => _showPsk = !_showPsk),
                  ),
                  if (server.psk.isNotEmpty)
                    IconButton(
                      icon: Icon(Icons.copy, size: 18,
                          color: kGold.withOpacity(0.5)),
                      onPressed: () {
                        Clipboard.setData(ClipboardData(text: server.psk));
                        ScaffoldMessenger.of(context).showSnackBar(
                          const SnackBar(content: Text('PSK copied')),
                        );
                      },
                    ),
                ],
              ),
            ),
          ),
          Card(
            child: ListTile(
              leading: const Icon(Icons.verified_user, size: 20, color: kGold),
              title: const Text('Certificate PIN',
                  style: TextStyle(fontSize: 13, color: Colors.grey)),
              subtitle: Text(
                server.certPin != null && server.certPin!.isNotEmpty
                    ? '${server.certPin!.substring(0, 16)}...'
                    : 'Not set (less secure)',
                style: TextStyle(
                  fontFamily: 'monospace',
                  fontSize: 12,
                  color: server.certPin != null && server.certPin!.isNotEmpty
                      ? kGoldLight
                      : Colors.orange,
                ),
              ),
            ),
          ),

          const SizedBox(height: 24),

          // Cover Traffic
          _sectionTitle('🎭 Cover Traffic'),
          _infoTile(
            'Status',
            server.coverEnabled ? 'Enabled' : 'Disabled',
            Icons.masks,
          ),
          _infoTile('Pattern', server.shapingPattern, Icons.pattern),
          _infoTile(
            'Created',
            server.createdAt.toString().substring(0, 16),
            Icons.calendar_today,
          ),

          if (server.coverEnabled) ...[
            const SizedBox(height: 12),
            ...server.coverDomains.map(
              (d) => Card(
                child: ListTile(
                  leading: const Icon(Icons.public, size: 20, color: kGold),
                  title: Text(d.domain, style: const TextStyle(color: kGoldLight)),
                  trailing: Text(
                    '${d.weight}%',
                    style: TextStyle(color: kGold.withOpacity(0.5)),
                  ),
                ),
              ),
            ),
          ],

          // Actions
          const SizedBox(height: 32),
          Row(
            children: [
              Expanded(
                child: OutlinedButton.icon(
                  onPressed: () async {
                    final provider = context.read<AppProvider>();
                    final ping = await provider.pingServer(server);
                    if (context.mounted) {
                      ScaffoldMessenger.of(context).showSnackBar(
                        SnackBar(
                          content: Text(
                            ping > 0 ? 'Ping: ${ping}ms' : 'Ping: timeout',
                          ),
                        ),
                      );
                    }
                  },
                  icon: const Icon(Icons.speed),
                  label: const Text('Ping'),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: FilledButton.icon(
                  onPressed: () => Navigator.push(
                    context,
                    MaterialPageRoute(
                      builder: (_) => ExportScreen(server: server),
                    ),
                  ),
                  icon: const Icon(Icons.share),
                  label: const Text('Export'),
                ),
              ),
            ],
          ),
          const SizedBox(height: 12),
          OutlinedButton.icon(
            onPressed: () {
              final provider = context.read<AppProvider>();
              provider.setActiveServer(server.id);
              ScaffoldMessenger.of(context).showSnackBar(
                SnackBar(content: Text('${server.name} set as active')),
              );
            },
            icon: const Icon(Icons.check_circle_outline),
            label: const Text('Set as Active Server'),
          ),

          // Security Warning
          if (server.psk.isEmpty) ...[
            const SizedBox(height: 24),
            Card(
              color: Colors.red.withOpacity(0.1),
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Row(
                  children: [
                    const Icon(Icons.warning, color: Colors.red),
                    const SizedBox(width: 12),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          const Text(
                            'Security Warning',
                            style: TextStyle(
                              fontWeight: FontWeight.w600,
                              color: Colors.red,
                            ),
                          ),
                          const SizedBox(height: 4),
                          Text(
                            'PSK is not set. Connection will not be secure. '
                            'Edit this server and add a PSK.',
                            style: TextStyle(
                              color: Colors.red.withOpacity(0.7),
                              fontSize: 12,
                            ),
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ],

          const SizedBox(height: 32),
        ],
      ),
    );
  }

  Widget _sectionTitle(String title) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: Text(
        title,
        style: const TextStyle(
          fontSize: 18,
          fontWeight: FontWeight.w600,
          color: kGold,
        ),
      ),
    );
  }

  Widget _infoTile(String label, String value, IconData icon) {
    return Card(
      child: ListTile(
        leading: Icon(icon, size: 20, color: kGold),
        title: Text(label,
            style: const TextStyle(fontSize: 13, color: Colors.grey)),
        trailing: Text(
          value,
          style: const TextStyle(
            fontWeight: FontWeight.w600,
            color: kGoldLight,
          ),
        ),
      ),
    );
  }
}
