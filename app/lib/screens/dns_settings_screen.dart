// app/lib/screens/dns_settings_screen.dart
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:guarch/models/server_config.dart';
import 'package:guarch/providers/app_provider.dart';
import 'package:guarch/app.dart';

class DnsSettingsScreen extends StatefulWidget {
  final ServerConfig server;

  const DnsSettingsScreen({super.key, required this.server});

  @override
  State<DnsSettingsScreen> createState() => _DnsSettingsScreenState();
}

class _DnsSettingsScreenState extends State<DnsSettingsScreen> {
  late TextEditingController _domainController;
  late TextEditingController _timeoutController;
  late TextEditingController _thresholdController;
  late List<TextEditingController> _serverControllers;
  
  late bool _enabled;
  late bool _autoSwitch;
  late String _mode;

  @override
  void initState() {
    super.initState();
    
    _enabled = widget.server.dnsFallbackEnabled;
    _autoSwitch = widget.server.dnsFallbackMode == 'auto';
    _mode = widget.server.dnsFallbackMode;
    
    _domainController = TextEditingController(text: widget.server.dnsFallbackDomain);
    _timeoutController = TextEditingController(text: widget.server.dnsFallbackTimeout.toString());
    _thresholdController = TextEditingController(text: widget.server.dnsFallbackSwitchThreshold.toString());
    
    _serverControllers = widget.server.dnsFallbackServers
        .map((s) => TextEditingController(text: s))
        .toList();
  }

  @override
  void dispose() {
    _domainController.dispose();
    _timeoutController.dispose();
    _thresholdController.dispose();
    for (var c in _serverControllers) {
      c.dispose();
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('DNS Fallback Settings'),
        actions: [
          IconButton(
            icon: const Icon(Icons.save),
            onPressed: _saveSettings,
          ),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          // ═══════════════════════════════════════════════════════════
          // Enable/Disable
          // ═══════════════════════════════════════════════════════════
          Card(
            child: SwitchListTile(
              secondary: const Text('🌐', style: TextStyle(fontSize: 24)),
              title: Text(
                'Enable DNS Fallback',
                style: TextStyle(
                  color: textSecondary(context),
                  fontWeight: FontWeight.w600,
                ),
              ),
              subtitle: Text(
                'Automatically switch to DNS tunnel when TLS fails',
                style: TextStyle(color: textMuted(context), fontSize: 12),
              ),
              value: _enabled,
              onChanged: (v) => setState(() => _enabled = v),
            ),
          ),

          if (_enabled) ...[
            const SizedBox(height: 24),

            // ═══════════════════════════════════════════════════════════
            // Basic Settings
            // ═══════════════════════════════════════════════════════════
            _sectionTitle(context, 'Basic Settings'),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    // Domain
                    TextField(
                      controller: _domainController,
                      decoration: InputDecoration(
                        labelText: 'Tunnel Domain',
                        hintText: 'tunnel.example.com',
                        helperText: 'Your authoritative DNS domain',
                        prefixIcon: const Icon(Icons.dns),
                        border: const OutlineInputBorder(),
                      ),
                    ),
                    
                    const SizedBox(height: 16),

                    // Auto Switch
                    SwitchListTile(
                      title: Text(
                        'Auto Switch',
                        style: TextStyle(color: textSecondary(context)),
                      ),
                      subtitle: Text(
                        'Automatically switch to DNS when TLS fails',
                        style: TextStyle(color: textMuted(context), fontSize: 12),
                      ),
                      value: _autoSwitch,
                      onChanged: (v) => setState(() {
                        _autoSwitch = v;
                        _mode = v ? 'auto' : 'manual';
                      }),
                    ),

                    const SizedBox(height: 16),

                    // Threshold
                    TextField(
                      controller: _thresholdController,
                      keyboardType: TextInputType.number,
                      decoration: InputDecoration(
                        labelText: 'Switch Threshold',
                        hintText: '3',
                        helperText: 'Switch to DNS after N TLS failures',
                        prefixIcon: const Icon(Icons.repeat),
                        border: const OutlineInputBorder(),
                      ),
                    ),

                    const SizedBox(height: 16),

                    // Timeout
                    TextField(
                      controller: _timeoutController,
                      keyboardType: TextInputType.number,
                      decoration: InputDecoration(
                        labelText: 'Query Timeout (seconds)',
                        hintText: '5',
                        helperText: 'Timeout for each DNS query',
                        prefixIcon: const Icon(Icons.timer),
                        border: const OutlineInputBorder(),
                      ),
                    ),
                  ],
                ),
              ),
            ),

