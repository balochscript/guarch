import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import 'package:guarch/app.dart';
import 'package:guarch/providers/app_provider.dart';
import 'package:guarch/models/server_config.dart';

class CoverSettingsScreen extends StatelessWidget {
  const CoverSettingsScreen({super.key});

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
                  'Hide real traffic patterns with decoy requests',
                  style: TextStyle(color: textMuted(context), fontSize: 13),
                ),
                value: provider.globalCoverEnabled,
                onChanged: (_) => provider.toggleGlobalCover(),
              ),

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

                const SizedBox(height: 24),
                _sectionTitle(context, 'Adaptive Settings'),
                Card(
                  child: Column(
                    children: [
                      SwitchListTile(
                        secondary: const Text('🔋', style: TextStyle(fontSize: 20)),
                        title: Text(
                          'Battery Aware',
                          style: TextStyle(color: textSecondary(context)),
                        ),
                        subtitle: Text(
                          'Reduce cover traffic on low battery (<30%)',
                          style: TextStyle(color: textMuted(context), fontSize: 12),
                        ),
                        value: provider.globalBatteryAware,
                        onChanged: (_) => provider.toggleGlobalBatteryAware(),
                      ),
                      Divider(height: 1, color: accentColor(context).withOpacity(0.1)),
                      ListTile(
                        leading: const Text('📊', style: TextStyle(fontSize: 20)),
                        title: Text(
                          'Current Battery',
                          style: TextStyle(color: textSecondary(context)),
                        ),
                        subtitle: Text(
                          '${provider.batteryLevel}% • ${_batteryStatus(provider.batteryLevel)}',
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
                    ],
                  ),
                ),

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
                            Icon(Icons.warning_amber, size: 48, color: Colors.orange),
                            const SizedBox(height: 12),
                            Text(
                              'No domains configured',
                              style: TextStyle(
                                color: textSecondary(context),
                                fontWeight: FontWeight.w600,
                              ),
                            ),
                            const SizedBox(height: 4),
                            Text(
                              'Add at least one domain',
                              style: TextStyle(color: textMuted(context), fontSize: 12),
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
                          'Weight: ${domain.weight} • Paths: ${domain.paths.length}',
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
                        '• Harder to detect VPN usage\n'
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
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Delete Domain'),
        content: Text('Remove "$domain" from cover domains?'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () {
              provider.removeGlobalCoverDomain(index);
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
                  content: Text('✅ Cover reset to defaults'),
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
