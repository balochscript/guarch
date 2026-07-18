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
import 'package:guarch/widgets/sponsor_banner.dart';
import 'package:flutter/services.dart';

class HomeScreen extends StatefulWidget {
  const HomeScreen({super.key});

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  int _currentIndex = 0;

  void _navigateToServers() {
    setState(() => _currentIndex = 1);
  }

  @override
  Widget build(BuildContext context) {
    final screens = [
      _HomeTab(onNavigateToServers: _navigateToServers),
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
  final VoidCallback onNavigateToServers;
  
  const _HomeTab({required this.onNavigateToServers});

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
                  if (provider.activeServer?.coverEnabled == true || stats.coverRequests > 0)
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

  void _showDonateDialog(BuildContext context) {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Row(
          children: [
            Icon(Icons.favorite, color: Colors.red[400]),
            const SizedBox(width: 12),
            const Text('Support Guarch VPN'),
          ],
        ),
        content: SingleChildScrollView(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              const Text(
                'Guarch VPN is free and open-source. Your donations help us:',
                style: TextStyle(height: 1.4),
              ),
              const SizedBox(height: 12),
              _bulletPoint('Maintain servers and infrastructure'),
              _bulletPoint('Develop new features'),
              _bulletPoint('Keep the project alive'),
              const SizedBox(height: 20),
              const Text(
                'Crypto Donations:',
                style: TextStyle(fontWeight: FontWeight.bold),
              ),
              const SizedBox(height: 12),
              _cryptoAddress(
                context,
                'Bitcoin (BTC)',
                'bc1q53y72zxhuxlsmx6uh8ga5pmslh9fgy9mfwvqvt',
                Icons.currency_bitcoin,
              ),
              const SizedBox(height: 8),
              _cryptoAddress(
                context,
                'Ethereum (ETH)',
                '0x37Afc5996621d22E8fa5f3f24652666F0a732f6E',
                Icons.monetization_on,
              ),
              const SizedBox(height: 8),
              _cryptoAddress(
                context,
                'USDT (TRC20) / TRX',
                'TUYMUcb8a3S4o8s2jch4ikEnfr1xPM5rjm',
                Icons.attach_money,
              ),
              const SizedBox(height: 8),
              _cryptoAddress(
                context,
                'USDT (Polygon)',
                '0x37Afc5996621d22E8fa5f3f24652666F0a732f6E',
                Icons.paid,
              ),
              const SizedBox(height: 8),
              _cryptoAddress(
                context,
                'TON',
                'UQC2jcvDdnCFXqNwlWbGEkaOitIoIIyTMuKKqsb8pvalgHtP',
                Icons.diamond,
              ),
            ],
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Close'),
          ),
        ],
      ),
    );
  }

  Widget _bulletPoint(String text) {
    return Padding(
      padding: const EdgeInsets.only(left: 8, bottom: 4),
      child: Row(
        children: [
          const Icon(Icons.check_circle, size: 16, color: Colors.green),
          const SizedBox(width: 8),
          Expanded(child: Text(text, style: const TextStyle(fontSize: 13))),
        ],
      ),
    );
  }

  Widget _cryptoAddress(BuildContext context, String name, String address, IconData icon) {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surfaceVariant.withOpacity(0.3),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(icon, size: 18, color: Colors.amber[800]),
              const SizedBox(width: 8),
              Text(
                name,
                style: const TextStyle(
                  fontWeight: FontWeight.w600,
                  fontSize: 13,
                ),
              ),
              const Spacer(),
              InkWell(
                onTap: () {
                  Clipboard.setData(ClipboardData(text: address));
                  ScaffoldMessenger.of(context).showSnackBar(
                    SnackBar(
                      content: Text('$name address copied!'),
                      duration: const Duration(seconds: 2),
                    ),
                  );
                },
                child: const Icon(Icons.copy, size: 16, color: Colors.blue),
              ),
            ],
          ),
          const SizedBox(height: 4),
          Text(
            address,
            style: TextStyle(
              fontSize: 11,
              color: Theme.of(context).textTheme.bodySmall?.color,
              fontFamily: 'monospace',
            ),
          ),
        ],
      ),
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
          child: ClipRRect(
            borderRadius: BorderRadius.circular(6),
            child: Image.asset(
              'assets/icon.png',
              width: 40,
              height: 40,
              fit: BoxFit.cover,
              errorBuilder: (context, error, stackTrace) {
                return Icon(
                  Icons.shield,
                  size: 40,
                  color: accentColor(context),
                );
              },
            ),
          ),
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
                'Guarch your activity!',
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
        IconButton(
          icon: Icon(Icons.favorite, color: Colors.red[400], size: 28),
          onPressed: () => _showDonateDialog(context),
          tooltip: 'Support Guarch',
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
            'Tap here to add a server',
            style: TextStyle(color: textMuted(context)),
          ),
          trailing: Icon(
            Icons.arrow_forward_ios,
            size: 16,
            color: accentColor(context),
          ),
          onTap: onNavigateToServers,
        ),
      );
    }

    return Card(
      child: InkWell(
        onTap: onNavigateToServers,
        borderRadius: BorderRadius.circular(12),
        child: Column(
          children: [
            ListTile(
              leading: Icon(
                _getProtocolIcon(server.protocol),
                size: 32,
                color: _getProtocolColor(server.protocol),
              ),
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
                      Icon(
                        _getProtocolIcon(server.protocol),
                        size: 14,
                        color: _getProtocolColor(server.protocol),
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
                        const Icon(Icons.shield_outlined, size: 14),
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
                        const Icon(Icons.theater_comedy, size: 14),
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
                        const Icon(Icons.dns, size: 14),
                        const SizedBox(width: 2),
                        Text(
                          'DNS',
                          style: TextStyle(
                            color: textMuted(context),
                            fontSize: 11,
                          ),
                        ),
                      ],
                      if (server.protocol == 'grouk' && server.groukFecEnabled) ...[
                        const SizedBox(width: 8),
                        const Icon(Icons.shield, size: 14, color: Colors.purple),
                        const SizedBox(width: 2),
                        Text(
                          'FEC',
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
                    Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Icon(
                          Icons.battery_alert,
                          size: 12,
                          color: Colors.orange,
                        ),
                        const SizedBox(width: 2),
                        const Text(
                          'Low',
                          style: TextStyle(fontSize: 10, color: Colors.orange),
                        ),
                      ],
                    ),
                ],
              ),
              isThreeLine: true,
            ),
            
            if (server.metadata != null)
              _buildMetadataFooter(context, server.metadata),
          ],
        ),
      ),
    );
  }

  IconData _getProtocolIcon(String protocol) {
    switch (protocol.toLowerCase()) {
      case 'guarch':
        return Icons.security;
      case 'grouk':
        return Icons.flash_on;
      case 'zhip':
        return Icons.bolt;
      default:
        return Icons.shield;
    }
  }

  Color _getProtocolColor(String protocol) {
    switch (protocol.toLowerCase()) {
      case 'guarch':
        return Colors.blue;
      case 'grouk':
        return Colors.purple;
      case 'zhip':
        return Colors.amber;
      default:
        return Colors.grey;
    }
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
                Icon(
                  Icons.campaign,
                  size: 16,
                  color: metadata.announcement.color,
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
                Icon(
                  Icons.data_usage,
                  size: 14,
                  color: accentColor(context),
                ),
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
                Icon(
                  metadata.isExpired ? Icons.error : Icons.calendar_today,
                  size: 14,
                  color: metadata.isExpired ? Colors.red : accentColor(context),
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
        text = 'Guarch Activated';
        color = Colors.green;
        break;
      case VpnStatus.disconnecting:
        text = 'De-Guarching...';
        color = textPrimary(context);
        break;
      case VpnStatus.error:
        text = 'Failed - Tap to Stop VPN';
        color = Colors.red;
        break;
    }
    return Row(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        if (status == VpnStatus.connected)
          Icon(Icons.shield, size: 18, color: Colors.green),
        if (status == VpnStatus.connected) const SizedBox(width: 6),
        if (status == VpnStatus.error)
          const Icon(Icons.error, size: 18, color: Colors.red),
        if (status == VpnStatus.error) const SizedBox(width: 6),
        Text(
          text,
          style: TextStyle(
            color: color,
            fontSize: 16,
            fontWeight: FontWeight.w600,
          ),
        ),
      ],
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
                Icon(Icons.theater_comedy, size: 20, color: accentColor(context)),
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
                        '${stats.coverRequests.toCompactString()} requests • ${stats.activityEmoji} ${stats.activityText}',
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
                  Icon(Icons.shield_outlined, size: 20, color: accentColor(context)),
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
                          '${stats.currentSNI} • ${stats.sniSwitches.toCompactString()} switches',
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
                  const Icon(Icons.dns, size: 20, color: Colors.orange),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        const Text(
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

            if (stats.fecEnabled && stats.fecRecv > 0) ...[
              const SizedBox(height: 12),
              Divider(color: accentColor(context).withOpacity(0.1)),
              const SizedBox(height: 12),
              Row(
                children: [
                  const Icon(Icons.healing, size: 20, color: Colors.purple),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          'FEC Active (Grouk)',
                          style: TextStyle(
                            fontWeight: FontWeight.w600,
                            color: textSecondary(context),
                          ),
                        ),
                        Text(
                          'Recovered ${stats.fecRecovered.toCompactString()}/${stats.fecRecv.toCompactString()} packets (${stats.fecRecoveryRate.toStringAsFixed(1)}%)',
                          style: TextStyle(
                            color: textMuted(context),
                            fontSize: 12,
                          ),
                        ),
                      ],
                    ),
                  ),
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                    decoration: BoxDecoration(
                      color: stats.fecRecoveryRate > 0 ? Colors.green.withOpacity(0.2) : Colors.grey.withOpacity(0.2),
                      borderRadius: BorderRadius.circular(8),
                    ),
                    child: Text(
                      stats.fecRecoveryRate > 0 ? 'Working' : 'Idle',
                      style: TextStyle(
                        fontSize: 10,
                        fontWeight: FontWeight.w600,
                        color: stats.fecRecoveryRate > 0 ? Colors.green : Colors.grey,
                      ),
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
                  Icon(
                    provider.batteryLevel < 30 ? Icons.battery_alert : Icons.data_saver_on,
                    size: 20,
                    color: accentColor(context),
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
