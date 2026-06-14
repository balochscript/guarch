import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import 'package:guarch/app.dart';
import 'package:guarch/providers/app_provider.dart';
import 'package:guarch/models/server_config.dart';

class SNISettingsScreen extends StatelessWidget {
  const SNISettingsScreen({super.key});

  void _onToggleSni(bool value, AppProvider provider, BuildContext context) {
    if (value && provider.globalSniDomains.isEmpty) {
      _showNoDomainsDialog(context, provider);
    } else {
      provider.toggleGlobalSni();
    }
  }

  void _showNoDomainsDialog(BuildContext context, AppProvider provider) {
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
          'SNI Protection requires at least one domain to work.\n\n'
          'Add a domain first before enabling this feature.',
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
              _showAddDomainDialog(context, provider);
            },
            icon: const Icon(Icons.add, size: 18),
            label: const Text('Add Domain'),
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
            title: const Text('SNI Protection'),
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
                secondary: Icon(Icons.shield, color: accentColor(context)),
                title: Text(
                  'Enable SNI Protection',
                  style: TextStyle(
                    color: textSecondary(context),
                    fontWeight: FontWeight.w600,
                  ),
                ),
                subtitle: Text(
                  provider.globalSniDomains.isEmpty && !provider.globalSniEnabled
                      ? 'Add at least one SNI domain to enable this feature'
                      : 'Bypass SNI-based censorship and filtering',
                  style: TextStyle(
                    color: provider.globalSniDomains.isEmpty && !provider.globalSniEnabled
                        ? Colors.orange
                        : textMuted(context),
                    fontSize: 13,
                  ),
                ),
                value: provider.globalSniEnabled,
                onChanged: (value) => _onToggleSni(value, provider, context),
              ),

              if (!provider.globalSniEnabled && provider.globalSniDomains.isEmpty) ...[
                const SizedBox(height: 16),
                Container(
                  padding: const EdgeInsets.all(16),
                  decoration: BoxDecoration(
                    color: Colors.orange.withOpacity(0.1),
                    borderRadius: BorderRadius.circular(12),
                    border: Border.all(color: Colors.orange.withOpacity(0.3)),
                  ),
                  child: Row(
                    children: [
                      const Icon(Icons.info_outline, color: Colors.orange, size: 24),
                      const SizedBox(width: 16),
                      Expanded(
                        child: Text(
                          'SNI Protection is disabled because no domains are configured. Add at least one domain below to enable.',
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
                  _sectionTitle(context, 'SNI Domains (${provider.globalSniDomains.length})'),
                  TextButton.icon(
                    onPressed: () => _showAddDomainDialog(context, provider),
                    icon: const Icon(Icons.add, size: 18),
                    label: const Text('Add'),
                  ),
                ],
              ),

              if (provider.globalSniDomains.isEmpty)
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
                            provider.globalSniEnabled
                                ? 'SNI protection is enabled but has no targets!'
                                : 'Add at least one domain to enable SNI protection',
                            style: TextStyle(
                              color: provider.globalSniEnabled ? Colors.red : textMuted(context),
                              fontSize: 12,
                            ),
                          ),
                        ],
                      ),
                    ),
                  ),
                )
              else
                ...provider.globalSniDomains.asMap().entries.map((entry) {
                  final index = entry.key;
                  final domain = entry.value;
                  return Card(
                    margin: const EdgeInsets.only(bottom: 8),
                    child: ListTile(
                      leading: Icon(
                        domain.fallback ? Icons.backup : Icons.language,
                        color: accentColor(context),
                      ),
                      title: Text(
                        domain.domain,
                        style: TextStyle(
                          color: textSecondary(context),
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                      subtitle: Text(
                        'Weight: ${domain.weight}${domain.checkHealth ? " • Health check" : ""}${domain.fallback ? " • Fallback" : ""}',
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

              if (provider.globalSniEnabled) ...[
                const SizedBox(height: 24),
                _sectionTitle(context, 'Selection Mode'),
                Card(
                  child: Column(
                    children: [
                      RadioListTile<String>(
                        title: Text('Random', style: TextStyle(color: textSecondary(context))),
                        subtitle: Text('Pick domain randomly', style: TextStyle(color: textMuted(context), fontSize: 12)),
                        value: 'random',
                        groupValue: provider.globalSniMode,
                        onChanged: (v) => provider.setGlobalSniMode(v!),
                      ),
                      Divider(height: 1, color: accentColor(context).withOpacity(0.1)),
                      RadioListTile<String>(
                        title: Text('Weighted', style: TextStyle(color: textSecondary(context))),
                        subtitle: Text('Pick based on weight (recommended)', style: TextStyle(color: textMuted(context), fontSize: 12)),
                        value: 'weighted',
                        groupValue: provider.globalSniMode,
                        onChanged: (v) => provider.setGlobalSniMode(v!),
                      ),
                      Divider(height: 1, color: accentColor(context).withOpacity(0.1)),
                      RadioListTile<String>(
                        title: Text('Sequential', style: TextStyle(color: textSecondary(context))),
                        subtitle: Text('Rotate in order', style: TextStyle(color: textMuted(context), fontSize: 12)),
                        value: 'sequential',
                        groupValue: provider.globalSniMode,
                        onChanged: (v) => provider.setGlobalSniMode(v!),
                      ),
                      Divider(height: 1, color: accentColor(context).withOpacity(0.1)),
                      RadioListTile<String>(
                        title: Text('Single', style: TextStyle(color: textSecondary(context))),
                        subtitle: Text('Always use first domain', style: TextStyle(color: textMuted(context), fontSize: 12)),
                        value: 'single',
                        groupValue: provider.globalSniMode,
                        onChanged: (v) => provider.setGlobalSniMode(v!),
                      ),
                    ],
                  ),
                ),

                const SizedBox(height: 24),
                _sectionTitle(context, 'Rotation Interval'),
                Card(
                  child: Padding(
                    padding: const EdgeInsets.all(16),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Row(
                          mainAxisAlignment: MainAxisAlignment.spaceBetween,
                          children: [
                            Text(
                              'Change domain every',
                              style: TextStyle(color: textSecondary(context)),
                            ),
                            Container(
                              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                              decoration: BoxDecoration(
                                color: accentColor(context).withOpacity(0.1),
                                borderRadius: BorderRadius.circular(8),
                              ),
                              child: Text(
                                '${provider.globalSniRotationMinutes} min',
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
                          value: provider.globalSniRotationMinutes.toDouble(),
                          min: 1,
                          max: 30,
                          divisions: 29,
                          label: '${provider.globalSniRotationMinutes} min',
                          onChanged: (v) => provider.setGlobalSniRotationMinutes(v.toInt()),
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
                            'About SNI Protection',
                            style: TextStyle(
                              fontWeight: FontWeight.w600,
                              color: textSecondary(context),
                            ),
                          ),
                        ],
                      ),
                      const SizedBox(height: 12),
                      Text(
                        '• SNI (Server Name Indication) is part of TLS\n'
                        '• Some countries block based on SNI field\n'
                        '• This rotates SNI to evade filtering\n'
                        '• Use trusted domains (google, cloudflare)\n'
                        '• Weighted mode: higher weight = more usage\n'
                        '• Health check: auto-detect blocked domains\n'
                        '• Fallback: used when primary fails',
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

  void _showAddDomainDialog(BuildContext context, AppProvider provider) {
    final domainController = TextEditingController();
    final weightController = TextEditingController(text: '10');
    bool checkHealth = true;
    bool fallback = false;

    showDialog(
      context: context,
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setState) => AlertDialog(
          title: const Text('Add SNI Domain'),
          content: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                TextField(
                  controller: domainController,
                  decoration: const InputDecoration(
                    labelText: 'Domain',
                    hintText: 'google.com',
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
                CheckboxListTile(
                  title: const Text('Health Check'),
                  value: checkHealth,
                  onChanged: (v) => setState(() => checkHealth = v ?? true),
                  controlAffinity: ListTileControlAffinity.leading,
                  contentPadding: EdgeInsets.zero,
                ),
                CheckboxListTile(
                  title: const Text('Fallback Domain'),
                  value: fallback,
                  onChanged: (v) => setState(() => fallback = v ?? false),
                  controlAffinity: ListTileControlAffinity.leading,
                  contentPadding: EdgeInsets.zero,
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
                
                if (domain.isEmpty) {
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(content: Text('Domain cannot be empty')),
                  );
                  return;
                }
                
                provider.addGlobalSniDomain(SNIDomain(
                  domain: domain,
                  weight: weight,
                  checkHealth: checkHealth,
                  fallback: fallback,
                ));
                
                Navigator.pop(ctx);
              },
              child: const Text('Add'),
            ),
          ],
        ),
      ),
    );
  }

  void _showEditDomainDialog(BuildContext context, AppProvider provider, int index, SNIDomain domain) {
    final domainController = TextEditingController(text: domain.domain);
    final weightController = TextEditingController(text: domain.weight.toString());
    bool checkHealth = domain.checkHealth;
    bool fallback = domain.fallback;

    showDialog(
      context: context,
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setState) => AlertDialog(
          title: const Text('Edit SNI Domain'),
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
                CheckboxListTile(
                  title: const Text('Health Check'),
                  value: checkHealth,
                  onChanged: (v) => setState(() => checkHealth = v ?? true),
                  controlAffinity: ListTileControlAffinity.leading,
                  contentPadding: EdgeInsets.zero,
                ),
                CheckboxListTile(
                  title: const Text('Fallback Domain'),
                  value: fallback,
                  onChanged: (v) => setState(() => fallback = v ?? false),
                  controlAffinity: ListTileControlAffinity.leading,
                  contentPadding: EdgeInsets.zero,
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
                
                if (newDomain.isEmpty) {
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(content: Text('Domain cannot be empty')),
                  );
                  return;
                }
                
                provider.updateGlobalSniDomain(index, SNIDomain(
                  domain: newDomain,
                  weight: weight,
                  checkHealth: checkHealth,
                  fallback: fallback,
                ));
                
                Navigator.pop(ctx);
              },
              child: const Text('Save'),
            ),
          ],
        ),
      ),
    );
  }

  void _confirmDelete(BuildContext context, AppProvider provider, int index, String domain) {
    final isLastDomain = provider.globalSniDomains.length == 1;
    
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Delete Domain'),
        content: Text(
          isLastDomain
              ? 'Remove "$domain" from SNI domains?\n\n'
                'This is the last domain. SNI protection will be automatically disabled.'
              : 'Remove "$domain" from SNI domains?',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () {
              provider.removeGlobalSniDomain(index);
              
              if (isLastDomain && provider.globalSniEnabled) {
                provider.toggleGlobalSni();
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(
                    content: const Text('SNI protection disabled - no domains left'),
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
          'This will reset SNI settings to default values:\n\n'
          '• Mode: weighted\n'
          '• Rotation: 5 minutes\n'
          '• Domains: google, cloudflare, microsoft, apple\n\n'
          'Continue?',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () {
              provider.resetSniToDefaults();
              Navigator.pop(ctx);
              ScaffoldMessenger.of(context).showSnackBar(
                const SnackBar(
                  content: Text('✅ SNI reset to defaults'),
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
