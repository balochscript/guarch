import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:battery_plus/battery_plus.dart';
import 'package:guarch/app.dart';
import 'package:guarch/providers/app_provider.dart';
import 'package:guarch/screens/about_screen.dart';
import 'package:guarch/screens/advanced_settings_screen.dart';
import 'package:guarch/screens/sni_settings_screen.dart';
import 'package:guarch/screens/cover_settings_screen.dart';
import 'package:guarch/screens/dns_settings_screen.dart';
import 'package:guarch/models/connection_state.dart';
import 'package:url_launcher/url_launcher.dart';

class SettingsScreen extends StatefulWidget {
  const SettingsScreen({super.key});

  @override
  State<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends State<SettingsScreen> {
  final Battery _battery = Battery();
  int _batteryLevel = 100;

  @override
  void initState() {
    super.initState();
    _initBattery();
  }

  Future<void> _initBattery() async {
    try {
      final level = await _battery.batteryLevel;
      if (mounted) {
        setState(() {
          _batteryLevel = level;
        });
        final provider = Provider.of<AppProvider>(context, listen: false);
        provider.setBatteryLevel(level);
      }

      _battery.onBatteryStateChanged.listen((BatteryState state) async {
        final level = await _battery.batteryLevel;
        if (mounted) {
          setState(() {
            _batteryLevel = level;
          });
          final provider = Provider.of<AppProvider>(context, listen: false);
          provider.setBatteryLevel(level);
        }
      });
    } catch (e) {
      debugPrint('Battery init failed: $e');
    }
  }

  void _onToggleSni(bool value, AppProvider provider) {
    if (value && provider.globalSniDomains.isEmpty) {
      showDialog(
        context: context,
        builder: (ctx) => AlertDialog(
          title: Row(
            children: [
              Icon(Icons.warning_amber, color: Colors.orange, size: 28),
              const SizedBox(width: 12),
              const Expanded(child: Text('No SNI Domains')),
            ],
          ),
          content: const Text(
            'SNI Rotation requires at least one domain.\n\n'
            'Add domains in SNI Protection settings first.',
            style: TextStyle(height: 1.4),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(ctx),
              child: const Text('Cancel'),
            ),
            FilledButton.icon(
              onPressed: () {
                Navigator.pop(ctx);
                Navigator.push(
                  context,
                  MaterialPageRoute(builder: (_) => const SNISettingsScreen()),
                );
              },
              icon: const Icon(Icons.settings, size: 18),
              label: const Text('Configure'),
            ),
          ],
        ),
      );
    } else {
      provider.toggleGlobalSni();
    }
  }

