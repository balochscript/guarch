import 'package:flutter/material.dart';

class AboutScreen extends StatelessWidget {
  const AboutScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('About Guarch')),
      body: ListView(
        padding: const EdgeInsets.all(24),
        children: [
          const Center(
            child: Text('🎯', style: TextStyle(fontSize: 80)),
          ),
          const SizedBox(height: 16),
          const Center(
            child: Text(
              'Guarch Protocol',
              style: TextStyle(fontSize: 28, fontWeight: FontWeight.bold),
            ),
          ),
          const SizedBox(height: 8),
          const Center(
            child: Text(
              'Version 1.0.0',
              style: TextStyle(color: Colors.grey),
            ),
          ),
          const SizedBox(height: 32),
          _infoCard(
            '🏹',
            'What is Guarch?',
            'Guarch is a Balochi word for a traditional hunting technique. '
                'The hunter hides behind a cloth and moves alongside the prey undetected. '
                'Similarly, this protocol hides your traffic behind normal-looking internet activity.',
          ),
          const SizedBox(height: 12),
          _infoCard(
            '🎭',
            'Cover Traffic',
            'Guarch sends real HTTPS requests to popular websites like Google, '
                'Microsoft, and GitHub. Your actual traffic is mixed with these requests, '
                'making it indistinguishable from normal browsing.',
          ),
          const SizedBox(height: 12),
          _infoCard(
            '🔐',
            'Encryption',
            'All data is encrypted using X25519 key exchange and '
                'ChaCha20-Poly1305 authenticated encryption, wrapped in TLS 1.3.',
          ),
          const SizedBox(height: 12),
          _infoCard(
            '🛡️',
            'Anti-Detection',
            'If someone probes the server, they see a normal-looking CDN website. '
                'Suspicious connection attempts are rate-limited and served decoy content.',
          ),
          const SizedBox(height: 32),
          const Center(
            child: Text(
              'Made with ❤️ for internet freedom',
              style: TextStyle(color: Colors.grey),
            ),
          ),
          const SizedBox(height: 32),
        ],
      ),
    );
  }

  Widget _infoCard(String emoji, String title, String description) {
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
                Text(
                  title,
                  style: const TextStyle(
                    fontSize: 16,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 8),
            Text(
              description,
              style: const TextStyle(color: Colors.grey, height: 1.5),
            ),
          ],
        ),
      ),
    );
  }
}
