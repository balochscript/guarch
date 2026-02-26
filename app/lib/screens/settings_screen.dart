import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import 'package:guarch/app.dart';
import 'package:guarch/providers/app_provider.dart';
import 'package:guarch/screens/import_screen.dart';
import 'package:guarch/screens/about_screen.dart';
import 'package:url_launcher/url_launcher.dart';

class SettingsScreen extends StatelessWidget {
  const SettingsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Consumer<AppProvider>(
      builder: (context, provider, _) {
        return Scaffold(
          appBar: AppBar(title: const Text('Settings')),
          body: ListView(
            padding: const EdgeInsets.all(16),
            children: [
              // ═══ Appearance ═══
              _sectionTitle('Appearance'),
              Card(
                child: ListTile(
                  leading: Icon(
                    provider.isDarkMode ? Icons.dark_mode : Icons.light_mode,
                    color: kGold,
                  ),
                  title: const Text('Dark Mode',
                      style: TextStyle(color: kGoldLight)),
                  trailing: Switch(
                    value: provider.isDarkMode,
                    onChanged: (_) => provider.toggleTheme(),
                  ),
                ),
              ),

              // ═══ Import / Export ═══
              const SizedBox(height: 24),
              _sectionTitle('Import / Export'),
              Card(
                child: Column(
                  children: [
                    ListTile(
                      leading: const Icon(Icons.input, color: kGold),
                      title: const Text('Import Config',
                          style: TextStyle(color: kGoldLight)),
                      subtitle: Text(
                        'From guarch://, grouk://, zhip:// link or JSON',
                        style: TextStyle(
                            color: kGold.withOpacity(0.4), fontSize: 12),
                      ),
                      trailing: Icon(Icons.arrow_forward_ios,
                          size: 16, color: kGold.withOpacity(0.4)),
                      onTap: () => Navigator.push(
                        context,
                        MaterialPageRoute(
                            builder: (_) => const ImportScreen()),
                      ),
                    ),
                    Divider(height: 1, color: kGold.withOpacity(0.1)),
                    ListTile(
                      leading:
                          const Icon(Icons.content_paste, color: kGold),
                      title: const Text('Quick Import from Clipboard',
                          style: TextStyle(color: kGoldLight)),
                      trailing: Icon(Icons.arrow_forward_ios,
                          size: 16, color: kGold.withOpacity(0.4)),
                      onTap: () =>
                          _importClipboard(context, provider),
                    ),
                  ],
                ),
              ),

              // ═══ Connection ═══
              const SizedBox(height: 24),
              _sectionTitle('Connection'),
              Card(
                child: Column(
                  children: [
                    ListTile(
                      leading: const Icon(Icons.speed, color: kGold),
                      title: const Text('Ping All Servers',
                          style: TextStyle(color: kGoldLight)),
                      trailing: Icon(Icons.arrow_forward_ios,
                          size: 16, color: kGold.withOpacity(0.4)),
                      onTap: () {
                        provider.pingAllServers();
                        ScaffoldMessenger.of(context).showSnackBar(
                          const SnackBar(
                              content: Text('Pinging all servers...')),
                        );
                      },
                    ),
                    Divider(height: 1, color: kGold.withOpacity(0.1)),
                    // ✅ جدید: نمایش تعداد سرورهای هر پروتکل
                    ListTile(
                      leading: const Icon(Icons.router, color: kGold),
                      title: const Text('Server Stats',
                          style: TextStyle(color: kGoldLight)),
                      subtitle: Text(
                        _serverStats(provider),
                        style: TextStyle(
                            color: kGold.withOpacity(0.4), fontSize: 12),
                      ),
                    ),
                  ],
                ),
              ),

              // ═══ Protocols Info ═══
              const SizedBox(height: 24),
              _sectionTitle('Protocols'),
              Card(
                child: Column(
                  children: [
                    _protocolTile(
                      '🏹',
                      'Guarch',
                      'TLS 1.3 / TCP — Maximum stealth',
                      'Cover traffic, traffic shaping, decoy server',
                    ),
                    Divider(height: 1, color: kGold.withOpacity(0.1)),
                    _protocolTile(
                      '🌩️',
                      'Grouk',
                      'Raw UDP — Maximum speed',
                      'Custom reliable UDP, AIMD congestion control',
                    ),
                    Divider(height: 1, color: kGold.withOpacity(0.1)),
                    _protocolTile(
                      '⚡',
                      'Zhip',
                      'QUIC / UDP — Balanced',
                      'HTTP/3 transport, 0-RTT, cover traffic',
                    ),
                  ],
                ),
              ),

              // ═══ About ═══
              const SizedBox(height: 24),
              _sectionTitle('About'),
              Card(
                child: Column(
                  children: [
                    ListTile(
                      leading: const Text('🎯',
                          style: TextStyle(fontSize: 24)),
                      title: const Text('About Guarch',
                          style: TextStyle(color: kGoldLight)),
                      subtitle: Text(
                        'Protocols, encryption, and anti-detection',
                        style: TextStyle(color: kGold.withOpacity(0.4)),
                      ),
                      trailing: Icon(Icons.arrow_forward_ios,
                          size: 16, color: kGold.withOpacity(0.4)),
                      onTap: () => Navigator.push(
                        context,
                        MaterialPageRoute(
                            builder: (_) => const AboutScreen()),
                      ),
                    ),
                    Divider(height: 1, color: kGold.withOpacity(0.1)),
                    ListTile(
                      leading: const Icon(Icons.code, color: kGold),
                      title: const Text('Source Code',
                          style: TextStyle(color: kGoldLight)),
                      subtitle: Text(
                        'github.com/balochscript/guarch',
                        style: TextStyle(color: kGold.withOpacity(0.4)),
                      ),
                      trailing: Icon(Icons.open_in_new,
                          size: 16, color: kGold.withOpacity(0.4)),
                      onTap: () => launchUrl(
                        Uri.parse(
                            'https://github.com/balochscript/guarch'),
                      ),
                    ),
                    Divider(height: 1, color: kGold.withOpacity(0.1)),
                    ListTile(
                      leading: const Text('📱',
                          style: TextStyle(fontSize: 24)),
                      title: const Text('Version',
                          style: TextStyle(color: kGoldLight)),
                      trailing: Text(
                        '1.0.0',
                        style:
                            TextStyle(color: kGold.withOpacity(0.5)),
                      ),
                    ),
                    Divider(height: 1, color: kGold.withOpacity(0.1)),
                    ListTile(
                      leading: const Text('🏹',
                          style: TextStyle(fontSize: 24)),
                      title: const Text('Protocols',
                          style: TextStyle(color: kGoldLight)),
                      trailing: Text(
                        'Guarch • Grouk • Zhip',
                        style:
                            TextStyle(color: kGold.withOpacity(0.5), fontSize: 12),
                      ),
                    ),
                  ],
                ),
              ),

              const SizedBox(height: 32),
            ],
          ),
        );
      },
    );
  }

  Widget _sectionTitle(String title) {
    return Padding(
      padding: const EdgeInsets.only(left: 4, bottom: 8),
      child: Text(
        title,
        style: const TextStyle(
          fontSize: 14,
          fontWeight: FontWeight.w600,
          color: kGold,
        ),
      ),
    );
  }

  Widget _protocolTile(
      String emoji, String name, String subtitle, String details) {
    return ListTile(
      leading: Text(emoji, style: const TextStyle(fontSize: 24)),
      title: Text(name, style: const TextStyle(color: kGoldLight)),
      subtitle: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(subtitle,
              style: TextStyle(
                  color: kGold.withOpacity(0.5), fontSize: 12)),
          const SizedBox(height: 2),
          Text(details,
              style: TextStyle(
                  color: kGold.withOpacity(0.3), fontSize: 11)),
        ],
      ),
      isThreeLine: true,
    );
  }

  String _serverStats(AppProvider provider) {
    if (provider.servers.isEmpty) return 'No servers configured';

    final counts = <String, int>{};
    for (final s in provider.servers) {
      counts[s.protocol] = (counts[s.protocol] ?? 0) + 1;
    }

    final parts = <String>[];
    if (counts.containsKey('guarch')) {
      parts.add('🏹 ${counts['guarch']} Guarch');
    }
    if (counts.containsKey('grouk')) {
      parts.add('🌩️ ${counts['grouk']} Grouk');
    }
    if (counts.containsKey('zhip')) {
      parts.add('⚡ ${counts['zhip']} Zhip');
    }

    return '${provider.servers.length} servers: ${parts.join(', ')}';
  }

  void _importClipboard(
      BuildContext context, AppProvider provider) async {
    final data = await Clipboard.getData(Clipboard.kTextPlain);
    if (data?.text != null && data!.text!.isNotEmpty) {
      provider.importConfig(data.text!);
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text('Config imported from clipboard'),
            backgroundColor: Colors.green,
          ),
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
}
