import 'package:flutter/material.dart';
import 'package:guarch/app.dart';

class AboutScreen extends StatelessWidget {
  const AboutScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('About Guarch')),
      body: ListView(
        padding: const EdgeInsets.all(24),
        children: [
          const Center(child: Text('🎯', style: TextStyle(fontSize: 80))),
          const SizedBox(height: 16),
          const Center(
            child: Text('Guarch', style: TextStyle(fontSize: 28, fontWeight: FontWeight.bold, color: kGold)),
          ),
          const SizedBox(height: 8),
          Center(child: Text('Version 1.0.0', style: TextStyle(color: kGold.withOpacity(0.4)))),

          const SizedBox(height: 32),
          _infoCard('🏹', 'What is Guarch?',
            'Guarch is a Balochi word for a traditional hunting technique. '
            'The hunter hides behind a cloth and moves alongside the prey undetected. '
            'Similarly, this project hides your traffic behind normal-looking internet activity.'),

          const SizedBox(height: 24),
          _sectionTitle('Three Protocols'),

          const SizedBox(height: 12),
          _protocolCard('🏹', 'Guarch', 'TLS 1.3 / TCP', 'Maximum Stealth', [
            'Cover traffic with real HTTPS requests',
            'Traffic shaping to mimic web browsing',
            'Multiplexed streams over encrypted TLS',
            'Decoy website for probe resistance',
            'Best for: heavily censored networks',
          ], Colors.green),

          const SizedBox(height: 12),
          _protocolCard('🌩️', 'Grouk', 'Raw UDP', 'Maximum Speed', [
            'Custom reliable UDP transport',
            'AIMD congestion control',
            'Session-based multiplexing',
            'Automatic retransmission',
            'Best for: speed-critical applications',
          ], kGold),

          const SizedBox(height: 12),
          _protocolCard('⚡', 'Zhip', 'QUIC / UDP', 'Balanced', [
            'QUIC protocol (HTTP/3 transport)',
            '0-RTT connection resumption',
            'Built-in congestion control',
            'Cover traffic support',
            'Best for: general use',
          ], Colors.blue),

          const SizedBox(height: 24),
          _sectionTitle('Security'),

          const SizedBox(height: 12),
          _infoCard('🔐', 'Encryption',
            'All protocols use X25519 key exchange and ChaCha20-Poly1305 '
            'authenticated encryption. PSK (Pre-Shared Key) provides an additional '
            'layer of authentication. Certificate pinning prevents MITM attacks.'),

          const SizedBox(height: 12),
          _infoCard('🎭', 'Cover Traffic',
            'Guarch and Zhip send real HTTPS requests to popular websites like Google, '
            'Microsoft, and GitHub. Your actual traffic is mixed with these requests, '
            'making it indistinguishable from normal browsing. Traffic shaping mimics '
            'real browser patterns with randomized timing and padding.'),

          const SizedBox(height: 12),
          _infoCard('🛡️', 'Anti-Detection',
            'If someone probes the server, they see a normal-looking CDN website (FastEdge CDN). '
            'Suspicious connection attempts are rate-limited and served decoy content. '
            'The server behaves exactly like nginx/1.24.0 to passive observers.'),

          const SizedBox(height: 12),
          _infoCard('🔄', 'Anti-Replay',
            'All packets include monotonic sequence numbers. Replayed packets are '
            'detected and rejected. Key rotation occurs automatically after sending '
            '1 billion messages or 64 GB of data.'),

          const SizedBox(height: 32),
          Center(child: Text('Made with ❤️ for internet freedom', style: TextStyle(color: kGold.withOpacity(0.4)))),
          const SizedBox(height: 8),
          Center(child: Text('github.com/balochscript/guarch', style: TextStyle(color: kGold.withOpacity(0.3), fontSize: 12))),
          const SizedBox(height: 32),
        ],
      ),
    );
  }

  Widget _sectionTitle(String title) {
    return Text(title, style: const TextStyle(fontSize: 20, fontWeight: FontWeight.bold, color: kGold));
  }

  Widget _protocolCard(String emoji, String name, String transport, String focus, List<String> features, Color accentColor) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Row(children: [
            Text(emoji, style: const TextStyle(fontSize: 28)),
            const SizedBox(width: 12),
            Expanded(
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Text(name, style: const TextStyle(fontSize: 18, fontWeight: FontWeight.bold, color: kGoldLight)),
                const SizedBox(height: 2),
                Text(transport, style: TextStyle(color: kGold.withOpacity(0.5), fontSize: 13)),
              ]),
            ),
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
              decoration: BoxDecoration(color: accentColor.withOpacity(0.15), borderRadius: BorderRadius.circular(8)),
              child: Text(focus, style: TextStyle(color: accentColor, fontSize: 11, fontWeight: FontWeight.w600)),
            ),
          ]),
          const SizedBox(height: 12),
          ...features.map((f) => Padding(
            padding: const EdgeInsets.only(bottom: 4),
            child: Row(children: [
              Icon(Icons.check_circle, size: 14, color: accentColor.withOpacity(0.7)),
              const SizedBox(width: 8),
              Expanded(child: Text(f, style: TextStyle(color: kGold.withOpacity(0.6), fontSize: 13))),
            ]),
          )),
        ]),
      ),
    );
  }

  Widget _infoCard(String emoji, String title, String description) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Row(children: [
            Text(emoji, style: const TextStyle(fontSize: 24)),
            const SizedBox(width: 12),
            Text(title, style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w600, color: kGold)),
          ]),
          const SizedBox(height: 8),
          Text(description, style: TextStyle(color: kGold.withOpacity(0.6), height: 1.5)),
        ]),
      ),
    );
  }
}
