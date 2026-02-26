import 'package:flutter/material.dart';
import 'package:guarch/app.dart';
import 'package:guarch/models/server_config.dart';

class ServerCard extends StatelessWidget {
  final ServerConfig server;
  final bool isActive;
  final VoidCallback? onTap;
  final VoidCallback? onPing;
  final VoidCallback? onShare;
  final VoidCallback? onDelete;

  const ServerCard({
    super.key,
    required this.server,
    this.isActive = false,
    this.onTap,
    this.onPing,
    this.onShare,
    this.onDelete,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: isActive
            ? BorderSide(color: accentColor(context), width: 2)
            : BorderSide.none,
      ),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(16),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Row(
            children: [
              Container(
                width: 48,
                height: 48,
                decoration: BoxDecoration(
                  color: _statusColor.withOpacity(0.15),
                  borderRadius: BorderRadius.circular(12),
                ),
                child: Center(
                  child: Text(server.pingEmoji, style: const TextStyle(fontSize: 24)),
                ),
              ),
              const SizedBox(width: 16),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Text(server.name,
                            style: TextStyle(
                                fontWeight: FontWeight.w600,
                                fontSize: 15,
                                color: textSecondary(context))),
                        if (isActive) ...[
                          const SizedBox(width: 8),
                          Container(
                            padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                            decoration: BoxDecoration(
                              color: accentColor(context).withOpacity(0.2),
                              borderRadius: BorderRadius.circular(6),
                            ),
                            child: Text('ACTIVE',
                                style: TextStyle(
                                    fontSize: 9,
                                    fontWeight: FontWeight.bold,
                                    color: accentColor(context))),
                          ),
                        ],
                      ],
                    ),
                    const SizedBox(height: 4),
                    Text(server.fullAddress,
                        style: TextStyle(fontSize: 12, color: textMuted(context))),
                    if (server.coverEnabled)
                      Padding(
                        padding: const EdgeInsets.only(top: 4),
                        child: Row(children: [
                          const Text('🎭', style: TextStyle(fontSize: 12)),
                          const SizedBox(width: 4),
                          Text('Cover traffic enabled',
                              style: TextStyle(fontSize: 11, color: textMuted(context))),
                        ]),
                      ),
                  ],
                ),
              ),
              Column(
                crossAxisAlignment: CrossAxisAlignment.end,
                children: [
                  Text(server.pingText,
                      style: TextStyle(
                          fontWeight: FontWeight.bold,
                          color: _statusColor,
                          fontSize: 14)),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }

  Color get _statusColor {
    if (server.ping == null) return Colors.grey;
    if (server.ping! < 0) return Colors.red;
    if (server.ping! < 100) return Colors.green;
    if (server.ping! < 300) return Colors.orange;
    return Colors.red;
  }
}
