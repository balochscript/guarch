import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import 'package:qr_flutter/qr_flutter.dart';
import 'package:guarch/app.dart';
import 'package:guarch/models/server_config.dart';
import 'package:guarch/providers/app_provider.dart';
import 'package:guarch/screens/add_server_screen.dart';
import 'package:guarch/screens/export_screen.dart';

class ServerDetailScreen extends StatefulWidget {
  final ServerConfig server;
  const ServerDetailScreen({super.key, required this.server});

  @override
  State<ServerDetailScreen> createState() => _ServerDetailScreenState();
}

class _ServerDetailScreenState extends State<ServerDetailScreen> {
  bool _showPsk = false;
  bool _isTesting = false;

  @override
  Widget build(BuildContext context) {
    final server = widget.server;

    return Scaffold(
      appBar: AppBar(
        title: Text(server.name),
        actions: [
          IconButton(
            icon: const Icon(Icons.edit),
            onPressed: () => Navigator.push(
              context,
              MaterialPageRoute(
                builder: (_) => AddServerScreen(server: server),
              ),
            ),
          ),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.all(24),
        children: [
          Center(
            child: Text(server.pingEmoji, style: const TextStyle(fontSize: 64)),
          ),
          const SizedBox(height: 16),
          Center(
            child: Text(
              server.name,
              style: TextStyle(
                fontSize: 24,
                fontWeight: FontWeight.bold,
                color: textSecondary(context),
              ),
            ),
          ),
          const SizedBox(height: 8),
          Center(
            child: Text(
              server.protocolLabel,
              style: TextStyle(
                color: textMuted(context),
                fontSize: 14,
              ),
            ),
          ),

          const SizedBox(height: 32),
          _sectionTitle(context, '🎯 Connection'),
          _infoTile(context, 'Address', server.address, Icons.dns),
          _infoTile(context, 'Port', server.port.toString(), Icons.numbers),

          if (server.transport != null) ...[
            const SizedBox(height: 24),
            _sectionTitle(context, '🚀 Transport'),
            
            Card(
              child: ListTile(
                leading: Icon(
                  _getTransportIcon(server.transport!.type),
                  size: 20,
                  color: accentColor(context),
                ),
                title: Text(
                  'Type',
                  style: TextStyle(fontSize: 13, color: textMuted(context)),
                ),
                subtitle: Text(
                  server.transport!.displayText,
                  style: TextStyle(color: textSecondary(context)),
                ),
                trailing: _getTransportBadge(server.transport!.type),
              ),
            ),
            
            if (server.transport!.host != null) 
              Card(
                child: ListTile(
                  leading: Icon(Icons.public, size: 20, color: accentColor(context)),
                  title: Text(
                    'Domain Fronting',
                    style: TextStyle(fontSize: 13, color: textMuted(context)),
                  ),
                  subtitle: Text(
                    server.transport!.host!,
                    style: TextStyle(
                      fontFamily: 'monospace',
                      color: textSecondary(context),
                    ),
                  ),
                ),
              ),
            
            if (server.transport!.path != null && server.transport!.type == 'websocket')
              Card(
                child: ListTile(
                  leading: Icon(Icons.route, size: 20, color: accentColor(context)),
                  title: Text(
                    'WebSocket Path',
                    style: TextStyle(fontSize: 13, color: textMuted(context)),
                  ),
                  subtitle: Text(
                    server.transport!.path!,
                    style: TextStyle(
                      fontFamily: 'monospace',
                      color: textSecondary(context),
                    ),
                  ),
                ),
              ),
            
            if (server.transport!.fallbackOrder != null && 
                server.transport!.fallbackOrder!.isNotEmpty)
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(16),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        children: [
                          Icon(Icons.swap_vert, size: 18, color: accentColor(context)),
                          const SizedBox(width: 8),
                          Text(
                            'Fallback Order',
                            style: TextStyle(
                              fontWeight: FontWeight.w600,
                              fontSize: 13,
                              color: textSecondary(context),
                            ),
                          ),
                        ],
                      ),
                      const SizedBox(height: 8),
                      Wrap(
                        spacing: 8,
                        runSpacing: 8,
                        children: server.transport!.fallbackOrder!.map((type) {
                          return Chip(
                            label: Text(
                              type.toUpperCase(),
                              style: const TextStyle(fontSize: 10),
                            ),
                            padding: const EdgeInsets.symmetric(horizontal: 8),
                            materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
                          );
                        }).toList(),
                      ),
                    ],
                  ),
                ),
              ),
          ],