  void _onToggleCover(bool value, AppProvider provider) {
    if (value && provider.globalCoverDomains.isEmpty) {
      showDialog(
        context: context,
        builder: (ctx) => AlertDialog(
          title: Row(
            children: [
              Icon(Icons.warning_amber, color: Colors.orange, size: 28),
              const SizedBox(width: 12),
              const Expanded(child: Text('No Cover Domains')),
            ],
          ),
          content: const Text(
            'Cover Traffic requires at least one domain.\n\n'
            'Add domains in Cover Traffic settings first.',
            style: TextStyle(height: 1.4),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(ctx),
              child: const Text('Cancel'),
            ),
            FilledButton.icon(
              onPressed: () {
                Navigator.pop(ctx);
                Navigator.push(
                  context,
                  MaterialPageRoute(builder: (_) => const CoverSettingsScreen()),
                );
              },
              icon: const Icon(Icons.settings, size: 18),
              label: const Text('Configure'),
            ),
          ],
        ),
      );
    } else {
      provider.toggleGlobalCover();
    }
  }

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
                      leading: Icon(
                        _batteryLevel < 20
                            ? Icons.battery_alert
                            : _batteryLevel < 50
                                ? Icons.battery_3_bar
                                : _batteryLevel < 80
                                    ? Icons.battery_5_bar
                                    : Icons.battery_full,
                        color: _batteryLevel < 30
                            ? Colors.orange
                            : Colors.green,
                        size: 28,
                      ),
                      title: Text(
                        'Battery Level',
                        style: TextStyle(color: textSecondary(context)),
                      ),
                      subtitle: Text(
                        '$_batteryLevel% • ${_batteryStatus(_batteryLevel)}',
                        style: TextStyle(color: textMuted(context), fontSize: 12),
                      ),
                      trailing: Text(
                        '$_batteryLevel%',
                        style: TextStyle(
                          color: _batteryLevel < 30
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
                      secondary: Icon(
                        Icons.data_saver_on,
                        color: accentColor(context),
                        size: 28,
                      ),
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
              _sectionTitle(context, 'Connection'),
              Card(
                child: Column(
                  children: [
                    ListTile(
                      leading: Icon(Icons.shield_outlined, color: accentColor(context)),
                      title: Text(
                        'SNI Protection',
                        style: TextStyle(color: textSecondary(context)),
                      ),
                      subtitle: Text(
                        provider.globalSniEnabled
                            ? '${provider.globalSniDomains.length} domains • ${provider.globalSniMode} mode'
                            : provider.globalSniDomains.isEmpty
                                ? 'No domains - tap to configure'
                                : 'Disabled',
                        style: TextStyle(
                          color: !provider.globalSniEnabled && provider.globalSniDomains.isEmpty
                              ? Colors.orange
                              : textMuted(context),
                          fontSize: 12,
                        ),
                      ),
                      trailing: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Switch(
                            value: provider.globalSniEnabled,
                            onChanged: (value) => _onToggleSni(value, provider),
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
                            : provider.globalCoverDomains.isEmpty
                                ? 'No domains - tap to configure'
                                : 'Disabled',
                        style: TextStyle(
                          color: !provider.globalCoverEnabled && provider.globalCoverDomains.isEmpty
                              ? Colors.orange
                              : textMuted(context),
                          fontSize: 12,
                        ),
                      ),
                      trailing: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Switch(
                            value: provider.globalCoverEnabled,
                            onChanged: (value) => _onToggleCover(value, provider),
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
                      leading: Icon(
                        Icons.phone_android,
                        color: accentColor(context),
                        size: 24,
                      ),
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
                      Icons.security,
                      Colors.blue,
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
                      Icons.flash_on,
                      Colors.purple,
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
                      Icons.bolt,
                      Colors.amber,
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
                      leading: Icon(
                        Icons.info,
                        color: accentColor(context),
                        size: 28,
                      ),
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
                      leading: Icon(
                        Icons.code,
                        color: accentColor(context),
                      ),
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
                  ],
                ),
              ),

              const SizedBox(height: 32),
              Center(
                child: Column(
                  children: [
                    Text(
                      'Guarch',
                      style: TextStyle(
                        color: textMuted(context),
                        fontSize: 14,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    const SizedBox(height: 4),
                    Text(
                      'Version 1.0.1',
                      style: TextStyle(
                        color: textMuted(context).withOpacity(0.7),
                        fontSize: 12,
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
    IconData icon,
    Color iconColor,
    String name,
    String subtitle,
    String details,
  ) {
    return ListTile(
      leading: Icon(
        icon,
        color: iconColor,
        size: 28,
      ),
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
      parts.add('${counts['guarch']} Guarch');
    }
    if (counts.containsKey('grouk')) {
      parts.add('${counts['grouk']} Grouk');
    }
    if (counts.containsKey('zhip')) {
      parts.add('${counts['zhip']} Zhip');
    }

    return '${provider.servers.length} servers: ${parts.join(' • ')}';
  }

  String _batteryStatus(int level) {
    if (level < 15) return 'Critical';
    if (level < 30) return 'Low (cover reduced)';
    if (level < 50) return 'Medium';
    return 'Good';
  }
}
