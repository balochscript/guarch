import 'package:flutter/material.dart';
import 'package:guarch/app.dart';
import 'package:guarch/services/guarch_engine.dart';

class AboutScreen extends StatelessWidget {
  const AboutScreen({super.key});

  Future<Map<String, String>> _getVersionInfo() async {
    try {
      final engineVersion = await GuarchEngine.getVersion();
      return {
        'app': '1.0.1',
        'engine': engineVersion,
      };
    } catch (e) {
      return {
        'app': '1.0.1',
        'engine': 'Unknown',
      };
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('About Guarch')),
      body: ListView(
        padding: const EdgeInsets.all(24),
        children: [
          const Center(child: Text('🎯', style: TextStyle(fontSize: 80))),
          const SizedBox(height: 16),
          Center(
            child: Text(
              'Guarch',
              style: TextStyle(
                fontSize: 28,
                fontWeight: FontWeight.bold,
                color: textPrimary(context),
              ),
            ),
          ),
          const SizedBox(height: 8),
          FutureBuilder<Map<String, String>>(
            future: _getVersionInfo(),
            builder: (context, snapshot) {
              if (!snapshot.hasData) {
                return Center(
                  child: Text(
                    'Loading version...',
                    style: TextStyle(color: textMuted(context)),
                  ),
                );
              }
              final versions = snapshot.data!;
              return Column(
                children: [
                  Text(
                    'App: ${versions['app']}',
                    style: TextStyle(color: textMuted(context)),
                  ),
                  Text(
                    'Engine: ${versions['engine']}',
                    style: TextStyle(color: textMuted(context)),
                  ),
                ],
              );
            },
          ),
          const SizedBox(height: 32),
          _infoCard(
            context,
            '🏹',
            'What is Guarch?',
            'Guarch is a Balochi word for a traditional hunting technique. '
                'The hunter hides behind a cloth and moves alongside the prey undetected. '
                'Similarly, this project hides your traffic behind normal-looking internet activity.',
          ),
          const SizedBox(height: 16),
          _infoCard(
            context,
            '🌐',
            'System-wide VPN',
            'Guarch works as a full VPN on Android — all apps (Telegram, Instagram, '
                'Chrome, etc.) are routed through the encrypted tunnel automatically. '
                'No manual proxy configuration needed. iOS support is coming soon.',
          ),
          const SizedBox(height: 24),
          _sectionTitle(context, 'Three Protocols'),
          const SizedBox(height: 12),
          _protocolCard(
            context,
            '🏹',
            'Guarch',
            'TLS 1.3 / TCP',
            'Maximum Stealth',
            [
              'Cover traffic with real HTTPS requests',
              'Traffic shaping to mimic web browsing',
              'Multiplexed streams over encrypted TLS',
              'Decoy website for probe resistance',
              'Best for: heavily censored networks',
            ],
            Colors.green,
          ),
          const SizedBox(height: 12),
          _protocolCard(
            context,
            '🌩️',
            'Grouk',
            'Raw UDP',
            'Maximum Speed',
            [
              'Custom reliable UDP transport',
              'AIMD congestion control',
              'Session-based multiplexing',
              'Automatic retransmission',
              'Best for: speed-critical applications',
            ],
            accentColor(context),
          ),
          const SizedBox(height: 12),
          _protocolCard(
            context,
            '⚡',
            'Zhip',
            'QUIC / UDP',
            'Balanced',
            [
              'QUIC protocol (HTTP/3 transport)',
              '0-RTT connection resumption',
              'Built-in congestion control',
              'Cover traffic support',
              'Best for: general use',
            ],
            Colors.blue,
          ),
          const SizedBox(height: 24),
          _sectionTitle(context, 'Security'),
          const SizedBox(height: 12),
          _infoCard(
            context,
            '🔐',
            'Encryption',
            'All protocols use X25519 key exchange and ChaCha20-Poly1305 '
                'authenticated encryption. PSK (Pre-Shared Key) provides an additional '
                'layer of authentication. Certificate pinning prevents MITM attacks.',
          ),
          const SizedBox(height: 12),
          _infoCard(
            context,
            '🎭',
            'Cover Traffic',
            'Guarch and Zhip send real HTTPS requests to popular websites like Google, '
                'Microsoft, and GitHub. Your actual traffic is mixed with these requests, '
                'making it harder to distinguish from normal browsing. Traffic shaping mimics '
                'real browser patterns with randomized timing and padding.',
          ),
          const SizedBox(height: 12),
          _infoCard(
            context,
            '🛡️',
            'Anti-Detection',
            'If someone probes the server, they see a normal-looking CDN website (FastEdge CDN). '
                'Suspicious connection attempts are rate-limited and served decoy content. '
                'The server behaves exactly like nginx/1.24.0 to passive observers.',
          ),
          const SizedBox(height: 12),
          _infoCard(
            context,
            '🔄',
            'Anti-Replay',
            'All packets include monotonic sequence numbers. Replayed packets are '
                'detected and rejected. Key rotation occurs automatically after sending '
                '1 billion messages or 64 GB of data.',
          ),
          const SizedBox(height: 24),
          _sectionTitle(context, 'Advanced Features (v1.0.1)'),
          const SizedBox(height: 12),
          _infoCard(
            context,
            '🔄',
            'SNI Rotation',
            'Server Name Indication (SNI) rotation automatically changes the domain '
                'name sent in TLS handshakes every 5 minutes. This makes your connection '
                'appear to be accessing different legitimate websites over time. Supports '
                'random, weighted, sequential, and single modes with health checking.',
          ),
          const SizedBox(height: 12),
          _infoCard(
            context,
            '🔌',
            'DNS Fallback',
            'When TLS connections are blocked, Guarch can automatically switch to '
                'tunneling data over DNS queries. This survival mode works even in heavily '
                'restricted networks where all ports except DNS (53) are blocked. Speed is '
                'reduced (~50 Kbps) but provides connectivity when nothing else works.',
          ),
          const SizedBox(height: 12),
          _infoCard(
            context,
            '🔋',
            'Battery-Aware Mode',
            'Automatically reduces cover traffic generation when device battery is low '
                '(below 30% by default). This helps extend battery life while maintaining '
                'connection security. Threshold is configurable per server.',
          ),
          const SizedBox(height: 12),
          _infoCard(
            context,
            '💾',
            'Data Saver Mode',
            'Halves cover traffic rate and reduces packet padding to save bandwidth on '
                'metered connections. Perfect for limited mobile data plans. Can be toggled '
                'on/off from settings at any time.',
          ),
          const SizedBox(height: 12),
          _infoCard(
            context,
            '📊',
            'Adaptive Cover Traffic',
            'Cover traffic automatically adjusts based on your real usage. Four activity '
                'levels (idle, light, medium, heavy) with automatic switching. Heavy usage '
                'generates more cover requests to blend in, while idle periods reduce overhead.',
          ),
          const SizedBox(height: 12),
          _infoCard(
            context,
            '🎯',
            'Per-Server Configuration',
            'Each server can have custom SNI domains, cover domains, DNS fallback settings, '
                'and battery/data saver preferences. Configure different strategies for different '
                'network environments (home, office, public WiFi, mobile data).',
          ),
          const SizedBox(height: 12),
          _infoCard(
            context,
            '⚙️',
            'Advanced Settings',
            'Fine-tune connection parameters: timeouts (5-60s), retry attempts (1-10), '
                'retry delay (1-30s), handshake timeout (10-120s), keep-alive interval (10-300s), '
                'and buffer size (16-128 KB). Optimize for your network conditions.',
          ),
          const SizedBox(height: 12),
          _infoCard(
            context,
            '🐛',
            'Debug Mode',
            'Developer-friendly debug mode with detailed logging. View real-time Go engine '
                'logs, Kotlin/Flutter logs, and crash reports. Perfect for troubleshooting '
                'connection issues. Can be enabled/disabled from settings.',
          ),
          const SizedBox(height: 32),
          Center(
            child: Text(
              'Made with ❤️ for internet freedom',
              style: TextStyle(color: textMuted(context)),
            ),
          ),
          const SizedBox(height: 8),
          Center(
            child: Text(
              'github.com/balochscript/guarch',
              style: TextStyle(
                color: textMuted(context).withOpacity(0.6),
                fontSize: 12,
              ),
            ),
          ),
          const SizedBox(height: 32),
        ],
      ),
    );
  }

  Widget _sectionTitle(BuildContext context, String title) {
    return Text(
      title,
      style: TextStyle(
        fontSize: 20,
        fontWeight: FontWeight.bold,
        color: textPrimary(context),
      ),
    );
  }

  Widget _protocolCard(
    BuildContext context,
    String emoji,
    String name,
    String transport,
    String focus,
    List<String> features,
    Color color,
  ) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Text(emoji, style: const TextStyle(fontSize: 28)),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        name,
                        style: TextStyle(
                          fontSize: 18,
                          fontWeight: FontWeight.bold,
                          color: textSecondary(context),
                        ),
                      ),
                      const SizedBox(height: 2),
                      Text(
                        transport,
                        style: TextStyle(
                          color: textMuted(context),
                          fontSize: 13,
                        ),
                      ),
                    ],
                  ),
                ),
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                  decoration: BoxDecoration(
                    color: color.withOpacity(0.15),
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Text(
                    focus,
                    style: TextStyle(
                      color: color,
                      fontSize: 11,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 12),
            ...features.map(
              (f) => Padding(
                padding: const EdgeInsets.only(bottom: 4),
                child: Row(
                  children: [
                    Icon(
                      Icons.check_circle,
                      size: 14,
                      color: color.withOpacity(0.7),
                    ),
                    const SizedBox(width: 8),
                    Expanded(
                      child: Text(
                        f,
                        style: TextStyle(
                          color: textMuted(context),
                          fontSize: 13,
                        ),
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _infoCard(
    BuildContext context,
    String emoji,
    String title,
    String description,
  ) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Text(emoji, style: const TextStyle(fontSize: 24)),
                const SizedBox(width: 12),
                Expanded(
                  child: Text(
                    title,
                    style: TextStyle(
                      fontSize: 16,
                      fontWeight: FontWeight.w600,
                      color: textPrimary(context),
                    ),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 8),
            Text(
              description,
              style: TextStyle(
                color: textMuted(context),
                height: 1.5,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
