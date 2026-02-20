import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import 'package:guarch/models/server_config.dart';
import 'package:guarch/providers/app_provider.dart';
import 'package:share_plus/share_plus.dart';

class ExportScreen extends StatelessWidget {
  final ServerConfig server;

  const ExportScreen({super.key, required this.server});

  @override
  Widget build(BuildContext context) {
    final provider = context.read<AppProvider>();
    final link = provider.exportConfig(server);
    final json = provider.exportConfigJson(server);

    return Scaffold(
      appBar: AppBar(title: const Text('Export Config')),
      body: ListView(
        padding: const EdgeInsets.all(24),
        children: [
          const Text(
            '🔗 Guarch Link',
            style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600),
          ),
          const SizedBox(height: 8),
          const Text(
            'Share this link with others',
            style: TextStyle(color: Colors.grey),
          ),
          const SizedBox(height: 12),
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  SelectableText(
                    link,
                    style: const TextStyle(
                      fontFamily: 'monospace',
                      fontSize: 12,
                    ),
                  ),
                  const SizedBox(height: 12),
                  Row(
                    children: [
                      Expanded(
                        child: OutlinedButton.icon(
                          onPressed: () {
                            Clipboard.setData(ClipboardData(text: link));
                            ScaffoldMessenger.of(context).showSnackBar(
                              const SnackBar(content: Text('Link copied!')),
                            );
                          },
                          icon: const Icon(Icons.copy, size: 18),
                          label: const Text('Copy'),
                        ),
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        child: FilledButton.icon(
                          onPressed: () => Share.share(link),
                          icon: const Icon(Icons.share, size: 18),
                          label: const Text('Share'),
                          style: FilledButton.styleFrom(
                            backgroundColor: const Color(0xFF6C5CE7),
                          ),
                        ),
                      ),
                    ],
                  ),
                ],
              ),
            ),
          ),
          const SizedBox(height: 32),
          const Text(
            '📋 JSON Config',
            style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600),
          ),
          const SizedBox(height: 8),
          const Text(
            'Full configuration in JSON format',
            style: TextStyle(color: Colors.grey),
          ),
          const SizedBox(height: 12),
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  SelectableText(
                    json,
                    style: const TextStyle(
                      fontFamily: 'monospace',
                      fontSize: 11,
                    ),
                  ),
                  const SizedBox(height: 12),
                  Row(
                    children: [
                      Expanded(
                        child: OutlinedButton.icon(
                          onPressed: () {
                            Clipboard.setData(ClipboardData(text: json));
                            ScaffoldMessenger.of(context).showSnackBar(
                              const SnackBar(content: Text('JSON copied!')),
                            );
                          },
                          icon: const Icon(Icons.copy, size: 18),
                          label: const Text('Copy'),
                        ),
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        child: FilledButton.icon(
                          onPressed: () => Share.share(json),
                          icon: const Icon(Icons.share, size: 18),
                          label: const Text('Share'),
                          style: FilledButton.styleFrom(
                            backgroundColor: const Color(0xFF6C5CE7),
                          ),
                        ),
                      ),
                    ],
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}
