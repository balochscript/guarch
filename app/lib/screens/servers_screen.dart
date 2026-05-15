import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import 'package:guarch/app.dart';
import 'package:guarch/providers/app_provider.dart';
import 'package:guarch/models/server_config.dart';
import 'package:guarch/screens/add_server_screen.dart';
import 'package:guarch/screens/server_detail_screen.dart';
import 'package:share_plus/share_plus.dart';
import 'package:qr_flutter/qr_flutter.dart';
import 'package:qr_code_scanner/qr_code_scanner.dart';
import 'package:image_picker/image_picker.dart';
import 'dart:io';

class ServersScreen extends StatelessWidget {
  const ServersScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Consumer<AppProvider>(
      builder: (context, provider, _) {
        return Scaffold(
          appBar: AppBar(
            title: const Text('Servers'),
            actions: [
              PopupMenuButton<String>(
                icon: const Icon(Icons.speed),
                tooltip: 'Ping options',
                onSelected: (value) {
                  if (value == 'tcping') {
                    provider.pingAllServers(includeRealDelay: false);
                  } else if (value == 'real') {
                    provider.pingAllServers(includeRealDelay: true);
                    ScaffoldMessenger.of(context).showSnackBar(
                      const SnackBar(
                        content: Text('Real delay test may take longer...'),
                        duration: Duration(seconds: 2),
                      ),
                    );
                  }
                },
                itemBuilder: (context) => [
                  const PopupMenuItem(
                    value: 'tcping',
                    child: Row(
                      children: [
                        Icon(Icons.bolt, size: 18),
                        SizedBox(width: 8),
                        Text('TCPing (Fast)'),
                      ],
                    ),
                  ),
                  const PopupMenuItem(
                    value: 'real',
                    child: Row(
                      children: [
                        Icon(Icons.auto_graph, size: 18),
                        SizedBox(width: 8),
                        Text('Real Delay (Accurate)'),
                      ],
                    ),
                  ),
                ],
              ),
            ],
          ),
          body: provider.servers.isEmpty
              ? _buildEmpty(context)
              : _buildList(context, provider),
          floatingActionButton: FloatingActionButton(
            onPressed: () => _showAddServerMenu(context, provider),
            child: const Icon(Icons.add),
          ),
        );
      },
    );
  }

  Widget _buildEmpty(BuildContext context) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(
            Icons.dns_outlined,
            size: 80,
            color: accentColor(context).withOpacity(0.3),
          ),
          const SizedBox(height: 16),
          Text(
            'No servers yet',
            style: TextStyle(
              color: textMuted(context),
              fontSize: 18,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            'Add a server or import a config',
            style: TextStyle(
              color: textMuted(context).withOpacity(0.6),
            ),
          ),
          const SizedBox(height: 24),
          FilledButton.icon(
            onPressed: () => _showAddServerMenu(context, Provider.of<AppProvider>(context, listen: false)),
            icon: const Icon(Icons.add),
            label: const Text('Add Server'),
          ),
        ],
      ),
    );
  }

  Widget _buildList(BuildContext context, AppProvider provider) {
    return ListView.builder(
      padding: const EdgeInsets.all(16),
      itemCount: provider.servers.length,
      itemBuilder: (context, index) {
        final server = provider.servers[index];
        final isActive = provider.activeServer?.id == server.id;

        return Card(
          margin: const EdgeInsets.only(bottom: 12),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(16),
            side: isActive
                ? BorderSide(color: accentColor(context), width: 2)
                : BorderSide.none,
          ),
          child: InkWell(
            borderRadius: BorderRadius.circular(16),
            onTap: () => provider.setActiveServer(server.id),
            onLongPress: () => Navigator.push(
              context,
              MaterialPageRoute(
                builder: (_) => ServerDetailScreen(server: server),
              ),
            ),
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                children: [
                  Row(
                    children: [
                      Text(
                        server.pingEmoji,
                        style: const TextStyle(fontSize: 28),
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Row(
                              children: [
                                Flexible(
                                  child: Text(
                                    server.name,
                                    style: TextStyle(
                                      fontWeight: FontWeight.w600,
                                      fontSize: 16,
                                      color: textSecondary(context),
                                    ),
                                    overflow: TextOverflow.ellipsis,
                                  ),
                                ),
                                if (isActive) ...[
                                  const SizedBox(width: 8),
                                  Container(
                                    padding: const EdgeInsets.symmetric(
                                      horizontal: 8,
                                      vertical: 2,
                                    ),
                                    decoration: BoxDecoration(
                                      color: accentColor(context).withOpacity(0.2),
                                      borderRadius: BorderRadius.circular(8),
                                    ),
                                    child: Text(
                                      'Active',
                                      style: TextStyle(
                                        fontSize: 10,
                                        color: accentColor(context),
                                        fontWeight: FontWeight.w600,
                                      ),
                                    ),
                                  ),
                                ],
                              ],
                            ),
                            const SizedBox(height: 4),
                            Row(
                              children: [
                                Text(
                                  server.protocolEmoji,
                                  style: const TextStyle(fontSize: 12),
                                ),
                                const SizedBox(width: 4),
                                Text(
                                  server.protocolLabel,
                                  style: TextStyle(
                                    color: textMuted(context),
                                    fontSize: 11,
                                    fontWeight: FontWeight.w500,
                                  ),
                                ),
                                const SizedBox(width: 8),
                                Expanded(
                                  child: Text(
                                    server.fullAddress,
                                    style: TextStyle(
                                      color: textMuted(context).withOpacity(0.7),
                                      fontSize: 13,
                                    ),
                                    overflow: TextOverflow.ellipsis,
                                  ),
                                ),
                              ],
                            ),
                          ],
                        ),
                      ),
                      Column(
                        crossAxisAlignment: CrossAxisAlignment.end,
                        children: [
                          Row(
                            children: [
                              const Icon(
                                Icons.bolt,
                                size: 12,
                                color: Colors.grey,
                              ),
                              const SizedBox(width: 4),
                              Text(
                                server.pingText,
                                style: TextStyle(
                                  fontWeight: FontWeight.w600,
                                  fontSize: 13,
                                  color: _pingColor(context, server.ping),
                                ),
                              ),
                            ],
                          ),
                          if (server.realDelay != null) ...[
                            const SizedBox(height: 2),
                            Row(
                              children: [
                                const Icon(
                                  Icons.auto_graph,
                                  size: 12,
                                  color: Colors.grey,
                                ),
                                const SizedBox(width: 4),
                                Text(
                                  server.realDelayText,
                                  style: TextStyle(
                                    fontWeight: FontWeight.w600,
                                    fontSize: 11,
                                    color: _pingColor(context, server.realDelay),
                                  ),
                                ),
                              ],
                            ),
                          ],
                          if (server.coverEnabled) ...[
                            const SizedBox(height: 2),
                            Text(
                              '🎭 ${server.coverDomains.length} sites',
                              style: TextStyle(
                                fontSize: 10,
                                color: textMuted(context),
                              ),
                            ),
                          ],
                          if (server.lastTested != null) ...[
                            const SizedBox(height: 2),
                            Text(
                              server.lastTestedText,
                              style: TextStyle(
                                fontSize: 9,
                                color: textMuted(context).withOpacity(0.5),
                              ),
                            ),
                          ],
                        ],
                      ),
                    ],
                  ),
                  const SizedBox(height: 8),
                  const Divider(height: 1, thickness: 0.5),
                  const SizedBox(height: 4),
                  Row(
                    mainAxisAlignment: MainAxisAlignment.end,
                    children: [
                      _actionButton(
                        context,
                        Icons.bolt,
                        'TCPing',
                        () async {
                          final ping = await provider.pingServer(server);
                          provider.updateServer(server.copyWith(
                            ping: ping,
                            lastTested: DateTime.now(),
                          ));
                        },
                      ),
                      _actionButton(
                        context,
                        Icons.auto_graph,
                        'Real Delay',
                        () async {
                          final delay = await provider.testRealDelay(server);
                          provider.updateServer(server.copyWith(
                            realDelay: delay,
                            lastTested: DateTime.now(),
                          ));
                        },
                      ),
                      _actionButton(
                        context,
                        Icons.share,
                        'Share',
                        () => _showShareMenu(context, provider, server),
                      ),
                      _actionButton(
                        context,
                        Icons.edit,
                        'Edit',
                        () => _openEditServer(context, server),
                      ),
                      _actionButton(
                        context,
                        Icons.delete_outline,
                        'Delete',
                        () => _confirmDelete(context, provider, server),
                        color: Colors.red.withOpacity(0.7),
                      ),
                    ],
                  ),
                ],
              ),
            ),
          ),
        );
      },
    );
  }

  Widget _actionButton(
    BuildContext context,
    IconData icon,
    String tooltip,
    VoidCallback onTap, {
    Color? color,
  }) {
    return IconButton(
      icon: Icon(
        icon,
        size: 18,
        color: color ?? textMuted(context),
      ),
      tooltip: tooltip,
      onPressed: onTap,
      visualDensity: VisualDensity.compact,
    );
  }

  Color _pingColor(BuildContext context, int? ping) {
    if (ping == null) return textMuted(context);
    if (ping < 0) return Colors.red;
    if (ping < 100) return Colors.green;
    if (ping < 300) return accentColor(context);
    return Colors.red;
  }

  void _showAddServerMenu(BuildContext context, AppProvider provider) {
    showModalBottomSheet(
      context: context,
      builder: (ctx) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            ListTile(
              leading: const Icon(Icons.content_paste),
              title: const Text('Import from Clipboard'),
              onTap: () async {
                Navigator.pop(ctx);
                final data = await Clipboard.getData(Clipboard.kTextPlain);
                if (data?.text != null && data!.text!.isNotEmpty) {
                  provider.importConfig(data.text!);
                  if (context.mounted) {
                    ScaffoldMessenger.of(context).showSnackBar(
                      const SnackBar(content: Text('Config imported')),
                    );
                  }
                } else {
                  if (context.mounted) {
                    ScaffoldMessenger.of(context).showSnackBar(
                      const SnackBar(content: Text('Clipboard is empty')),
                    );
                  }
                }
              },
            ),
            ListTile(
              leading: const Icon(Icons.qr_code_scanner),
              title: const Text('Scan QR Code'),
              onTap: () {
                Navigator.pop(ctx);
                _scanQRCode(context, provider);
              },
            ),
            ListTile(
              leading: const Icon(Icons.photo_library),
              title: const Text('Pick QR from Gallery'),
              onTap: () {
                Navigator.pop(ctx);
                _pickQRFromGallery(context, provider);
              },
            ),
            ListTile(
              leading: const Icon(Icons.add),
              title: const Text('Create Manually'),
              onTap: () {
                Navigator.pop(ctx);
                Navigator.push(
                  context,
                  MaterialPageRoute(builder: (_) => const AddServerScreen()),
                );
              },
            ),
          ],
        ),
      ),
    );
  }

  void _showShareMenu(BuildContext context, AppProvider provider, ServerConfig server) {
    showModalBottomSheet(
      context: context,
      builder: (ctx) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            ListTile(
              leading: const Icon(Icons.copy),
              title: const Text('Copy URI'),
              onTap: () {
                Navigator.pop(ctx);
                Clipboard.setData(ClipboardData(text: provider.exportConfig(server)));
                ScaffoldMessenger.of(context).showSnackBar(
                  const SnackBar(content: Text('URI copied')),
                );
              },
            ),
            ListTile(
              leading: const Icon(Icons.share),
              title: const Text('Share URI'),
              onTap: () {
                Navigator.pop(ctx);
                Share.share(provider.exportConfig(server));
              },
            ),
            ListTile(
              leading: const Icon(Icons.code),
              title: const Text('Copy JSON'),
              onTap: () {
                Navigator.pop(ctx);
                Clipboard.setData(ClipboardData(text: provider.exportConfigJson(server)));
                ScaffoldMessenger.of(context).showSnackBar(
                  const SnackBar(content: Text('JSON copied')),
                );
              },
            ),
            ListTile(
              leading: const Icon(Icons.share),
              title: const Text('Share JSON'),
              onTap: () {
                Navigator.pop(ctx);
                Share.share(provider.exportConfigJson(server));
              },
            ),
            ListTile(
              leading: const Icon(Icons.qr_code),
              title: const Text('Share QR Code'),
              onTap: () {
                Navigator.pop(ctx);
                _showQRCode(context, provider.exportConfig(server), server.name);
              },
            ),
          ],
        ),
      ),
    );
  }

  void _showQRCode(BuildContext context, String data, String title) {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(title),
        content: SizedBox(
          width: 280,
          height: 280,
          child: QrImageView(
            data: data,
            version: QrVersions.auto,
            backgroundColor: Colors.white,
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

  void _scanQRCode(BuildContext context, AppProvider provider) {
    Navigator.push(
      context,
      MaterialPageRoute(
        builder: (_) => Scaffold(
          appBar: AppBar(title: const Text('Scan QR Code')),
          body: QRView(
            key: GlobalKey(debugLabel: 'QR'),
            onQRViewCreated: (controller) {
              controller.scannedDataStream.listen((scanData) {
                if (scanData.code != null) {
                  controller.dispose();
                  Navigator.pop(context);
                  provider.importConfig(scanData.code!);
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(content: Text('QR Code scanned')),
                  );
                }
              });
            },
          ),
        ),
      ),
    );
  }

  Future<void> _pickQRFromGallery(BuildContext context, AppProvider provider) async {
    final picker = ImagePicker();
    final image = await picker.pickImage(source: ImageSource.gallery);
    
    if (image != null) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('QR parsing not yet implemented')),
      );
    }
  }

  void _openEditServer(BuildContext context, ServerConfig server) {
    Navigator.push(
      context,
      MaterialPageRoute(builder: (_) => AddServerScreen(server: server)),
    );
  }

  void _confirmDelete(
    BuildContext context,
    AppProvider provider,
    ServerConfig server,
  ) {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(
          'Delete Server',
          style: TextStyle(color: textPrimary(context)),
        ),
        content: Text(
          'Delete "${server.name}"?\nThis action cannot be undone.',
          style: TextStyle(color: textSecondary(context)),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: Text(
              'Cancel',
              style: TextStyle(color: textMuted(context)),
            ),
          ),
          TextButton(
            onPressed: () {
              provider.removeServer(server.id);
              Navigator.pop(ctx);
              ScaffoldMessenger.of(context).showSnackBar(
                SnackBar(
                  content: Text('${server.name} deleted'),
                  action: SnackBarAction(
                    label: 'Undo',
                    onPressed: () => provider.addServer(server),
                  ),
                ),
              );
            },
            style: TextButton.styleFrom(foregroundColor: Colors.red),
            child: const Text('Delete'),
          ),
        ],
      ),
    );
  }
}
