import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import 'package:guarch/app.dart';
import 'package:guarch/providers/app_provider.dart';

class DNSSettingsScreen extends StatelessWidget {
  const DNSSettingsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Consumer<AppProvider>(
      builder: (context, provider, _) {
        return Scaffold(
          appBar: AppBar(
            title: const Text('DNS & Security'),
            actions: [
              IconButton(
                icon: const Icon(Icons.restore),
                onPressed: () => _showResetDialog(context, provider),
                tooltip: 'Reset to Defaults',
              ),
            ],
          ),
          body: ListView(
            padding: const EdgeInsets.all(16),
            children: [
              _sectionTitle(context, 'DNS Fallback'),
              Card(
                child: Column(
                  children: [
                    SwitchListTile(
                      secondary: Icon(Icons.dns, color: accentColor(context)),
                      title: Text(
                        'Enable DNS Fallback',
                        style: TextStyle(
                          color: textSecondary(context),
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                      subtitle: Text(
                        'Fallback to DNS tunnel if TLS fails',
                        style: TextStyle(color: textMuted(context), fontSize: 13),
                      ),
                      value: provider.globalDnsEnabled,
                      onChanged: (_) => provider.toggleGlobalDns(),
                    ),
                    
                    if (provider.globalDnsEnabled) ...[
                      Divider(height: 1, color: accentColor(context).withOpacity(0.1)),
                      
                      Padding(
                        padding: const EdgeInsets.all(16),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              'Tunnel Domain',
                              style: TextStyle(
                                color: textSecondary(context),
                                fontSize: 13,
                                fontWeight: FontWeight.w600,
                              ),
                            ),
                            const SizedBox(height: 8),
                            TextField(
                              controller: TextEditingController(text: provider.globalDnsDomain),
                              decoration: const InputDecoration(
                                hintText: 'tunnel.example.com',
                                border: OutlineInputBorder(),
                              ),
                              onSubmitted: (v) => provider.setGlobalDnsDomain(v.trim()),
                            ),
                          ],
                        ),
                      ),
                      
                      Divider(height: 1, color: accentColor(context).withOpacity(0.1)),
                      
                      Padding(
                        padding: const EdgeInsets.all(16),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Row(
                              mainAxisAlignment: MainAxisAlignment.spaceBetween,
                              children: [
                                Text(
                                  'DNS Servers (${provider.globalDnsServers.length})',
                                  style: TextStyle(
                                    color: textSecondary(context),
                                    fontSize: 13,
                                    fontWeight: FontWeight.w600,
                                  ),
                                ),
                                TextButton.icon(
                                  onPressed: () => _showAddServerDialog(context, provider),
                                  icon: const Icon(Icons.add, size: 18),
                                  label: const Text('Add'),
                                ),
                              ],
                            ),
                            const SizedBox(height: 8),
                            ...provider.globalDnsServers.asMap().entries.map((entry) {
                              final index = entry.key;
                              final server = entry.value;
                              return Card(
                                margin: const EdgeInsets.only(bottom: 4),
                                child: ListTile(
                                  dense: true,
                                  leading: const Icon(Icons.dns, size: 20),
                                  title: Text(server, style: const TextStyle(fontSize: 13)),
                                  trailing: IconButton(
                                    icon: const Icon(Icons.delete, size: 18),
                                    color: Colors.red,
                                    onPressed: () => provider.removeGlobalDnsServer(index),
                                  ),
                                ),
                              );
                            }).toList(),
                          ],
                        ),
                      ),
                      
                      Divider(height: 1, color: accentColor(context).withOpacity(0.1)),
                      
                      Padding(
                        padding: const EdgeInsets.all(16),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Row(
                              mainAxisAlignment: MainAxisAlignment.spaceBetween,
                              children: [
                                Text(
                                  'Switch Threshold',
                                  style: TextStyle(color: textSecondary(context)),
                                ),
                                Container(
                                  padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                                  decoration: BoxDecoration(
                                    color: accentColor(context).withOpacity(0.1),
                                    borderRadius: BorderRadius.circular(8),
                                  ),
                                  child: Text(
                                    '${provider.globalDnsSwitchThreshold} fails',
                                    style: TextStyle(
                                      fontWeight: FontWeight.w600,
                                      color: accentColor(context),
                                    ),
                                  ),
                                ),
                              ],
                            ),
                            const SizedBox(height: 8),
                            Slider(
                              value: provider.globalDnsSwitchThreshold.toDouble(),
                              min: 1,
                              max: 10,
                              divisions: 9,
                              label: '${provider.globalDnsSwitchThreshold} fails',
                              onChanged: (v) => provider.setGlobalDnsSwitchThreshold(v.toInt()),
                            ),
                            Text(
                              'Switch to next DNS server after this many failures',
                              style: TextStyle(
                                color: textMuted(context),
                                fontSize: 11,
                              ),
                            ),
                          ],
                        ),
                      ),
                    ],
                  ],
                ),
              ),

