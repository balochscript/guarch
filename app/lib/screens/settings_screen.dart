import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import 'package:guarch/app.dart';
import 'package:guarch/providers/app_provider.dart';
import 'package:guarch/screens/import_screen.dart';
import 'package:guarch/screens/about_screen.dart';
import 'package:guarch/screens/advanced_settings_screen.dart';
import 'package:guarch/screens/sni_settings_screen.dart';
import 'package:guarch/screens/cover_settings_screen.dart';
import 'package:guarch/screens/dns_settings_screen.dart';
import 'package:guarch/models/connection_state.dart';
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
              _sectionTitle(context, 'Appearance'),
              Card(
                child: Column(
                  children: [
                    ListTile(
                      leading: Icon(
                        provider.isDarkMode ? Icons.dark_mode : Icons.light_mode,
                        color: accentColor(context),
                      ),
                      title: Text(
                        'Dark Mode',
                        style: TextStyle(color: textSecondary(context)),
                      ),
                      trailing: Switch(
                        value: provider.isDarkMode,
                        onChanged: (_) => provider.toggleTheme(),
                      ),
                    ),
                    Divider(
                      height: 1,
                      color: accentColor(context).withOpacity(0.1),
                    ),
                    ListTile(
                      leading: Icon(
                        Icons.bug_report,
                        color: provider.debugModeEnabled
                            ? Colors.orange
                            : accentColor(context),
                      ),
                      title: Text(
                        'Debug Mode',
                        style: TextStyle(color: textSecondary(context)),
                      ),
                      subtitle: Text(
                        provider.debugModeEnabled
                            ? 'Debug button visible on home screen'
                            : 'Debug button hidden',
                        style: TextStyle(color: textMuted(context), fontSize: 12),
                      ),
                      trailing: Switch(
                        value: provider.debugModeEnabled,
                        onChanged: (_) => provider.toggleDebugMode(),
                      ),
                    ),
                  ],
                ),
              ),

              const SizedBox(height: 24),
              _sectionTitle(context, 'Battery & Data'),
              Card(
                child: Column(
                  children: [
                    ListTile(
                      leading: const Text('🔋', style: TextStyle(fontSize: 24)),
                      title: Text(
                        'Battery Level',
                        style: TextStyle(color: textSecondary(context)),
                      ),
                      subtitle: Text(
                        '${provider.batteryLevel}% • ${_batteryStatus(provider.batteryLevel)}',
                        style: TextStyle(color: textMuted(context), fontSize: 12),
                      ),
                      trailing: Text(
                        '${provider.batteryLevel}%',
                        style: TextStyle(
                          color: provider.batteryLevel < 30
                              ? Colors.orange
                              : Colors.green,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                    ),
                    Divider(
                      height: 1,
                      color: accentColor(context).withOpacity(0.1),
                    ),
                    SwitchListTile(
                      secondary: const Text('💾', style: TextStyle(fontSize: 24)),
                      title: Text(
                        'Data Saver Mode',
                        style: TextStyle(color: textSecondary(context)),
                      ),
                      subtitle: Text(
                        'Halve cover traffic to save bandwidth',
                        style: TextStyle(color: textMuted(context), fontSize: 12),
                      ),
                      value: provider.dataSaverEnabled,
                      onChanged: (_) => provider.toggleDataSaver(),
                    ),
                  ],
                ),
              ),

              const SizedBox(height: 24),
              _sectionTitle(context, 'Import / Export'),
              Card(
                child: Column(
                  children: [
                    ListTile(
                      leading: Icon(Icons.input, color: accentColor(context)),
                      title: Text(
                        'Import Config',
                        style: TextStyle(color: textSecondary(context)),
                      ),
                      subtitle: Text(
                        'From guarch://, grouk://, zhip:// link or JSON',
                        style: TextStyle(color: textMuted(context), fontSize: 12),
                      ),
                      trailing: Icon(
                        Icons.arrow_forward_ios,
                        size: 16,
                        color: textMuted(context),
                      ),
                      onTap: () => Navigator.push(
                        context,
                        MaterialPageRoute(builder: (_) => const ImportScreen()),
                      ),
                    ),
                    Divider(
                      height: 1,
                      color: accentColor(context).withOpacity(0.1),
                    ),
                    ListTile(
                      leading: Icon(
                        Icons.content_paste,
                        color: accentColor(context),
                      ),
                      title: Text(
                        'Quick Import from Clipboard',
                        style: TextStyle(color: textSecondary(context)),
                      ),
                      trailing: Icon(
                        Icons.arrow_forward_ios,
                        size: 16,
                        color: textMuted(context),
                      ),
                      onTap: () => _importClipboard(context, provider),
                    ),
                  ],
                ),
              ),

              const SizedBox(height: 24),
              _sectionTitle(context, 'Connection'),
              Card(
                child: Column(
                  children: [
                    ListTile(
                      leading: Icon(Icons.shield, color: accentColor(context)),
                      title: Text(
                        'SNI Protection',
                        style: TextStyle(color: textSecondary(context)),
                      ),
                      subtitle: Text(
                        provider.globalSniEnabled
                            ? '${provider.globalSniDomains.length} domains • ${provider.globalSniMode} mode'
                            : 'Disabled',
                        style: TextStyle(color: textMuted(context), fontSize: 12),
                      ),
                      trailing: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Switch(
                            value: provider.globalSniEnabled,
                            onChanged: (_) => provider.toggleGlobalSni(),
                          ),
                          const SizedBox(width: 8),
                          Icon(
                            Icons.arrow_forward_ios,
                            size: 16,
                            color: textMuted(context),
                          ),
                        ],
                      ),
                      onTap: () => Navigator.push(
                        context,
                        MaterialPageRoute(builder: (_) => const SNISettingsScreen()),
                      ),
                    ),
                    Divider(
                      height: 1,
                      color: accentColor(context).withOpacity(0.1),
                    ),
                    ListTile(
                      leading: Icon(Icons.theater_comedy, color: accentColor(context)),
                      title: Text(
                        'Cover Traffic',
                        style: TextStyle(color: textSecondary(context)),
                      ),
                      subtitle: Text(
                        provider.globalCoverEnabled
                            ? '${provider.globalCoverDomains.length} domains • ${provider.globalCoverMode} mode'
                            : 'Disabled',
                        style: TextStyle(color: textMuted(context), fontSize: 12),
                      ),
                      trailing: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Switch(
                            value: provider.globalCoverEnabled,
                            onChanged: (_) => provider.toggleGlobalCover(),
                          ),
                          const SizedBox(width: 8),
                          Icon(
                            Icons.arrow_forward_ios,
                            size: 16,
                            color: textMuted(context),
                          ),
                        ],
                      ),
                      onTap: () => Navigator.push(
                        context,
                        MaterialPageRoute(builder: (_) => const CoverSettingsScreen()),
                      ),
                    ),
                    Divider(
                      height: 1,
                      color: accentColor(context).withOpacity(0.1),
                    ),
                    ListTile(
                      leading: Icon(Icons.dns, color: accentColor(context)),
                      title: Text(
                        'DNS & Security',
                        style: TextStyle(color: textSecondary(context)),
                      ),
                      subtitle: Text(
                        provider.globalDnsEnabled
                            ? 'DNS: ${provider.globalDnsDomain} • UTLS: ${provider.globalUtlsEnabled ? "ON" : "OFF"}'
                            : 'DNS disabled • UTLS: ${provider.globalUtlsEnabled ? "ON" : "OFF"}',
                        style: TextStyle(color: textMuted(context), fontSize: 12),
                      ),
                      trailing: Icon(
                        Icons.arrow_forward_ios,
                        size: 16,
                        color: textMuted(context),
                      ),
                      onTap: () => Navigator.push(
                        context,
                        MaterialPageRoute(builder: (_) => const DNSSettingsScreen()),
                      ),
                    ),
                    Divider(
                      height: 1,
                      color: accentColor(context).withOpacity(0.1),
                    ),
                    ListTile(
                      leading: Icon(Icons.speed, color: accentColor(context)),
                      title: Text(
                        'Ping All Servers',
                        style: TextStyle(color: textSecondary(context)),
                      ),
                      trailing: Icon(
                        Icons.arrow_forward_ios,
                        size: 16,
                        color: textMuted(context),
                      ),
                      onTap: () {
                        provider.pingAllServers();
                        ScaffoldMessenger.of(context).showSnackBar(
                          const SnackBar(content: Text('Pinging all servers...')),
                        );
                      },
                    ),
                    Divider(
                      height: 1,
                      color: accentColor(context).withOpacity(0.1),
                    ),
                    ListTile(
                      leading: Icon(Icons.router, color: accentColor(context)),
                      title: Text(
                        'Server Stats',
                        style: TextStyle(color: textSecondary(context)),
                      ),
                      subtitle: Text(
                        _serverStats(provider),
                        style: TextStyle(color: textMuted(context), fontSize: 12),
                      ),
                    ),
                    Divider(
                      height: 1,
                      color: accentColor(context).withOpacity(0.1),
                    ),
                    ListTile(
                      leading: Icon(Icons.tune, color: accentColor(context)),
                      title: Text(
                        'Advanced Settings',
                        style: TextStyle(color: textSecondary(context)),
                      ),
                      subtitle: Text(
                        'Timeout, retry, buffer size, etc.',
                        style: TextStyle(color: textMuted(context), fontSize: 12),
                      ),
                      trailing: Icon(
                        Icons.arrow_forward_ios,
                        size: 16,
                        color: textMuted(context),
                      ),
                      onTap: () => Navigator.push(
                        context,
                        MaterialPageRoute(
                          builder: (_) => const AdvancedSettingsScreen(),
                        ),
                      ),
                    ),
                  ],
                ),
              ),

              const SizedBox(height: 24),
              _sectionTitle(context, 'Connection Mode'),
              Card(
                child: Column(
                  children: [
                    SwitchListTile(
                      secondary: Icon(
                        provider.vpnModeEnabled ? Icons.vpn_lock : Icons.settings_ethernet,
                        color: accentColor(context),
                      ),
                      title: Text(
                        provider.vpnModeEnabled ? 'VPN Mode (System-wide)' : 'Proxy Mode (SOCKS5)',
                        style: TextStyle(color: textSecondary(context)),
                      ),
                      subtitle: Text(
                        provider.vpnModeEnabled
                            ? 'All device traffic routes through VPN tunnel'
                            : 'Only proxy-aware apps • Listening on 127.0.0.1:7070',
                        style: TextStyle(color: textMuted(context), fontSize: 12),
                      ),
                      value: provider.vpnModeEnabled,
                      onChanged: (_) {
                        if (provider.status == VpnStatus.connected) {
                          showDialog(
                            context: context,
                            builder: (ctx) => AlertDialog(
                              title: const Text('Reconnection Required'),
                              content: const Text(
                                'Changing connection mode requires disconnecting first. Continue?',
                              ),
                              actions: [
                                TextButton(
                                  onPressed: () => Navigator.pop(ctx),
                                  child: const Text('Cancel'),
                                ),
                                TextButton(
                                  onPressed: () {
                                    Navigator.pop(ctx);
                                    provider.disconnect().then((_) {
                                      provider.toggleVpnMode();
                                    });
                                  },
                                  child: const Text('Disconnect & Change'),
                                ),
                              ],
                            ),
                          );
                        } else {
                          provider.toggleVpnMode();
                        }
                      },
                    ),
                    
                    if (!provider.vpnModeEnabled)
                      Container(
                        margin: const EdgeInsets.all(12),
                        padding: const EdgeInsets.all(12),
                        decoration: BoxDecoration(
                          color: Colors.orange.withOpacity(0.1),
                          borderRadius: BorderRadius.circular(8),
                          border: Border.all(color: Colors.orange.withOpacity(0.3)),
                        ),
                        child: Row(
                          children: [
                            const Icon(Icons.info_outline, color: Colors.orange, size: 20),
                            const SizedBox(width: 12),
                            Expanded(
                              child: Text(
                                'Apps must support SOCKS5 proxy (127.0.0.1:7070) to use this mode',
                                style: TextStyle(
                                  color: textSecondary(context),
                                  fontSize: 12,
                                ),
                              ),
                            ),
                          ],
                        ),
                      ),
                    
                    Divider(
                      height: 1,
                      color: accentColor(context).withOpacity(0.1),
                    ),
                    
                    ListTile(
                      leading: const Text('📱', style: TextStyle(fontSize: 20)),
                      title: Text(
                        'Platform',
                        style: TextStyle(color: textSecondary(context)),
                      ),
                      trailing: Text(
                        'Android (iOS coming soon)',
                        style: TextStyle(color: textMuted(context), fontSize: 12),
                      ),
                    ),
                  ],
                ),
              ),

              const SizedBox(height: 24),
              _sectionTitle(context, 'Protocols'),
              Card(
                child: Column(
                  children: [
                    _protocolTile(
                      context,
                      '🏹',
                      'Guarch',
                      'TLS 1.3 / TCP — Maximum stealth',
                      'Cover traffic, traffic shaping, decoy server',
                    ),
                    Divider(
                      height: 1,
                      color: accentColor(context).withOpacity(0.1),
                    ),
                    _protocolTile(
                      context,
                      '🌩️',
                      'Grouk',
                      'Raw UDP — Maximum speed',
                      'Custom reliable UDP, AIMD congestion control',
                    ),
                    Divider(
                      height: 1,
                      color: accentColor(context).withOpacity(0.1),
                    ),
                    _protocolTile(
                      context,
                      '⚡',
                      'Zhip',
                      'QUIC / UDP — Balanced',
                      'HTTP/3 transport, 0-RTT, cover traffic',
                    ),
                  ],
                ),
              ),

              const SizedBox(height: 24),
              _sectionTitle(context, 'About'),
              Card(
                child: Column(
                  children: [
                    ListTile(
                      leading: const Text('🎯', style: TextStyle(fontSize: 24)),
                      title: Text(
                        'About Guarch',
                        style: TextStyle(color: textSecondary(context)),
                      ),
                      subtitle: Text(
                        'Protocols, encryption, and anti-detection',
                        style: TextStyle(color: textMuted(context)),
                      ),
                      trailing: Icon(
                        Icons.arrow_forward_ios,
                        size: 16,
                        color: textMuted(context),
                      ),
                      onTap: () => Navigator.push(
                        context,
                        MaterialPageRoute(builder: (_) => const AboutScreen()),
                      ),
                    ),
                    Divider(
                      height: 1,
                      color: accentColor(context).withOpacity(0.1),
                    ),
                    ListTile(
                      leading: Icon(Icons.code, color: accentColor(context)),
                      title: Text(
                        'Source Code',
                        style: TextStyle(color: textSecondary(context)),
                      ),
                      subtitle: Text(
                        'github.com/balochscript/guarch',
                        style: TextStyle(color: textMuted(context)),
                      ),
                      trailing: Icon(
                        Icons.open_in_new,
                        size: 16,
                        color: textMuted(context),
                      ),
                      onTap: () => launchUrl(
                        Uri.parse('https://github.com/balochscript/guarch'),
                      ),
                    ),
                    Divider(
                      height: 1,
                      color: accentColor(context).withOpacity(0.1),
                    ),
                    ListTile(
                      leading: const Text('📱', style: TextStyle(fontSize: 24)),
                      title: Text(
                        'Version',
                        style: TextStyle(color: textSecondary(context)),
                      ),
                      trailing: Text(
                        '1.0.1',
                        style: TextStyle(color: textMuted(context)),
                      ),
                    ),
                    Divider(
                      height: 1,
                      color: accentColor(context).withOpacity(0.1),
                    ),
                    ListTile(
                      leading: const Text('🏹', style: TextStyle(fontSize: 24)),
                      title: Text(
                        'Protocols',
                        style: TextStyle(color: textSecondary(context)),
                      ),
                      trailing: Text(
                        'Guarch • Grouk • Zhip',
                        style: TextStyle(
                          color: textMuted(context),
                          fontSize: 12,
                        ),
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

  Widget _protocolTile(
    BuildContext context,
    String emoji,
    String name,
    String subtitle,
    String details,
  ) {
    return ListTile(
      leading: Text(emoji, style: const TextStyle(fontSize: 24)),
      title: Text(name, style: TextStyle(color: textSecondary(context))),
      subtitle: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            subtitle,
            style: TextStyle(color: textMuted(context), fontSize: 12),
          ),
          const SizedBox(height: 2),
          Text(
            details,
            style: TextStyle(
              color: textMuted(context).withOpacity(0.6),
              fontSize: 11,
            ),
          ),
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

  String _batteryStatus(int level) {
    if (level < 15) return 'Critical';
    if (level < 30) return 'Low (cover reduced)';
    if (level < 50) return 'Medium';
    return 'Good';
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
