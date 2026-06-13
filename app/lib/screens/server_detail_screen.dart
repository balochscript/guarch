import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter/rendering.dart';
import 'dart:ui' as ui;
import 'dart:io';
import 'package:provider/provider.dart';
import 'package:qr_flutter/qr_flutter.dart';
import 'package:share_plus/share_plus.dart';
import 'package:path_provider/path_provider.dart';
import 'package:guarch/app.dart';
import 'package:guarch/models/server_config.dart';
import 'package:guarch/providers/app_provider.dart';
import 'package:guarch/screens/add_server_screen.dart';

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
            child: Icon(
              _getProtocolIcon(server.protocol),
              size: 64,
              color: _getProtocolColor(server.protocol),
            ),
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
          _sectionTitle(context, 'Connection', Icons.link),
          _infoTile(context, 'Address', server.address, Icons.dns),
          _infoTile(context, 'Port', server.port.toString(), Icons.numbers),

          if (server.protocol == 'grouk') ...[
            const SizedBox(height: 24),
            _sectionTitle(context, 'Grouk Settings', Icons.flash_on),
            
            Card(
              child: Column(
                children: [
                  ListTile(
                    leading: Icon(
                      Icons.shield,
                      size: 20,
                      color: server.groukFecEnabled ? Colors.purple : Colors.grey,
                    ),
                    title: Text(
                      'FEC (Forward Error Correction)',
                      style: TextStyle(fontSize: 14, color: textSecondary(context)),
                    ),
                    subtitle: Text(
                      server.groukFecEnabled 
                          ? 'Enabled - Recovers lost UDP packets'
                          : 'Disabled',
                      style: TextStyle(fontSize: 12, color: textMuted(context)),
                    ),
                    trailing: Icon(
                      server.groukFecEnabled ? Icons.check_circle : Icons.cancel,
                      color: server.groukFecEnabled ? Colors.green : Colors.grey,
                      size: 20,
                    ),
                  ),
                  
                  if (server.groukFecEnabled) ...[
                    Divider(height: 1, color: accentColor(context).withOpacity(0.1)),
                    Padding(
                      padding: const EdgeInsets.all(16),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Container(
                            padding: const EdgeInsets.all(12),
                            decoration: BoxDecoration(
                              color: Colors.purple.withOpacity(0.1),
                              borderRadius: BorderRadius.circular(8),
                              border: Border.all(color: Colors.purple.withOpacity(0.3)),
                            ),
                            child: Row(
                              children: [
                                const Icon(Icons.info_outline, size: 16, color: Colors.purple),
                                const SizedBox(width: 12),
                                Expanded(
                                  child: Text(
                                    'Implementation: Simple XOR (can recover 1 packet per group)',
                                    style: TextStyle(fontSize: 11, color: textMuted(context), height: 1.3),
                                  ),
                                ),
                              ],
                            ),
                          ),
                          
                          const SizedBox(height: 16),
                          
                          Row(
                            children: [
                              Expanded(
                                child: Column(
                                  crossAxisAlignment: CrossAxisAlignment.start,
                                  children: [
                                    Row(
                                      children: [
                                        const Icon(Icons.grain, size: 16, color: Colors.blue),
                                        const SizedBox(width: 8),
                                        Text(
                                          'Data Shards',
                                          style: TextStyle(
                                            fontSize: 13,
                                            fontWeight: FontWeight.w600,
                                            color: textSecondary(context),
                                          ),
                                        ),
                                      ],
                                    ),
                                    const SizedBox(height: 4),
                                    Text(
                                      '${server.groukFecDataShards} packets',
                                      style: TextStyle(
                                        fontSize: 16,
                                        fontWeight: FontWeight.bold,
                                        color: Colors.blue,
                                      ),
                                    ),
                                    const SizedBox(height: 2),
                                    Text(
                                      'Group size (in use)',
                                      style: TextStyle(fontSize: 10, color: textMuted(context)),
                                    ),
                                  ],
                                ),
                              ),
                              
                              Container(
                                width: 1,
                                height: 50,
                                color: accentColor(context).withOpacity(0.2),
                              ),
                              
                              Expanded(
                                child: Column(
                                  crossAxisAlignment: CrossAxisAlignment.start,
                                  children: [
                                    Row(
                                      children: [
                                        const SizedBox(width: 16),
                                        const Icon(Icons.backup, size: 16, color: Colors.grey),
                                        const SizedBox(width: 8),
                                        Text(
                                          'Parity Shards',
                                          style: TextStyle(
                                            fontSize: 13,
                                            fontWeight: FontWeight.w600,
                                            color: textSecondary(context),
                                          ),
                                        ),
                                      ],
                                    ),
                                    const SizedBox(height: 4),
                                    Padding(
                                      padding: const EdgeInsets.only(left: 16),
                                      child: Text(
                                        '${server.groukFecParityShards}',
                                        style: const TextStyle(
                                          fontSize: 16,
                                          fontWeight: FontWeight.bold,
                                          color: Colors.grey,
                                        ),
                                      ),
                                    ),
                                    const SizedBox(height: 2),
                                    Padding(
                                      padding: const EdgeInsets.only(left: 16),
                                      child: Text(
                                        'Reserved for future',
                                        style: TextStyle(fontSize: 10, color: textMuted(context), fontStyle: FontStyle.italic),
                                      ),
                                    ),
                                  ],
                                ),
                              ),
                            ],
                          ),
                          
                          const SizedBox(height: 16),
                          
                          Container(
                            padding: const EdgeInsets.all(10),
                            decoration: BoxDecoration(
                              color: Colors.green.withOpacity(0.1),
                              borderRadius: BorderRadius.circular(6),
                            ),
                            child: Row(
                              children: [
                                const Icon(Icons.check_circle, size: 14, color: Colors.green),
                                const SizedBox(width: 8),
                                Expanded(
                                  child: Text(
                                    'Can recover 1 lost packet per ${server.groukFecDataShards} packets',
                                    style: TextStyle(fontSize: 11, color: textMuted(context)),
                                  ),
                                ),
                              ],
                            ),
                          ),
                        ],
                      ),
                    ),
                  ],
                ],
              ),
            ),
          ],

          if (server.transport != null) ...[
            const SizedBox(height: 24),
            _sectionTitle(context, 'Transport', Icons.rocket_launch),
            
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
          _sectionTitle(context, 'Latency', Icons.speed),

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
          _sectionTitle(context, 'Security', Icons.lock),

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

          if (server.protocol != 'grouk')
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
          _sectionTitle(context, 'Advanced Features', Icons.tune),

          Card(
            child: Column(
              children: [
                if (server.protocol != 'grouk') ...[
                  _featureTile(
                    context,
                    Icons.shield_outlined,
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
                    Icons.theater_comedy,
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
                    Icons.dns,
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
                ],
                _featureTile(
                  context,
                  Icons.battery_charging_full,
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
                  Icons.data_saver_on,
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
            _sectionTitle(context, 'Cover Domains', Icons.theater_comedy),
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
            _sectionTitle(context, 'SNI Domains', Icons.shield_outlined),
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
                  onPressed: () => _showExportDialog(context, server),
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
      _sectionTitle(context, 'Service Info', Icons.info),
      
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
                Icon(
                  Icons.campaign,
                  size: 24,
                  color: metadata.announcement!.color,
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

  Widget _sectionTitle(BuildContext context, String title, IconData icon) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: Row(
        children: [
          Icon(icon, size: 20, color: accentColor(context)),
          const SizedBox(width: 8),
          Text(
            title,
            style: TextStyle(
              fontSize: 18,
              fontWeight: FontWeight.w600,
              color: textPrimary(context),
            ),
          ),
        ],
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
    IconData icon,
    String title,
    String subtitle,
    bool enabled,
  ) {
    return ListTile(
      leading: Icon(icon, size: 20, color: accentColor(context)),
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

  void _showExportDialog(BuildContext context, ServerConfig server) {
    final provider = context.read<AppProvider>();
    final link = provider.exportConfig(server);
    final json = provider.exportConfigJson(server);

    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      builder: (context) => DraggableScrollableSheet(
        initialChildSize: 0.7,
        minChildSize: 0.5,
        maxChildSize: 0.95,
        expand: false,
        builder: (context, scrollController) => Container(
          padding: const EdgeInsets.all(24),
          child: ListView(
            controller: scrollController,
            children: [
              Row(
                children: [
                  Icon(
                    Icons.share,
                    size: 28,
                    color: accentColor(context),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          'Export ${server.name}',
                          style: Theme.of(context).textTheme.titleLarge?.copyWith(
                                fontWeight: FontWeight.bold,
                              ),
                        ),
                        Text(
                          'Share this configuration',
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
              
              if (server.psk.isNotEmpty) ...[
                const SizedBox(height: 16),
                Card(
                  color: Colors.orange.withOpacity(0.1),
                  child: Padding(
                    padding: const EdgeInsets.all(12),
                    child: Row(
                      children: [
                        const Icon(Icons.warning_amber, color: Colors.orange, size: 20),
                        const SizedBox(width: 12),
                        Expanded(
                          child: Text(
                            'Contains PSK. Share only through secure channels.',
                            style: TextStyle(
                              color: Colors.orange.shade800,
                              fontSize: 12,
                            ),
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
              ],

              const SizedBox(height: 24),
              Row(
                children: [
                  Icon(Icons.link, size: 20, color: accentColor(context)),
                  const SizedBox(width: 8),
                  Text(
                    '${server.protocol.toUpperCase()} Link',
                    style: TextStyle(
                      fontSize: 16,
                      fontWeight: FontWeight.w600,
                      color: textPrimary(context),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 8),
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(12),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      SelectableText(
                        link,
                        style: TextStyle(
                          fontFamily: 'monospace',
                          fontSize: 11,
                          color: textSecondary(context),
                        ),
                      ),
                      const SizedBox(height: 12),
                      Row(
                        children: [
                          Expanded(
                            child: OutlinedButton.icon(
                              onPressed: () {
                                Clipboard.setData(ClipboardData(text: link));
                                ScaffoldMessenger.of(context).showSnackBar(
                                  const SnackBar(content: Text('Link copied!')),
                                );
                              },
                              icon: const Icon(Icons.copy, size: 16),
                              label: const Text('Copy'),
                            ),
                          ),
                          const SizedBox(width: 8),
                          Expanded(
                            child: FilledButton.icon(
                              onPressed: () => Share.share(link),
                              icon: const Icon(Icons.share, size: 16),
                              label: const Text('Share'),
                            ),
                          ),
                        ],
                      ),
                    ],
                  ),
                ),
              ),

              const SizedBox(height: 24),
              Row(
                children: [
                  Icon(Icons.code, size: 20, color: accentColor(context)),
                  const SizedBox(width: 8),
                  Text(
                    'JSON Config',
                    style: TextStyle(
                      fontSize: 16,
                      fontWeight: FontWeight.w600,
                      color: textPrimary(context),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 8),
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(12),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      ConstrainedBox(
                        constraints: const BoxConstraints(maxHeight: 200),
                        child: SingleChildScrollView(
                          child: SelectableText(
                            json,
                            style: TextStyle(
                              fontFamily: 'monospace',
                              fontSize: 10,
                              color: textSecondary(context),
                            ),
                          ),
                        ),
                      ),
                      const SizedBox(height: 12),
                      Row(
                        children: [
                          Expanded(
                            child: OutlinedButton.icon(
                              onPressed: () {
                                Clipboard.setData(ClipboardData(text: json));
                                ScaffoldMessenger.of(context).showSnackBar(
                                  const SnackBar(content: Text('JSON copied!')),
                                );
                              },
                              icon: const Icon(Icons.copy, size: 16),
                              label: const Text('Copy'),
                            ),
                          ),
                          const SizedBox(width: 8),
                          Expanded(
                            child: FilledButton.icon(
                              onPressed: () => Share.share(json),
                              icon: const Icon(Icons.share, size: 16),
                              label: const Text('Share'),
                            ),
                          ),
                        ],
                      ),
                    ],
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  void _showQRCode(BuildContext context, ServerConfig server) {
    final GlobalKey qrKey = GlobalKey();
    final uri = server.toShareString();

    showDialog(
      context: context,
      builder: (dialogContext) => Dialog(
        child: Container(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Row(
                children: [
                  Icon(
                    _getProtocolIcon(server.protocol),
                    size: 32,
                    color: _getProtocolColor(server.protocol),
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
                          'Scan or share this QR code',
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
                    onPressed: () => Navigator.pop(dialogContext),
                  ),
                ],
              ),
              const SizedBox(height: 24),
              RepaintBoundary(
                key: qrKey,
                child: Container(
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
                            content: Text('Config URI copied'),
                            duration: Duration(seconds: 2),
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
                      onPressed: () async {
                        try {
                          final boundary = qrKey.currentContext!.findRenderObject() as RenderRepaintBoundary;
                          final image = await boundary.toImage(pixelRatio: 3.0);
                          final byteData = await image.toByteData(format: ui.ImageByteFormat.png);
                          final pngBytes = byteData!.buffer.asUint8List();

                          final tempDir = await getTemporaryDirectory();
                          final fileName = 'guarch_${server.name.replaceAll(' ', '_')}_qr.png';
                          final file = File('${tempDir.path}/$fileName');
                          await file.writeAsBytes(pngBytes);

                          await Share.shareXFiles(
                            [XFile(file.path)],
                            text: 'Guarch VPN Config: ${server.name}\n\n$uri',
                            subject: 'Guarch VPN Configuration',
                          );

                          if (context.mounted) {
                            Navigator.pop(dialogContext);
                          }
                        } catch (e) {
                          if (context.mounted) {
                            ScaffoldMessenger.of(context).showSnackBar(
                              SnackBar(
                                content: Text('Share failed: $e'),
                                backgroundColor: Colors.red,
                              ),
                            );
                          }
                        }
                      },
                      icon: const Icon(Icons.share, size: 18),
                      label: const Text('Share QR'),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 12),
              OutlinedButton.icon(
                onPressed: () async {
                  try {
                    final boundary = qrKey.currentContext!.findRenderObject() as RenderRepaintBoundary;
                    final image = await boundary.toImage(pixelRatio: 3.0);
                    final byteData = await image.toByteData(format: ui.ImageByteFormat.png);
                    final pngBytes = byteData!.buffer.asUint8List();

                    final tempDir = await getTemporaryDirectory();
                    final fileName = 'guarch_${server.name.replaceAll(' ', '_')}_qr.png';
                    final file = File('${tempDir.path}/$fileName');
                    await file.writeAsBytes(pngBytes);

                    if (context.mounted) {
                      ScaffoldMessenger.of(context).showSnackBar(
                        SnackBar(
                          content: Text('QR saved to ${file.path}'),
                          duration: const Duration(seconds: 3),
                          action: SnackBarAction(
                            label: 'Share',
                            onPressed: () async {
                              await Share.shareXFiles([XFile(file.path)]);
                            },
                          ),
                        ),
                      );
                    }
                  } catch (e) {
                    if (context.mounted) {
                      ScaffoldMessenger.of(context).showSnackBar(
                        SnackBar(
                          content: Text('Save failed: $e'),
                          backgroundColor: Colors.red,
                        ),
                      );
                    }
                  }
                },
                icon: const Icon(Icons.download, size: 18),
                label: const Text('Save Image'),
                style: OutlinedButton.styleFrom(
                  minimumSize: const Size(double.infinity, 40),
                ),
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
                        'Share via encrypted channels only (Signal, Telegram)',
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
