import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:guarch/app.dart';
import 'package:guarch/providers/app_provider.dart';

class AdvancedSettingsScreen extends StatelessWidget {
  const AdvancedSettingsScreen({super.key});

  @override
  Widget build(BuildContext context) {
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
              // ═══════════════════════════════════════════════
              // Connection Timeouts
              // ═══════════════════════════════════════════════
              _sectionTitle(context, 'Connection Timeouts'),
              Card(
                child: Column(
                  children: [
                    _buildSliderTile(
                      context,
                      icon: Icons.timer,
                      title: 'Connection Timeout',
                      subtitle: 'Max time to establish connection',
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
                      icon: Icons.handshake,
                      title: 'Handshake Timeout',
                      subtitle: 'Max time for protocol handshake',
                      value: provider.handshakeTimeout.toDouble(),
                      min: 10,
                      max: 120,
                      divisions: 11,
                      label: '${provider.handshakeTimeout}s',
                      onChanged: (v) => provider.setHandshakeTimeout(v.toInt()),
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

              // ═══════════════════════════════════════════════
              // Retry Settings
              // ═══════════════════════════════════════════════
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

              // ═══════════════════════════════════════════════
              // Performance
              // ═══════════════════════════════════════════════
              const SizedBox(height: 24),
              _sectionTitle(context, 'Performance'),
              Card(
                child: _buildBufferSizeTile(context, provider),
              ),

              // ═══════════════════════════════════════════════
              // Info Box
              // ═══════════════════════════════════════════════
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

              // ═══════════════════════════════════════════════
              // Current Values Summary
              // ═══════════════════════════════════════════════
              const SizedBox(height: 24),
              _sectionTitle(context, 'Current Configuration'),
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(16),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      _buildInfoRow(
                        'Connection Timeout',
                        '${provider.connectionTimeout}s',
                      ),
                      const SizedBox(height: 8),
                      _buildInfoRow(
                        'Handshake Timeout',
                        '${provider.handshakeTimeout}s',
                      ),
                      const SizedBox(height: 8),
                      _buildInfoRow(
                        'Keep-Alive Interval',
                        '${provider.keepAliveInterval}s',
                      ),
                      const SizedBox(height: 8),
                      _buildInfoRow(
                        'Max Retry Attempts',
                        '${provider.maxRetryAttempts}',
                      ),
                      const SizedBox(height: 8),
                      _buildInfoRow(
                        'Retry Delay',
                        '${provider.retryDelay}s',
                      ),
                      const SizedBox(height: 8),
                      _buildInfoRow(
                        'Buffer Size',
                        '${provider.bufferSize ~/ 1024} KB',
                      ),
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
            onPressed: () {
              provider.setConnectionTimeout(15);
              provider.setHandshakeTimeout(30);
              provider.setKeepAliveInterval(30);
              provider.setMaxRetryAttempts(3);
              provider.setRetryDelay(5);
              provider.setBufferSize(32768);
              Navigator.pop(context);
              ScaffoldMessenger.of(context).showSnackBar(
                const SnackBar(
                  content: Text('Settings reset to defaults'),
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
