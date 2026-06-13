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
  
  bool _groukFecEnabled = false;
  int _groukFecDataShards = 10;
  int _groukFecParityShards = 3;
  
  String _transportType = 'direct';
  
  late String _sniMode;
  late String _dnsFallbackMode;
  late List<CoverDomain> _coverDomains;
  late List<SNIDomain> _sniDomains;

  bool get isEditing => widget.server != null;

  String get _protocolDescription {
    switch (_protocol) {
      case 'grouk':
        return 'Fast raw UDP tunnel';
      case 'zhip':
        return 'QUIC-based tunnel with HTTP/3';
      default:
        return 'TLS 1.3 encrypted tunnel';
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
    
    _groukFecEnabled = widget.server?.groukFecEnabled ?? false;
    _groukFecDataShards = widget.server?.groukFecDataShards ?? 10;
    _groukFecParityShards = widget.server?.groukFecParityShards ?? 3;
    
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
          padding: const EdgeInsets.all(16),
          children: [
            _sectionTitle(context, 'Server Information'),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  children: [
                    TextFormField(
                      controller: _nameController,
                      decoration: InputDecoration(
                        labelText: 'Server Name',
                        hintText: 'e.g. Germany Server',
                        prefixIcon: Icon(Icons.label_outline, color: accentColor(context)),
                      ),
                      validator: (v) => v == null || v.isEmpty ? 'Name required' : null,
                    ),
                    const SizedBox(height: 16),
                    
                    TextFormField(
                      controller: _addressController,
                      decoration: InputDecoration(
                        labelText: 'Server Address',
                        hintText: 'IP or domain',
                        prefixIcon: Icon(Icons.dns_outlined, color: accentColor(context)),
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
                            decoration: InputDecoration(
                              labelText: 'Port',
                              hintText: '8443',
                              prefixIcon: Icon(Icons.settings_input_component, color: accentColor(context)),
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
                            decoration: InputDecoration(
                              labelText: 'Protocol',
                              prefixIcon: Icon(Icons.vpn_lock, color: accentColor(context)),
                            ),
                            items: const [
                              DropdownMenuItem(value: 'guarch', child: Text('Guarch (TLS)')),
                              DropdownMenuItem(value: 'grouk', child: Text('Grouk (UDP)')),
                              DropdownMenuItem(value: 'zhip', child: Text('Zhip (QUIC)')),
                            ],
                            onChanged: (v) => setState(() => _protocol = v!),
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 4),
                    Padding(
                      padding: const EdgeInsets.only(left: 12),
                      child: Align(
                        alignment: Alignment.centerLeft,
                        child: Text(
                          _protocolDescription,
                          style: TextStyle(color: textMuted(context), fontSize: 12),
                        ),
                      ),
                    ),
                  ],
                ),
              ),
            ),

            const SizedBox(height: 24),
            _sectionTitle(context, 'Security'),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
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
                        prefixIcon: Icon(Icons.key, color: accentColor(context)),
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
                    
                    if (_protocol != 'grouk') ...[
                      const SizedBox(height: 16),
                      TextFormField(
                        controller: _pinController,
                        decoration: InputDecoration(
                          labelText: 'Certificate PIN (optional)',
                          hintText: 'SHA-256 hash from server output',
                          prefixIcon: Icon(Icons.verified_user_outlined, color: accentColor(context)),
                          helperText: 'Protects against MITM attacks',
                          helperStyle: TextStyle(color: textMuted(context), fontSize: 11),
                        ),
                        style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
                      ),
                    ],
                  ],
                ),
              ),
            ),

            if (_protocol == 'grouk') ...[
              const SizedBox(height: 24),
              _sectionTitle(context, 'Grouk Settings'),
              Card(
                child: Column(
                  children: [
                    SwitchListTile(
                      secondary: Icon(Icons.shield_outlined, color: accentColor(context)),
                      title: Text('FEC (Forward Error Correction)', style: TextStyle(color: textSecondary(context))),
                      subtitle: Text('Recover lost UDP packets without retransmission', style: TextStyle(color: textMuted(context), fontSize: 12)),
                      value: _groukFecEnabled,
                      onChanged: (v) => setState(() => _groukFecEnabled = v),
                    ),
                    
                    if (_groukFecEnabled) ...[
                      Divider(height: 1, color: accentColor(context).withOpacity(0.1)),
                      Padding(
                        padding: const EdgeInsets.all(16),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text('Data Shards: $_groukFecDataShards', style: TextStyle(fontSize: 14, color: textSecondary(context))),
                            Slider(
                              value: _groukFecDataShards.toDouble(),
                              min: 4,
                              max: 20,
                              divisions: 16,
                              label: '$_groukFecDataShards',
                              onChanged: (v) => setState(() => _groukFecDataShards = v.toInt()),
                            ),
                            
                            const SizedBox(height: 8),
                            Text('Parity Shards: $_groukFecParityShards', style: TextStyle(fontSize: 14, color: textSecondary(context))),
                            Slider(
                              value: _groukFecParityShards.toDouble(),
                              min: 1,
                              max: 10,
                              divisions: 9,
                              label: '$_groukFecParityShards',
                              onChanged: (v) => setState(() => _groukFecParityShards = v.toInt()),
                            ),
                            
                            const SizedBox(height: 8),
                            Container(
                              padding: const EdgeInsets.all(12),
                              decoration: BoxDecoration(
                                color: _getFECRecommendationColor().withOpacity(0.1),
                                borderRadius: BorderRadius.circular(8),
                                border: Border.all(color: _getFECRecommendationColor().withOpacity(0.3)),
                              ),
                              child: Row(
                                children: [
                                  Icon(_getFECRecommendationIcon(), size: 18, color: _getFECRecommendationColor()),
                                  const SizedBox(width: 12),
                                  Expanded(
                                    child: Text(_getFECRecommendationText(), style: TextStyle(fontSize: 12, color: textMuted(context), height: 1.4)),
                                  ),
                                ],
                              ),
                            ),
                          ],
                        ),
                      ),
                    ],
                  ],
                ),
              ),
            ],

            if (_protocol != 'grouk') ...[
              const SizedBox(height: 24),
              _sectionTitle(context, 'Transport Settings'),
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(16),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      DropdownButtonFormField<String>(
                        value: _transportType,
                        decoration: InputDecoration(
                          labelText: 'Transport Type',
                          prefixIcon: Icon(Icons.swap_horiz, color: accentColor(context)),
                          helperText: 'How data is sent through tunnel',
                        ),
                        items: const [
                          DropdownMenuItem(value: 'direct', child: Text('Direct')),
                          DropdownMenuItem(value: 'websocket', child: Text('WebSocket')),
                          DropdownMenuItem(value: 'http2', child: Text('HTTP/2')),
                          DropdownMenuItem(value: 'dns', child: Text('DNS Tunnel')),
                        ],
                        onChanged: (v) => setState(() => _transportType = v!),
                      ),
                      
                      const SizedBox(height: 12),
                      Container(
                        padding: const EdgeInsets.all(12),
                        decoration: BoxDecoration(
                          color: _getTransportColor().withOpacity(0.1),
                          borderRadius: BorderRadius.circular(8),
                          border: Border.all(color: _getTransportColor().withOpacity(0.3)),
                        ),
                        child: Row(
                          children: [
                            Icon(_getTransportIcon(), size: 18, color: _getTransportColor()),
                            const SizedBox(width: 12),
                            Expanded(
                              child: Text(_getTransportDescription(), style: TextStyle(fontSize: 12, color: textMuted(context), height: 1.4)),
                            ),
                          ],
                        ),
                      ),
                      
                      if (_transportType == 'websocket' || _transportType == 'http2') ...[
                        const SizedBox(height: 16),
                        Divider(color: accentColor(context).withOpacity(0.1)),
                        const SizedBox(height: 16),
                        
                        Text('Advanced Settings', style: TextStyle(fontWeight: FontWeight.w600, color: textSecondary(context))),
                        const SizedBox(height: 12),
                        
                        TextFormField(
                          controller: _transportHostController,
                          decoration: InputDecoration(
                            labelText: 'Domain Fronting (optional)',
                            hintText: 'e.g., cloudflare.com',
                            prefixIcon: Icon(Icons.public, color: accentColor(context)),
                            helperText: 'Pretend to connect to this domain',
                          ),
                          keyboardType: TextInputType.url,
                        ),
                        
                        if (_transportType == 'websocket') ...[
                          const SizedBox(height: 16),
                          TextFormField(
                            controller: _transportPathController,
                            decoration: InputDecoration(
                              labelText: 'WebSocket Path',
                              hintText: '/ws',
                              prefixIcon: Icon(Icons.route, color: accentColor(context)),
                              helperText: 'URL path for connection',
                            ),
                          ),
                        ],
                      ],
                    ],
                  ),
                ),
              ),

              const SizedBox(height: 24),
              _sectionTitle(context, 'Connection'),
              Card(
                child: Column(
                  children: [
                    ListTile(
                      leading: Icon(Icons.shield_outlined, color: accentColor(context)),
                      title: Text('SNI Rotation', style: TextStyle(color: textSecondary(context))),
                      subtitle: Text(_sniEnabled ? '${_sniDomains.length} domains • $_sniMode mode' : 'Disabled', style: TextStyle(color: textMuted(context), fontSize: 12)),
                      trailing: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Switch(value: _sniEnabled, onChanged: (v) => setState(() => _sniEnabled = v)),
                          const SizedBox(width: 8),
                          Icon(Icons.arrow_forward_ios, size: 16, color: textMuted(context)),
                        ],
                      ),
                      onTap: () => setState(() => _sniEnabled = !_sniEnabled),
                    ),
                    
                    if (_sniEnabled) ...[
                      Divider(height: 1, color: accentColor(context).withOpacity(0.1)),
                      Padding(
                        padding: const EdgeInsets.all(16),
                        child: _buildSNISection(),
                      ),
                    ],
                    
                    Divider(height: 1, color: accentColor(context).withOpacity(0.1)),
                    ListTile(
                      leading: Icon(Icons.theater_comedy, color: accentColor(context)),
                      title: Text('Cover Traffic', style: TextStyle(color: textSecondary(context))),
                      subtitle: Text(_coverEnabled ? '${_coverDomains.length} domains' : 'Disabled', style: TextStyle(color: textMuted(context), fontSize: 12)),
                      trailing: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Switch(value: _coverEnabled, onChanged: (v) => setState(() => _coverEnabled = v)),
                          const SizedBox(width: 8),
                          Icon(Icons.arrow_forward_ios, size: 16, color: textMuted(context)),
                        ],
                      ),
                      onTap: () => setState(() => _coverEnabled = !_coverEnabled),
                    ),
                    
                    if (_coverEnabled) ...[
                      Divider(height: 1, color: accentColor(context).withOpacity(0.1)),
                      Padding(
                        padding: const EdgeInsets.all(16),
                        child: _buildCoverSection(),
                      ),
                    ],
                    
                    Divider(height: 1, color: accentColor(context).withOpacity(0.1)),
                    ListTile(
                      leading: Icon(Icons.dns, color: accentColor(context)),
                      title: Text('DNS Fallback', style: TextStyle(color: textSecondary(context))),
                      subtitle: Text(_dnsFallbackEnabled ? '$_dnsFallbackMode mode' : 'Disabled', style: TextStyle(color: textMuted(context), fontSize: 12)),
                      trailing: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Switch(value: _dnsFallbackEnabled, onChanged: (v) => setState(() => _dnsFallbackEnabled = v)),
                          const SizedBox(width: 8),
                          Icon(Icons.arrow_forward_ios, size: 16, color: textMuted(context)),
                        ],
                      ),
                      onTap: () => setState(() => _dnsFallbackEnabled = !_dnsFallbackEnabled),
                    ),
                    
                    if (_dnsFallbackEnabled) ...[
                      Divider(height: 1, color: accentColor(context).withOpacity(0.1)),
                      Padding(
                        padding: const EdgeInsets.all(16),
                        child: _buildDNSSection(),
                      ),
                    ],
                  ],
                ),
              ),
            ],

            const SizedBox(height: 32),
            FilledButton(
              onPressed: _save,
              style: FilledButton.styleFrom(
                padding: const EdgeInsets.symmetric(vertical: 16),
              ),
              child: Text(isEditing ? 'Save Changes' : 'Add Server', style: const TextStyle(fontSize: 16)),
            ),
            const SizedBox(height: 16),
          ],
        ),
      ),
    );
  }

  Widget _sectionTitle(BuildContext context, String title) {
    return Padding(
      padding: const EdgeInsets.only(left: 4, bottom: 8),
      child: Text(
        title,
        style: TextStyle(
          fontSize: 14,
          fontWeight: FontWeight.w600,
          color: textPrimary(context),
        ),
      ),
    );
  }

  Color _getFECRecommendationColor() {
    final ratio = _groukFecParityShards / _groukFecDataShards;
    if (ratio < 0.2) return Colors.orange;
    if (ratio > 0.5) return Colors.orange;
    return Colors.green;
  }

  IconData _getFECRecommendationIcon() {
    final ratio = _groukFecParityShards / _groukFecDataShards;
    if (ratio < 0.2 || ratio > 0.5) return Icons.warning_amber;
    return Icons.check_circle;
  }

  String _getFECRecommendationText() {
    final ratio = _groukFecParityShards / _groukFecDataShards;
    final overhead = ((_groukFecDataShards + _groukFecParityShards) / _groukFecDataShards * 100 - 100).toStringAsFixed(0);
    
    if (ratio < 0.2) {
      return 'Low redundancy: Can recover up to $_groukFecParityShards lost packets per $_groukFecDataShards. Overhead: +$overhead%';
    } else if (ratio > 0.5) {
      return 'High redundancy: Can recover up to $_groukFecParityShards lost packets per $_groukFecDataShards. Overhead: +$overhead%';
    } else {
      return 'Balanced: Can recover up to $_groukFecParityShards lost packets per $_groukFecDataShards. Overhead: +$overhead%';
    }
  }

  Color _getTransportColor() {
    switch (_transportType) {
      case 'websocket': return Colors.green;
      case 'http2': return Colors.orange;
      case 'dns': return Colors.red;
      default: return Colors.blue;
    }
  }

  IconData _getTransportIcon() {
    switch (_transportType) {
      case 'websocket': return Icons.wifi;
      case 'http2': return Icons.flash_on;
      case 'dns': return Icons.dns;
      default: return Icons.cable;
    }
  }

  String _getTransportDescription() {
    switch (_transportType) {
      case 'direct': return 'Direct TCP connection. Fastest method.';
      case 'websocket': return 'WebSocket over HTTPS. Good for censored networks.';
      case 'http2': return 'HTTP/2 multiplexing. Advanced bypass technique.';
      case 'dns': return 'Tunnel via DNS queries. Very slow but works everywhere.';
      default: return '';
    }
  }

  Widget _buildSNISection() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        DropdownButtonFormField<String>(
          value: _sniMode,
          decoration: InputDecoration(
            labelText: 'Selection Mode',
            prefixIcon: Icon(Icons.shuffle, color: accentColor(context)),
          ),
          items: const [
            DropdownMenuItem(value: 'random', child: Text('Random')),
            DropdownMenuItem(value: 'weighted', child: Text('Weighted')),
            DropdownMenuItem(value: 'sequential', child: Text('Sequential')),
            DropdownMenuItem(value: 'single', child: Text('Single')),
          ],
          onChanged: (v) => setState(() => _sniMode = v!),
        ),
        
        const SizedBox(height: 16),
        Row(
          children: [
            Expanded(
              child: TextField(
                controller: _sniDomainController,
                decoration: InputDecoration(
                  hintText: 'e.g. www.google.com',
                  prefixIcon: Icon(Icons.public, size: 20, color: accentColor(context)),
                  isDense: true,
                ),
                keyboardType: TextInputType.url,
                onSubmitted: (_) => _addSNIDomain(),
              ),
            ),
            const SizedBox(width: 8),
            IconButton.filled(onPressed: _addSNIDomain, icon: const Icon(Icons.add)),
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
          ],
        ),
        
        if (_sniDomains.isNotEmpty) ...[
          const SizedBox(height: 16),
          Divider(color: accentColor(context).withOpacity(0.1)),
          const SizedBox(height: 8),
          
          Text('Active SNI Domains:', style: TextStyle(fontWeight: FontWeight.w600, fontSize: 13, color: textSecondary(context))),
          const SizedBox(height: 8),

          ..._sniDomains.asMap().entries.map((entry) {
            final index = entry.key;
            final domain = entry.value;
            
            return Padding(
              padding: const EdgeInsets.only(bottom: 4),
              child: Row(
                children: [
                  Icon(domain.fallback ? Icons.shield : domain.checkHealth ? Icons.check_circle : Icons.circle_outlined, size: 16, color: domain.fallback ? Colors.blue : domain.checkHealth ? Colors.green : Colors.grey),
                  const SizedBox(width: 8),
                  Expanded(child: Text(domain.domain, style: TextStyle(fontSize: 14, color: textSecondary(context)))),
                  
                  if (domain.fallback)
                    Container(
                      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                      decoration: BoxDecoration(color: Colors.blue.withOpacity(0.2), borderRadius: BorderRadius.circular(4)),
                      child: const Text('Fallback', style: TextStyle(fontSize: 10, color: Colors.blue)),
                    ),
                  
                  if (domain.checkHealth && !domain.fallback)
                    Container(
                      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                      decoration: BoxDecoration(color: Colors.green.withOpacity(0.2), borderRadius: BorderRadius.circular(4)),
                      child: const Text('Health', style: TextStyle(fontSize: 10, color: Colors.green)),
                    ),
                  
                  const SizedBox(width: 8),
                  InkWell(
                    onTap: () => setState(() {
                      _sniDomains.removeAt(index);
                      _recalculateSNIWeights();
                    }),
                    child: const Padding(padding: EdgeInsets.all(4), child: Icon(Icons.close, size: 16, color: Colors.red)),
                  ),
                ],
              ),
            );
          }).toList(),
        ],
        
        if (_sniDomains.isEmpty)
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 8),
            child: Text('No SNI domains added yet.', style: TextStyle(color: Colors.orange, fontSize: 12)),
          ),
      ],
    );
  }

  Widget _buildCoverSection() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('Add websites you normally visit', style: TextStyle(color: textMuted(context), fontSize: 12)),
        const SizedBox(height: 16),
        
        Row(
          children: [
            Expanded(
              child: TextField(
                controller: _domainController,
                decoration: InputDecoration(
                  hintText: 'e.g. google.com',
                  prefixIcon: Icon(Icons.public, size: 20, color: accentColor(context)),
                  isDense: true,
                ),
                keyboardType: TextInputType.url,
                onSubmitted: (_) => _addDomain(),
              ),
            ),
            const SizedBox(width: 8),
            IconButton.filled(onPressed: _addDomain, icon: const Icon(Icons.add)),
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
            _quickAddChip('cloudflare.com'),
          ],
        ),
        
        if (_coverDomains.isNotEmpty) ...[
          const SizedBox(height: 16),
          Divider(color: accentColor(context).withOpacity(0.1)),
          const SizedBox(height: 8),
          
          Text('Active Cover Domains:', style: TextStyle(fontWeight: FontWeight.w600, fontSize: 13, color: textSecondary(context))),
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
                  Expanded(child: Text(domain.domain, style: TextStyle(fontSize: 14, color: textSecondary(context)))),
                  Text('${domain.weight}%', style: TextStyle(color: textMuted(context), fontSize: 12)),
                  const SizedBox(width: 4),
                  InkWell(
                    onTap: () => setState(() {
                      _coverDomains.removeAt(index);
                      _recalculateWeights();
                    }),
                    child: const Padding(padding: EdgeInsets.all(4), child: Icon(Icons.close, size: 16, color: Colors.red)),
                  ),
                ],
              ),
            );
          }).toList(),
        ],
        
        if (_coverDomains.isEmpty)
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 8),
            child: Text('No domains added yet.', style: TextStyle(color: Colors.orange, fontSize: 12)),
          ),
        
        const SizedBox(height: 16),
        Divider(color: accentColor(context).withOpacity(0.1)),
        const SizedBox(height: 16),
        
        SwitchListTile(
          secondary: Icon(Icons.battery_saver, color: accentColor(context)),
          title: Text('Battery-Aware Mode', style: TextStyle(color: textSecondary(context))),
          subtitle: Text('Reduce cover traffic when battery is low', style: TextStyle(color: textMuted(context), fontSize: 12)),
          value: _batteryAwareEnabled,
          onChanged: (v) => setState(() => _batteryAwareEnabled = v),
          contentPadding: EdgeInsets.zero,
        ),
        
        SwitchListTile(
          secondary: Icon(Icons.data_saver_on, color: accentColor(context)),
          title: Text('Data Saver Mode', style: TextStyle(color: textSecondary(context))),
          subtitle: Text('Halve cover rate to save bandwidth', style: TextStyle(color: textMuted(context), fontSize: 12)),
          value: _dataSaverEnabled,
          onChanged: (v) => setState(() => _dataSaverEnabled = v),
          contentPadding: EdgeInsets.zero,
        ),
      ],
    );
  }

  Widget _buildDNSSection() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        DropdownButtonFormField<String>(
          value: _dnsFallbackMode,
          decoration: InputDecoration(
            labelText: 'Fallback Mode',
            prefixIcon: Icon(Icons.settings_input_antenna, color: accentColor(context)),
          ),
          items: const [
            DropdownMenuItem(value: 'auto', child: Text('Auto')),
            DropdownMenuItem(value: 'manual', child: Text('Manual')),
          ],
          onChanged: (v) => setState(() => _dnsFallbackMode = v!),
        ),
        
        const SizedBox(height: 16),
        TextFormField(
          controller: _dnsDomainController,
          decoration: InputDecoration(
            labelText: 'DNS Tunnel Domain (optional)',
            hintText: 'tunnel.yourdomain.com',
            prefixIcon: Icon(Icons.dns, color: accentColor(context)),
            helperText: 'Your authoritative DNS domain',
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
              const Icon(Icons.info_outline, color: Colors.orange, size: 20),
              const SizedBox(width: 12),
              Expanded(
                child: Text('DNS fallback is slow. Use only when TLS is blocked.', style: TextStyle(color: textMuted(context), fontSize: 12)),
              ),
            ],
          ),
        ),
      ],
    );
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
    if (_transportType != 'direct' && _protocol != 'grouk') {
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

    GroukConfig? groukConfig;
    if (_protocol == 'grouk') {
      groukConfig = GroukConfig(
        enableFEC: _groukFecEnabled,
        fecDataShards: _groukFecDataShards,
        fecParityShards: _groukFecParityShards,
      );
    }

    if (isEditing) {
      provider.updateServer(
        widget.server!.copyWith(
          name: _nameController.text.trim(),
          address: _addressController.text.trim(),
          port: int.parse(_portController.text.trim()),
          psk: psk,
          certPin: (_protocol != 'grouk' && pin.isNotEmpty) ? pin : null,
          protocol: _protocol,
          transport: transport,
          grouk: groukConfig,
          coverEnabled: _protocol != 'grouk' ? _coverEnabled : false,
          coverDomains: _protocol != 'grouk' ? List.from(_coverDomains) : [],
          sniEnabled: _protocol != 'grouk' ? _sniEnabled : false,
          sniMode: _protocol != 'grouk' ? _sniMode : 'weighted',
          sniDomains: _protocol != 'grouk' ? List.from(_sniDomains) : [],
          dnsFallbackEnabled: _protocol != 'grouk' ? _dnsFallbackEnabled : false,
          dnsFallbackMode: _protocol != 'grouk' ? _dnsFallbackMode : 'auto',
          batteryAwareEnabled: _protocol != 'grouk' ? _batteryAwareEnabled : true,
          dataSaverEnabled: _protocol != 'grouk' ? _dataSaverEnabled : false,
        ),
      );
    } else {
      final server = ServerConfig(
        id: DateTime.now().millisecondsSinceEpoch.toString(),
        name: _nameController.text.trim(),
        address: _addressController.text.trim(),
        port: int.parse(_portController.text.trim()),
        psk: psk,
        certPin: (_protocol != 'grouk' && pin.isNotEmpty) ? pin : null,
        protocol: _protocol,
        transport: transport,
        grouk: groukConfig,
        coverEnabled: _protocol != 'grouk' ? _coverEnabled : false,
        coverDomains: _protocol != 'grouk' ? List.from(_coverDomains) : [],
        sniEnabled: _protocol != 'grouk' ? _sniEnabled : false,
        sniMode: _protocol != 'grouk' ? _sniMode : 'weighted',
        sniDomains: _protocol != 'grouk' ? List.from(_sniDomains) : [],
        dnsFallbackEnabled: _protocol != 'grouk' ? _dnsFallbackEnabled : false,
        dnsFallbackMode: _protocol != 'grouk' ? _dnsFallbackMode : 'auto',
        batteryAwareEnabled: _protocol != 'grouk' ? _batteryAwareEnabled : true,
        dataSaverEnabled: _protocol != 'grouk' ? _dataSaverEnabled : false,
      );
      provider.addServer(server);
      provider.setActiveServer(server.id);
    }

    Navigator.pop(context);
  }
}
