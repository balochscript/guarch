import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

class SponsorBanner extends StatelessWidget {
  const SponsorBanner({super.key});

  @override
  Widget build(BuildContext context) {
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(
          color: Colors.amber.withOpacity(0.3),
          width: 1,
        ),
      ),
      color: Colors.amber.withOpacity(0.05),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(Icons.favorite, color: Colors.red[400], size: 18),
                const SizedBox(width: 8),
                Text(
                  'Support Guarch VPN',
                  style: TextStyle(
                    fontWeight: FontWeight.w600,
                    color: Theme.of(context).textTheme.bodyLarge?.color,
                    fontSize: 14,
                  ),
                ),
                const Spacer(),
                TextButton.icon(
                  onPressed: () => _showDonateDialog(context),
                  icon: const Icon(Icons.volunteer_activism, size: 16),
                  label: const Text('Donate'),
                  style: TextButton.styleFrom(
                    foregroundColor: Colors.amber[800],
                    padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 4),
            Text(
              'Help us keep Guarch free and open-source',
              style: TextStyle(
                color: Theme.of(context).textTheme.bodySmall?.color,
                fontSize: 12,
              ),
            ),
          ],
        ),
      ),
    );
  }

  void _showDonateDialog(BuildContext context) {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Row(
          children: [
            Icon(Icons.favorite, color: Colors.red),
            const SizedBox(width: 12),
            const Text('Support Guarch VPN'),
          ],
        ),
        content: SingleChildScrollView(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              const Text(
                'Guarch VPN is free and open-source. Your donations help us:',
                style: TextStyle(height: 1.4),
              ),
              const SizedBox(height: 12),
              _bulletPoint('Maintain servers and infrastructure'),
              _bulletPoint('Develop new features'),
              _bulletPoint('Keep the project alive'),
              const SizedBox(height: 20),
              const Text(
                'Crypto Donations:',
                style: TextStyle(fontWeight: FontWeight.bold),
              ),
              const SizedBox(height: 12),
              _cryptoAddress(
                context,
                'Bitcoin (BTC)',
                'bc1q53y72zxhuxlsmx6uh8ga5pmslh9fgy9mfwvqvt',
                Icons.currency_bitcoin,
              ),
              const SizedBox(height: 8),
              _cryptoAddress(
                context,
                'Ethereum (ETH)',
                '0x37Afc5996621d22E8fa5f3f24652666F0a732f6E',
                Icons.monetization_on,
              ),
              const SizedBox(height: 8),
              _cryptoAddress(
                context,
                'USDT (TRC20) / TRX',
                'TUYMUcb8a3S4o8s2jch4ikEnfr1xPM5rjm',
                Icons.attach_money,
              ),
              const SizedBox(height: 8),
              _cryptoAddress(
                context,
                'USDT (Polygon)',
                '0x37Afc5996621d22E8fa5f3f24652666F0a732f6E',
                Icons.paid,
              ),
              const SizedBox(height: 8),
              _cryptoAddress(
                context,
                'TON',
                'UQC2jcvDdnCFXqNwlWbGEkaOitIoIIyTMuKKqsb8pvalgHtP',
                Icons.diamond,
              ),
            ],
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

  Widget _bulletPoint(String text) {
    return Padding(
      padding: const EdgeInsets.only(left: 8, bottom: 4),
      child: Row(
        children: [
          const Icon(Icons.check_circle, size: 16, color: Colors.green),
          const SizedBox(width: 8),
          Expanded(child: Text(text, style: const TextStyle(fontSize: 13))),
        ],
      ),
    );
  }

  Widget _cryptoAddress(BuildContext context, String name, String address, IconData icon) {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surfaceVariant.withOpacity(0.3),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(icon, size: 18, color: Colors.amber[800]),
              const SizedBox(width: 8),
              Text(
                name,
                style: const TextStyle(
                  fontWeight: FontWeight.w600,
                  fontSize: 13,
                ),
              ),
              const Spacer(),
              InkWell(
                onTap: () {
                  Clipboard.setData(ClipboardData(text: address));
                  ScaffoldMessenger.of(context).showSnackBar(
                    SnackBar(
                      content: Text('$name address copied!'),
                      duration: const Duration(seconds: 2),
                    ),
                  );
                },
                child: const Icon(Icons.copy, size: 16, color: Colors.blue),
              ),
            ],
          ),
          const SizedBox(height: 4),
          Text(
            address,
            style: TextStyle(
              fontSize: 11,
              color: Theme.of(context).textTheme.bodySmall?.color,
              fontFamily: 'monospace',
            ),
          ),
        ],
      ),
    );
  }
}
