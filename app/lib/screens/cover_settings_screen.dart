import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import 'package:guarch/app.dart';
import 'package:guarch/providers/app_provider.dart';
import 'package:guarch/models/server_config.dart';

class CoverSettingsScreen extends StatelessWidget {
  const CoverSettingsScreen({super.key});

  void _onToggleCover(bool value, AppProvider provider, BuildContext context) {
    if (value && provider.globalCoverDomains.isEmpty) {
      _showDefaultDomainsDialog(context, provider);
    } else {
      provider.toggleGlobalCover();
    }
  }

  void _showDefaultDomainsDialog(BuildContext context, AppProvider provider) {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Row(
          children: [
            Icon(Icons.info_outline, color: Colors.blue, size: 28),
            const SizedBox(width: 12),
            const Expanded(child: Text('No Cover Domains'))
          ],
        ),
        content: const Text(
          'No custom domains configured.\n\n'
          'Default trusted domains will be used:\n'
          '• www.google.com\n'
          '• www.microsoft.com\n'
          '• github.com\n\n'
          'You can add custom domains later.',
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
              provider.setGlobalCoverDomains([
                CoverDomain(domain: 'www.google.com', weight: 30, paths: ['/', '/search']),
                CoverDomain(domain: 'www.microsoft.com', weight: 20, paths: ['/', '/windows']),
                CoverDomain(domain: 'github.com', weight: 15, paths: ['/', '/explore']),
              ]);
              provider.toggleGlobalCover();
            },
            icon: const Icon(Icons.check, size: 18),
            label: const Text('Use Defaults'),
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Consumer<AppProvider>(
      builder: (context, provider, _) {
        return Scaffold(
          appBar: AppBar(
            title: const Text('Cover Traffic'),
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
              SwitchListTile(
                secondary: Icon(Icons.theater_comedy, color: accentColor(context)),
                title: Text(
                  'Enable Cover Traffic',
                  style: TextStyle(
                    color: textSecondary(context),
                    fontWeight: FontWeight.w600,
                  ),
                ),
                subtitle: Text(
                  provider.globalCoverDomains.isEmpty && !provider.globalCoverEnabled
                      ? 'Default trusted domains will be used if enabled'
                      : 'Hide real traffic patterns with decoy requests',
                  style: TextStyle(
                    color: provider.globalCoverDomains.isEmpty && !provider.globalCoverEnabled
                        ? Colors.blue
                        : textMuted(context),
                    fontSize: 13,
                  ),
                ),
                value: provider.globalCoverEnabled,
                onChanged: (value) => _onToggleCover(value, provider, context),
              ),

              if (!provider.globalCoverEnabled && provider.globalCoverDomains.isEmpty) ...[
                const SizedBox(height: 16),
                Container(
                  padding: const EdgeInsets.all(16),
                  decoration: BoxDecoration(
                    color: Colors.blue.withOpacity(0.1),
                    borderRadius: BorderRadius.circular(12),
                    border: Border.all(color: Colors.blue.withOpacity(0.3)),
                  ),
                  child: Row(
                    children: [
                      const Icon(Icons.info_outline, color: Colors.blue, size: 24),
                      const SizedBox(width: 16),
                      Expanded(
                        child: Text(
                          'You can enable Cover Traffic without adding domains. Default trusted domains (Google, Microsoft, GitHub) will be used automatically.',
                          style: TextStyle(
                            color: textSecondary(context),
                            fontSize: 13,
                            height: 1.4,
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
              ],

              const SizedBox(height: 24),
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  _sectionTitle(context, 'Cover Domains (${provider.globalCoverDomains.length})'),
                  TextButton.icon(
                    onPressed: () => _showAddDomainDialog(context, provider),
                    icon: const Icon(Icons.add, size: 18),
                    label: const Text('Add'),
                  ),
                ],
              ),

              if (provider.globalCoverDomains.isEmpty)
                Card(
                  child: Padding(
                    padding: const EdgeInsets.all(24),
                    child: Center(
                      child: Column(
                        children: [
                          Icon(Icons.info_outline, size: 48, color: Colors.blue),
                          const SizedBox(height: 12),
                          Text(
                            'No custom domains configured',
                            style: TextStyle(
                              color: textSecondary(context),
                              fontWeight: FontWeight.w600,
                            ),
                          ),
                          const SizedBox(height: 4),
                          Text(
                            provider.globalCoverEnabled
                                ? 'Using default trusted domains'
                                : 'Default domains will be used when enabled',
                            style: TextStyle(
                              color: provider.globalCoverEnabled ? Colors.blue : textMuted(context),
                              fontSize: 12,
                            ),
                          ),
                        ],
                      ),
                    ),
                  ),
                )
              else
                ...provider.globalCoverDomains.asMap().entries.map((entry) {
                  final index = entry.key;
                  final domain = entry.value;
                  return Card(
                    margin: const EdgeInsets.only(bottom: 8),
                    child: ListTile(
                      leading: Icon(Icons.web, color: accentColor(context)),
                      title: Text(
                        domain.domain,
                        style: TextStyle(
                          color: textSecondary(context),
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                      subtitle: Text(
                        'Weight: ${domain.weight} - Paths: ${domain.paths.length}',
                        style: TextStyle(color: textMuted(context), fontSize: 12),
                      ),
                      trailing: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          IconButton(
                            icon: const Icon(Icons.edit, size: 20),
                            onPressed: () => _showEditDomainDialog(context, provider, index, domain),
                          ),
                          IconButton(
                            icon: const Icon(Icons.delete, size: 20),
                            color: Colors.red,
                            onPressed: () => _confirmDelete(context, provider, index, domain.domain),
                          ),
                        ],
                      ),
                    ),
                  );
                }).toList(),

              if (provider.globalCoverEnabled) ...[
                const SizedBox(height: 24),
                _sectionTitle(context, 'Cover Mode'),
                Card(
                  child: Column(
                    children: [
                      RadioListTile<String>(
                        title: Text('Stealth', style: TextStyle(color: textSecondary(context))),
                        subtitle: Text('Maximum cover traffic (high data usage)', style: TextStyle(color: textMuted(context), fontSize: 12)),
                        value: 'stealth',
                        groupValue: provider.globalCoverMode,
                        onChanged: (v) => provider.setGlobalCoverMode(v!),
                      ),
                      Divider(height: 1, color: accentColor(context).withOpacity(0.1)),
                      RadioListTile<String>(
                        title: Text('Balanced', style: TextStyle(color: textSecondary(context))),
                        subtitle: Text('Moderate cover traffic (recommended)', style: TextStyle(color: textMuted(context), fontSize: 12)),
                        value: 'balanced',
                        groupValue: provider.globalCoverMode,
                        onChanged: (v) => provider.setGlobalCoverMode(v!),
                      ),
                      Divider(height: 1, color: accentColor(context).withOpacity(0.1)),
                      RadioListTile<String>(
                        title: Text('Fast', style: TextStyle(color: textSecondary(context))),
                        subtitle: Text('Minimal cover traffic (low data usage)', style: TextStyle(color: textMuted(context), fontSize: 12)),
                        value: 'fast',
                        groupValue: provider.globalCoverMode,
                        onChanged: (v) => provider.setGlobalCoverMode(v!),
                      ),
                      Divider(height: 1, color: accentColor(context).withOpacity(0.1)),
                      RadioListTile<String>(
                        title: Text('Auto', style: TextStyle(color: textSecondary(context))),
                        subtitle: Text('Adaptive based on activity', style: TextStyle(color: textMuted(context), fontSize: 12)),
                        value: 'auto',
                        groupValue: provider.globalCoverMode,
                        onChanged: (v) => provider.setGlobalCoverMode(v!),
                      ),
                    ],
                  ),
                ),
                _buildModeInfoBox(context, provider.globalCoverMode),

                const SizedBox(height: 24),
                _sectionTitle(context, 'Traffic Obfuscation'),
                Card(
                  child: Column(
                    children: [
                      SwitchListTile(
                        secondary: Icon(Icons.lock, color: accentColor(context), size: 20),
                        title: Text(
                          'Packet Padding',
                          style: TextStyle(color: textSecondary(context)),
                        ),
                        subtitle: Text(
                          'Add random bytes to hide real packet sizes',
                          style: TextStyle(color: textMuted(context), fontSize: 12),
                        ),
                        value: provider.globalPaddingEnabled,
                        onChanged: (_) => provider.toggleGlobalPadding(),
                      ),
                      
                      if (provider.globalPaddingEnabled) ...[
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
                                    'Max Padding Size',
                                    style: TextStyle(color: textSecondary(context)),
                                  ),
                                  Container(
                                    padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                                    decoration: BoxDecoration(
                                      color: accentColor(context).withOpacity(0.1),
                                      borderRadius: BorderRadius.circular(8),
                                    ),
                                    child: Text(
                                      '${provider.globalMaxPadding} bytes',
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
                                value: provider.globalMaxPadding.toDouble(),
                                min: 64,
                                max: 2048,
                                divisions: 10,
                                label: '${provider.globalMaxPadding} bytes',
                                onChanged: (v) => provider.setGlobalMaxPadding(v.toInt()),
                              ),
                              Text(
                                'Smart padding rounds to web-realistic sizes. Larger = better obfuscation but more bandwidth.',
                                style: TextStyle(
                                  color: textMuted(context),
                                  fontSize: 11,
                                  height: 1.4,
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
                _sectionTitle(context, 'Traffic Shaping Pattern'),
                Card(
                  child: Column(
                    children: [
                      RadioListTile<String>(
                        title: Row(
                          children: [
                            Icon(Icons.public, size: 20, color: Colors.blue),
                            const SizedBox(width: 12),
                            Text('Web Browsing', style: TextStyle(color: textSecondary(context))),
                          ],
                        ),
                        subtitle: Text(
                          'Normal website browsing (default)',
                          style: TextStyle(color: textMuted(context), fontSize: 12),
                        ),
                        value: 'web',
                        groupValue: provider.globalTrafficPattern,
                        onChanged: (v) => provider.setGlobalTrafficPattern(v!),
                      ),
                      Divider(height: 1, color: accentColor(context).withOpacity(0.1)),
                      RadioListTile<String>(
                        title: Row(
                          children: [
                            Icon(Icons.video_library, size: 20, color: Colors.red),
                            const SizedBox(width: 12),
                            Text('Video Streaming', style: TextStyle(color: textSecondary(context))),
                          ],
                        ),
                        subtitle: Text(
                          'Simulate YouTube/Netflix traffic',
                          style: TextStyle(color: textMuted(context), fontSize: 12),
                        ),
                        value: 'video',
                        groupValue: provider.globalTrafficPattern,
                        onChanged: (v) => provider.setGlobalTrafficPattern(v!),
                      ),
                      Divider(height: 1, color: accentColor(context).withOpacity(0.1)),
                      RadioListTile<String>(
                        title: Row(
                          children: [
                            Icon(Icons.file_download, size: 20, color: Colors.green),
                            const SizedBox(width: 12),
                            Text('File Download', style: TextStyle(color: textSecondary(context))),
                          ],
                        ),
                        subtitle: Text(
                          'Simulate large file download',
                          style: TextStyle(color: textMuted(context), fontSize: 12),
                        ),
                        value: 'download',
                        groupValue: provider.globalTrafficPattern,
                        onChanged: (v) => provider.setGlobalTrafficPattern(v!),
                      ),
                    ],
                  ),
                ),

                const SizedBox(height: 24),
                _sectionTitle(context, 'Adaptive Settings'),
                Card(
                  child: Column(
                    children: [
                      SwitchListTile(
                        secondary: Icon(Icons.battery_charging_full, size: 20, color: Colors.green),
                        title: Text(
                          'Battery Aware',
                          style: TextStyle(color: textSecondary(context)),
                        ),
                        subtitle: Text(
                          'Reduce cover traffic on low battery',
                          style: TextStyle(color: textMuted(context), fontSize: 12),
                        ),
                        value: provider.globalBatteryAware,
                        onChanged: (_) => provider.toggleGlobalBatteryAware(),
                      ),
                      Divider(height: 1, color: accentColor(context).withOpacity(0.1)),
                      ListTile(
                        leading: Icon(Icons.bar_chart, size: 20, color: accentColor(context)),
                        title: Text(
                          'Current Battery',
                          style: TextStyle(color: textSecondary(context)),
                        ),
                        subtitle: Text(
                          '${provider.batteryLevel}% - ${_batteryStatus(provider.batteryLevel)}',
                          style: TextStyle(color: textMuted(context), fontSize: 12),
                        ),
                        trailing: Text(
                          '${provider.batteryLevel}%',
                          style: TextStyle(
                            color: provider.batteryLevel < 30 ? Colors.orange : Colors.green,
                            fontWeight: FontWeight.w600,
                          ),
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
                                  'Low Battery Threshold',
                                  style: TextStyle(
                                    color: textSecondary(context),
                                    fontWeight: FontWeight.w600,
                                  ),
                                ),
                                Container(
                                  padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                                  decoration: BoxDecoration(
                                    color: accentColor(context).withOpacity(0.1),
                                    borderRadius: BorderRadius.circular(8),
                                  ),
                                  child: Text(
                                    '${provider.globalBatteryThreshold}%',
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
                              value: provider.globalBatteryThreshold.toDouble(),
                              min: 10,
                              max: 50,
                              divisions: 8,
                              label: '${provider.globalBatteryThreshold}%',
                              onChanged: (v) => provider.setGlobalBatteryThreshold(v.toInt()),
                            ),
                            Text(
                              'Reduce cover traffic when battery drops below this level',
                              style: TextStyle(
                                color: textMuted(context),
                                fontSize: 11,
                              ),
                            ),
                          ],
                        ),
                      ),
                    ],
                  ),
                ),

                const SizedBox(height: 24),
                _sectionTitle(context, 'Activity Monitor'),
                Card(
                  child: ListTile(
                    leading: Icon(
                      Icons.speed,
                      color: _getActivityColor(provider.stats.activityLevel),
                      size: 28,
                    ),
                    title: Text(
                      'Current Activity Level',
                      style: TextStyle(color: textSecondary(context)),
                    ),
                    subtitle: Text(
                      'Automatically adjusts cover traffic based on usage',
                      style: TextStyle(color: textMuted(context), fontSize: 12),
                    ),
                    trailing: Container(
                      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                      decoration: BoxDecoration(
                        color: _getActivityColor(provider.stats.activityLevel).withOpacity(0.2),
                        borderRadius: BorderRadius.circular(8),
                        border: Border.all(
                          color: _getActivityColor(provider.stats.activityLevel),
                          width: 1,
                        ),
                      ),
                      child: Text(
                        _getActivityText(provider.stats.activityLevel).toUpperCase(),
                        style: TextStyle(
                          fontWeight: FontWeight.w600,
                          fontSize: 12,
                          color: _getActivityColor(provider.stats.activityLevel),
                        ),
                      ),
                    ),
                  ),
                ),

                const SizedBox(height: 24),
                _sectionTitle(context, 'Advanced Settings'),
                Card(
                  child: Padding(
                    padding: const EdgeInsets.all(16),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Row(
                          mainAxisAlignment: MainAxisAlignment.spaceBetween,
                          children: [
                            Expanded(
                              child: Column(
                                crossAxisAlignment: CrossAxisAlignment.start,
                                children: [
                                  Text(
                                    'Hysteresis Delay',
                                    style: TextStyle(
                                      color: textSecondary(context),
                                      fontWeight: FontWeight.w600,
                                    ),
                                  ),
                                  const SizedBox(height: 4),
                                  Text(
                                    'Wait time before changing activity level',
                                    style: TextStyle(
                                      color: textMuted(context),
                                      fontSize: 11,
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
                                '${provider.globalHysteresisDelay}s',
                                style: TextStyle(
                                  fontWeight: FontWeight.w600,
                                  color: accentColor(context),
                                ),
                              ),
                            ),
                          ],
                        ),
                        const SizedBox(height: 12),
                        Slider(
                          value: provider.globalHysteresisDelay.toDouble(),
                          min: 10,
                          max: 60,
                          divisions: 10,
                          label: '${provider.globalHysteresisDelay}s',
                          onChanged: (v) => provider.setGlobalHysteresisDelay(v.toInt()),
                        ),
                        Text(
                          'Prevents rapid switching between activity levels. Higher = more stable.',
                          style: TextStyle(
                            color: textMuted(context),
                            fontSize: 11,
                            height: 1.3,
                          ),
                        ),
                      ],
                    ),
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
                            'About Cover Traffic',
                            style: TextStyle(
                              fontWeight: FontWeight.w600,
                              color: textSecondary(context),
                            ),
                          ),
                        ],
                      ),
                      const SizedBox(height: 12),
                      Text(
                        '• Sends fake HTTP requests to real websites\n'
                        '• Makes your traffic look like normal browsing\n'
                        '• Packet padding hides real packet sizes\n'
                        '• Traffic pattern simulates different behaviors\n'
                        '• Activity monitor auto-adjusts cover rate\n'
                        '• Higher modes = more data usage\n'
                        '• Auto mode adjusts based on your activity\n'
                        '• Battery aware reduces traffic on low battery\n'
                        '• Use popular domains (google, microsoft)',
                        style: TextStyle(
                          color: textMuted(context),
                          fontSize: 13,
                          height: 1.5,
                        ),
                      ),
                    ],
                  ),
                ),
              ],

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

  String _batteryStatus(int level) {
    if (level < 15) return 'Critical (cover stopped)';
    if (level < 30) return 'Low (cover reduced)';
    if (level < 50) return 'Medium';
    return 'Good';
  }

  Color _getActivityColor(String level) {
    switch (level.toLowerCase()) {
      case 'idle':
        return Colors.grey;
      case 'light':
        return Colors.blue;
      case 'medium':
        return Colors.orange;
      case 'heavy':
        return Colors.red;
      default:
        return Colors.amber;
    }
  }

  String _getActivityText(String level) {
    if (level.isEmpty) return 'Unknown';
    return level[0].toUpperCase() + level.substring(1).toLowerCase();
  }

  void _showAddDomainDialog(BuildContext context, AppProvider provider) {
    final domainController = TextEditingController();
    final weightController = TextEditingController(text: '10');
    final pathsController = TextEditingController(text: '/, /search');

    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Add Cover Domain'),
        content: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: domainController,
                decoration: const InputDecoration(
                  labelText: 'Domain',
                  hintText: 'www.google.com',
                  border: OutlineInputBorder(),
                ),
                autofocus: true,
              ),
              const SizedBox(height: 16),
              TextField(
                controller: weightController,
                decoration: const InputDecoration(
                  labelText: 'Weight',
                  border: OutlineInputBorder(),
                ),
                keyboardType: TextInputType.number,
                inputFormatters: [FilteringTextInputFormatter.digitsOnly],
              ),
              const SizedBox(height: 16),
              TextField(
                controller: pathsController,
                decoration: const InputDecoration(
                  labelText: 'Paths (comma separated)',
                  hintText: '/, /search, /news',
                  border: OutlineInputBorder(),
                ),
                maxLines: 2,
              ),
            ],
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () {
              final domain = domainController.text.trim();
              final weight = int.tryParse(weightController.text) ?? 10;
              final paths = pathsController.text
                  .split(',')
                  .map((p) => p.trim())
                  .where((p) => p.isNotEmpty)
                  .toList();
              
              if (domain.isEmpty) {
                ScaffoldMessenger.of(context).showSnackBar(
                  const SnackBar(content: Text('Domain cannot be empty')),
                );
                return;
              }
              
              if (paths.isEmpty) {
                ScaffoldMessenger.of(context).showSnackBar(
                  const SnackBar(content: Text('Add at least one path')),
                );
                return;
              }
              
              provider.addGlobalCoverDomain(CoverDomain(
                domain: domain,
                weight: weight,
                paths: paths,
              ));
              
              Navigator.pop(ctx);
            },
            child: const Text('Add'),
          ),
        ],
      ),
    );
  }

  void _showEditDomainDialog(BuildContext context, AppProvider provider, int index, CoverDomain domain) {
    final domainController = TextEditingController(text: domain.domain);
    final weightController = TextEditingController(text: domain.weight.toString());
    final pathsController = TextEditingController(text: domain.paths.join(', '));

    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Edit Cover Domain'),
        content: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: domainController,
                decoration: const InputDecoration(
                  labelText: 'Domain',
                  border: OutlineInputBorder(),
                ),
              ),
              const SizedBox(height: 16),
              TextField(
                controller: weightController,
                decoration: const InputDecoration(
                  labelText: 'Weight',
                  border: OutlineInputBorder(),
                ),
                keyboardType: TextInputType.number,
                inputFormatters: [FilteringTextInputFormatter.digitsOnly],
              ),
              const SizedBox(height: 16),
              TextField(
                controller: pathsController,
                decoration: const InputDecoration(
                  labelText: 'Paths (comma separated)',
                  border: OutlineInputBorder(),
                ),
                maxLines: 2,
              ),
            ],
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () {
              final newDomain = domainController.text.trim();
              final weight = int.tryParse(weightController.text) ?? 10;
              final paths = pathsController.text
                  .split(',')
                  .map((p) => p.trim())
                  .where((p) => p.isNotEmpty)
                  .toList();
              
              if (newDomain.isEmpty) {
                ScaffoldMessenger.of(context).showSnackBar(
                  const SnackBar(content: Text('Domain cannot be empty')),
                );
                return;
              }
              
              if (paths.isEmpty) {
                ScaffoldMessenger.of(context).showSnackBar(
                  const SnackBar(content: Text('Add at least one path')),
                );
                return;
              }
              
              provider.updateGlobalCoverDomain(index, CoverDomain(
                domain: newDomain,
                weight: weight,
                paths: paths,
              ));
              
              Navigator.pop(ctx);
            },
            child: const Text('Save'),
          ),
        ],
      ),
    );
  }

  void _confirmDelete(BuildContext context, AppProvider provider, int index, String domain) {
    final isLastDomain = provider.globalCoverDomains.length == 1;
    
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Delete Domain'),
        content: Text(
          isLastDomain
              ? 'Remove "$domain" from cover domains?\n\n'
                'This is the last domain. Cover traffic will be automatically disabled.'
              : 'Remove "$domain" from cover domains?',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () {
              provider.removeGlobalCoverDomain(index);
              
              if (isLastDomain && provider.globalCoverEnabled) {
                provider.toggleGlobalCover();
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(
                    content: const Text('Cover traffic disabled - no domains left'),
                    backgroundColor: Colors.orange,
                  ),
                );
              }
              
              Navigator.pop(ctx);
            },
            style: FilledButton.styleFrom(backgroundColor: Colors.red),
            child: const Text('Delete'),
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
          'This will reset cover settings to default values:\n\n'
          '• Mode: balanced\n'
          '• Battery aware: ON\n'
          '• Domains: google, microsoft, github\n\n'
          'Continue?',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () {
              provider.resetCoverToDefaults();
              Navigator.pop(ctx);
              ScaffoldMessenger.of(context).showSnackBar(
                const SnackBar(
                  content: Text('Cover reset to defaults'),
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

  Widget _buildModeInfoBox(BuildContext context, String mode) {
    String text;
    IconData icon;
    Color color;

    switch (mode) {
      case 'stealth':
        text = '• High-intensity Cover Traffic (6 domains)\n'
            '• Large random packet padding (up to 1024 bytes)\n'
            '• High-frequency heartbeat (every 2 seconds) to mask idle times\n'
            '• Highly effective against severe DPI, ML models, and national firewall checks, but has higher bandwidth overhead.';
        icon = Icons.security;
        color = Colors.blue;
        break;
      case 'balanced':
        text = '• Moderate Cover Traffic (3 domains)\n'
            '• Medium packet padding (up to 256 bytes)\n'
            '• Medium heartbeat frequency (every 10 seconds)\n'
            '• Recommended for general daily use. Provides excellent DPI protection with minimal impact on latency and bandwidth.';
        icon = Icons.verified_user;
        color = Colors.green;
        break;
      case 'fast':
        text = '• Cover traffic is completely disabled\n'
            '• Packet padding is set to zero\n'
            '• Standard traffic shaping and heartbeat\n'
            '• Perfect for maximum raw speed, online gaming, and downloading when severe censorship is not actively present on the network.';
        icon = Icons.flash_on;
        color = Colors.amber;
        break;
      default:
        text = '• Adaptive mode adjusts cover traffic on the fly\n'
            '• Matches your real data usage dynamically\n'
            '• Saves battery when idle, increases obfuscation when actively browsing.';
        icon = Icons.autorenew;
        color = Colors.purple;
        break;
    }

    return Container(
      margin: const EdgeInsets.only(top: 12),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: color.withOpacity(0.1),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: color.withOpacity(0.3)),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(icon, color: color, size: 24),
          const SizedBox(width: 16),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  '${mode.toUpperCase()} MODE DETAILS',
                  style: TextStyle(
                    fontWeight: FontWeight.bold,
                    fontSize: 12,
                    color: color,
                    letterSpacing: 1.1,
                  ),
                ),
                const SizedBox(height: 8),
                Text(
                  text,
                  style: TextStyle(
                    fontSize: 12,
                    height: 1.4,
                    color: textSecondary(context),
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
