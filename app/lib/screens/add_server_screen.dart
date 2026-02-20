import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:guarch/app.dart';
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
  final _domainController = TextEditingController();
  bool _coverEnabled = true;
  late List<CoverDomain> _coverDomains;

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
    _coverDomains =
        widget.server?.coverDomains ?? ServerConfig.defaultCoverDomains();
  }

  @override
  void dispose() {
    _nameController.dispose();
    _addressController.dispose();
    _portController.dispose();
    _domainController.dispose();
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
            const Row(
              children: [
                Text('🎯', style: TextStyle(fontSize: 24)),
                SizedBox(width: 8),
                Text(
                  'Server Information',
                  style: TextStyle(
                    fontSize: 18,
                    fontWeight: FontWeight.w600,
                    color: kGold,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 20),
            TextFormField(
              controller: _nameController,
              decoration: const InputDecoration(
                labelText: 'Server Name',
                hintText: 'e.g. Germany Server',
                prefixIcon: Icon(Icons.label_outline),
              ),
              validator: (v) =>
                  v == null || v.isEmpty ? 'Name required' : null,
            ),
            const SizedBox(height: 16),
            TextFormField(
              controller: _addressController,
              decoration: const InputDecoration(
                labelText: 'Server Address',
                hintText: 'IP or domain (e.g. 1.2.3.4)',
                prefixIcon: Icon(Icons.dns_outlined),
              ),
              keyboardType: TextInputType.url,
              validator: (v) =>
                  v == null || v.isEmpty ? 'Address required' : null,
            ),
            const SizedBox(height: 16),
            TextFormField(
              controller: _portController,
              decoration: const InputDecoration(
                labelText: 'Port',
                hintText: '8443',
                prefixIcon: Icon(Icons.numbers),
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
            const SizedBox(height: 32),

            // Cover Traffic
            Row(
              children: [
                const Text('🎭', style: TextStyle(fontSize: 24)),
                const SizedBox(width: 8),
                const Expanded(
                  child: Text(
                    'Cover Traffic',
                    style: TextStyle(
                      fontSize: 18,
                      fontWeight: FontWeight.w600,
                      color: kGold,
                    ),
                  ),
                ),
                Switch(
                  value: _coverEnabled,
                  onChanged: (v) => setState(() => _coverEnabled = v),
                ),
              ],
            ),
            const SizedBox(height: 4),
            Text(
              'Send real requests to popular sites to hide your traffic',
              style: TextStyle(color: kGold.withOpacity(0.5), fontSize: 13),
            ),

            if (_coverEnabled) ...[
              const SizedBox(height: 16),
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(16),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Text(
                        'Cover Domains',
                        style: TextStyle(
                          fontWeight: FontWeight.w600,
                          color: kGoldLight,
                        ),
                      ),
                      const SizedBox(height: 4),
                      Text(
                        'Add websites you normally visit',
                        style: TextStyle(
                          color: kGold.withOpacity(0.4),
                          fontSize: 12,
                        ),
                      ),
                      const SizedBox(height: 16),

                      // فیلد اضافه کردن
                      Row(
                        children: [
                          Expanded(
                            child: TextField(
                              controller: _domainController,
                              decoration: const InputDecoration(
                                hintText: 'e.g. google.com',
                                prefixIcon: Icon(Icons.public, size: 20),
                                isDense: true,
                                contentPadding: EdgeInsets.symmetric(
                                  horizontal: 12,
                                  vertical: 12,
                                ),
                              ),
                              keyboardType: TextInputType.url,
                              onSubmitted: (_) => _addDomain(),
                            ),
                          ),
                          const SizedBox(width: 8),
                          IconButton.filled(
                            onPressed: _addDomain,
                            icon: const Icon(Icons.add),
                          ),
                        ],
                      ),
                      const SizedBox(height: 12),

                      // پیشنهادات
                      Wrap(
                        spacing: 8,
                        runSpacing: 8,
                        children: [
                          _quickAddChip('google.com'),
                          _quickAddChip('youtube.com'),
                          _quickAddChip('instagram.com'),
                          _quickAddChip('twitter.com'),
                          _quickAddChip('facebook.com'),
                          _quickAddChip('reddit.com'),
                          _quickAddChip('linkedin.com'),
                          _quickAddChip('apple.com'),
                          _quickAddChip('cloudflare.com'),
                          _quickAddChip('netflix.com'),
                        ],
                      ),
                      const SizedBox(height: 16),
                      Divider(color: kGold.withOpacity(0.1)),
                      const SizedBox(height: 8),

                      Text(
                        'Active Cover Domains:',
                        style: TextStyle(
                          fontWeight: FontWeight.w600,
                          fontSize: 13,
                          color: kGoldLight,
                        ),
                      ),
                      const SizedBox(height: 8),

                      ..._coverDomains.asMap().entries.map((entry) {
                        final index = entry.key;
                        final domain = entry.value;
                        return Padding(
                          padding: const EdgeInsets.only(bottom: 4),
                          child: Row(
                            children: [
                              const Icon(Icons.check_circle,
                                  size: 16, color: Colors.green),
                              const SizedBox(width: 8),
                              Expanded(
                                child: Text(
                                  domain.domain,
                                  style: const TextStyle(
                                    fontSize: 14,
                                    color: kGoldLight,
                                  ),
                                ),
                              ),
                              Text(
                                '${domain.weight}%',
                                style: TextStyle(
                                  color: kGold.withOpacity(0.4),
                                  fontSize: 12,
                                ),
                              ),
                              const SizedBox(width: 4),
                              InkWell(
                                onTap: () {
                                  setState(() {
                                    _coverDomains.removeAt(index);
                                    _recalculateWeights();
                                  });
                                },
                                child: const Padding(
                                  padding: EdgeInsets.all(4),
                                  child: Icon(Icons.close,
                                      size: 16, color: Colors.red),
                                ),
                              ),
                            ],
                          ),
                        );
                      }),

                      if (_coverDomains.isEmpty)
                        const Padding(
                          padding: EdgeInsets.symmetric(vertical: 8),
                          child: Text(
                            'No domains added. Add at least one.',
                            style:
                                TextStyle(color: Colors.orange, fontSize: 12),
                          ),
                        ),
                    ],
                  ),
                ),
              ),
            ],

            const SizedBox(height: 32),
            FilledButton(
              onPressed: _save,
              style: FilledButton.styleFrom(
                padding: const EdgeInsets.symmetric(vertical: 16),
              ),
              child: Text(
                isEditing ? 'Save Changes' : 'Add Server',
                style: const TextStyle(fontSize: 16),
              ),
            ),
            const SizedBox(height: 16),
          ],
        ),
      ),
    );
  }

  Widget _quickAddChip(String domain) {
    final exists = _coverDomains.any(
      (d) => d.domain == domain || d.domain == 'www.$domain',
    );
    return ActionChip(
      avatar: Icon(
        exists ? Icons.check : Icons.add,
        size: 16,
        color: exists ? Colors.green : kGold,
      ),
      label: Text(domain, style: const TextStyle(fontSize: 12)),
      onPressed: exists
          ? null
          : () {
              setState(() {
                _coverDomains.add(CoverDomain(domain: domain));
                _recalculateWeights();
              });
            },
    );
  }

  void _addDomain() {
    final domain = _domainController.text.trim().toLowerCase();
    if (domain.isEmpty) return;
    String clean =
        domain.replaceAll('https://', '').replaceAll('http://', '');
    if (clean.endsWith('/')) clean = clean.substring(0, clean.length - 1);
    if (_coverDomains.any((d) => d.domain == clean)) {
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text('$clean already exists')));
      return;
    }
    setState(() {
      _coverDomains.add(CoverDomain(domain: clean));
      _recalculateWeights();
      _domainController.clear();
    });
  }

  void _recalculateWeights() {
    if (_coverDomains.isEmpty) return;
    final w = 100 ~/ _coverDomains.length;
    final r = 100 % _coverDomains.length;
    for (var i = 0; i < _coverDomains.length; i++) {
      _coverDomains[i].weight = w + (i < r ? 1 : 0);
    }
  }

  void _save() {
    if (!_formKey.currentState!.validate()) return;
    if (_coverEnabled && _coverDomains.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Add at least one cover domain'),
          backgroundColor: Colors.orange,
        ),
      );
      return;
    }
    final provider = context.read<AppProvider>();
    if (isEditing) {
      provider.updateServer(widget.server!.copyWith(
        name: _nameController.text.trim(),
        address: _addressController.text.trim(),
        port: int.parse(_portController.text.trim()),
        coverEnabled: _coverEnabled,
        coverDomains: List.from(_coverDomains),
      ));
    } else {
      final server = ServerConfig(
        id: DateTime.now().millisecondsSinceEpoch.toString(),
        name: _nameController.text.trim(),
        address: _addressController.text.trim(),
        port: int.parse(_portController.text.trim()),
        coverEnabled: _coverEnabled,
        coverDomains: List.from(_coverDomains),
      );
      provider.addServer(server);
      provider.setActiveServer(server.id);
    }
    Navigator.pop(context);
  }
}
