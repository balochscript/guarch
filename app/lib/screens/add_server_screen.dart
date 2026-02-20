import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:guarch/providers/app_provider.dart';
import 'package:guarch/models/server_config.dart';

class AddServerScreen extends StatefulWidget {
  final ServerConfig? server;

  const AddServerScreen({super.key, this.server});

  @override
  State<AddServerScreen> createState() => _AddServerScreenState();
}

class _AddServerScreenState extends State<AddServerScreen> {
  final _formKey = GlobalKey<FormState>();
  late TextEditingController _nameController;
  late TextEditingController _addressController;
  late TextEditingController _portController;
  bool _coverEnabled = true;

  bool get isEditing => widget.server != null;

  @override
  void initState() {
    super.initState();
    _nameController = TextEditingController(text: widget.server?.name ?? '');
    _addressController =
        TextEditingController(text: widget.server?.address ?? '');
    _portController =
        TextEditingController(text: (widget.server?.port ?? 8443).toString());
    _coverEnabled = widget.server?.coverEnabled ?? true;
  }

  @override
  void dispose() {
    _nameController.dispose();
    _addressController.dispose();
    _portController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(isEditing ? 'Edit Server' : 'Add Server'),
      ),
      body: Form(
        key: _formKey,
        child: ListView(
          padding: const EdgeInsets.all(24),
          children: [
            const Text(
              '🎯 Server Information',
              style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600),
            ),
            const SizedBox(height: 20),
            TextFormField(
              controller: _nameController,
              decoration: InputDecoration(
                labelText: 'Server Name',
                hintText: 'My Server',
                prefixIcon: const Icon(Icons.label_outline),
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(12),
                ),
              ),
              validator: (v) =>
                  v == null || v.isEmpty ? 'Name required' : null,
            ),
            const SizedBox(height: 16),
            TextFormField(
              controller: _addressController,
              decoration: InputDecoration(
                labelText: 'Server Address',
                hintText: '1.2.3.4 or domain.com',
                prefixIcon: const Icon(Icons.dns_outlined),
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(12),
                ),
              ),
              keyboardType: TextInputType.url,
              validator: (v) =>
                  v == null || v.isEmpty ? 'Address required' : null,
            ),
            const SizedBox(height: 16),
            TextFormField(
              controller: _portController,
              decoration: InputDecoration(
                labelText: 'Port',
                hintText: '8443',
                prefixIcon: const Icon(Icons.numbers),
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(12),
                ),
              ),
              keyboardType: TextInputType.number,
              validator: (v) {
                if (v == null || v.isEmpty) return 'Port required';
                final port = int.tryParse(v);
                if (port == null || port < 1 || port > 65535) {
                  return 'Invalid port (1-65535)';
                }
                return null;
              },
            ),
            const SizedBox(height: 24),
            const Text(
              '🎭 Cover Traffic',
              style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600),
            ),
            const SizedBox(height: 12),
            Card(
              child: ListTile(
                title: const Text('Enable Cover Traffic'),
                subtitle: const Text(
                  'Send real requests to popular sites to blend in',
                ),
                trailing: Switch(
                  value: _coverEnabled,
                  onChanged: (v) => setState(() => _coverEnabled = v),
                  activeColor: const Color(0xFF6C5CE7),
                ),
              ),
            ),
            if (_coverEnabled) ...[
              const SizedBox(height: 8),
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(16),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Text(
                        'Cover Domains',
                        style: TextStyle(fontWeight: FontWeight.w600),
                      ),
                      const SizedBox(height: 8),
                      _coverItem('google.com', 30),
                      _coverItem('microsoft.com', 20),
                      _coverItem('github.com', 15),
                      _coverItem('stackoverflow.com', 15),
                      _coverItem('wikipedia.org', 10),
                      _coverItem('amazon.com', 10),
                    ],
                  ),
                ),
              ),
            ],
            const SizedBox(height: 32),
            FilledButton(
              onPressed: _save,
              style: FilledButton.styleFrom(
                backgroundColor: const Color(0xFF6C5CE7),
                padding: const EdgeInsets.symmetric(vertical: 16),
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(12),
                ),
              ),
              child: Text(
                isEditing ? 'Save Changes' : 'Add Server',
                style: const TextStyle(fontSize: 16),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _coverItem(String domain, int weight) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        children: [
          const Icon(Icons.check_circle, size: 16, color: Colors.green),
          const SizedBox(width: 8),
          Expanded(child: Text(domain)),
          Text(
            '$weight%',
            style: const TextStyle(color: Colors.grey, fontSize: 12),
          ),
        ],
      ),
    );
  }

  void _save() {
    if (!_formKey.currentState!.validate()) return;

    final provider = context.read<AppProvider>();

    if (isEditing) {
      final updated = widget.server!.copyWith(
        name: _nameController.text.trim(),
        address: _addressController.text.trim(),
        port: int.parse(_portController.text.trim()),
        coverEnabled: _coverEnabled,
      );
      provider.updateServer(updated);
    } else {
      final server = ServerConfig(
        id: DateTime.now().millisecondsSinceEpoch.toString(),
        name: _nameController.text.trim(),
        address: _addressController.text.trim(),
        port: int.parse(_portController.text.trim()),
        coverEnabled: _coverEnabled,
      );
      provider.addServer(server);
      provider.setActiveServer(server.id);
    }

    Navigator.pop(context);
  }
}
