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
              _sectionTitle(context, 'Appearance'),
              Card(
                child: ListTile(
                  leading: Icon(
                    provider.isDarkMode ? Icons.dark_mode : Icons.light_mode,
                    color: accentColor(context),
                  ),
                  title: Text('Dark Mode',
                      style: TextStyle(color: textSecondary(context))),
                  trailing: Switch(
                    value: provider.isDarkMode,
                    onChanged: (_) => provider.toggleTheme(),
                  ),
                ),
              ),

              // ═══ Import / Export ═══
              const SizedBox(height: 24),
              _sectionTitle(context, 'Import / Export'),
              Card(
                child: Column(
                  children: [
                    ListTile(
                      leading: Icon(Icons.input, color: accentColor(context)),
                      title: Text('Import Config',
                          style: TextStyle(color: textSecondary(context))),
                      subtitle: Text(
                        'From guarch://, grouk://, zhip:// link or JSON',
                        style: TextStyle(color: textMuted(context), fontSize: 12),
                      ),
                      trailing: Icon(Icons.arrow_forward_ios,
                          size: 16, color: textMuted(context)),
                      onTap: () => Navigator.push(
                        context,
                        MaterialPageRoute(builder: (_) => const ImportScreen()),
                      ),
                    ),
                    Divider(height: 1, color: accentColor(context).withOpacity(0.1)),
                    ListTile(
                      leading: Icon(Icons.content_paste, color: accentColor(context)),
                      title: Text('Quick Import from Clipboard',
                          style: TextStyle(color: textSecondary(context))),
                      trailing: Icon(Icons.arrow_forward_ios,
                          size: 16, color: textMuted(context)),
                      onTap: () => _importClipboard(context, provider),
                    ),
                  ],
                ),
              ),

              // ═══ Connection ═══
              const SizedBox(height: 24),
              _sectionTitle(context, 'Connection'),
              Card(
                child: Column(
                  children: [
                    ListTile(
                      leading: Icon(Icons.speed, color: accentColor(context)),
                      title: Text('Ping All Servers',
                          style: TextStyle(color: textSecondary(context))),
                      trailing: Icon(Icons.arrow_forward_ios,
                          size: 16, color: textMuted(context)),
                      onTap: () {
                        provider.pingAllServers();
                        ScaffoldMessenger.of(context).showSnackBar(
                          const SnackBar(content: Text('Pinging all servers...')),
                        );
                      },
                    ),
                    Divider(height: 1, color: accentColor(context).withOpacity(0.1)),
                    ListTile(
                      leading: Icon(Icons.router, color: accentColor(context)),
                      title: Text('Server Stats',
                          style: TextStyle(color: textSecondary(context))),
                      subtitle: Text(
                        _serverStats(provider),
                        style: TextStyle(color: textMuted(context), fontSize: 12),
                      ),
                    ),
                  ],
                ),
              ),

              // ═══ VPN Mode ═══
              const SizedBox(height: 24),
              _sectionTitle(context, 'VPN Mode'),
              Card(
                child: Column(
                  children: [
                    ListTile(
                      leading: Icon(Icons.vpn_lock, color: accentColor(context)),
                      title: Text('System-wide VPN',
                          style: TextStyle(color: textSecondary(context))),
                      subtitle: Text(
                        'Routes all device traffic through tunnel',
                        style: TextStyle(color: textMuted(context), fontSize: 12),
                      ),
                      trailing: Icon(Icons.check_circle, 
                          color: Colors.green, size: 20),
                    ),
                    Divider(height: 1, color: accentColor(context).withOpacity(0.1)),
                    ListTile(
                      leading: const Text('📱', style: TextStyle(fontSize: 20)),
                      title: Text('Platform',
                          style: TextStyle(color: textSecondary(context))),
                      trailing: Text(
                        'Android (iOS coming soon)',
                        style: TextStyle(color: textMuted(context), fontSize: 12),
                      ),
                    ),
                  ],
                ),
              ),

              // ═══ Protocols Info ═══
              const SizedBox(height: 24),
              _sectionTitle(context, 'Protocols'),
              Card(
                child: Column(
                  children: [
                    _protocolTile(context,
                      '🏹', 'Guarch',
                      'TLS 1.3 / TCP — Maximum stealth',
                      'Cover traffic, traffic shaping, decoy server',
                    ),
                    Divider(height: 1, color: accentColor(context).withOpacity(0.1)),
                    _protocolTile(context,
                      '🌩️', 'Grouk',
                      'Raw UDP — Maximum speed',
                      'Custom reliable UDP, AIMD congestion control',
                    ),
                    Divider(height: 1, color: accentColor(context).withOpacity(0.1)),
                    _protocolTile(context,
                      '⚡', 'Zhip',
                      'QUIC / UDP — Balanced',
                      'HTTP/3 transport, 0-RTT, cover traffic',
                    ),
                  ],
                ),
              ),

              // ═══ About ═══
              const SizedBox(height: 24),
              _sectionTitle(context, 'About'),
              Card(
                child: Column(
                  children: [
                    ListTile(
                      leading: const Text('🎯', style: TextStyle(fontSize: 24)),
                      title: Text('About Guarch',
                          style: TextStyle(color: textSecondary(context))),
                      subtitle: Text(
                        'Protocols, encryption, and anti-detection',
                        style: TextStyle(color: textMuted(context)),
                      ),
                      trailing: Icon(Icons.arrow_forward_ios,
                          size: 16, color: textMuted(context)),
                      onTap: () => Navigator.push(
                        context,
                        MaterialPageRoute(builder: (_) => const AboutScreen()),
                      ),
                    ),
                    Divider(height: 1, color: accentColor(context).withOpacity(0.1)),
                    ListTile(
                      leading: Icon(Icons.code, color: accentColor(context)),
                      title: Text('Source Code',
                          style: TextStyle(color: textSecondary(context))),
                      subtitle: Text(
                        'github.com/balochscript/guarch',
                        style: TextStyle(color: textMuted(context)),
                      ),
                      trailing: Icon(Icons.open_in_new,
                          size: 16, color: textMuted(context)),
                      onTap: () => launchUrl(
                        Uri.parse('https://github.com/balochscript/guarch'),
                      ),
                    ),
                    Divider(height: 1, color: accentColor(context).withOpacity(0.1)),
                    ListTile(
                      leading: const Text('📱', style: TextStyle(fontSize: 24)),
                      title: Text('Version',
                          style: TextStyle(color: textSecondary(context))),
                      trailing: Text('1.0.0',
                          style: TextStyle(color: textMuted(context))),
                    ),
                    Divider(height: 1, color: accentColor(context).withOpacity(0.1)),
                    ListTile(
                      leading: const Text('🏹', style: TextStyle(fontSize: 24)),
                      title: Text('Protocols',
                          style: TextStyle(color: textSecondary(context))),
                      trailing: Text(
                        'Guarch • Grouk • Zhip',
                        style: TextStyle(color: textMuted(context), fontSize: 12),
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

  Widget _sectionTitle(BuildContext context, String title) {
    return Padding(
      padding: const EdgeInsets.only(left: 4, bottom: 8),
      child: Text(
        title,
        style: TextStyle(
          fontSize: 14,
          fontWeight: FontWeight.w600,
          color: textPrimary(context),
        ),
      ),
    );
  }

  Widget _protocolTile(BuildContext context,
      String emoji, String name, String subtitle, String details) {
    return ListTile(
      leading: Text(emoji, style: const TextStyle(fontSize: 24)),
      title: Text(name, style: TextStyle(color: textSecondary(context))),
      subtitle: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(subtitle,
              style: TextStyle(color: textMuted(context), fontSize: 12)),
          const SizedBox(height: 2),
          Text(details,
              style: TextStyle(
                  color: textMuted(context).withOpacity(0.6), fontSize: 11)),
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
    if (counts.containsKey('guarch')) parts.add('🏹 ${counts['guarch']} Guarch');
    if (counts.containsKey('grouk')) parts.add('🌩️ ${counts['grouk']} Grouk');
    if (counts.containsKey('zhip')) parts.add('⚡ ${counts['zhip']} Zhip');

    return '${provider.servers.length} servers: ${parts.join(', ')}';
  }

  void _importClipboard(BuildContext context, AppProvider provider) async {
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
