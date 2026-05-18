import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:guarch/app.dart';
import 'package:guarch/providers/app_provider.dart';
import 'package:guarch/models/connection_state.dart';
import 'package:guarch/screens/servers_screen.dart';
import 'package:guarch/screens/settings_screen.dart';
import 'package:guarch/screens/logs_screen.dart';
import 'package:guarch/screens/log_viewer_screen.dart';
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
                _buildHeader(context, provider),
                const SizedBox(height: 16),
                _buildServerInfo(context, server, provider),
                const Spacer(),
                ConnectionButton(
                  status: status,
                  onTap: () {
                    if (status == VpnStatus.error) {
                      provider.disconnect();
                    } else if (server == null) {
                      ScaffoldMessenger.of(context).showSnackBar(
                        const SnackBar(
                          content: Text('Please add and select a server first'),
                        ),
                      );
                    } else {
                      provider.toggleConnection();
                    }
                  },
                ),
                const SizedBox(height: 16),
                _buildStatusText(context, status),
                const Spacer(),
                if (status == VpnStatus.connected) ...[
                  StatsCard(stats: stats),
                  const SizedBox(height: 16),
                  _buildEnhancedStats(context, stats, provider),
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

  Widget _buildHeader(BuildContext context, AppProvider provider) {
    return Row(
      children: [
        Container(
          padding: const EdgeInsets.all(10),
          decoration: BoxDecoration(
            color: accentColor(context).withOpacity(0.15),
            borderRadius: BorderRadius.circular(12),
          ),
          child: const Text('🎯', style: TextStyle(fontSize: 24)),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                'Guarch',
                style: Theme.of(context).textTheme.titleLarge?.copyWith(
                      fontWeight: FontWeight.bold,
                      color: textPrimary(context),
                    ),
              ),
              Text(
                'Hidden like a Balochi hunter',
                style: Theme.of(context).textTheme.bodySmall?.copyWith(
                      color: textMuted(context),
                    ),
              ),
            ],
          ),
        ),
        if (provider.debugModeEnabled)
          IconButton(
            icon: const Icon(Icons.bug_report, color: Colors.orange, size: 28),
            onPressed: () => Navigator.push(
              context,
              MaterialPageRoute(builder: (_) => const LogViewerScreen()),
            ),
            tooltip: 'Debug Logs',
          ),
      ],
    );
  }

  Widget _buildServerInfo(
      BuildContext context, dynamic server, AppProvider provider) {
    if (server == null) {
      return Card(
        child: ListTile(
          leading: Icon(Icons.add_circle_outline, color: accentColor(context)),
          title: Text(
            'No server selected',
            style: TextStyle(color: textSecondary(context)),
          ),
          subtitle: Text(
            'Go to Servers tab to add one',
            style: TextStyle(color: textMuted(context)),
          ),
          trailing:
              Icon(Icons.arrow_forward_ios, size: 16, color: accentColor(context)),
        ),
      );
    }

    return Card(
      child: Column(
        children: [
          ListTile(
            leading: Text(server.pingEmoji, style: const TextStyle(fontSize: 24)),
            title: Text(
              server.name,
              style: TextStyle(
                fontWeight: FontWeight.w600,
                color: textSecondary(context),
              ),
            ),
            subtitle: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  server.fullAddress,
                  style: TextStyle(color: textMuted(context)),
                ),
                const SizedBox(height: 4),
                Row(
                  children: [
                    Text(
                      server.protocolEmoji,
                      style: const TextStyle(fontSize: 14),
                    ),
                    const SizedBox(width: 4),
                    Text(
                      server.protocol.toUpperCase(),
                      style: TextStyle(
                        color: textMuted(context),
                        fontSize: 11,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    if (server.sniEnabled) ...[
                      const SizedBox(width: 8),
                      const Text('🔄', style: TextStyle(fontSize: 12)),
                      const SizedBox(width: 2),
                      Text(
                        'SNI',
                        style: TextStyle(
                          color: textMuted(context),
                          fontSize: 11,
                        ),
                      ),
                    ],
                    if (server.coverEnabled) ...[
                      const SizedBox(width: 8),
                      const Text('🎭', style: TextStyle(fontSize: 12)),
                      const SizedBox(width: 2),
                      Text(
                        'Cover',
                        style: TextStyle(
                          color: textMuted(context),
                          fontSize: 11,
                        ),
                      ),
                    ],
                    if (server.dnsFallbackEnabled) ...[
                      const SizedBox(width: 8),
                      const Text('🔌', style: TextStyle(fontSize: 12)),
                      const SizedBox(width: 2),
                      Text(
                        'DNS',
                        style: TextStyle(
                          color: textMuted(context),
                          fontSize: 11,
                        ),
                      ),
                    ],
                  ],
                ),
              ],
            ),
            trailing: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              crossAxisAlignment: CrossAxisAlignment.end,
              children: [
                Text(
                  server.pingText,
                  style: TextStyle(
                    color: server.ping != null &&
                            server.ping! > 0 &&
                            server.ping! < 100
                        ? Colors.green
                        : textMuted(context),
                    fontWeight: FontWeight.w600,
                  ),
                ),
                if (provider.isConnected &&
                    server.batteryAwareEnabled &&
                    provider.batteryLevel < 30)
                  const Text(
                    '🔋 Low',
                    style: TextStyle(fontSize: 10, color: Colors.orange),
                  ),
              ],
            ),
            isThreeLine: true,
          ),
          
          if (server.metadata != null)
            _buildMetadataFooter(context, server.metadata),
        ],
      ),
    );
  }

  Widget _buildMetadataFooter(BuildContext context, dynamic metadata) {
    final hasAnnouncement = metadata.announcement?.enabled == true && 
                           metadata.announcement?.text != null;
    final hasQuota = metadata.quota != null && metadata.quota.unlimited == false;
    final hasExpiry = metadata.expiresAt != null;
    
    if (!hasAnnouncement && !hasQuota && !hasExpiry) {
      return const SizedBox.shrink();
    }

    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: Theme.of(context).brightness == Brightness.dark
            ? Colors.grey.shade900
            : Colors.grey.shade50,
        borderRadius: const BorderRadius.vertical(bottom: Radius.circular(12)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (hasAnnouncement)
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  metadata.announcement.icon,
                  style: const TextStyle(fontSize: 16),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    metadata.announcement.text,
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                      fontSize: 12,
                      height: 1.3,
                      color: textSecondary(context),
                    ),
                  ),
                ),
              ],
            ),
          
          if (hasQuota) ...[
            if (hasAnnouncement) const SizedBox(height: 10),
            Row(
              children: [
                const Text('📊', style: TextStyle(fontSize: 14)),
                const SizedBox(width: 8),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      ClipRRect(
                        borderRadius: BorderRadius.circular(2),
                        child: LinearProgressIndicator(
                          value: metadata.quota.usagePercent / 100,
                          backgroundColor: Colors.grey.shade300,
                          valueColor: AlwaysStoppedAnimation<Color>(
                            metadata.quota.progressColor,
                          ),
                          minHeight: 4,
                        ),
                      ),
                      const SizedBox(height: 4),
                      Text(
                        '${metadata.quota.remainingFormatted} remaining of ${metadata.quota.totalFormatted}',
                        style: TextStyle(
                          fontSize: 10,
                          color: textMuted(context),
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ],
          
          if (hasExpiry) ...[
            if (hasAnnouncement || hasQuota) const SizedBox(height: 8),
            Row(
              children: [
                Text(
                  metadata.isExpired ? '⏰' : '📅',
                  style: const TextStyle(fontSize: 14),
                ),
                const SizedBox(width: 8),
                Text(
                  metadata.isExpired 
                      ? 'Config expired'
                      : 'Expires: ${metadata.expiryText}',
                  style: TextStyle(
                    fontSize: 11,
                    color: metadata.isExpired ? Colors.red : textMuted(context),
                  ),
                ),
              ],
            ),
          ],
        ],
      ),
    );
  }

  Widget _buildStatusText(BuildContext context, VpnStatus status) {
    String text;
    Color color;
    switch (status) {
      case VpnStatus.disconnected:
        text = 'Tap to Guarch';
        color = textMuted(context);
        break;
      case VpnStatus.connecting:
        text = 'Guarching...';
        color = textPrimary(context);
        break;
      case VpnStatus.connected:
        text = '🎯 Guarch Activated';
        color = Colors.green;
        break;
      case VpnStatus.disconnecting:
        text = 'De-Guarching...';
        color = textPrimary(context);
        break;
      case VpnStatus.error:
        text = '❌ Failed - Tap to Stop VPN';
        color = Colors.red;
        break;
    }
    return Text(
      text,
      style: TextStyle(
        color: color,
        fontSize: 16,
        fontWeight: FontWeight.w600,
      ),
    );
  }

  Widget _buildEnhancedStats(
      BuildContext context, ConnectionStats stats, AppProvider provider) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                const Text('🎭', style: TextStyle(fontSize: 20)),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'Cover Traffic Active',
                        style: TextStyle(
                          fontWeight: FontWeight.w600,
                          color: textSecondary(context),
                        ),
                      ),
                      Text(
                        '${stats.coverRequests} requests • ${stats.activityEmoji} ${stats.activityText}',
                        style: TextStyle(
                          color: textMuted(context),
                          fontSize: 12,
                        ),
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

            if (stats.currentSNI.isNotEmpty) ...[
              const SizedBox(height: 12),
              Divider(color: accentColor(context).withOpacity(0.1)),
              const SizedBox(height: 12),
              Row(
                children: [
                  const Text('🔄', style: TextStyle(fontSize: 20)),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          'Current SNI',
                          style: TextStyle(
                            fontWeight: FontWeight.w600,
                            color: textSecondary(context),
                          ),
                        ),
                        Text(
                          '${stats.currentSNI} • ${stats.sniSwitches} switches',
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
            ],

            if (stats.dnsFallbackUsed) ...[
              const SizedBox(height: 12),
              Divider(color: accentColor(context).withOpacity(0.1)),
              const SizedBox(height: 12),
              Row(
                children: [
                  const Text('🔌', style: TextStyle(fontSize: 20)),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          'DNS Fallback Mode',
                          style: TextStyle(
                            fontWeight: FontWeight.w600,
                            color: Colors.orange,
                          ),
                        ),
                        Text(
                          'Reduced speed — TLS blocked',
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
            ],

            if (provider.batteryLevel < 30 ||
                provider.dataSaverEnabled) ...[
              const SizedBox(height: 12),
              Divider(color: accentColor(context).withOpacity(0.1)),
              const SizedBox(height: 12),
              Row(
                children: [
                  Text(
                    provider.batteryLevel < 30 ? '🔋' : '💾',
                    style: const TextStyle(fontSize: 20),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Text(
                      provider.batteryLevel < 30
                          ? 'Low Battery Mode (${provider.batteryLevel}%)'
                          : 'Data Saver Active',
                      style: TextStyle(
                        fontSize: 13,
                        color: textMuted(context),
                      ),
                    ),
                  ),
                ],
              ),
            ],
          ],
        ),
      ),
    );
  }
}
