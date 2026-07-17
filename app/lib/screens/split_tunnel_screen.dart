import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:guarch/app.dart';
import 'package:guarch/providers/app_provider.dart';

class SplitTunnelScreen extends StatefulWidget {
  const SplitTunnelScreen({super.key});

  @override
  State<SplitTunnelScreen> createState() => _SplitTunnelScreenState();
}

class _SplitTunnelScreenState extends State<SplitTunnelScreen> {
  bool _isWhitelist = true;
  String _searchQuery = '';
  List<Map<String, String>> _allApps = [];
  List<String> _selectedApps = [];
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _loadApps();
  }

  Future<void> _loadApps() async {
    final provider = context.read<AppProvider>();
    try {
      final apps = await provider.getInstalledApps();
      setState(() {
        _allApps = apps;
        _selectedApps = List<String>.from(provider.allowedApps);
        _loading = false;
      });
    } catch (e) {
      setState(() => _loading = false);
    }
  }

  List<Map<String, String>> get _filteredApps {
    if (_searchQuery.isEmpty) return _allApps;
    return _allApps.where((app) {
      final name = app['name']?.toLowerCase() ?? '';
      final pkg = app['packageName']?.toLowerCase() ?? '';
      return name.contains(_searchQuery.toLowerCase()) ||
          pkg.contains(_searchQuery.toLowerCase());
    }).toList();
  }

  void _toggleMode(bool isWhitelist) {
    setState(() {
      _isWhitelist = isWhitelist;
      _selectedApps = [];
    });
  }

  void _save() {
    final provider = context.read<AppProvider>();
    if (_isWhitelist) {
      provider.setAllowedApps(_selectedApps);
    } else {
      provider.setDisallowedApps(_selectedApps);
    }
    Navigator.pop(context);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('App Split Tunneling'),
        actions: [
          TextButton(
            onPressed: _save,
            child: const Text(
              'Save',
              style: TextStyle(fontWeight: FontWeight.bold),
            ),
          ),
        ],
      ),
      body: Column(
        children: [
          Container(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Container(
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: Colors.blue.withOpacity(0.1),
                    borderRadius: BorderRadius.circular(12),
                    border: Border.all(color: Colors.blue.withOpacity(0.3)),
                  ),
                  child: Row(
                    children: [
                      const Icon(Icons.info_outline, color: Colors.blue, size: 20),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Text(
                          _isWhitelist
                              ? 'Whitelist: Only selected apps use VPN. Other apps bypass VPN.'
                              : 'Blacklist: All apps use VPN except selected ones.',
                          style: TextStyle(
                            color: textSecondary(context),
                            fontSize: 13,
                            height: 1.4,
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
                const SizedBox(height: 16),
                Row(
                  children: [
                    Expanded(
                      child: _ModeButton(
                        label: 'Whitelist',
                        icon: Icons.check_circle_outline,
                        isSelected: _isWhitelist,
                        onTap: () => _toggleMode(true),
                      ),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: _ModeButton(
                        label: 'Blacklist',
                        icon: Icons.block,
                        isSelected: !_isWhitelist,
                        onTap: () => _toggleMode(false),
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 16),
                Row(
                  children: [
                    Expanded(
                      child: TextField(
                        decoration: InputDecoration(
                          hintText: 'Search apps...',
                          prefixIcon: const Icon(Icons.search, size: 20),
                          border: OutlineInputBorder(
                            borderRadius: BorderRadius.circular(12),
                          ),
                          isDense: true,
                          contentPadding: const EdgeInsets.symmetric(
                            horizontal: 12,
                            vertical: 12,
                          ),
                        ),
                        onChanged: (value) {
                          setState(() => _searchQuery = value);
                        },
                      ),
                    ),
                    const SizedBox(width: 12),
                    Container(
                      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                      decoration: BoxDecoration(
                        color: accentColor(context).withOpacity(0.1),
                        borderRadius: BorderRadius.circular(12),
                      ),
                      child: Text(
                        '${_selectedApps.length}',
                        style: TextStyle(
                          color: accentColor(context),
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                    ),
                  ],
                ),
              ],
            ),
          ),
          Expanded(
            child: _loading
                ? const Center(child: CircularProgressIndicator())
                : _filteredApps.isEmpty
                    ? Center(
                        child: Column(
                          mainAxisAlignment: MainAxisAlignment.center,
                          children: [
                            Icon(Icons.search_off, size: 64, color: Colors.grey[400]),
                            const SizedBox(height: 16),
                            Text(
                              'No apps found',
                              style: TextStyle(
                                color: Colors.grey[600],
                                fontSize: 16,
                              ),
                            ),
                          ],
                        ),
                      )
                    : ListView.builder(
                        itemCount: _filteredApps.length,
                        itemBuilder: (context, index) {
                          final app = _filteredApps[index];
                          final name = app['name'] ?? '';
                          final pkg = app['packageName'] ?? '';
                          final isChecked = _selectedApps.contains(pkg);

                          return ListTile(
                            leading: CircleAvatar(
                              backgroundColor: accentColor(context).withOpacity(0.1),
                              child: Icon(
                                Icons.apps,
                                color: accentColor(context),
                                size: 20,
                              ),
                            ),
                            title: Text(
                              name,
                              style: TextStyle(
                                color: textSecondary(context),
                                fontWeight: FontWeight.w600,
                                fontSize: 14,
                              ),
                            ),
                            subtitle: Text(
                              pkg,
                              style: TextStyle(
                                color: textMuted(context),
                                fontSize: 11,
                              ),
                            ),
                            trailing: Checkbox(
                              value: isChecked,
                              onChanged: (val) {
                                setState(() {
                                  if (val == true) {
                                    _selectedApps.add(pkg);
                                  } else {
                                    _selectedApps.remove(pkg);
                                  }
                                });
                              },
                            ),
                            onTap: () {
                              setState(() {
                                if (isChecked) {
                                  _selectedApps.remove(pkg);
                                } else {
                                  _selectedApps.add(pkg);
                                }
                              });
                            },
                          );
                        },
                      ),
          ),
        ],
      ),
    );
  }
}

class _ModeButton extends StatelessWidget {
  final String label;
  final IconData icon;
  final bool isSelected;
  final VoidCallback onTap;

  const _ModeButton({
    required this.label,
    required this.icon,
    required this.isSelected,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(12),
      child: Container(
        padding: const EdgeInsets.symmetric(vertical: 12),
        decoration: BoxDecoration(
          color: isSelected ? accentColor(context) : Colors.grey.withOpacity(0.1),
          borderRadius: BorderRadius.circular(12),
          border: Border.all(
            color: isSelected ? accentColor(context) : Colors.grey.withOpacity(0.3),
          ),
        ),
        child: Row(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(
              icon,
              size: 18,
              color: isSelected ? Colors.white : Colors.grey[700],
            ),
            const SizedBox(width: 8),
            Text(
              label,
              style: TextStyle(
                color: isSelected ? Colors.white : Colors.grey[700],
                fontWeight: FontWeight.w600,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
