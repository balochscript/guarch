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
    FlutterLog.d('Provider', '=== init START ===');
    try {
      _prefs = await SharedPreferences.getInstance();
      _isDarkMode = _prefs.getBool('dark_mode') ?? true;
      _activeServerId = _prefs.getString('active_server');
      _loadServers();
      FlutterLog.d('Provider', 'Prefs loaded. servers=${_servers.length} active=$_activeServerId');
    } catch (e) {
      FlutterLog.e('Provider', 'Prefs FAILED', e);
    }

    try {
      await _engine.init();
      FlutterLog.d('Provider', 'Engine init done');
    } catch (e) {
      FlutterLog.e('Provider', 'Engine init FAILED', e);
    }

    _statusSub = _engine.statusStream.listen((status) {
      FlutterLog.d('Provider', 'Status event: $status');
      switch (status) {
        case 'connected':
          if (_status != VpnStatus.connected) {
            _status = VpnStatus.connected;
            _connectTime ??= DateTime.now();
            _startStatsTimer();
            notifyListeners();
          }
          break;
        case 'disconnected':
          _status = VpnStatus.disconnected;
          _stopStatsTimer();
          _connectTime = null;
          _stats = const ConnectionStats();
          notifyListeners();
          break;
        default:
          break;
      }
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
      FlutterLog.d('EngineLog', msg);
      _addLog(msg);
      notifyListeners();
    });

    if (_servers.isNotEmpty) {
      _addLog('Auto-pinging ${_servers.length} servers...');
      notifyListeners();
      pingAllServers();
    }

    FlutterLog.d('Provider', '=== init DONE ===');
  }

  void _loadServers() {
    try {
      final data = _prefs.getString('servers');
      if (data != null) {
        final list = jsonDecode(data) as List;
        _servers = list.map((j) => ServerConfig.fromJson(j)).toList();
      }
    } catch (e) {
      FlutterLog.e('Provider', 'loadServers FAILED', e);
    }
    notifyListeners();
  }

  Future<void> _saveServers() async {
    try {
      final data = jsonEncode(_servers.map((s) => s.toJson()).toList());
      await _prefs.setString('servers', data);
    } catch (e) {
      FlutterLog.e('Provider', 'saveServers FAILED', e);
    }
  }

  void addServer(ServerConfig server) {
    _servers.add(server);
    _saveServers();
    _addLog('Server added: ${server.name} (${server.fullAddress})');
    notifyListeners();
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
    try {
      final name = _servers.firstWhere((s) => s.id == id).name;
      _servers.removeWhere((s) => s.id == id);
      if (_activeServerId == id) {
        _activeServerId = null;
        _prefs.remove('active_server');
      }
      _saveServers();
      _addLog('Server removed: $name');
    } catch (e) {
      FlutterLog.e('Provider', 'removeServer FAILED', e);
    }
    notifyListeners();
  }

  void setActiveServer(String id) {
    _activeServerId = id;
    _prefs.setString('active_server', id);
    _addLog('Active: ${activeServer?.name}');
    notifyListeners();
  }

  // ═══════════════════════════════
  // NPV-Style Connect
  // ═══════════════════════════════

  Future<void> connect() async {
    FlutterLog.d('Provider', '=== connect() ===');

    if (activeServer == null) {
      FlutterLog.w('Provider', '  No active server');
      return;
    }
    if (_status == VpnStatus.connecting || _status == VpnStatus.connected) {
      FlutterLog.w('Provider', '  Already ${_status.name}');
      return;
    }

    final server = activeServer!;
    FlutterLog.d('Provider', '  ${server.protocol} → ${server.fullAddress}');

    if (server.psk.isEmpty) {
      _addLog('Error: PSK is required');
      _status = VpnStatus.error;
      notifyListeners();
      await Future.delayed(const Duration(seconds: 2));
      _status = VpnStatus.disconnected;
      notifyListeners();
      return;
    }

    // ← فوراً connecting نشون بده
    _status = VpnStatus.connecting;
    _addLog('Guarching to ${server.name}...');
    notifyListeners();

    try {
      // engine.connect الان NPV-style هست:
      // VPN فوری شروع میشه → true برمیگرده
      // اتصال سرور پشت صحنه انجام میشه
      final success = await _engine.connect(
        serverAddr: server.address,
        serverPort: server.port,
        psk: server.psk,
        certPin: server.certPin,
        listenPort: server.listenPort,
        coverEnabled: server.coverEnabled,
        protocol: server.protocol,
      ).timeout(
        const Duration(seconds: 10),
        onTimeout: () {
          FlutterLog.w('Provider', '  Connect timeout - assuming VPN failed');
          return false;
        },
      );

      FlutterLog.d('Provider', '  Result: $success');

      if (success) {
        // ← NPV-style: فوراً "connected" نشون بده
        _status = VpnStatus.connected;
        _connectTime = DateTime.now();
        _startStatsTimer();
        _addLog('🎯 Guarch activated');
      } else {
        _status = VpnStatus.error;
        _addLog('Guarch failed');
        if (!_engine.isNativeAvailable) {
          _addLog('Native engine not available');
        }
        notifyListeners();
        await Future.delayed(const Duration(seconds: 2));
        _status = VpnStatus.disconnected;
      }
    } catch (e) {
      FlutterLog.e('Provider', '  Connect FAILED', e);
      _addLog('Error: $e');
      _status = VpnStatus.error;
      notifyListeners();
      await Future.delayed(const Duration(seconds: 2));
      _status = VpnStatus.disconnected;
    }

    notifyListeners();
  }

  Future<void> disconnect() async {
    FlutterLog.d('Provider', '=== disconnect() ===');
    if (_status != VpnStatus.connected && _status != VpnStatus.connecting) {
      return;
    }

    _status = VpnStatus.disconnecting;
    _addLog('De-Guarching...');
    notifyListeners();

    try {
      await _engine.disconnect().timeout(
        const Duration(seconds: 5),
        onTimeout: () => true,
      );
    } catch (e) {
      FlutterLog.e('Provider', 'disconnect error', e);
    }

    _status = VpnStatus.disconnected;
    _stats = const ConnectionStats();
    _connectTime = null;
    _stopStatsTimer();
    _addLog('Guarch deactivated');
    notifyListeners();
  }

  void toggleConnection() {
    FlutterLog.d('Provider', '>>> toggleConnection (${_status.name})');
    if (_status == VpnStatus.connected) {
      disconnect();
    } else if (_status == VpnStatus.disconnected || _status == VpnStatus.error) {
      connect();
    }
    // اگه connecting یا disconnecting هست، کاری نکن
  }

  // ═══════════════════════════════
  // Ping
  // ═══════════════════════════════

  Future<int> pingServer(ServerConfig server) async {
    _addLog('Pinging ${server.name}...');
    notifyListeners();
    final ping = await _engine.ping(server.address, server.port);
    _addLog(ping > 0 ? '${server.name}: ${ping}ms ✅' : '${server.name}: unreachable ❌');
    notifyListeners();
    return ping;
  }

  Future<void> pingAllServers() async {
    for (var i = 0; i < _servers.length; i++) {
      final ping = await pingServer(_servers[i]);
      _servers[i] = _servers[i].copyWith(ping: ping);
      notifyListeners();
    }
    _saveServers();
  }

  // ═══════════════════════════════
  // Import / Export
  // ═══════════════════════════════

  void importConfig(String data) {
    try {
      ServerConfig server;
      if (data.startsWith('guarch://') || data.startsWith('grouk://') || data.startsWith('zhip://')) {
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
    _statsTimer?.cancel();
    _statsTimer = Timer.periodic(const Duration(seconds: 1), (_) {
      if (_connectTime != null && _status == VpnStatus.connected) {
        _stats = _stats.copyWith(duration: DateTime.now().difference(_connectTime!));
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
    if (_logs.length > 200) _logs = _logs.sublist(0, 200);
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
