import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter/rendering.dart';
import 'dart:ui' as ui;
import 'dart:io';
import 'package:provider/provider.dart';
import 'package:guarch/app.dart';
import 'package:guarch/providers/app_provider.dart';
import 'package:guarch/models/server_config.dart';
import 'package:guarch/screens/add_server_screen.dart';
import 'package:guarch/screens/server_detail_screen.dart';
import 'package:share_plus/share_plus.dart';
import 'package:qr_flutter/qr_flutter.dart';
import 'package:mobile_scanner/mobile_scanner.dart';
import 'package:image_picker/image_picker.dart';
import 'package:path_provider/path_provider.dart';
import 'package:google_mlkit_barcode_scanning/google_mlkit_barcode_scanning.dart' as mlkit;

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
                      Container(
                        padding: const EdgeInsets.all(8),
                        decoration: BoxDecoration(
                          color: _getProtocolColor(server.protocol).withOpacity(0.15),
                          borderRadius: BorderRadius.circular(12),
                        ),
                        child: Icon(
                          _getProtocolIcon(server.protocol),
                          size: 28,
                          color: _getProtocolColor(server.protocol),
                        ),
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
                                Icon(
                                  _getProtocolIcon(server.protocol),
                                  size: 12,
                                  color: _getProtocolColor(server.protocol),
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
                            Row(
                              children: [
                                Icon(
                                  Icons.theater_comedy,
                                  size: 10,
                                  color: textMuted(context),
                                ),
                                const SizedBox(width: 2),
                                Text(
                                  '${server.coverDomains.length} sites',
                                  style: TextStyle(
                                    fontSize: 10,
                                    color: textMuted(context),
                                  ),
                                ),
                              ],
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

  void _scanQRCode(BuildContext context, AppProvider provider) {
    final scannerController = MobileScannerController();
    bool hasScanned = false;

    Navigator.push(
      context,
      MaterialPageRoute(
        builder: (scanContext) => Scaffold(
          appBar: AppBar(
            title: const Text('Scan QR Code'),
            backgroundColor: Colors.black,
            actions: [
              IconButton(
                icon: const Icon(Icons.flash_on),
                onPressed: () => scannerController.toggleTorch(),
              ),
              IconButton(
                icon: const Icon(Icons.flip_camera_ios),
                onPressed: () => scannerController.switchCamera(),
              ),
            ],
          ),
          body: Stack(
            children: [
              MobileScanner(
                controller: scannerController,
                onDetect: (capture) {
                  if (hasScanned) return;

                  final List<Barcode> barcodes = capture.barcodes;
                  for (final barcode in barcodes) {
                    if (barcode.rawValue != null && barcode.rawValue!.isNotEmpty) {
                      hasScanned = true;
                      scannerController.dispose();

                      Navigator.of(scanContext).pop();

                      provider.importConfig(barcode.rawValue!);

                      if (context.mounted) {
                        ScaffoldMessenger.of(context).showSnackBar(
                          const SnackBar(
                            content: Text('✅ QR Code scanned successfully'),
                            backgroundColor: Colors.green,
                            duration: Duration(seconds: 2),
                          ),
                        );
                      }
                      break;
                    }
                  }
                },
              ),
              Positioned(
                bottom: 0,
                left: 0,
                right: 0,
                child: Container(
                  padding: const EdgeInsets.all(24),
                  decoration: BoxDecoration(
                    gradient: LinearGradient(
                      begin: Alignment.bottomCenter,
                      end: Alignment.topCenter,
                      colors: [
                        Colors.black.withOpacity(0.8),
                        Colors.transparent,
                      ],
                    ),
                  ),
                  child: const Column(
                    children: [
                      Icon(
                        Icons.qr_code_scanner,
                        size: 48,
                        color: Colors.white,
                      ),
                      SizedBox(height: 12),
                      Text(
                        'Point camera at QR code',
                        style: TextStyle(
                          color: Colors.white,
                          fontSize: 16,
                          fontWeight: FontWeight.w500,
                        ),
                      ),
                      SizedBox(height: 4),
                      Text(
                        'Will automatically scan when detected',
                        style: TextStyle(
                          color: Colors.white70,
                          fontSize: 13,
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    ).then((_) {
      if (!hasScanned) {
        scannerController.dispose();
      }
    });
  }

  Future<void> _pickQRFromGallery(BuildContext context, AppProvider provider) async {
    try {
      final picker = ImagePicker();
      final image = await picker.pickImage(source: ImageSource.gallery);
      
      if (image == null) {
        return;
      }

      if (context.mounted) {
        showDialog(
          context: context,
          barrierDismissible: false,
          builder: (ctx) => const Center(
            child: Card(
              child: Padding(
                padding: EdgeInsets.all(24.0),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    CircularProgressIndicator(),
                    SizedBox(height: 16),
                    Text('Reading QR code...'),
                  ],
                ),
              ),
            ),
          ),
        );
      }

      final inputImage = mlkit.InputImage.fromFilePath(image.path);
      final barcodeScanner = mlkit.BarcodeScanner();

      try {
        final List<mlkit.Barcode> barcodes = await barcodeScanner.processImage(inputImage);
        
        if (context.mounted) {
          Navigator.pop(context);
        }

        if (barcodes.isEmpty) {
          if (context.mounted) {
            ScaffoldMessenger.of(context).showSnackBar(
              const SnackBar(
                content: Text('❌ No QR code found in image'),
                backgroundColor: Colors.red,
                duration: Duration(seconds: 3),
              ),
            );
          }
          return;
        }

        String? qrData;
        for (final barcode in barcodes) {
          if (barcode.rawValue != null && barcode.rawValue!.isNotEmpty) {
            qrData = barcode.rawValue;
            break;
          }
        }

        if (qrData != null) {
          provider.importConfig(qrData);
          
          if (context.mounted) {
            ScaffoldMessenger.of(context).showSnackBar(
              const SnackBar(
                content: Text('✅ QR code imported successfully'),
                backgroundColor: Colors.green,
                duration: Duration(seconds: 2),
              ),
            );
          }
        } else {
          if (context.mounted) {
            ScaffoldMessenger.of(context).showSnackBar(
              const SnackBar(
                content: Text('❌ Could not read QR code data'),
                backgroundColor: Colors.red,
                duration: Duration(seconds: 3),
              ),
            );
          }
        }
      } finally {
        barcodeScanner.close();
      }
    } catch (e) {
      if (context.mounted) {
        Navigator.pop(context);
        
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('❌ Error reading QR: ${e.toString()}'),
            backgroundColor: Colors.red,
            duration: const Duration(seconds: 4),
          ),
        );
      }
    }
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
              title: const Text('Show QR Code'),
              onTap: () {
                Navigator.pop(ctx);
                _showQRCodeDialog(context, provider, server);
              },
            ),
          ],
        ),
      ),
    );
  }

  void _showQRCodeDialog(BuildContext context, AppProvider provider, ServerConfig server) {
    final GlobalKey qrKey = GlobalKey();
    final uri = provider.exportConfig(server);

    showDialog(
      context: context,
      builder: (ctx) => Dialog(
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
                    onPressed: () => Navigator.pop(ctx),
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
                          final file = File('${tempDir.path}/guarch_${server.name.replaceAll(' ', '_')}_qr.png');
                          await file.writeAsBytes(pngBytes);

                          await Share.shareXFiles(
                            [XFile(file.path)],
                            text: 'Guarch VPN Config: ${server.name}\n\n$uri',
                            subject: 'Guarch VPN Configuration',
                          );

                          if (context.mounted) {
                            Navigator.pop(ctx);
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
                    final file = File('${tempDir.path}/guarch_${server.name.replaceAll(' ', '_')}_qr.png');
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