          const SizedBox(height: 24),
          _sectionTitle(context, '⏱️ Latency'),

          Card(
            child: Column(
              children: [
                ListTile(
                  leading: Icon(Icons.bolt, size: 20, color: accentColor(context)),
                  title: Text(
                    'TCPing',
                    style: TextStyle(fontSize: 13, color: textMuted(context)),
                  ),
                  subtitle: Text(
                    'TCP socket connection time',
                    style: TextStyle(fontSize: 11, color: textMuted(context)),
                  ),
                  trailing: Column(
                    mainAxisAlignment: MainAxisAlignment.center,
                    crossAxisAlignment: CrossAxisAlignment.end,
                    children: [
                      Text(
                        server.pingText,
                        style: TextStyle(
                          fontWeight: FontWeight.w600,
                          color: _latencyColor(context, server.ping),
                        ),
                      ),
                      if (server.lastTested != null)
                        Text(
                          _formatLastTested(server.lastTested!),
                          style: TextStyle(
                            fontSize: 10,
                            color: textMuted(context),
                          ),
                        ),
                    ],
                  ),
                ),
                Divider(
                  height: 1,
                  color: accentColor(context).withOpacity(0.1),
                ),
                ListTile(
                  leading: Icon(
                    Icons.auto_graph,
                    size: 20,
                    color: accentColor(context),
                  ),
                  title: Text(
                    'Real Delay',
                    style: TextStyle(fontSize: 13, color: textMuted(context)),
                  ),
                  subtitle: Text(
                    'Full handshake + packet round-trip',
                    style: TextStyle(fontSize: 11, color: textMuted(context)),
                  ),
                  trailing: Column(
                    mainAxisAlignment: MainAxisAlignment.center,
                    crossAxisAlignment: CrossAxisAlignment.end,
                    children: [
                      Text(
                        server.realDelayText,
                        style: TextStyle(
                          fontWeight: FontWeight.w600,
                          color: _latencyColor(context, server.realDelay),
                        ),
                      ),
                      if (server.lastTested != null)
                        Text(
                          _formatLastTested(server.lastTested!),
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
          ),

          const SizedBox(height: 12),
          Row(
            children: [
              Expanded(
                child: OutlinedButton.icon(
                  onPressed: _isTesting ? null : () => _runTCPing(context),
                  icon: _isTesting
                      ? const SizedBox(
                          width: 14,
                          height: 14,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        )
                      : const Icon(Icons.bolt, size: 18),
                  label: const Text('TCPing'),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: FilledButton.icon(
                  onPressed: _isTesting ? null : () => _runRealDelay(context),
                  icon: _isTesting
                      ? const SizedBox(
                          width: 14,
                          height: 14,
                          child: CircularProgressIndicator(
                            strokeWidth: 2,
                            color: Colors.white,
                          ),
                        )
                      : const Icon(Icons.auto_graph, size: 18),
                  label: const Text('Real Delay'),
                ),
              ),
            ],
          ),

          const SizedBox(height: 24),
          _sectionTitle(context, '🔐 Security'),

          Card(
            child: ListTile(
              leading: Icon(Icons.key, size: 20, color: accentColor(context)),
              title: Text(
                'PSK',
                style: TextStyle(fontSize: 13, color: textMuted(context)),
              ),
              subtitle: Text(
                _showPsk
                    ? (server.psk.isEmpty ? 'Not set ⚠️' : server.psk)
                    : (server.psk.isEmpty
                        ? 'Not set ⚠️'
                        : '••••••••••••'),
                style: TextStyle(
                  fontFamily: 'monospace',
                  fontSize: 12,
                  color: server.psk.isEmpty
                      ? Colors.red
                      : textSecondary(context),
                ),
              ),
              trailing: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  IconButton(
                    icon: Icon(
                      _showPsk ? Icons.visibility_off : Icons.visibility,
                      size: 18,
                      color: textMuted(context),
                    ),
                    onPressed: () => setState(() => _showPsk = !_showPsk),
                  ),
                  if (server.psk.isNotEmpty)
                    IconButton(
                      icon: Icon(
                        Icons.copy,
                        size: 18,
                        color: textMuted(context),
                      ),
                      onPressed: () {
                        Clipboard.setData(ClipboardData(text: server.psk));
                        ScaffoldMessenger.of(context).showSnackBar(
                          const SnackBar(content: Text('PSK copied')),
                        );
                      },
                    ),
                ],
              ),
            ),
          ),

          Card(
            child: ListTile(
              leading: Icon(
                Icons.verified_user,
                size: 20,
                color: accentColor(context),
              ),
              title: Text(
                'Certificate PIN',
                style: TextStyle(fontSize: 13, color: textMuted(context)),
              ),
              subtitle: Text(
                server.certPin != null && server.certPin!.isNotEmpty
                    ? '${server.certPin!.substring(0, 16)}...'
                    : 'Not set (less secure)',
                style: TextStyle(
                  fontFamily: 'monospace',
                  fontSize: 12,
                  color: server.certPin != null && server.certPin!.isNotEmpty
                      ? textSecondary(context)
                      : Colors.orange,
                ),
              ),
            ),
          ),

          const SizedBox(height: 24),
          _sectionTitle(context, '🎯 Advanced Features'),

          Card(
            child: Column(
              children: [
                _featureTile(
                  context,
                  '🔄',
                  'SNI Rotation',
                  server.sniEnabled
                      ? 'Enabled (${server.sniMode}, ${server.sniDomains.length} domains)'
                      : 'Disabled',
                  server.sniEnabled,
                ),
                Divider(
                  height: 1,
                  color: accentColor(context).withOpacity(0.1),
                ),
                _featureTile(
                  context,
                  '🎭',
                  'Cover Traffic',
                  server.coverEnabled
                      ? 'Enabled (${server.coverDomains.length} domains)'
                      : 'Disabled',
                  server.coverEnabled,
                ),
                Divider(
                  height: 1,
                  color: accentColor(context).withOpacity(0.1),
                ),
                _featureTile(
                  context,
                  '🔌',
                  'DNS Fallback',
                  server.dnsFallbackEnabled
                      ? 'Enabled (${server.dnsFallbackMode})'
                      : 'Disabled',
                  server.dnsFallbackEnabled,
                ),
                Divider(
                  height: 1,
                  color: accentColor(context).withOpacity(0.1),
                ),
                _featureTile(
                  context,
                  '🔋',
                  'Battery-Aware',
                  server.batteryAwareEnabled ? 'Enabled' : 'Disabled',
                  server.batteryAwareEnabled,
                ),
                Divider(
                  height: 1,
                  color: accentColor(context).withOpacity(0.1),
                ),
                _featureTile(
                  context,
                  '💾',
                  'Data Saver',
                  server.dataSaverEnabled ? 'Enabled' : 'Disabled',
                  server.dataSaverEnabled,
                ),
              ],
            ),
          ),

          if (server.metadata != null) ..._buildMetadataSection(context, server.metadata!),

          if (server.coverEnabled) ...[
            const SizedBox(height: 24),
            _sectionTitle(context, '🎭 Cover Domains'),
            ...server.coverDomains.map(
              (d) => Card(
                child: ListTile(
                  leading: Icon(
                    Icons.public,
                    size: 20,
                    color: accentColor(context),
                  ),
                  title: Text(
                    d.domain,
                    style: TextStyle(color: textSecondary(context)),
                  ),
                  trailing: Text(
                    '${d.weight}%',
                    style: TextStyle(color: textMuted(context)),
                  ),
                ),
              ),
            ),
          ],

          if (server.sniEnabled) ...[
            const SizedBox(height: 24),
            _sectionTitle(context, '🔄 SNI Domains'),
            ...server.sniDomains.where((d) => d.checkHealth).map(
                  (d) => Card(
                    child: ListTile(
                      leading: Icon(
                        Icons.language,
                        size: 20,
                        color: accentColor(context),
                      ),
                      title: Text(
                        d.domain,
                        style: TextStyle(color: textSecondary(context)),
                      ),
                      trailing: server.sniMode == 'weighted'
                          ? Text(
                              '${d.weight}%',
                              style: TextStyle(color: textMuted(context)),
                            )
                          : null,
                    ),
                  ),
                ),
          ],

          const SizedBox(height: 32),
          Row(
            children: [
              Expanded(
                child: OutlinedButton.icon(
                  onPressed: () => _showQRCode(context, server),
                  icon: const Icon(Icons.qr_code),
                  label: const Text('QR Code'),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: OutlinedButton.icon(
                  onPressed: () => Navigator.push(
                    context,
                    MaterialPageRoute(
                      builder: (_) => ExportScreen(server: server),
                    ),
                  ),
                  icon: const Icon(Icons.share),
                  label: const Text('Export'),
                ),
              ),
            ],
          ),
          const SizedBox(height: 12),
          Row(
            children: [
              Expanded(
                child: FilledButton.icon(
                  onPressed: () {
                    final provider = context.read<AppProvider>();
                    provider.setActiveServer(server.id);
                    ScaffoldMessenger.of(context).showSnackBar(
                      SnackBar(
                        content: Text('${server.name} set as active'),
                      ),
                    );
                  },
                  icon: const Icon(Icons.check_circle_outline),
                  label: const Text('Set Active'),
                ),
              ),
            ],
          ),

          if (server.psk.isEmpty) ...[
            const SizedBox(height: 24),
            Card(
              color: Colors.red.withOpacity(0.1),
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Row(
                  children: [
                    const Icon(Icons.warning, color: Colors.red),
                    const SizedBox(width: 12),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          const Text(
                            'Security Warning',
                            style: TextStyle(
                              fontWeight: FontWeight.w600,
                              color: Colors.red,
                            ),
                          ),
                          const SizedBox(height: 4),
                          Text(
                            'PSK is not set. Edit this server and add a PSK.',
                            style: TextStyle(
                              color: Colors.red.withOpacity(0.7),
                              fontSize: 12,
                            ),
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ],

          const SizedBox(height: 32),
        ],
      ),
    );
  }

  List<Widget> _buildMetadataSection(BuildContext context, Metadata metadata) {
    return [
      const SizedBox(height: 24),
      _sectionTitle(context, '📋 Service Info'),
      
      if (metadata.country != null)
        _infoTile(context, 'Location', metadata.country!, Icons.location_on),
      
      if (metadata.ispHint != null)
        _infoTile(context, 'Provider', metadata.ispHint!, Icons.cloud),
      
      if (metadata.expiresAt != null)
        Card(
          color: metadata.isExpired 
              ? Colors.red.withOpacity(0.1) 
              : null,
          child: ListTile(
            leading: Icon(
              metadata.isExpired ? Icons.error : Icons.calendar_today,
              size: 20,
              color: metadata.isExpired ? Colors.red : accentColor(context),
            ),
            title: Text(
              'Expires',
              style: TextStyle(fontSize: 13, color: textMuted(context)),
            ),
            trailing: Text(
              metadata.expiryText,
              style: TextStyle(
                fontWeight: FontWeight.w600,
                color: metadata.isExpired ? Colors.red : textSecondary(context),
              ),
            ),
          ),
        ),
      
      if (metadata.quota != null && !metadata.quota!.unlimited) ...[
        Card(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Icon(Icons.data_usage, size: 20, color: accentColor(context)),
                    const SizedBox(width: 12),
                    Expanded(
                      child: Text(
                        'Data Quota',
                        style: TextStyle(
                          fontSize: 14,
                          fontWeight: FontWeight.w600,
                          color: textSecondary(context),
                        ),
                      ),
                    ),
                    Text(
                      '${metadata.quota!.usagePercent.toStringAsFixed(0)}%',
                      style: TextStyle(
                        fontWeight: FontWeight.bold,
                        color: metadata.quota!.progressColor,
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 12),
                ClipRRect(
                  borderRadius: BorderRadius.circular(4),
                  child: LinearProgressIndicator(
                    value: metadata.quota!.usagePercent / 100,
                    backgroundColor: Colors.grey.shade300,
                    valueColor: AlwaysStoppedAnimation<Color>(
                      metadata.quota!.progressColor,
                    ),
                    minHeight: 8,
                  ),
                ),
                const SizedBox(height: 8),
                Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    Text(
                      'Used: ${metadata.quota!.usedFormatted}',
                      style: TextStyle(fontSize: 11, color: textMuted(context)),
                    ),
                    Text(
                      'Remaining: ${metadata.quota!.remainingFormatted}',
                      style: TextStyle(fontSize: 11, color: textMuted(context)),
                    ),
                  ],
                ),
              ],
            ),
          ),
        ),
      ],
      
      if (metadata.announcement != null && 
          metadata.announcement!.enabled &&
          metadata.announcement!.text != null)
        Card(
          color: metadata.announcement!.color.withOpacity(0.1),
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  metadata.announcement!.icon,
                  style: const TextStyle(fontSize: 24),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'Announcement',
                        style: TextStyle(
                          fontWeight: FontWeight.w600,
                          color: metadata.announcement!.color,
                        ),
                      ),
                      const SizedBox(height: 4),
                      Text(
                        metadata.announcement!.text!,
                        style: TextStyle(
                          fontSize: 13,
                          color: textSecondary(context),
                          height: 1.4,
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ),
      
      if (metadata.notes != null && metadata.notes!.isNotEmpty)
        Card(
          child: ListTile(
            leading: Icon(Icons.notes, size: 20, color: accentColor(context)),
            title: Text(
              'Notes',
              style: TextStyle(fontSize: 13, color: textMuted(context)),
            ),
            subtitle: Text(
              metadata.notes!,
              style: TextStyle(fontSize: 12, color: textSecondary(context)),
            ),
          ),
        ),
      
      if (metadata.tags != null && metadata.tags!.isNotEmpty)
        Card(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Icon(Icons.label, size: 18, color: accentColor(context)),
                    const SizedBox(width: 8),
                    Text(
                      'Tags',
                      style: TextStyle(
                        fontSize: 13,
                        fontWeight: FontWeight.w600,
                        color: textMuted(context),
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 8),
                Wrap(
                  spacing: 8,
                  runSpacing: 8,
                  children: metadata.tags!.map((tag) {
                    return Chip(
                      label: Text(tag, style: const TextStyle(fontSize: 11)),
                      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                      materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
                    );
                  }).toList(),
                ),
              ],
            ),
          ),
        ),
    ];
  }

  Widget _sectionTitle(BuildContext context, String title) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: Text(
        title,
        style: TextStyle(
          fontSize: 18,
          fontWeight: FontWeight.w600,
          color: textPrimary(context),
        ),
      ),
    );
  }

  Widget _infoTile(
    BuildContext context,
    String label,
    String value,
    IconData icon,
  ) {
    return Card(
      child: ListTile(
        leading: Icon(icon, size: 20, color: accentColor(context)),
        title: Text(
          label,
          style: TextStyle(fontSize: 13, color: textMuted(context)),
        ),
        trailing: Text(
          value,
          style: TextStyle(
            fontWeight: FontWeight.w600,
            color: textSecondary(context),
          ),
        ),
      ),
    );
  }

  Widget _featureTile(
    BuildContext context,
    String emoji,
    String title,
    String subtitle,
    bool enabled,
  ) {
    return ListTile(
      leading: Text(emoji, style: const TextStyle(fontSize: 20)),
      title: Text(
        title,
        style: TextStyle(
          fontSize: 14,
          color: textSecondary(context),
        ),
      ),
      subtitle: Text(
        subtitle,
        style: TextStyle(
          fontSize: 12,
          color: textMuted(context),
        ),
      ),
      trailing: Icon(
        enabled ? Icons.check_circle : Icons.cancel,
        color: enabled ? Colors.green : Colors.grey,
        size: 20,
      ),
    );
  }

  IconData _getTransportIcon(String type) {
    switch (type) {
      case 'websocket':
        return Icons.wifi;
      case 'http2':
        return Icons.flash_on;
      case 'dns':
        return Icons.dns;
      default:
        return Icons.cable;
    }
  }

  Widget _getTransportBadge(String type) {
    Color color;
    String label;
    
    switch (type) {
      case 'websocket':
        color = Colors.green;
        label = 'Bypass';
        break;
      case 'http2':
        color = Colors.orange;
        label = 'Experimental';
        break;
      case 'dns':
        color = Colors.red;
        label = 'Slow';
        break;
      default:
        color = Colors.blue;
        label = 'Fast';
    }
    
    return Chip(
      label: Text(label, style: const TextStyle(fontSize: 10, color: Colors.white)),
      backgroundColor: color,
      padding: const EdgeInsets.symmetric(horizontal: 8),
      materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
    );
  }

  Color _latencyColor(BuildContext context, int? latency) {
    if (latency == null) return textMuted(context);
    if (latency < 0) return Colors.red;
    if (latency < 100) return Colors.green;
    if (latency < 300) return Colors.orange;
    return Colors.red;
  }

  String _formatLastTested(DateTime time) {
    final diff = DateTime.now().difference(time);
    if (diff.inMinutes < 1) return 'just now';
    if (diff.inMinutes < 60) return '${diff.inMinutes}m ago';
    if (diff.inHours < 24) return '${diff.inHours}h ago';
    return '${diff.inDays}d ago';
  }

  Future<void> _runTCPing(BuildContext context) async {
    setState(() => _isTesting = true);

    final provider = context.read<AppProvider>();
    final ping = await provider.pingServer(widget.server);

    provider.updateServer(
      widget.server.copyWith(
        ping: ping,
        lastTested: DateTime.now(),
      ),
    );

    setState(() => _isTesting = false);

    if (context.mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            ping > 0 ? 'TCPing: ${ping}ms' : 'TCPing: timeout',
          ),
          backgroundColor: ping > 0 ? Colors.green : Colors.red,
        ),
      );
    }
  }

  Future<void> _runRealDelay(BuildContext context) async {
    setState(() => _isTesting = true);

    final provider = context.read<AppProvider>();
    final delay = await provider.testRealDelay(widget.server);

    provider.updateServer(
      widget.server.copyWith(
        realDelay: delay,
        lastTested: DateTime.now(),
      ),
    );

    setState(() => _isTesting = false);

    if (context.mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            delay > 0 ? 'Real Delay: ${delay}ms' : 'Real Delay: timeout',
          ),
          backgroundColor: delay > 0 ? Colors.green : Colors.red,
        ),
      );
    }
  }

  void _showQRCode(BuildContext context, ServerConfig server) {
    final uri = server.toShareString();

    showDialog(
      context: context,
      builder: (context) => Dialog(
        child: Container(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Row(
                children: [
                  Text(
                    server.protocolEmoji,
                    style: const TextStyle(fontSize: 32),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          'Share ${server.name}',
                          style: Theme.of(context).textTheme.titleLarge?.copyWith(
                                fontWeight: FontWeight.bold,
                              ),
                        ),
                        Text(
                          'Scan this QR code',
                          style: TextStyle(
                            fontSize: 13,
                            color: textMuted(context),
                          ),
                        ),
                      ],
                    ),
                  ),
                  IconButton(
                    icon: const Icon(Icons.close),
                    onPressed: () => Navigator.pop(context),
                  ),
                ],
              ),
              const SizedBox(height: 24),
              Container(
                padding: const EdgeInsets.all(16),
                decoration: BoxDecoration(
                  color: Colors.white,
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(
                    color: Theme.of(context).brightness == Brightness.dark
                        ? Colors.grey.shade700
                        : Colors.grey.shade300,
                  ),
                ),
                child: QrImageView(
                  data: uri,
                  version: QrVersions.auto,
                  size: 280,
                  backgroundColor: Colors.white,
                  errorCorrectionLevel: QrErrorCorrectLevel.M,
                ),
              ),
              const SizedBox(height: 24),
              Row(
                children: [
                  Expanded(
                    child: OutlinedButton.icon(
                      onPressed: () {
                        Clipboard.setData(ClipboardData(text: uri));
                        ScaffoldMessenger.of(context).showSnackBar(
                          const SnackBar(
                            content: Text('Config URI copied to clipboard'),
                          ),
                        );
                      },
                      icon: const Icon(Icons.copy, size: 18),
                      label: const Text('Copy URI'),
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: FilledButton.icon(
                      onPressed: () {
                        Navigator.pop(context);
                      },
                      icon: const Icon(Icons.check, size: 18),
                      label: const Text('Done'),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 16),
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: Colors.blue.withOpacity(0.1),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Row(
                  children: [
                    Icon(
                      Icons.info_outline,
                      size: 18,
                      color: accentColor(context),
                    ),
                    const SizedBox(width: 8),
                    Expanded(
                      child: Text(
                        'Share via encrypted channels only (Signal, Telegram secret chat)',
                        style: TextStyle(
                          fontSize: 11,
                          color: textMuted(context),
                        ),
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
