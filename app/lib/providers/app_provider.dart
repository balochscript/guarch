import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:guarch/models/server_config.dart';
import 'package:guarch/models/connection_state.dart';
import 'package:guarch/services/guarch_engine.dart';

class AppProvider extends ChangeNotifier {
  late SharedPreferences _prefs;
  final GuarchEngine _engine = GuarchEngine();

  List<ServerConfig> _servers = [];
  VpnStatus _status = VpnStatus.disconnected;
  ConnectionStats _stats = const ConnectionStats();
  bool _isDarkMode = true;
  String? _activeServerId;
  List<String> _logs = [];
  Timer? _statsTimer;
  DateTime? _connectTime;
  StreamSubscription? _statusSub;
  StreamSubscription? _statsSub;
  StreamSubscription? _logSub;

  List<ServerConfig> get servers => _servers;
  VpnStatus get status => _status;
  ConnectionStats get stats => _stats;
  bool get isDarkMode => _isDarkMode;
  List<String> get logs => _logs;
  bool get isConnected => _status == VpnStatus.connected;

  ServerConfig? get activeServer {
    if (_activeServerId == null) return null;
    try {
      return _servers.firstWhere((s) => s.id == _activeServerId);
    } catch (_) {
      return null;
    }
  }

  Future<void> init() async {
    _prefs = await SharedPreferences.getInstance();
    _isDarkMode = _prefs.getBool('dark_mode') ?? true;
    _activeServerId = _prefs.getString('active_server');
    _loadServers();

    await _engine.init();

    _statusSub = _engine.statusStream.listen((status) {
      switch (status) {
        case 'connected':
          _status = VpnStatus.connected;
          _connectTime = DateTime.now();
          _startStatsTimer();
          break;
        case 'connecting':
          _status = VpnStatus.connecting;
          break;
        case 'disconnecting':
          _status = VpnStatus.disconnecting;
          break;
        case 'disconnected':
          _status = VpnStatus.disconnected;
          _stopStatsTimer();
          _connectTime = null;
          _stats = const ConnectionStats();
          break;
        default:
          _status = VpnStatus.disconnected;
      }
      notifyListeners();
    });

    _statsSub = _engine.statsStream.listen((data) {
      _stats = _stats.copyWith(
        uploadSpeed: data['upload_speed'] as int? ?? 0,    
        downloadSpeed: data['download_speed'] as int? ?? 0,   
        totalUpload: data['total_upload'] as int? ?? 0,
        totalDownload: data['total_download'] as int? ?? 0,
        coverRequests: data['cover_requests'] as int? ?? 0,
        duration: Duration(seconds: data['duration_seconds'] as int? ?? 0),
      );
      notifyListeners();
    });

    _logSub = _engine.logStream.listen((msg) {
      _addLog(msg);
      notifyListeners();
    });

    // پینگ خودکار هنگام شروع
    if (_servers.isNotEmpty) {
      _addLog('Auto-pinging ${_servers.length} servers...');
      notifyListeners();
      pingAllServers();
    }
  }

  void _loadServers() {
    final data = _prefs.getString('servers');
    if (data != null) {
      final list = jsonDecode(data) as List;
      _servers = list.map((j) => ServerConfig.fromJson(j)).toList();
    }
    notifyListeners();
  }

  Future<void> _saveServers() async {
    final data = jsonEncode(_servers.map((s) => s.toJson()).toList());
    await _prefs.setString('servers', data);
  }

  void addServer(ServerConfig server) {
    _servers.add(server);
    _saveServers();
    _addLog('Server added: ${server.name} (${server.fullAddress})');
    if (server.coverEnabled) {
      _addLog('Cover: ${server.coverDomains.map((d) => d.domain).join(", ")}');
    }
    notifyListeners();

    // پینگ خودکار
    pingServer(server).then((ping) {
      final index = _servers.indexWhere((s) => s.id == server.id);
      if (index >= 0) {
        _servers[index] = _servers[index].copyWith(ping: ping);
        _saveServers();
        notifyListeners();
      }
    });
  }

  void updateServer(ServerConfig server) {
    final index = _servers.indexWhere((s) => s.id == server.id);
    if (index >= 0) {
      _servers[index] = server;
      _saveServers();
      notifyListeners();
    }
  }

  void removeServer(String id) {
    final name = _servers.firstWhere((s) => s.id == id).name;
    _servers.removeWhere((s) => s.id == id);
    if (_activeServerId == id) {
      _activeServerId = null;
      _prefs.remove('active_server');
    }
    _saveServers();
    _addLog('Server removed: $name');
    notifyListeners();
  }

  void setActiveServer(String id) {
    _activeServerId = id;
    _prefs.setString('active_server', id);
    _addLog('Active: ${activeServer?.name}');
    notifyListeners();
  }

