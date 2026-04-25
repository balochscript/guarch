import 'package:flutter/material.dart';
import 'package:guarch/app.dart';
import 'package:guarch/models/server_config.dart';

class ServerTile extends StatelessWidget {
  final ServerConfig server;
  final bool isActive;
  final VoidCallback? onTap;
  final VoidCallback? onLongPress;
  final List<Widget>? actions;

  const ServerTile({
    super.key,
    required this.server,
    this.isActive = false,
    this.onTap,
    this.onLongPress,
    this.actions,
  });

  @override
  Widget build(BuildContext context) {
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
        onTap: onTap,
        onLongPress: onLongPress,
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            children: [
              // Header row
              Row(
                children: [
                  // Protocol emoji
                  Text(
                    server.pingEmoji,
                    style: const TextStyle(fontSize: 28),
                  ),
                  const SizedBox(width: 12),

                  // Server info
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

                  // Latency info
                  Column(
                    crossAxisAlignment: CrossAxisAlignment.end,
                    children: [
                      // TCPing
                      Row(
                        children: [
                          const Icon(Icons.bolt, size: 12, color: Colors.grey),
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
                      // Real Delay
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
                      // Features
                      if (server.coverEnabled ||
                          server.sniEnabled ||
                          server.dnsFallbackEnabled) ...[
                        const SizedBox(height: 2),
                        Row(
                          children: [
                            if (server.sniEnabled)
                              const Text('🔄', style: TextStyle(fontSize: 10)),
                            if (server.coverEnabled)
                              const Text('🎭', style: TextStyle(fontSize: 10)),
                            if (server.dnsFallbackEnabled)
                              const Text('🔌', style: TextStyle(fontSize: 10)),
                          ],
                        ),
                      ],
                    ],
                  ),
                ],
              ),

              // Actions row
              if (actions != null && actions!.isNotEmpty) ...[
                const SizedBox(height: 8),
                Row(
                  mainAxisAlignment: MainAxisAlignment.end,
                  children: actions!,
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }

  Color _pingColor(BuildContext context, int? ping) {
    if (ping == null) return textMuted(context);
    if (ping < 0) return Colors.red;
    if (ping < 100) return Colors.green;
    if (ping < 300) return Colors.orange;
    return Colors.red;
  }
}
