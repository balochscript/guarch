import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import 'package:guarch/providers/app_provider.dart';
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
              _sectionTitle('Appearance'),
              Card(
                child: SwitchListTile(
                  leading: Icon(
                    provider.isDarkMode ? Icons.dark_mode : Icons.light_mode,
                  ),
                  title: const Text('Dark Mode'),
                  value: provider.isDarkMode,
                  onChanged: (_) => provider.toggleTheme(),
                  activeColor: const Color(0xFF6C5CE7),
                ),
              ),
              const SizedBox(height: 24),
              _sectionTitle('Import / Export'),
              Card(
                child: Column(
                  children: [
                    ListTile(
                      leading: const Icon(Icons.content_paste),
                      title: const Text('Import from Clipboard'),
                      subtitle: const Text('Paste guarch:// or JSON config'),
                      trailing: const Icon(Icons.arrow_forward_ios, size: 16),
                      onTap: () => _importClipboard(context, provider),
                    ),
                    const Divider(height: 1),
                    ListTile(
                      leading: const Icon(Icons.input),
                      title: const Text('Import from Text'),
                      subtitle: const Text('Enter config manually'),
                      trailing: const Icon(Icons.arrow_forward_ios, size: 16),
                      onTap: () => _showImportDialog(context, provider),
                    ),
                  ],
                ),
              ),
              const SizedBox(height: 24),
              _sectionTitle('Connection'),
              Card(
                child: Column(
                  children: [
                    ListTile(
                      leading: const Icon(Icons.speed),
                      title: const Text('Ping All Servers'),
                      trailing: const Icon(Icons.arrow_forward_ios, size: 16),
                      onTap: () {
                        provider.pingAllServers();
                        ScaffoldMessenger.of(context).showSnackBar(
                          const SnackBar(content: Text('Pinging all servers...')),
                        );
                      },
                    ),
                  ],
                ),
              ),
              const SizedBox(height: 24),
              _sectionTitle('About'),
              Card(
                child: Column(
                  children: [
                    const ListTile(
                      leading: Text('🎯', style: TextStyle(fontSize: 24)),
                      title: Text('Guarch Protocol'),
                      subtitle: Text('Version 1.0.0'),
                    ),
                    const Divider(height: 1),
                    ListTile(
                      leading: const Icon(Icons.code),
                      title: const Text('Source Code'),
                      subtitle: const Text('github.com/ppooria/guarch'),
                      trailing: const Icon(Icons.open_in_new, size: 16),
                      onTap: () => launchUrl(
                        Uri.parse('https://github.com/ppooria/guarch'),
                      ),
                    ),
                    const Divider(height: 1),
                    const ListTile(
                      leading: Text('🏹', style: TextStyle(fontSize: 24)),
                      title: Text('Name Origin'),
                      subtitle: Text(
                        'Guarch is a Balochi hunting technique where a hunter hides behind a cloth to approach prey undetected.',
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

  Widget _sectionTitle(String title) {
    return Padding(
      padding: const EdgeInsets.only(left: 4, bottom: 8),
      child: Text(
        title,
        style: const TextStyle(
          fontSize: 14,
          fontWeight: FontWeight.w600,
          color: Color(0xFF6C5CE7),
        ),
      ),
    );
  }

  void _importClipboard(BuildContext context, AppProvider provider) async {
    final data = await Clipboard.getData(Clipboard.kTextPlain);
    if (data?.text != null && data!.text!.isNotEmpty) {
      provider.importConfig(data.text!);
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Config imported from clipboard')),
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

  void _showImportDialog(BuildContext context, AppProvider provider) {
    final controller = TextEditingController();

    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Import Config'),
        content: TextField(
          controller: controller,
          maxLines: 5,
          decoration: InputDecoration(
            hintText: 'Paste guarch:// link or JSON config here',
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(12),
            ),
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () {
              if (controller.text.isNotEmpty) {
                provider.importConfig(controller.text);
                Navigator.pop(ctx);
                ScaffoldMessenger.of(context).showSnackBar(
                  const SnackBar(content: Text('Config imported')),
                );
              }
            },
            child: const Text('Import'),
          ),
        ],
      ),
    );
  }
}
