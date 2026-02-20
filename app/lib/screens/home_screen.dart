import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:guarch/app.dart';
import 'package:guarch/providers/app_provider.dart';
import 'package:guarch/models/connection_state.dart';
import 'package:guarch/screens/servers_screen.dart';
import 'package:guarch/screens/settings_screen.dart';
import 'package:guarch/screens/logs_screen.dart';
import 'package:guarch/widgets/connection_button.dart';
import 'package:guarch/widgets/stats_card.dart';

class HomeScreen extends StatefulWidget {
  const HomeScreen({super.key});

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  int _currentIndex = 0;

  @override
  Widget build(BuildContext context) {
    final screens = [
      const _HomeTab(),
      const ServersScreen(),
      const LogsScreen(),
      const SettingsScreen(),
    ];

    return Scaffold(
      body: screens[_currentIndex],
      bottomNavigationBar: NavigationBar(
        selectedIndex: _currentIndex,
        onDestinationSelected: (i) => setState(() => _currentIndex = i),
        destinations: const [
          NavigationDestination(
            icon: Icon(Icons.home_outlined),
            selectedIcon: Icon(Icons.home),
            label: 'Home',
          ),
          NavigationDestination(
            icon: Icon(Icons.dns_outlined),
            selectedIcon: Icon(Icons.dns),
            label: 'Servers',
          ),
          NavigationDestination(
            icon: Icon(Icons.article_outlined),
            selectedIcon: Icon(Icons.article),
            label: 'Logs',
          ),
          NavigationDestination(
            icon: Icon(Icons.settings_outlined),
            selectedIcon: Icon(Icons.settings),
            label: 'Settings',
          ),
        ],
      ),
    );
  }
}

class _HomeTab extends StatelessWidget {
  const _HomeTab();

  @override
  Widget build(BuildContext context) {
    return Consumer<AppProvider>(
      builder: (context, provider, _) {
        final status = provider.status;
        final server = provider.activeServer;
        final stats = provider.stats;

        return SafeArea(
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 24),
            child: Column(
              children: [
                const SizedBox(height: 20),
                _buildHeader(context),
                const SizedBox(height: 16),
                _buildServerInfo(context, server),
                const Spacer(),
                ConnectionButton(
                  status: status,
                  onTap: () {
                    if (server == null) {
                      ScaffoldMessenger.of(context).showSnackBar(
                        const SnackBar(
                          content: Text('Please add and select a server first'),
                        ),
                      );
                      return;
                    }
                    provider.toggleConnection();
                  },
                ),
                const SizedBox(height: 16),
                _buildStatusText(status),
                const Spacer(),
                if (status == VpnStatus.connected) ...[
                  StatsCard(stats: stats),
                  const SizedBox(height: 16),
                  _buildCoverInfo(stats),
                ] else
                  const SizedBox(height: 120),
                const SizedBox(height: 20),
              ],
            ),
          ),
        );
      },
    );
  }

  Widget _buildHeader(BuildContext context) {
    return Row(
      children: [
        Container(
          padding: const EdgeInsets.all(10),
          decoration: BoxDecoration(
            color: kGold.withOpacity(0.15),
            borderRadius: BorderRadius.circular(12),
          ),
          child: const Text('🎯', style: TextStyle(fontSize: 24)),
        ),
        const SizedBox(width: 12),
        Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              'Guarch',
              style: Theme.of(context).textTheme.titleLarge?.copyWith(
                    fontWeight: FontWeight.bold,
                    color: kGold,
                  ),
            ),
            Text(
              'Hidden like a Balochi hunter',
              style: Theme.of(context).textTheme.bodySmall?.copyWith(
                    color: kGold.withOpacity(0.5),
                  ),
            ),
          ],
        ),
      ],
    );
  }

  Widget _buildServerInfo(BuildContext context, dynamic server) {
    if (server == null) {
      return Card(
        child: ListTile(
          leading: Icon(Icons.add_circle_outline, color: kGold),
          title: Text('No server selected',
              style: TextStyle(color: kGoldLight)),
          subtitle: Text('Go to Servers tab to add one',
              style: TextStyle(color: kGold.withOpacity(0.5))),
          trailing: Icon(Icons.arrow_forward_ios, size: 16, color: kGold),
        ),
      );
    }

    return Card(
      child: ListTile(
        leading: Text(server.pingEmoji, style: const TextStyle(fontSize: 24)),
        title: Text(
          server.name,
          style: const TextStyle(
            fontWeight: FontWeight.w600,
            color: kGoldLight,
          ),
        ),
        subtitle: Text(
          server.fullAddress,
          style: TextStyle(color: kGold.withOpacity(0.5)),
        ),
        trailing: Text(
          server.pingText,
          style: TextStyle(
            color: server.ping != null && server.ping! > 0 && server.ping! < 100
                ? Colors.green
                : kGold.withOpacity(0.5),
            fontWeight: FontWeight.w600,
          ),
        ),
      ),
    );
  }

  Widget _buildStatusText(VpnStatus status) {
    String text;
    Color color;

    switch (status) {
      case VpnStatus.disconnected:
        text = 'Tap to connect';
        color = kGold.withOpacity(0.5);
        break;
      case VpnStatus.connecting:
        text = 'Connecting...';
        color = kGold;
        break;
      case VpnStatus.connected:
        text = 'Connected & Protected';
        color = Colors.green;
        break;
      case VpnStatus.disconnecting:
        text = 'Disconnecting...';
        color = kGold;
        break;
      case VpnStatus.error:
        text = 'Connection failed';
        color = Colors.red;
        break;
    }

    return Text(
      text,
      style: TextStyle(color: color, fontSize: 16, fontWeight: FontWeight.w500),
    );
  }

  Widget _buildCoverInfo(ConnectionStats stats) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Row(
          children: [
            const Text('🎭', style: TextStyle(fontSize: 20)),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Text(
                    'Cover Traffic Active',
                    style: TextStyle(fontWeight: FontWeight.w600, color: kGoldLight),
                  ),
                  Text(
                    '${stats.coverRequests} cover requests sent',
                    style: TextStyle(color: kGold.withOpacity(0.5), fontSize: 12),
                  ),
                ],
              ),
            ),
            Container(
              width: 8,
              height: 8,
              decoration: const BoxDecoration(
                color: Colors.green,
                shape: BoxShape.circle,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
