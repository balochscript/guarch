import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import 'package:guarch/app.dart';
import 'package:guarch/providers/app_provider.dart';
import 'package:guarch/models/app_settings.dart';

class AdvancedSettingsScreen extends StatefulWidget {
  const AdvancedSettingsScreen({super.key});

  @override
  State<AdvancedSettingsScreen> createState() => _AdvancedSettingsScreenState();
}

class _AdvancedSettingsScreenState extends State<AdvancedSettingsScreen> {
  int _socksPort = 7070;
  int _dialTimeout = 30;
  int _handshakeTimeout = 15;
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _loadSettings();
  }

  Future<void> _loadSettings() async {
    final settings = await AppSettings.load();
    setState(() {
      _socksPort = settings.socksPort;
      _dialTimeout = settings.dialTimeout;
      _handshakeTimeout = settings.handshakeTimeout;
      _loading = false;
    });
  }

  Future<void> _saveSocksPort(int port) async {
    await AppSettings.setSocksPort(port);
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('✓ SOCKS port updated to $port')),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_loading) {
      return Scaffold(
        appBar: AppBar(title: const Text('Advanced Settings')),
        body: const Center(child: CircularProgressIndicator()),
      );
    }

    return Consumer<AppProvider>(
      builder: (context, provider, _) {
        return Scaffold(
          appBar: AppBar(
            title: const Text('Advanced Settings'),
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
              _sectionTitle(context, 'Local Proxy Settings'),
              Card(
                child: _buildSocksPortTile(context),
              ),

              const SizedBox(height: 24),
              _sectionTitle(context, 'Transport Timeouts'),
              Card(
                child: Column(
                  children: [
                    _buildSliderTile(
                      context,
                      icon: Icons.timer,
                      title: 'Dial Timeout',
                      subtitle: 'Max time to establish TCP connection',
                      value: _dialTimeout.toDouble(),
                      min: 10,
                      max: 120,
                      divisions: 11,
                      label: '${_dialTimeout}s',
                      onChanged: (v) async {
                        setState(() => _dialTimeout = v.toInt());
                        await AppSettings.setDialTimeout(_dialTimeout);
                      },
                    ),
                    Divider(
                      height: 1,
                      color: accentColor(context).withOpacity(0.1),
                    ),
                    _buildSliderTile(
                      context,
                      icon: Icons.handshake,
                      title: 'Handshake Timeout',
                      subtitle: 'Max time for TLS/protocol handshake',
                      value: _handshakeTimeout.toDouble(),
                      min: 5,
                      max: 60,
                      divisions: 11,
                      label: '${_handshakeTimeout}s',
                      onChanged: (v) async {
                        setState(() => _handshakeTimeout = v.toInt());
                        await AppSettings.setHandshakeTimeout(_handshakeTimeout);
                      },
                    ),
                  ],
                ),
              ),

              const SizedBox(height: 24),
              _sectionTitle(context, 'Engine Settings'),
              Card(
                child: Column(
                  children: [
                    _buildSliderTile(
                      context,
                      icon: Icons.access_time,
                      title: 'Connection Timeout',
                      subtitle: 'Max time for overall connection attempt',
                      value: provider.connectionTimeout.toDouble(),
                      min: 5,
                      max: 60,
                      divisions: 11,
                      label: '${provider.connectionTimeout}s',
                      onChanged: (v) => provider.setConnectionTimeout(v.toInt()),
                    ),
                    Divider(
                      height: 1,
                      color: accentColor(context).withOpacity(0.1),
                    ),
                    _buildSliderTile(
                      context,
                      icon: Icons.favorite,
                      title: 'Keep-Alive Interval',
                      subtitle: 'Heartbeat frequency to maintain connection',
                      value: provider.keepAliveInterval.toDouble(),
                      min: 10,
                      max: 300,
                      divisions: 29,
                      label: '${provider.keepAliveInterval}s',
                      onChanged: (v) => provider.setKeepAliveInterval(v.toInt()),
                    ),
                  ],
                ),
              ),

              const SizedBox(height: 24),
              _sectionTitle(context, 'Retry Settings'),
              Card(
                child: Column(
                  children: [
                    _buildSliderTile(
                      context,
                      icon: Icons.replay,
                      title: 'Max Retry Attempts',
                      subtitle: 'Number of retries before giving up',
                      value: provider.maxRetryAttempts.toDouble(),
                      min: 1,
                      max: 10,
                      divisions: 9,
                      label: '${provider.maxRetryAttempts}',
                      onChanged: (v) => provider.setMaxRetryAttempts(v.toInt()),
                    ),
                    Divider(
                      height: 1,
                      color: accentColor(context).withOpacity(0.1),
                    ),
                    _buildSliderTile(
                      context,
                      icon: Icons.schedule,
                      title: 'Retry Delay',
                      subtitle: 'Wait time between retry attempts',
                      value: provider.retryDelay.toDouble(),
                      min: 1,
                      max: 30,
                      divisions: 29,
                      label: '${provider.retryDelay}s',
                      onChanged: (v) => provider.setRetryDelay(v.toInt()),
                    ),
                  ],
                ),
              ),

              const SizedBox(height: 24),
              _sectionTitle(context, 'Performance'),
              Card(
                child: _buildBufferSizeTile(context, provider),
              ),

              const SizedBox(height: 24),
              Container(
                padding: const EdgeInsets.all(16),
                decoration: BoxDecoration(
                  color: accentColor(context).withOpacity(0.1),
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(
                    color: accentColor(context).withOpacity(0.3),
                  ),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Icon(
                          Icons.info_outline,
                          color: accentColor(context),
                          size: 20,
                        ),
                        const SizedBox(width: 8),
                        Text(
                          'Advanced Settings Info',
                          style: TextStyle(
                            fontWeight: FontWeight.w600,
                            color: textSecondary(context),
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 12),
                    Text(
                      '• SOCKS port: Change if port 7070 conflicts\n'
                      '• Dial timeout: TCP connection establishment\n'
                      '• Handshake timeout: TLS/protocol handshake\n'
                      '• Connection timeout: Overall attempt (engine-level)\n'
                      '• Lower timeouts = faster failure detection\n'
                      '• Higher timeouts = better on slow networks\n'
                      '• More retries = more resilient\n'
                      '• Larger buffer = higher memory usage\n'
                      '• Keep-alive prevents idle disconnects',
                      style: TextStyle(
                        color: textMuted(context),
                        fontSize: 13,
                        height: 1.5,
                      ),
                    ),
                  ],
                ),
              ),

              const SizedBox(height: 24),
              _sectionTitle(context, 'Current Configuration'),
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(16),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'Transport Settings',
                        style: TextStyle(
                          fontWeight: FontWeight.w600,
                          fontSize: 13,
                          color: textSecondary(context),
                        ),
                      ),
                      const SizedBox(height: 8),
                      _buildInfoRow('SOCKS Port', '$_socksPort'),
                      const SizedBox(height: 8),
                      _buildInfoRow('Dial Timeout', '${_dialTimeout}s'),
                      const SizedBox(height: 8),
                      _buildInfoRow('Handshake Timeout', '${_handshakeTimeout}s'),
                      const SizedBox(height: 16),
                      Divider(color: accentColor(context).withOpacity(0.1)),
                      const SizedBox(height: 16),
                      Text(
                        'Engine Settings',
                        style: TextStyle(
                          fontWeight: FontWeight.w600,
                          fontSize: 13,
                          color: textSecondary(context),
                        ),
                      ),
                      const SizedBox(height: 8),
                      _buildInfoRow('Connection Timeout', '${provider.connectionTimeout}s'),
                      const SizedBox(height: 8),
                      _buildInfoRow('Keep-Alive Interval', '${provider.keepAliveInterval}s'),
                      const SizedBox(height: 8),
                      _buildInfoRow('Max Retry Attempts', '${provider.maxRetryAttempts}'),
                      const SizedBox(height: 8),
                      _buildInfoRow('Retry Delay', '${provider.retryDelay}s'),
                      const SizedBox(height: 8),
                      _buildInfoRow('Buffer Size', '${provider.bufferSize ~/ 1024} KB'),
                    ],
                  ),
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

  Widget _buildSocksPortTile(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(Icons.settings_ethernet, color: accentColor(context), size: 20),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'SOCKS5 Proxy Port',
                      style: TextStyle(
                        fontWeight: FontWeight.w600,
                        color: textSecondary(context),
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      'Local port for SOCKS5 proxy (applies to all servers)',
                      style: TextStyle(
                        color: textMuted(context),
                        fontSize: 12,
                      ),
                    ),
                  ],
                ),
              ),
              SizedBox(
                width: 100,
                child: TextField(
                  controller: TextEditingController(text: _socksPort.toString()),
                  keyboardType: TextInputType.number,
                  textAlign: TextAlign.center,
                  inputFormatters: [
                    FilteringTextInputFormatter.digitsOnly,
                    LengthLimitingTextInputFormatter(5),
                  ],
                  decoration: const InputDecoration(
                    border: OutlineInputBorder(),
                    contentPadding: EdgeInsets.symmetric(horizontal: 8, vertical: 8),
                    hintText: '7070',
                  ),
                  onSubmitted: (value) {
                    int? port = int.tryParse(value);
                    if (port == null || port < 1024 || port > 65535) {
                      ScaffoldMessenger.of(context).showSnackBar(
                        const SnackBar(
                          content: Text('❌ Invalid port. Must be 1024-65535'),
                          backgroundColor: Colors.red,
                        ),
                      );
                      return;
                    }
                    setState(() => _socksPort = port);
                    _saveSocksPort(port);
                  },
                ),
              ),
            ],
          ),
          const SizedBox(height: 12),
          Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: Colors.blue.shade50,
              borderRadius: BorderRadius.circular(8),
            ),
            child: Row(
              children: [
                const Icon(Icons.info_outline, color: Colors.blue, size: 16),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    'ℹ️  Valid range: 1024-65535 | Avoid: 1080, 8080, 3128',
                    style: TextStyle(fontSize: 12, color: Colors.blue.shade900),
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildSliderTile(
    BuildContext context, {
    required IconData icon,
    required String title,
    required String subtitle,
    required double value,
    required double min,
    required double max,
    required int divisions,
    required String label,
    required ValueChanged<double> onChanged,
  }) {
    return Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(icon, color: accentColor(context), size: 20),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      title,
                      style: TextStyle(
                        fontWeight: FontWeight.w600,
                        color: textSecondary(context),
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      subtitle,
                      style: TextStyle(
                        color: textMuted(context),
                        fontSize: 12,
                      ),
                    ),
                  ],
                ),
              ),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                decoration: BoxDecoration(
                  color: accentColor(context).withOpacity(0.1),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Text(
                  label,
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
            value: value,
            min: min,
            max: max,
            divisions: divisions,
            label: label,
            onChanged: onChanged,
          ),
        ],
      ),
    );
  }

  Widget _buildBufferSizeTile(BuildContext context, AppProvider provider) {
    final sizes = [16384, 32768, 65536, 131072];
    final labels = ['16 KB', '32 KB', '64 KB', '128 KB'];

    return Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(Icons.storage, color: accentColor(context), size: 20),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'Buffer Size',
                      style: TextStyle(
                        fontWeight: FontWeight.w600,
                        color: textSecondary(context),
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      'Memory allocated for network operations',
                      style: TextStyle(
                        color: textMuted(context),
                        fontSize: 12,
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
          const SizedBox(height: 16),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: List.generate(sizes.length, (index) {
              final size = sizes[index];
              final label = labels[index];
              final isSelected = provider.bufferSize == size;

              return ChoiceChip(
                label: Text(label),
                selected: isSelected,
                onSelected: (selected) {
                  if (selected) {
                    provider.setBufferSize(size);
                  }
                },
              );
            }),
          ),
        ],
      ),
    );
  }

  Widget _buildInfoRow(String label, String value) {
    return Row(
      mainAxisAlignment: MainAxisAlignment.spaceBetween,
      children: [
        Text(
          label,
          style: const TextStyle(fontSize: 14),
        ),
        Text(
          value,
          style: const TextStyle(
            fontWeight: FontWeight.w600,
            fontSize: 14,
          ),
        ),
      ],
    );
  }

  void _showResetDialog(BuildContext context, AppProvider provider) {
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Reset to Defaults'),
        content: const Text(
          'This will reset all advanced settings to their default values.\n\n'
          'Are you sure?',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () async {
              provider.setConnectionTimeout(15);
              provider.setHandshakeTimeout(30);
              provider.setKeepAliveInterval(30);
              provider.setMaxRetryAttempts(3);
              provider.setRetryDelay(5);
              provider.setBufferSize(32768);
              
              await AppSettings.reset();
              final settings = await AppSettings.load();
              setState(() {
                _socksPort = settings.socksPort;
                _dialTimeout = settings.dialTimeout;
                _handshakeTimeout = settings.handshakeTimeout;
              });
              
              if (context.mounted) {
                Navigator.pop(context);
                ScaffoldMessenger.of(context).showSnackBar(
                  const SnackBar(
                    content: Text('✅ Settings reset to defaults'),
                    backgroundColor: Colors.green,
                  ),
                );
              }
            },
            child: const Text('Reset'),
          ),
        ],
      ),
    );
  }
}
