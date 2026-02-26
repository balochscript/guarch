import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import 'package:guarch/app.dart';
import 'package:guarch/providers/app_provider.dart';

class ImportScreen extends StatefulWidget {
  const ImportScreen({super.key});

  @override
  State<ImportScreen> createState() => _ImportScreenState();
}

class _ImportScreenState extends State<ImportScreen> with SingleTickerProviderStateMixin {
  late TabController _tabController;
  final _linkController = TextEditingController();
  final _jsonController = TextEditingController();

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 3, vsync: this);
  }

  @override
  void dispose() {
    _tabController.dispose();
    _linkController.dispose();
    _jsonController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Import Config'),
        bottom: TabBar(controller: _tabController, tabs: const [
          Tab(icon: Icon(Icons.link), text: 'Link'),
          Tab(icon: Icon(Icons.data_object), text: 'JSON'),
          Tab(icon: Icon(Icons.content_paste), text: 'Clipboard'),
        ]),
      ),
      body: TabBarView(controller: _tabController, children: [
        _buildLinkTab(),
        _buildJsonTab(),
        _buildClipboardTab(),
      ]),
    );
  }

  Widget _buildLinkTab() {
    return Padding(
      padding: const EdgeInsets.all(24),
      child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
        const Row(children: [
          Text('🔗', style: TextStyle(fontSize: 24)),
          SizedBox(width: 8),
          Text('Import from Link', style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600, color: kGold)),
        ]),
        const SizedBox(height: 8),
        Text('Paste a guarch://, grouk://, or zhip:// link', style: TextStyle(color: kGold.withOpacity(0.5))),
        const SizedBox(height: 16),
        Wrap(spacing: 8, runSpacing: 4, children: [
          _protocolChip('🏹 guarch://'),
          _protocolChip('🌩️ grouk://'),
          _protocolChip('⚡ zhip://'),
        ]),
        const SizedBox(height: 16),
        TextField(
          controller: _linkController,
          maxLines: 3,
          style: const TextStyle(color: kGoldLight, fontFamily: 'monospace', fontSize: 13),
          decoration: const InputDecoration(hintText: 'guarch://eyJuYW1lIjoi...', prefixIcon: Icon(Icons.link)),
        ),
        const SizedBox(height: 16),
        FilledButton(
          onPressed: () => _importData(_linkController.text),
          style: FilledButton.styleFrom(padding: const EdgeInsets.symmetric(vertical: 16)),
          child: const Text('Import', style: TextStyle(fontSize: 16)),
        ),
      ]),
    );
  }

  Widget _protocolChip(String label) {
    return Chip(
      label: Text(label, style: const TextStyle(fontSize: 11, fontFamily: 'monospace', color: kGoldLight)),
      backgroundColor: kDarkCard,
      side: BorderSide(color: kGold.withOpacity(0.2)),
      visualDensity: VisualDensity.compact,
    );
  }

  Widget _buildJsonTab() {
    return Padding(
      padding: const EdgeInsets.all(24),
      child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
        const Row(children: [
          Text('📋', style: TextStyle(fontSize: 24)),
          SizedBox(width: 8),
          Text('Import from JSON', style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600, color: kGold)),
        ]),
        const SizedBox(height: 8),
        Text('Paste JSON config. Use "protocol": "guarch"|"grouk"|"zhip"', style: TextStyle(color: kGold.withOpacity(0.5), fontSize: 12)),
        const SizedBox(height: 24),
        Expanded(
          child: TextField(
            controller: _jsonController,
            maxLines: null,
            expands: true,
            textAlignVertical: TextAlignVertical.top,
            style: const TextStyle(fontFamily: 'monospace', fontSize: 12, color: kGoldLight),
            decoration: const InputDecoration(
              hintText: '{\n  "name": "My Server",\n  "address": "1.2.3.4",\n  "port": 8443,\n  "protocol": "guarch",\n  "psk": "my-secret"\n}',
            ),
          ),
        ),
        const SizedBox(height: 16),
        FilledButton(
          onPressed: () => _importData(_jsonController.text),
          style: FilledButton.styleFrom(padding: const EdgeInsets.symmetric(vertical: 16)),
          child: const Text('Import', style: TextStyle(fontSize: 16)),
        ),
      ]),
    );
  }

  Widget _buildClipboardTab() {
    return Padding(
      padding: const EdgeInsets.all(24),
      child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
        Container(
          padding: const EdgeInsets.all(24),
          decoration: BoxDecoration(color: kGold.withOpacity(0.1), shape: BoxShape.circle),
          child: const Icon(Icons.content_paste, size: 48, color: kGold),
        ),
        const SizedBox(height: 24),
        const Text('Import from Clipboard', style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600, color: kGold)),
        const SizedBox(height: 8),
        Text('Copy a guarch://, grouk://, zhip:// link\nor JSON config, then tap below', textAlign: TextAlign.center, style: TextStyle(color: kGold.withOpacity(0.5))),
        const SizedBox(height: 24),
        FilledButton.icon(
          onPressed: _importFromClipboard,
          icon: const Icon(Icons.content_paste),
          label: const Text('Paste & Import'),
          style: FilledButton.styleFrom(padding: const EdgeInsets.symmetric(horizontal: 32, vertical: 16)),
        ),
      ]),
    );
  }

  void _importFromClipboard() async {
    final data = await Clipboard.getData(Clipboard.kTextPlain);
    if (data?.text != null && data!.text!.isNotEmpty) {
      _importData(data.text!);
    } else {
      if (mounted) ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('Clipboard is empty')));
    }
  }

  void _importData(String data) {
    if (data.trim().isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('Please enter config data')));
      return;
    }
    final provider = context.read<AppProvider>();
    provider.importConfig(data.trim());
    ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('Config imported successfully!'), backgroundColor: Colors.green));
    Navigator.pop(context);
  }
}
