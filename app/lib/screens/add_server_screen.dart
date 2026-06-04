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
  late TextEditingController _pskController;
  late TextEditingController _pinController;
  final _domainController = TextEditingController();
  final _sniDomainController = TextEditingController();
  final _dnsDomainController = TextEditingController();
  final _transportHostController = TextEditingController();
  final _transportPathController = TextEditingController();
  
  late String _protocol;
  bool _coverEnabled = false;
  bool _sniEnabled = false;
  bool _dnsFallbackEnabled = false;
  bool _batteryAwareEnabled = true;
  bool _dataSaverEnabled = false;
  bool _pskVisible = false;
  
  String _transportType = 'direct';
  
  late String _sniMode;
  late String _dnsFallbackMode;
  late List<CoverDomain> _coverDomains;
  late List<SNIDomain> _sniDomains;

  bool get isEditing => widget.server != null;

  String get _protocolDescription {
    switch (_protocol) {
      case 'grouk':
        return '🌩️ Fast raw UDP tunnel. Best for speed, less stealth. ⚠️ Beta';
      case 'zhip':
        return '⚡ QUIC-based tunnel. Good balance. ⚠️ Experimental';
      default:
        return '🏹 TLS 1.3 encrypted. Maximum stealth & stability. ✅ Recommended';
    }
  }

  @override
  void initState() {
    super.initState();
    _nameController = TextEditingController(text: widget.server?.name ?? '');
    _addressController = TextEditingController(text: widget.server?.address ?? '');
    _portController = TextEditingController(text: (widget.server?.port ?? 8443).toString());
    _pskController = TextEditingController(text: widget.server?.psk ?? '');
    _pinController = TextEditingController(text: widget.server?.certPin ?? '');
    
    _protocol = widget.server?.protocol ?? 'guarch';
    _coverEnabled = widget.server?.coverEnabled ?? false;
    _sniEnabled = widget.server?.sniEnabled ?? false;
    _dnsFallbackEnabled = widget.server?.dnsFallbackEnabled ?? false;
    _batteryAwareEnabled = widget.server?.batteryAwareEnabled ?? true;
    _dataSaverEnabled = widget.server?.dataSaverEnabled ?? false;
    
    _transportType = widget.server?.transport?.type ?? 'direct';
    _transportHostController.text = widget.server?.transport?.host ?? '';
    _transportPathController.text = widget.server?.transport?.path ?? '/';
    
    _sniMode = widget.server?.sniMode ?? 'weighted';
    _dnsFallbackMode = widget.server?.dnsFallbackMode ?? 'auto';
    _coverDomains = widget.server?.coverDomains.isNotEmpty == true
        ? List.from(widget.server!.coverDomains)
        : [];
    _sniDomains = widget.server?.sniDomains.isNotEmpty == true
        ? List.from(widget.server!.sniDomains)
        : [];
  }

  @override
  void dispose() {
    _nameController.dispose();
    _addressController.dispose();
    _portController.dispose();
    _pskController.dispose();
    _pinController.dispose();
    _domainController.dispose();
    _sniDomainController.dispose();
    _dnsDomainController.dispose();
    _transportHostController.dispose();
    _transportPathController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(isEditing ? 'Edit Server' : 'Add Server'),
        actions: [
          IconButton(
            icon: const Icon(Icons.check),
            onPressed: _save,
            tooltip: 'Save',
          ),
        ],
      ),
      body: Form(
        key: _formKey,
        child: ListView(
          padding: const EdgeInsets.all(24),
          children: [
            _buildSectionHeader('🎯 Server Information'),
            const SizedBox(height: 16),
            
            TextFormField(
              controller: _nameController,
              decoration: const InputDecoration(
                labelText: 'Server Name',
                hintText: 'e.g. Germany Server',
                prefixIcon: Icon(Icons.label_outline),
              ),
              validator: (v) => v == null || v.isEmpty ? 'Name required' : null,
            ),
            const SizedBox(height: 16),
            
            TextFormField(
              controller: _addressController,
              decoration: const InputDecoration(
                labelText: 'Server Address',
                hintText: 'IP or domain',
                prefixIcon: Icon(Icons.dns_outlined),
              ),
              keyboardType: TextInputType.url,
              validator: (v) => v == null || v.isEmpty ? 'Address required' : null,
            ),
            const SizedBox(height: 16),
            
            Row(
              children: [
                Expanded(
                  flex: 2,
                  child: TextFormField(
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
                        return 'Invalid port';
                      }
                      return null;
                    },
                  ),
                ),
                const SizedBox(width: 16),
                Expanded(
                  flex: 3,
                  child: DropdownButtonFormField<String>(
                    value: _protocol,
                    decoration: const InputDecoration(
                      labelText: 'Protocol',
                      prefixIcon: Icon(Icons.router),
                    ),
                    items: [
                      DropdownMenuItem(
                        value: 'guarch',
                        child: Row(
                          children: [
                            const Text('🏹 Guarch (TLS)'),
                            const SizedBox(width: 8),
                            Chip(
                              label: const Text('Stable', style: TextStyle(fontSize: 10, color: Colors.white)),
                              backgroundColor: Colors.green,
                              padding: const EdgeInsets.symmetric(horizontal: 4),
                              materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
                            ),
                          ],
                        ),
                      ),
                      DropdownMenuItem(
                        value: 'grouk',
                        child: Row(
                          children: [
                            const Text('🌩️ Grouk (UDP)'),
                            const SizedBox(width: 8),
                            Chip(
                              label: const Text('Beta', style: TextStyle(fontSize: 10, color: Colors.white)),
                              backgroundColor: Colors.orange,
                              padding: const EdgeInsets.symmetric(horizontal: 4),
                              materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
                            ),
                          ],
                        ),
                      ),
                      DropdownMenuItem(
                        value: 'zhip',
                        child: Row(
                          children: [
                            const Text('⚡ Zhip (QUIC)'),
                            const SizedBox(width: 8),
                            Chip(
                              label: const Text('Experimental', style: TextStyle(fontSize: 10, color: Colors.white)),
                              backgroundColor: Colors.red,
                              padding: const EdgeInsets.symmetric(horizontal: 4),
                              materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
                            ),
                          ],
                        ),
                      ),
                    ],
                    onChanged: (v) {
                      if (v == 'grouk' || v == 'zhip') {
                        _showProtocolWarning(v!);
                      } else {
                        setState(() => _protocol = v!);
                      }
                    },
                  ),
                ),
              ],
            ),
            const SizedBox(height: 4),
            Padding(
              padding: const EdgeInsets.only(left: 12),
              child: Text(
                _protocolDescription,
                style: TextStyle(color: textMuted(context), fontSize: 12),
              ),
            ),

            const SizedBox(height: 32),
            _buildSectionHeader('🔐 Security'),
            const SizedBox(height: 4),
            Text(
              'PSK is required for secure connection',
              style: TextStyle(color: textMuted(context), fontSize: 13),
            ),
            const SizedBox(height: 16),
            
            TextFormField(
              controller: _pskController,
              obscureText: !_pskVisible,
              decoration: InputDecoration(
                labelText: 'Pre-Shared Key (PSK)',
                hintText: 'Must match server PSK',
                prefixIcon: const Icon(Icons.key),
                suffixIcon: IconButton(
                  icon: Icon(
                    _pskVisible ? Icons.visibility_off : Icons.visibility,
                    color: textMuted(context),
                  ),
                  onPressed: () => setState(() => _pskVisible = !_pskVisible),
                ),
              ),
              validator: (v) {
                if (v == null || v.isEmpty) return 'PSK is required';
                if (v.length < 8) return 'PSK must be at least 8 characters';
                return null;
              },
            ),
            const SizedBox(height: 16),
            
            TextFormField(
              controller: _pinController,
              decoration: InputDecoration(
                labelText: 'Certificate PIN (optional)',
                hintText: 'SHA-256 hash from server output',
                prefixIcon: const Icon(Icons.verified_user_outlined),
                helperText: 'Protects against man-in-the-middle attacks',
                helperStyle: TextStyle(color: textMuted(context), fontSize: 11),
              ),
              style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
            ),

            const SizedBox(height: 32),
            _buildSectionHeader('🚀 Transport Settings'),
            const SizedBox(height: 4),
            Text(
              'How data is sent through the tunnel (bypass method)',
              style: TextStyle(color: textMuted(context), fontSize: 13),
            ),
            const SizedBox(height: 16),

            _buildTransportSection(),

            const SizedBox(height: 32),
            _buildToggleSection(
              '🔄 SNI Rotation',
              'Change Server Name Indication every 5 minutes',
              _sniEnabled,
              (v) => setState(() => _sniEnabled = v),
            ),

            if (_sniEnabled) ..._buildSNISection(),

            const SizedBox(height: 32),
            _buildToggleSection(
              '🎭 Cover Traffic',
              'Send real requests to popular sites to hide your traffic',
              _coverEnabled,
              (v) => setState(() => _coverEnabled = v),
            ),

            if (_coverEnabled) ..._buildCoverSection(),

            const SizedBox(height: 32),
            _buildToggleSection(
              '🔌 DNS Fallback',
              'Tunnel traffic over DNS when TLS is blocked (survival mode)',
              _dnsFallbackEnabled,
              (v) => setState(() => _dnsFallbackEnabled = v),
            ),

            if (_dnsFallbackEnabled) ..._buildDNSSection(),

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

  Widget _buildSectionHeader(String title) {
    return Row(
      children: [
        Text(
          title,
          style: TextStyle(
            fontSize: 18,
            fontWeight: FontWeight.w600,
            color: textPrimary(context),
          ),
        ),
      ],
    );
  }

  Widget _buildToggleSection(
    String title,
    String description,
    bool value,
    ValueChanged<bool> onChanged,
  ) {
    return Row(
      children: [
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                title,
                style: TextStyle(
                  fontSize: 18,
                  fontWeight: FontWeight.w600,
                  color: textPrimary(context),
                ),
              ),
              const SizedBox(height: 4),
              Text(
                description,
                style: TextStyle(color: textMuted(context), fontSize: 13),
              ),
            ],
          ),
        ),
        Switch(value: value, onChanged: onChanged),
      ],
    );
  }

  Widget _buildTransportSection() {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            DropdownButtonFormField<String>(
              value: _transportType,
              decoration: const InputDecoration(
                labelText: 'Transport Type',
                prefixIcon: Icon(Icons.swap_horiz),
                helperText: 'Method for sending data',
              ),
              items: const [
                DropdownMenuItem(
                  value: 'direct',
                  child: Text('🔌 Direct (Simple & Fast)'),
                ),
                DropdownMenuItem(
                  value: 'websocket',
                  child: Text('🌐 WebSocket (Bypass)'),
                ),
                DropdownMenuItem(
                  value: 'http2',
                  child: Text('⚡ HTTP/2 (Experimental)'),
                ),
                DropdownMenuItem(
                  value: 'dns',
                  child: Text('🔌 DNS Tunnel (Emergency)'),
                ),
              ],
              onChanged: (v) => setState(() => _transportType = v!),
            ),
            
            const SizedBox(height: 12),
            Container(
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: _getTransportColor().withOpacity(0.1),
                borderRadius: BorderRadius.circular(8),
                border: Border.all(
                  color: _getTransportColor().withOpacity(0.3),
                ),
              ),
              child: Row(
                children: [
                  Icon(
                    _getTransportIcon(),
                    size: 18,
                    color: _getTransportColor(),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Text(
                      _getTransportDescription(),
                      style: TextStyle(
                        fontSize: 12,
                        color: textMuted(context),
                        height: 1.4,
                      ),
                    ),
                  ),
                ],
              ),
            ),
            
            if (_transportType == 'websocket' || _transportType == 'http2') ...[
              const SizedBox(height: 16),
              Divider(color: accentColor(context).withOpacity(0.1)),
              const SizedBox(height: 16),
              
              Text(
                'Advanced Settings',
                style: TextStyle(
                  fontWeight: FontWeight.w600,
                  color: textSecondary(context),
                ),
              ),
              const SizedBox(height: 12),
              
              TextFormField(
                controller: _transportHostController,
                decoration: const InputDecoration(
                  labelText: 'Domain Fronting (optional)',
                  hintText: 'e.g., digikala.com',
                  prefixIcon: Icon(Icons.public),
                  helperText: 'Pretend to connect to this popular domain',
                ),
                keyboardType: TextInputType.url,
              ),
              
              if (_transportType == 'websocket') ...[
                const SizedBox(height: 16),
                TextFormField(
                  controller: _transportPathController,
                  decoration: const InputDecoration(
                    labelText: 'WebSocket Path',
                    hintText: '/ws',
                    prefixIcon: Icon(Icons.route),
                    helperText: 'URL path for WebSocket connection',
                  ),
                ),
              ],
              
              const SizedBox(height: 12),
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: Colors.blue.withOpacity(0.1),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Row(
                  children: [
                    const Icon(Icons.info_outline, size: 16, color: Colors.blue),
                    const SizedBox(width: 8),
                    Expanded(
                      child: Text(
                        'Domain Fronting helps bypass SNI-based filtering (Iran/China)',
                        style: TextStyle(fontSize: 11, color: textMuted(context)),
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }

  Color _getTransportColor() {
    switch (_transportType) {
      case 'websocket':
        return Colors.green;
      case 'http2':
        return Colors.orange;
      case 'dns':
        return Colors.red;
      default:
        return Colors.blue;
    }
  }

  IconData _getTransportIcon() {
    switch (_transportType) {
      case 'websocket':
        return Icons.wifi;
      case 'http2':
        return Icons.flash_on;
      case 'dns':
        return Icons.dns;
      default:
        return Icons.cable;
    }
  }

  String _getTransportDescription() {
    switch (_transportType) {
      case 'direct':
        return 'Direct TCP connection. Fastest but easily detected by DPI firewalls.';
      case 'websocket':
        return 'WebSocket over HTTPS. Looks like normal web browsing. Best for censored networks.';
      case 'http2':
        return 'HTTP/2 multiplexing. Advanced bypass technique. May be unstable.';
      case 'dns':
        return 'Tunnel via DNS queries (port 53). Very slow (~50 Kbps) but works everywhere.';
      default:
        return '';
    }
  }

  void _showProtocolWarning(String protocol) {
    final warnings = {
      'grouk': {
        'title': '⚠️ Grouk Protocol (Beta)',
        'content': '''This protocol is still in beta testing.

⚠️ Known Issues:
• UDP may be blocked on some networks (WiFi, corporate)
• NAT traversal problems on mobile
• Higher battery consumption
• Cover traffic less effective

✅ Use Grouk only if:
• You need maximum speed
• Your network allows UDP traffic
• You're testing/debugging

Recommended: Use Guarch for production.''',
      },
      'zhip': {
        'title': '⚠️ Zhip Protocol (Experimental)',
        'content': '''This protocol uses QUIC and is experimental.

⚠️ Known Issues:
• Not stable on all networks yet
• Some firewalls block QUIC (UDP port 443)
• Limited testing in production
• May have bugs

✅ Use Zhip only if:
• You're an advanced user
• Testing new features
• Contributing to development

Recommended: Use Guarch for reliable connections.''',
      },
    };

    final warning = warnings[protocol]!;

    showDialog(
      context: context,
      barrierDismissible: false,
      builder: (context) => AlertDialog(
        title: Text(warning['title']!),
        content: SingleChildScrollView(
          child: Text(
            warning['content']!,
            style: const TextStyle(height: 1.5),
          ),
        ),
        actions: [
          TextButton(
            onPressed: () {
              setState(() => _protocol = 'guarch');
              Navigator.pop(context);
            },
            child: const Text('Use Guarch (Safe)'),
          ),
          FilledButton(
            onPressed: () {
              setState(() => _protocol = protocol);
              Navigator.pop(context);
            },
            style: FilledButton.styleFrom(
              backgroundColor: Colors.orange,
            ),
            child: const Text('Continue Anyway'),
          ),
        ],
      ),
    );
  }

  List<Widget> _buildSNISection() {
    return [
      const SizedBox(height: 16),
      Card(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                'SNI Configuration',
                style: TextStyle(
                  fontWeight: FontWeight.w600,
                  color: textSecondary(context),
                ),
              ),
              const SizedBox(height: 16),
              
              DropdownButtonFormField<String>(
                value: _sniMode,
                decoration: const InputDecoration(
                  labelText: 'Selection Mode',
                  prefixIcon: Icon(Icons.shuffle),
                ),
                items: const [
                  DropdownMenuItem(value: 'random', child: Text('🎲 Random')),
                  DropdownMenuItem(value: 'weighted', child: Text('⚖️ Weighted (Recommended)')),
                  DropdownMenuItem(value: 'sequential', child: Text('🔄 Sequential')),
                  DropdownMenuItem(value: 'single', child: Text('📍 Single (No Rotation)')),
                ],
                onChanged: (v) => setState(() => _sniMode = v!),
              ),
              
              const SizedBox(height: 16),
              Row(
                children: [
                  Expanded(
                    child: TextField(
                      controller: _sniDomainController,
                      decoration: const InputDecoration(
                        hintText: 'e.g. www.google.com',
                        prefixIcon: Icon(Icons.public, size: 20),
                        isDense: true,
                      ),
                      keyboardType: TextInputType.url,
                      onSubmitted: (_) => _addSNIDomain(),
                    ),
                  ),
                  const SizedBox(width: 8),
                  IconButton.filled(
                    onPressed: _addSNIDomain,
                    icon: const Icon(Icons.add),
                  ),
                ],
              ),
              
              const SizedBox(height: 12),
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: [
                  _quickAddSNIChip('www.google.com'),
                  _quickAddSNIChip('www.microsoft.com'),
                  _quickAddSNIChip('github.com'),
                  _quickAddSNIChip('www.cloudflare.com'),
                  _quickAddSNIChip('stackoverflow.com'),
                  _quickAddSNIChip('learn.microsoft.com'),
                ],
              ),
              
              if (_sniDomains.isNotEmpty) ...[
                const SizedBox(height: 16),
                Divider(color: accentColor(context).withOpacity(0.1)),
                const SizedBox(height: 8),
                
                Text(
                  'Active SNI Domains:',
                  style: TextStyle(
                    fontWeight: FontWeight.w600,
                    fontSize: 13,
                    color: textSecondary(context),
                  ),
                ),
                const SizedBox(height: 8),

                ..._sniDomains.asMap().entries.map((entry) {
                  final index = entry.key;
                  final domain = entry.value;
                  
                  return Padding(
                    padding: const EdgeInsets.only(bottom: 4),
                    child: Row(
                      children: [
                        Icon(
                          domain.fallback ? Icons.shield : 
                          domain.checkHealth ? Icons.check_circle : Icons.circle_outlined,
                          size: 16,
                          color: domain.fallback ? Colors.blue :
                                 domain.checkHealth ? Colors.green : Colors.grey,
                        ),
                        const SizedBox(width: 8),
                        Expanded(
                          child: Text(
                            domain.domain,
                            style: TextStyle(fontSize: 14, color: textSecondary(context)),
                          ),
                        ),
                        
                        if (domain.fallback)
                          Container(
                            padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                            decoration: BoxDecoration(
                              color: Colors.blue.withOpacity(0.2),
                              borderRadius: BorderRadius.circular(4),
                            ),
                            child: const Text('Fallback', style: TextStyle(fontSize: 10, color: Colors.blue)),
                          ),
                        
                        if (domain.checkHealth && !domain.fallback)
                          Container(
                            padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                            decoration: BoxDecoration(
                              color: Colors.green.withOpacity(0.2),
                              borderRadius: BorderRadius.circular(4),
                            ),
                            child: const Text('Health', style: TextStyle(fontSize: 10, color: Colors.green)),
                          ),
                        
                        const SizedBox(width: 8),
                        InkWell(
                          onTap: () => setState(() {
                            _sniDomains.removeAt(index);
                            _recalculateSNIWeights();
                          }),
                          child: const Padding(
                            padding: EdgeInsets.all(4),
                            child: Icon(Icons.close, size: 16, color: Colors.red),
                          ),
                        ),
                      ],
                    ),
                  );
                }).toList(),
              ],
              
              if (_sniDomains.isEmpty)
                const Padding(
                  padding: EdgeInsets.symmetric(vertical: 8),
                  child: Text(
                    'No SNI domains added yet.',
                    style: TextStyle(color: Colors.orange, fontSize: 12),
                  ),
                ),
            ],
          ),
        ),
      ),
    ];
  }

  List<Widget> _buildCoverSection() {
    return [
      const SizedBox(height: 16),
      Card(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                'Cover Domains',
                style: TextStyle(fontWeight: FontWeight.w600, color: textSecondary(context)),
              ),
              const SizedBox(height: 4),
              Text(
                'Add websites you normally visit',
                style: TextStyle(color: textMuted(context), fontSize: 12),
              ),
              const SizedBox(height: 16),
              
              Row(
                children: [
                  Expanded(
                    child: TextField(
                      controller: _domainController,
                      decoration: const InputDecoration(
                        hintText: 'e.g. google.com',
                        prefixIcon: Icon(Icons.public, size: 20),
                        isDense: true,
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
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: [
                  _quickAddChip('google.com'),
                  _quickAddChip('microsoft.com'),
                  _quickAddChip('github.com'),
                  _quickAddChip('stackoverflow.com'),
                  _quickAddChip('cloudflare.com'),
                  _quickAddChip('youtube.com'),
                ],
              ),
              
              if (_coverDomains.isNotEmpty) ...[
                const SizedBox(height: 16),
                Divider(color: accentColor(context).withOpacity(0.1)),
                const SizedBox(height: 8),
                
                Text(
                  'Active Cover Domains:',
                  style: TextStyle(fontWeight: FontWeight.w600, fontSize: 13, color: textSecondary(context)),
                ),
                const SizedBox(height: 8),
                
                ..._coverDomains.asMap().entries.map((entry) {
                  final index = entry.key;
                  final domain = entry.value;
                  return Padding(
                    padding: const EdgeInsets.only(bottom: 4),
                    child: Row(
                      children: [
                        const Icon(Icons.check_circle, size: 16, color: Colors.green),
                        const SizedBox(width: 8),
                        Expanded(
                          child: Text(domain.domain, style: TextStyle(fontSize: 14, color: textSecondary(context))),
                        ),
                        Text('${domain.weight}%', style: TextStyle(color: textMuted(context), fontSize: 12)),
                        const SizedBox(width: 4),
                        InkWell(
                          onTap: () => setState(() {
                            _coverDomains.removeAt(index);
                            _recalculateWeights();
                          }),
                          child: const Padding(
                            padding: EdgeInsets.all(4),
                            child: Icon(Icons.close, size: 16, color: Colors.red),
                          ),
                        ),
                      ],
                    ),
                  );
                }).toList(),
              ],
              
              if (_coverDomains.isEmpty)
                const Padding(
                  padding: EdgeInsets.symmetric(vertical: 8),
                  child: Text('No domains added yet.', style: TextStyle(color: Colors.orange, fontSize: 12)),
                ),
            ],
          ),
        ),
      ),
      
      const SizedBox(height: 16),
      Card(
        child: Column(
          children: [
            SwitchListTile(
              secondary: const Text('🔋', style: TextStyle(fontSize: 20)),
              title: Text('Battery-Aware Mode', style: TextStyle(color: textSecondary(context))),
              subtitle: Text('Reduce cover traffic when battery is low', style: TextStyle(color: textMuted(context), fontSize: 12)),
              value: _batteryAwareEnabled,
              onChanged: (v) => setState(() => _batteryAwareEnabled = v),
            ),
            Divider(height: 1, color: accentColor(context).withOpacity(0.1)),
            SwitchListTile(
              secondary: const Text('💾', style: TextStyle(fontSize: 20)),
              title: Text('Data Saver Mode', style: TextStyle(color: textSecondary(context))),
              subtitle: Text('Halve cover rate to save bandwidth', style: TextStyle(color: textMuted(context), fontSize: 12)),
              value: _dataSaverEnabled,
              onChanged: (v) => setState(() => _dataSaverEnabled = v),
            ),
          ],
        ),
      ),
    ];
  }

  List<Widget> _buildDNSSection() {
    return [
      const SizedBox(height: 16),
      Card(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text('DNS Fallback Configuration', style: TextStyle(fontWeight: FontWeight.w600, color: textSecondary(context))),
              const SizedBox(height: 16),
              
              DropdownButtonFormField<String>(
                value: _dnsFallbackMode,
                decoration: const InputDecoration(
                  labelText: 'Fallback Mode',
                  prefixIcon: Icon(Icons.settings_input_antenna),
                ),
                items: const [
                  DropdownMenuItem(value: 'auto', child: Text('🔄 Auto (Switch on TLS fail)')),
                  DropdownMenuItem(value: 'manual', child: Text('📍 Manual (Always use DNS)')),
                ],
                onChanged: (v) => setState(() => _dnsFallbackMode = v!),
              ),
              
              const SizedBox(height: 16),
              TextFormField(
                controller: _dnsDomainController,
                decoration: const InputDecoration(
                  labelText: 'DNS Tunnel Domain (optional)',
                  hintText: 'tunnel.yourdomain.com',
                  prefixIcon: Icon(Icons.dns),
                  helperText: 'Your authoritative DNS domain for tunneling',
                ),
                keyboardType: TextInputType.url,
              ),
              
              const SizedBox(height: 12),
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: Colors.orange.withOpacity(0.1),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Row(
                  children: [
                    const Icon(Icons.warning_amber, color: Colors.orange, size: 20),
                    const SizedBox(width: 12),
                    Expanded(
                      child: Text(
                        'DNS fallback has reduced speed (~50 Kbps). Use only when TLS is blocked.',
                        style: TextStyle(color: textMuted(context), fontSize: 12),
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    ];
  }

  Widget _quickAddChip(String domain) {
    final exists = _coverDomains.any((d) => d.domain == domain || d.domain == 'www.$domain');
    return ActionChip(
      avatar: Icon(exists ? Icons.check : Icons.add, size: 16, color: exists ? Colors.green : accentColor(context)),
      label: Text(domain, style: const TextStyle(fontSize: 12)),
      onPressed: exists ? null : () => setState(() {
        _coverDomains.add(CoverDomain(domain: domain));
        _recalculateWeights();
      }),
    );
  }

  Widget _quickAddSNIChip(String domain) {
    final exists = _sniDomains.any((d) => d.domain == domain);
    final isFallbackDomain = ['www.cloudflare.com', 'www.microsoft.com'].contains(domain);
    return ActionChip(
      avatar: Icon(exists ? Icons.check : Icons.add, size: 16, color: exists ? Colors.green : accentColor(context)),
      label: Text(domain, style: const TextStyle(fontSize: 12)),
      onPressed: exists ? null : () => setState(() {
        _sniDomains.add(SNIDomain(domain: domain, checkHealth: !isFallbackDomain, fallback: isFallbackDomain));
        _recalculateSNIWeights();
      }),
    );
  }

  void _addDomain() {
    final domain = _domainController.text.trim().toLowerCase();
    if (domain.isEmpty) return;

    String clean = domain.replaceAll('https://', '').replaceAll('http://', '');
    if (clean.endsWith('/')) clean = clean.substring(0, clean.length - 1);

    if (_coverDomains.any((d) => d.domain == clean)) {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('$clean already exists')));
      return;
    }

    setState(() {
      _coverDomains.add(CoverDomain(domain: clean));
      _recalculateWeights();
      _domainController.clear();
    });
  }

  void _addSNIDomain() {
    final domain = _sniDomainController.text.trim().toLowerCase();
    if (domain.isEmpty) return;

    String clean = domain.replaceAll('https://', '').replaceAll('http://', '');
    if (clean.endsWith('/')) clean = clean.substring(0, clean.length - 1);

    if (_sniDomains.any((d) => d.domain == clean)) {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('$clean already exists')));
      return;
    }

    setState(() {
      _sniDomains.add(SNIDomain(domain: clean));
      _recalculateSNIWeights();
      _sniDomainController.clear();
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

  void _recalculateSNIWeights() {
    if (_sniDomains.isEmpty) return;
    final w = 100 ~/ _sniDomains.length;
    final r = 100 % _sniDomains.length;
    for (var i = 0; i < _sniDomains.length; i++) {
      _sniDomains[i].weight = w + (i < r ? 1 : 0);
    }
  }

  void _save() {
    if (!_formKey.currentState!.validate()) return;

    final provider = context.read<AppProvider>();
    final psk = _pskController.text.trim();
    final pin = _pinController.text.trim();
    
    TransportConfig? transport;
    if (_transportType != 'direct') {
      final host = _transportHostController.text.trim();
      final path = _transportPathController.text.trim();
      
      transport = TransportConfig(
        type: _transportType,
        host: host.isEmpty ? null : host,
        path: (_transportType == 'websocket' && path.isNotEmpty) ? path : null,
        useTls: true,
        fallbackOrder: _transportType == 'websocket' 
            ? ['http2', 'dns'] 
            : _transportType == 'http2'
                ? ['websocket', 'dns']
                : null,
      );
    }

    if (isEditing) {
      provider.updateServer(
        widget.server!.copyWith(
          name: _nameController.text.trim(),
          address: _addressController.text.trim(),
          port: int.parse(_portController.text.trim()),
          psk: psk,
          certPin: pin.isEmpty ? null : pin,
          protocol: _protocol,
          transport: transport,
          coverEnabled: _coverEnabled,
          coverDomains: List.from(_coverDomains),
          sniEnabled: _sniEnabled,
          sniMode: _sniMode,
          sniDomains: List.from(_sniDomains),
          dnsFallbackEnabled: _dnsFallbackEnabled,
          dnsFallbackMode: _dnsFallbackMode,
          batteryAwareEnabled: _batteryAwareEnabled,
          dataSaverEnabled: _dataSaverEnabled,
        ),
      );
    } else {
      final server = ServerConfig(
        id: DateTime.now().millisecondsSinceEpoch.toString(),
        name: _nameController.text.trim(),
        address: _addressController.text.trim(),
        port: int.parse(_portController.text.trim()),
        psk: psk,
        certPin: pin.isEmpty ? null : pin,
        protocol: _protocol,
        transport: transport,
        coverEnabled: _coverEnabled,
        coverDomains: List.from(_coverDomains),
        sniEnabled: _sniEnabled,
        sniMode: _sniMode,
        sniDomains: List.from(_sniDomains),
        dnsFallbackEnabled: _dnsFallbackEnabled,
        dnsFallbackMode: _dnsFallbackMode,
        batteryAwareEnabled: _batteryAwareEnabled,
        dataSaverEnabled: _dataSaverEnabled,
      );
      provider.addServer(server);
      provider.setActiveServer(server.id);
    }

    Navigator.pop(context);
  }
}
