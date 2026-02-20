import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import 'package:guarch/providers/app_provider.dart';
import 'package:guarch/screens/import_screen.dart';
import 'package:guarch/screens/about_screen.dart';
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
                child: ListTile(
                  leading: Icon(
                    provider.isDarkMode ? Icons.dark_mode : Icons.light_mode,
                  ),
                  title: const Text('Dark Mode'),
                  trailing: Switch(
                    value: provider.isDarkMode,
                    onChanged: (_) => provider.toggleTheme(),
                    activeColor: const Color(0xFF6C5CE7),
                  ),
                ),
              ),
              const SizedBox(height: 24),
              _sectionTitle('Import / Export'),
              Card(
                child: Column(
                  children: [
                    ListTile(
                      leading: const Icon(Icons.input),
                      title: const Text('Import Config'),
                      subtitle: const Text('From link, JSON, or clipboard'),
                      trailing: const Icon(Icons.arrow_forward_ios, size: 16),
                      onTap: () => Navigator.push(
                        context,
                        MaterialPageRoute(
                          builder: (_) => const ImportScreen(),
                        ),
                      ),
                    ),
                    const Divider(height: 1),
                    ListTile(
                      leading: const Icon(Icons.content_paste),
                      title: const Text('Quick Import from Clipboard'),
                      trailing: const Icon(Icons.arrow_forward_ios, size: 16),
                      onTap: () => _importClipboard(context, provider),
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
                          const SnackBar(
                            content: Text('Pinging all servers...'),
                          ),
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
                    ListTile(
                      leading: const Text('🎯', style: TextStyle(fontSize: 24)),
                      title: const Text('About Guarch'),
                      subtitle: const Text('Learn about the protocol'),
                      trailing: const Icon(Icons.arrow_forward_ios, size: 16),
                      onTap: () => Navigator.push(
                        context,
                        MaterialPageRoute(
                          builder: (_) => const AboutScreen(),
                        ),
                      ),
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
                      leading: Text('📱', style: TextStyle(fontSize: 24)),
                      title: Text('Version'),
                      trailing: Text('1.0.0'),
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
