import 'package:flutter/material.dart';
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
            Text(
              stats.durationText,
              style: const TextStyle(
                fontSize: 32,
                fontWeight: FontWeight.w300,
                letterSpacing: 4,
              ),
            ),
            const SizedBox(height: 16),
            Row(
              children: [
                Expanded(
                  child: _buildStat(
                    icon: Icons.arrow_upward,
                    color: Colors.orange,
                    speed: stats.uploadSpeedText,
                    total: stats.totalUploadText,
                    label: 'Upload',
                  ),
                ),
                Container(
                  width: 1,
                  height: 50,
                  color: Colors.grey.withOpacity(0.3),
                ),
                Expanded(
                  child: _buildStat(
                    icon: Icons.arrow_downward,
                    color: Colors.green,
                    speed: stats.downloadSpeedText,
                    total: stats.totalDownloadText,
                    label: 'Download',
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildStat({
    required IconData icon,
    required Color color,
    required String speed,
    required String total,
    required String label,
  }) {
    return Column(
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(icon, color: color, size: 16),
            const SizedBox(width: 4),
            Text(
              speed,
              style: TextStyle(
                color: color,
                fontWeight: FontWeight.w600,
                fontSize: 14,
              ),
            ),
          ],
        ),
        const SizedBox(height: 4),
        Text(
          total,
          style: const TextStyle(color: Colors.grey, fontSize: 11),
        ),
      ],
    );
  }
}