              const SizedBox(height: 24),
              _sectionTitle(context, 'TLS Fingerprinting (UTLS)'),
              Card(
                child: Column(
                  children: [
                    SwitchListTile(
                      secondary: Icon(Icons.fingerprint, color: accentColor(context)),
                      title: Text(
                        'Enable UTLS',
                        style: TextStyle(
                          color: textSecondary(context),
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                      subtitle: Text(
                        'Mimic real browser TLS fingerprint',
                        style: TextStyle(color: textMuted(context), fontSize: 13),
                      ),
                      value: provider.globalUtlsEnabled,
                      onChanged: (_) => provider.toggleGlobalUtls(),
                    ),
                    
                    if (provider.globalUtlsEnabled) ...[
                      Divider(height: 1, color: accentColor(context).withOpacity(0.1)),
                      
                      Padding(
                        padding: const EdgeInsets.all(16),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              'Fingerprint',
                              style: TextStyle(
                                color: textSecondary(context),
                                fontSize: 13,
                                fontWeight: FontWeight.w600,
                              ),
                            ),
                            const SizedBox(height: 8),
                            DropdownButtonFormField<String>(
                              value: provider.globalUtlsFingerprint,
                              decoration: const InputDecoration(
                                border: OutlineInputBorder(),
                              ),
                              items: [
                                'chrome_auto',
                                'chrome_120',
                                'chrome_119',
                                'firefox_121',
                                'firefox_120',
                                'edge_120',
                                'safari_17',
                              ].map((fp) {
                                return DropdownMenuItem(
                                  value: fp,
                                  child: Text(fp),
                                );
                              }).toList(),
                              onChanged: (v) {
                                if (v != null) provider.setGlobalUtlsFingerprint(v);
                              },
                            ),
                          ],
                        ),
                      ),
                    ],
                  ],
                ),
              ),

              const SizedBox(height: 24),
              _sectionTitle(context, 'Packet Fragmentation'),
              Card(
                child: Column(
                  children: [
                    SwitchListTile(
                      secondary: const Text('✂️', style: TextStyle(fontSize: 20)),
                      title: Text(
                        'Enable Fragmentation',
                        style: TextStyle(
                          color: textSecondary(context),
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                      subtitle: Text(
                        'Split packets to bypass DPI',
                        style: TextStyle(color: textMuted(context), fontSize: 13),
                      ),
                      value: provider.globalFragmentEnabled,
                      onChanged: (_) => provider.toggleGlobalFragment(),
                    ),
                    
                    if (provider.globalFragmentEnabled) ...[
                      Divider(height: 1, color: accentColor(context).withOpacity(0.1)),
                      
                      Padding(
                        padding: const EdgeInsets.all(16),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Row(
                              children: [
                                Expanded(
                                  child: Column(
                                    crossAxisAlignment: CrossAxisAlignment.start,
                                    children: [
                                      Text(
                                        'Min Size',
                                        style: TextStyle(
                                          color: textSecondary(context),
                                          fontSize: 13,
                                        ),
                                      ),
                                      const SizedBox(height: 4),
                                      Text(
                                        '${provider.globalFragmentMinSize} bytes',
                                        style: TextStyle(
                                          color: accentColor(context),
                                          fontWeight: FontWeight.w600,
                                        ),
                                      ),
                                    ],
                                  ),
                                ),
                                const SizedBox(width: 16),
                                Expanded(
                                  child: Column(
                                    crossAxisAlignment: CrossAxisAlignment.start,
                                    children: [
                                      Text(
                                        'Max Size',
                                        style: TextStyle(
                                          color: textSecondary(context),
                                          fontSize: 13,
                                        ),
                                      ),
                                      const SizedBox(height: 4),
                                      Text(
                                        '${provider.globalFragmentMaxSize} bytes',
                                        style: TextStyle(
                                          color: accentColor(context),
                                          fontWeight: FontWeight.w600,
                                        ),
                                      ),
                                    ],
                                  ),
                                ),
                              ],
                            ),
                            const SizedBox(height: 16),
                            Row(
                              children: [
                                Expanded(
                                  child: Column(
                                    children: [
                                      Slider(
                                        value: provider.globalFragmentMinSize.toDouble(),
                                        min: 64,
                                        max: 512,
                                        divisions: 7,
                                        label: '${provider.globalFragmentMinSize}',
                                        onChanged: (v) {
                                          final minSize = v.toInt();
                                          final maxSize = provider.globalFragmentMaxSize;
                                          if (minSize <= maxSize) {
                                            provider.setGlobalFragmentSizes(minSize, maxSize);
                                          }
                                        },
                                      ),
                                    ],
                                  ),
                                ),
                                const SizedBox(width: 16),
                                Expanded(
                                  child: Column(
                                    children: [
                                      Slider(
                                        value: provider.globalFragmentMaxSize.toDouble(),
                                        min: 128,
                                        max: 1024,
                                        divisions: 7,
                                        label: '${provider.globalFragmentMaxSize}',
                                        onChanged: (v) {
                                          final maxSize = v.toInt();
                                          final minSize = provider.globalFragmentMinSize;
                                          if (maxSize >= minSize) {
                                            provider.setGlobalFragmentSizes(minSize, maxSize);
                                          }
                                        },
                                      ),
                                    ],
                                  ),
                                ),
                              ],
                            ),
                          ],
                        ),
                      ),
                    ],
                  ],
                ),
              ),

              const SizedBox(height: 24),
              Container(
                padding: const EdgeInsets.all(16),
                decoration: BoxDecoration(
                  color: accentColor(context).withOpacity(0.1),
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(color: accentColor(context).withOpacity(0.3)),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Icon(Icons.info_outline, color: accentColor(context), size: 20),
                        const SizedBox(width: 8),
                        Text(
                          'About DNS & Security',
                          style: TextStyle(
                            fontWeight: FontWeight.w600,
                            color: textSecondary(context),
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 12),
                    Text(
                      '• DNS Fallback: Uses DNS tunnel if TLS blocked\n'
                      '• UTLS: Makes TLS look like real browser\n'
                      '• Fragmentation: Splits packets to evade DPI\n'
                      '• DNS slower (~50Kbps) but more reliable\n'
                      '• chrome_auto recommended for UTLS\n'
                      '• Fragment 64-256 bytes works best',
                      style: TextStyle(
                        color: textMuted(context),
                        fontSize: 13,
                        height: 1.5,
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

  void _showAddServerDialog(BuildContext context, AppProvider provider) {
    final controller = TextEditingController();

    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Add DNS Server'),
        content: TextField(
          controller: controller,
          decoration: const InputDecoration(
            labelText: 'Server Address',
            hintText: '8.8.8.8:53',
            border: OutlineInputBorder(),
          ),
          autofocus: true,
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () {
              final server = controller.text.trim();
              
              if (server.isEmpty) {
                ScaffoldMessenger.of(context).showSnackBar(
                  const SnackBar(content: Text('Server cannot be empty')),
                );
                return;
              }
              
              if (!server.contains(':')) {
                ScaffoldMessenger.of(context).showSnackBar(
                  const SnackBar(content: Text('Use format: IP:PORT (e.g. 8.8.8.8:53)')),
                );
                return;
              }
              
              provider.addGlobalDnsServer(server);
              Navigator.pop(ctx);
            },
            child: const Text('Add'),
          ),
        ],
      ),
    );
  }

  void _showResetDialog(BuildContext context, AppProvider provider) {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Reset to Defaults'),
        content: const Text(
          'This will reset DNS & Security settings:\n\n'
          '• DNS: tunnel.example.com\n'
          '• Servers: 8.8.8.8, 1.1.1.1\n'
          '• UTLS: chrome_auto\n'
          '• Fragment: 64-256 bytes\n\n'
          'Continue?',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () {
              provider.resetDnsToDefaults();
              provider.resetSecurityToDefaults();
              Navigator.pop(ctx);
              ScaffoldMessenger.of(context).showSnackBar(
                const SnackBar(
                  content: Text('✅ DNS & Security reset to defaults'),
                  backgroundColor: Colors.green,
                ),
              );
            },
            child: const Text('Reset'),
          ),
        ],
      ),
    );
  }
}
