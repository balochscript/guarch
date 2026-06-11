import 'package:flutter/material.dart';
import 'package:guarch/app.dart';
import 'package:guarch/models/connection_state.dart';

class StatsCard extends StatelessWidget {
  final ConnectionStats stats;

  const StatsCard({super.key, required this.stats});

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          children: [
            Row(
              children: [
                Expanded(
                  child: _buildSpeedIndicator(
                    context,
                    '↑ Upload',
                    stats.uploadSpeedText,
                    Colors.blue,
                  ),
                ),
                Container(
                  width: 1,
                  height: 40,
                  color: accentColor(context).withOpacity(0.2),
                ),
                Expanded(
                  child: _buildSpeedIndicator(
                    context,
                    '↓ Download',
                    stats.downloadSpeedText,
                    Colors.green,
                  ),
                ),
              ],
            ),

            const SizedBox(height: 16),
            Divider(color: accentColor(context).withOpacity(0.1)),
            const SizedBox(height: 16),

            Row(
              mainAxisAlignment: MainAxisAlignment.spaceAround,
              children: [
                _buildStatItem(
                  context,
                  'Total Up',
                  stats.totalUploadText,
                  Icons.arrow_upward,
                  Colors.blue,
                ),
                _buildStatItem(
                  context,
                  'Total Down',
                  stats.totalDownloadText,
                  Icons.arrow_downward,
                  Colors.green,
                ),
                _buildStatItem(
                  context,
                  'Duration',
                  stats.durationText,
                  Icons.access_time,
                  accentColor(context),
                ),
              ],
            ),

            if (stats.activeStreams > 0 || stats.totalConnections > 0) ...[
              const SizedBox(height: 16),
              Divider(color: accentColor(context).withOpacity(0.1)),
              const SizedBox(height: 16),
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceAround,
                children: [
                  _buildStatItem(
                    context,
                    'Streams',
                    '${stats.activeStreams}',
                    Icons.stream,
                    Colors.purple,
                  ),
                  _buildStatItem(
                    context,
                    'Connections',
                    '${stats.totalConnections}',
                    Icons.link,
                    Colors.orange,
                  ),
                ],
              ),
            ],

            if (stats.fecEnabled) ...[
              const SizedBox(height: 16),
              Divider(color: accentColor(context).withOpacity(0.1)),
              const SizedBox(height: 12),
              
              Row(
                children: [
                  const Icon(Icons.shield, size: 16, color: Colors.purple),
                  const SizedBox(width: 8),
                  Text(
                    'Forward Error Correction (Grouk)',
                    style: TextStyle(
                      fontSize: 13,
                      fontWeight: FontWeight.w600,
                      color: textSecondary(context),
                    ),
                  ),
                ],
              ),
              
              const SizedBox(height: 12),
              
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceAround,
                children: [
                  _buildStatItem(
                    context,
                    'Sent',
                    '${stats.fecSent}',
                    Icons.send,
                    Colors.blue,
                  ),
                  _buildStatItem(
                    context,
                    'Received',
                    '${stats.fecRecv}',
                    Icons.call_received,
                    Colors.green,
                  ),
                  _buildStatItem(
                    context,
                    'Recovered',
                    '${stats.fecRecovered}',
                    Icons.healing,
                    Colors.orange,
                  ),
                ],
              ),
              
              if (stats.fecRecv > 0) ...[
                const SizedBox(height: 12),
                Container(
                  padding: const EdgeInsets.all(8),
                  decoration: BoxDecoration(
                    color: stats.fecRecoveryRate > 0 
                        ? Colors.green.withOpacity(0.1) 
                        : Colors.grey.withOpacity(0.1),
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Row(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      Icon(
                        stats.fecRecoveryRate > 0 ? Icons.check_circle : Icons.info,
                        size: 14,
                        color: stats.fecRecoveryRate > 0 ? Colors.green : Colors.grey,
                      ),
                      const SizedBox(width: 6),
                      Text(
                        'Recovery Rate: ${stats.fecRecoveryRate.toStringAsFixed(1)}%',
                        style: TextStyle(
                          fontSize: 11,
                          fontWeight: FontWeight.w600,
                          color: stats.fecRecoveryRate > 0 
                              ? Colors.green 
                              : textMuted(context),
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ],
          ],
        ),
      ),
    );
  }

  Widget _buildSpeedIndicator(
    BuildContext context,
    String label,
    String value,
    Color color,
  ) {
    return Column(
      children: [
        Text(
          label,
          style: TextStyle(
            color: textMuted(context),
            fontSize: 12,
          ),
        ),
        const SizedBox(height: 4),
        Text(
          value,
          style: TextStyle(
            color: color,
            fontSize: 20,
            fontWeight: FontWeight.bold,
          ),
        ),
      ],
    );
  }

  Widget _buildStatItem(
    BuildContext context,
    String label,
    String value,
    IconData icon,
    Color color,
  ) {
    return Column(
      children: [
        Icon(icon, size: 20, color: color),
        const SizedBox(height: 4),
        Text(
          value,
          style: TextStyle(
            fontWeight: FontWeight.w600,
            fontSize: 14,
            color: textSecondary(context),
          ),
        ),
        const SizedBox(height: 2),
        Text(
          label,
          style: TextStyle(
            color: textMuted(context),
            fontSize: 11,
          ),
        ),
      ],
    );
  }
}
