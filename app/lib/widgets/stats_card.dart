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
            // Speed indicators
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

            // Total transferred
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

            // Advanced stats
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