    Future<void> connect() async {
    if (activeServer == null) return;
    if (_status == VpnStatus.connecting || _status == VpnStatus.connected) return;

    // ✅ بررسی PSK
    if (activeServer!.psk.isEmpty) {
      _addLog('Error: PSK is required. Edit server settings.');
      _status = VpnStatus.error;
      notifyListeners();
      await Future.delayed(const Duration(seconds: 2));
      _status = VpnStatus.disconnected;
      notifyListeners();
      return;
    }

    _status = VpnStatus.connecting;
    _addLog('Guarching to ${activeServer!.name}...');
    if (activeServer!.coverEnabled) {
      _addLog('Cover: ${activeServer!.coverDomains.map((d) => d.domain).join(", ")}');
    }
    notifyListeners();

    final success = await _engine.connect(
      serverAddr: activeServer!.address,
      serverPort: activeServer!.port,
      psk: activeServer!.psk,                   // ✅
      certPin: activeServer!.certPin,            // ✅
      listenPort: activeServer!.listenPort,      // ✅
      coverEnabled: activeServer!.coverEnabled,
    );

    if (!success) {
      _status = VpnStatus.error;
      _addLog('Guarch failed!');
      if (!_engine.isNativeAvailable) {          // ✅
        _addLog('Native engine not built. See docs for gomobile setup.');
      }
      notifyListeners();
      await Future.delayed(const Duration(seconds: 2));
      _status = VpnStatus.disconnected;
      notifyListeners();
    }
  }

  Future<void> disconnect() async {
    if (_status != VpnStatus.connected) return;

    _status = VpnStatus.disconnecting;
    _addLog('De-Guarching...');
    notifyListeners();

    await _engine.disconnect();

    _status = VpnStatus.disconnected;
    _stats = const ConnectionStats();
    _connectTime = null;
    _addLog('Guarch deactivated');
    notifyListeners();
  }

  void toggleConnection() {
    if (isConnected) {
      disconnect();
    } else {
      connect();
    }
  }

  Future<int> pingServer(ServerConfig server) async {
    _addLog('Pinging ${server.name} (${server.fullAddress})...');
    notifyListeners();

    final ping = await _engine.ping(server.address, server.port);

    if (ping > 0) {
      _addLog('${server.name}: ${ping}ms ✅');
    } else {
      _addLog('${server.name}: unreachable ❌');
    }
    notifyListeners();
    return ping;
  }

  Future<void> pingAllServers() async {
    _addLog('Pinging ${_servers.length} servers...');
    notifyListeners();

    for (var i = 0; i < _servers.length; i++) {
      final ping = await pingServer(_servers[i]);
      _servers[i] = _servers[i].copyWith(ping: ping);
      notifyListeners();
    }
    _saveServers();
    _addLog('Ping complete');
    notifyListeners();
  }

  void importConfig(String data) {
    try {
      ServerConfig server;
      if (data.startsWith('guarch://')) {
        server = ServerConfig.fromShareString(data);
      } else if (data.startsWith('{')) {
        final json = jsonDecode(data) as Map<String, dynamic>;
        json['id'] = DateTime.now().millisecondsSinceEpoch.toString();
        server = ServerConfig.fromJson(json);
      } else {
        server = ServerConfig.fromShareString(data);
      }
      if (server.address.isEmpty) {
        _addLog('Import failed: empty address');
        notifyListeners();
        return;
      }
      addServer(server);
      _addLog('Imported: ${server.name}');
    } catch (e) {
      _addLog('Import failed: $e');
      notifyListeners();
    }
  }

  String exportConfig(ServerConfig server) => server.toShareString();

  String exportConfigJson(ServerConfig server) {
    const encoder = JsonEncoder.withIndent('  ');
    return encoder.convert(server.toJson());
  }

  void toggleTheme() {
    _isDarkMode = !_isDarkMode;
    _prefs.setBool('dark_mode', _isDarkMode);
    notifyListeners();
  }

  void _startStatsTimer() {
    _statsTimer = Timer.periodic(const Duration(seconds: 1), (_) {
      if (_connectTime != null && _status == VpnStatus.connected) {
        _stats = _stats.copyWith(
          duration: DateTime.now().difference(_connectTime!),
        );
        notifyListeners();
      }
    });
  }

  void _stopStatsTimer() {
    _statsTimer?.cancel();
    _statsTimer = null;
  }

  void _addLog(String message) {
    final time = DateTime.now().toString().substring(11, 19);
    _logs.insert(0, '[$time] $message');
    if (_logs.length > 200) {
      _logs = _logs.sublist(0, 200);
    }
  }

  void clearLogs() {
    _logs.clear();
    notifyListeners();
  }

  @override
  void dispose() {
    _stopStatsTimer();
    _statusSub?.cancel();
    _statsSub?.cancel();
    _logSub?.cancel();
    _engine.dispose();
    super.dispose();
  }
}
