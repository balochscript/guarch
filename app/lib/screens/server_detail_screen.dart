import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:guarch/models/server_config.dart';
import 'package:guarch/providers/app_provider.dart';
import 'package:guarch/screens/add_server_screen.dart';
import 'package:guarch/screens/export_screen.dart';

class ServerDetailScreen extends StatelessWidget {
  final ServerConfig server;

  const ServerDetailScreen({super.key, required this.server});

  @override
  Widget build(BuildContext context) {
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
              ),
            ),
          ),
          const SizedBox(height: 32),
          _infoTile('Address', server.address, Icons.dns),
          _infoTile('Port', server.port.toString(), Icons.numbers),
          _infoTile('Ping', server.pingText, Icons.speed),
          _infoTile(
            'Cover Traffic',
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
            const SizedBox(height: 24),
            const Text(
              '🎭 Cover Domains',
              style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600),
            ),
            const SizedBox(height: 12),
            ...server.coverDomains.map(
              (d) => Card(
                child: ListTile(
                  leading: const Icon(Icons.public, size: 20),
                  title: Text(d.domain),
                  trailing: Text(
                    '${d.weight}%',
                    style: const TextStyle(color: Colors.grey),
                  ),
                ),
              ),
            ),
          ],
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
                  style: FilledButton.styleFrom(
                    backgroundColor: const Color(0xFF6C5CE7),
                  ),
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
          const SizedBox(height: 32),
        ],
      ),
    );
  }

  Widget _infoTile(String label, String value, IconData icon) {
    return Card(
      child: ListTile(
        leading: Icon(icon, size: 20, color: const Color(0xFF6C5CE7)),
        title: Text(label, style: const TextStyle(fontSize: 13, color: Colors.grey)),
        trailing: Text(
          value,
          style: const TextStyle(fontWeight: FontWeight.w600),
        ),
      ),
    );
  }
}