            const SizedBox(height: 24),

            // ═══════════════════════════════════════════════════════════
            // DNS Servers
            // ═══════════════════════════════════════════════════════════
            _sectionTitle(context, 'DNS Servers'),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'Upstream DNS servers for queries',
                      style: TextStyle(color: textMuted(context), fontSize: 12),
                    ),
                    const SizedBox(height: 12),

                    ..._serverControllers.asMap().entries.map((entry) {
                      int idx = entry.key;
                      TextEditingController controller = entry.value;
                      
                      return Padding(
                        padding: const EdgeInsets.only(bottom: 12),
                        child: Row(
                          children: [
                            Expanded(
                              child: TextField(
                                controller: controller,
                                decoration: InputDecoration(
                                  labelText: 'Server ${idx + 1}',
                                  hintText: '8.8.8.8:53',
                                  prefixIcon: const Icon(Icons.cloud),
                                  border: const OutlineInputBorder(),
                                ),
                              ),
                            ),
                            if (_serverControllers.length > 1)
                              IconButton(
                                icon: const Icon(Icons.delete, color: Colors.red),
                                onPressed: () => _removeServer(idx),
                              ),
                          ],
                        ),
                      );
                    }).toList(),

                    OutlinedButton.icon(
                      onPressed: _addServer,
                      icon: const Icon(Icons.add),
                      label: const Text('Add DNS Server'),
                    ),
                  ],
                ),
              ),
            ),

            const SizedBox(height: 24),

            // ═══════════════════════════════════════════════════════════
            // Info
            // ═══════════════════════════════════════════════════════════
            Card(
              color: Colors.orange.withOpacity(0.1),
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        const Icon(Icons.info, color: Colors.orange),
                        const SizedBox(width: 8),
                        Text(
                          'DNS Fallback Information',
                          style: TextStyle(
                            color: textSecondary(context),
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 8),
                    Text(
                      '• Speed: ~50 Kbps (very slow)\n'
                      '• Latency: 100-300ms higher than TLS\n'
                      '• Use only when TLS is completely blocked\n'
                      '• Requires authoritative DNS setup on server',
                      style: TextStyle(
                        color: textMuted(context),
                        fontSize: 12,
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ],
        ],
      ),
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

  void _addServer() {
    setState(() {
      _serverControllers.add(TextEditingController(text: '1.1.1.1:53'));
    });
  }

  void _removeServer(int index) {
    setState(() {
      _serverControllers[index].dispose();
      _serverControllers.removeAt(index);
    });
  }

  void _saveSettings() {
    final provider = Provider.of<AppProvider>(context, listen: false);
    
    // Validate
    if (_domainController.text.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Domain is required')),
      );
      return;
    }

    // Collect DNS servers
    final servers = _serverControllers
        .map((c) => c.text.trim())
        .where((s) => s.isNotEmpty)
        .toList();

    if (servers.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('At least one DNS server is required')),
      );
      return;
    }

    // Update server config
    final updatedServer = widget.server.copyWith(
      dnsFallbackEnabled: _enabled,
      dnsFallbackMode: _mode,
      dnsFallbackDomain: _domainController.text.trim(),
      dnsFallbackServers: servers,
      dnsFallbackTimeout: int.tryParse(_timeoutController.text) ?? 5,
      dnsFallbackSwitchThreshold: int.tryParse(_thresholdController.text) ?? 3,
    );

    provider.updateServer(updatedServer);

    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(
        content: Text('DNS settings saved'),
        backgroundColor: Colors.green,
      ),
    );

    Navigator.pop(context);
  }
}
