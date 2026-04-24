import 'dart:async';
import 'dart:convert';
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
  StreamSubscription? _errorSub;
  StreamSubscription? _sniSub;
  StreamSubscription? _dnsSub;

  // Battery and Data Saver
  int _batteryLevel = 100;
  bool _dataSaverEnabled = false;

  // Getters
  List<ServerConfig> get servers => _servers;
  VpnStatus get status => _status;
  ConnectionStats get stats => _stats;
  bool get isDarkMode => _isDarkMode;
  List<String> get logs => _logs;
  bool get isConnected => _status == VpnStatus.connected;
  int get batteryLevel => _batteryLevel;
  bool get dataSaverEnabled => _dataSaverEnabled;

  ServerConfig? get activeServer {
    if (_activeServerId == null) return null;
    try {
      return _servers.firstWhere((s) => s.id == _activeServerId);
    } catch (_) {
      return null;
    }
  }

  // ═══════════════════════════════════════════════════════════════
  // Initialization
  // ═══════════════════════════════════════════════════════════════

  Future<void> init() async {
    FlutterLog.d('Provider', '=== init START (v1.0.1) ===');
    
    try {
      _prefs = await SharedPreferences.getInstance();
      _isDarkMode = _prefs.getBool('dark_mode') ?? true;
      _activeServerId = _prefs.getString('active_server');
      _batteryLevel = _prefs.getInt('battery_level') ?? 100;
      _dataSaverEnabled = _prefs.getBool('data_saver') ?? false;
      
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

    // Status stream
    _statusSub = _engine.statusStream.listen((status) {
      FlutterLog.d('Provider', 'Status event: $status');
      switch (status) {
        case 'connected':
          if (_status != VpnStatus.connected) {
            _status = VpnStatus.connected;
            _connectTime ??= DateTime.now();
            _startStatsTimer();
            _addLog('✅ Connected successfully');
            notifyListeners();
          }
          break;
        case 'connecting':
          _status = VpnStatus.connecting;
          _addLog('🔄 Connecting...');
          notifyListeners();
          break;
        case 'disconnected':
          _status = VpnStatus.disconnected;
          _stopStatsTimer();
          _connectTime = null;
          _stats = const ConnectionStats();
          _addLog('⚫ Disconnected');
          notifyListeners();
          break;
        case 'error':
          _status = VpnStatus.error;
          _addLog('❌ Connection error');
          notifyListeners();
          break;
        default:
          break;
      }
    });

    // Stats stream
    _statsSub = _engine.statsStream.listen((data) {
      _stats = ConnectionStats.fromJson(data);
      
      // Update duration from connectTime if available
      if (_connectTime != null && _status == VpnStatus.connected) {
        _stats = _stats.copyWith(
          duration: DateTime.now().difference(_connectTime!),
        );
      }
      
      notifyListeners();
    });

    // Log stream
    _logSub = _engine.logStream.listen((msg) {
      FlutterLog.d('EngineLog', msg);
      _addLog(msg);
      notifyListeners();
    });

    // Error stream
    _errorSub = _engine.errorStream.listen((error) {
      FlutterLog.e('Engine', 'Error event: $error');
      _addLog('⚠️ Error: $error');
      
      if (_status == VpnStatus.connecting) {
        _status = VpnStatus.error;
        notifyListeners();
        
        // Auto-retry DNS fallback if enabled
        Future.delayed(const Duration(seconds: 2), () {
          if (activeServer?.dnsFallbackEnabled == true) {
            _addLog('🔄 Retrying with DNS fallback...');
            notifyListeners();
          } else {
            _status = VpnStatus.disconnected;
            notifyListeners();
          }
        });
      }
    });

    // SNI rotation stream
    _sniSub = _engine.sniStream.listen((newSNI) {
      FlutterLog.d('Provider', 'SNI rotated: $newSNI');
      _addLog('🔄 SNI: $newSNI');
      _stats = _stats.copyWith(
        currentSNI: newSNI,
        sniSwitches: _stats.sniSwitches + 1,
      );
      notifyListeners();
    });

    // DNS fallback stream
    _dnsSub = _engine.dnsFallbackStream.listen((enabled) {
      FlutterLog.d('Provider', 'DNS fallback: $enabled');
      if (enabled) {
        _addLog('⚠️ DNS Fallback Mode Active (Reduced Speed)');
      } else {
        _addLog('✅ TLS Mode Restored');
      }
      _stats = _stats.copyWith(dnsFallbackUsed: enabled);
      notifyListeners();
    });

    // Auto-ping servers
    if (_servers.isNotEmpty) {
      _addLog('📡 Auto-pinging ${_servers.length} servers...');
      notifyListeners();
      pingAllServers();
    }

    FlutterLog.d('Provider', '=== init DONE ===');
  }

  // ═══════════════════════════════════════════════════════════════
  // Server Management
  // ═══════════════════════════════════════════════════════════════

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
    _addLog('➕ Server added: ${server.name} (${server.fullAddress})');
    notifyListeners();
    
    // Auto-ping new server
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
      _addLog('✏️ Server updated: ${server.name}');
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
      _addLog('🗑️ Server removed: $name');
    } catch (e) {
      FlutterLog.e('Provider', 'removeServer FAILED', e);
    }
    notifyListeners();
  }

  void setActiveServer(String id) {
    _activeServerId = id;
    _prefs.setString('active_server', id);
    _addLog('⭐ Active: ${activeServer?.name}');
    notifyListeners();
  }

  // ═══════════════════════════════════════════════════════════════
  // Connection Control
  // ═══════════════════════════════════════════════════════════════

  Future<void> connect() async {
    FlutterLog.d('Provider', '=== connect() ===');

    if (activeServer == null) {
      FlutterLog.w('Provider', '  No active server');
      _addLog('❌ No server selected');
      notifyListeners();
      return;
    }
    
    if (_status == VpnStatus.connecting || _status == VpnStatus.connected) {
      FlutterLog.w('Provider', '  Already ${_status.name}');
      return;
    }

    final server = activeServer!;
    FlutterLog.d('Provider', '  ${server.protocol} → ${server.fullAddress}');

    if (server.psk.isEmpty) {
      _addLog('❌ Error: PSK is required');
      _status = VpnStatus.error;
      notifyListeners();
      await Future.delayed(const Duration(seconds: 2));
      _status = VpnStatus.disconnected;
      notifyListeners();
      return;
    }

    // Set status to connecting immediately
    _status = VpnStatus.connecting;
    _addLog('🏹 Connecting to ${server.name}...');
    _addLog('📡 Protocol: ${server.protocolLabel}');
    
    if (server.sniEnabled) {
      _addLog('🔄 SNI rotation: ${server.sniMode} (${server.sniDomains.length} domains)');
    }
    
    if (server.coverEnabled) {
      _addLog('🎭 Cover traffic: enabled (${server.coverDomains.length} domains)');
    }
    
    if (server.dnsFallbackEnabled) {
      _addLog('🔌 DNS fallback: enabled (${server.dnsFallbackMode})');
    }
    
    notifyListeners();

    try {
      // Load config from server
      final configJson = server.toJson();
      
      // Connect through engine
      final success = await _engine.connectWithConfig(
        jsonEncode(configJson),
      ).timeout(
        const Duration(seconds: 15),
        onTimeout: () {
          FlutterLog.w('Provider', '  Connect timeout');
          return false;
        },
      );

      FlutterLog.d('Provider', '  Result: $success');

      if (success) {
        _status = VpnStatus.connected;
        _connectTime = DateTime.now();
        _startStatsTimer();
        _addLog('✅ Connected successfully!');
      } else {
        _status = VpnStatus.error;
        _addLog('❌ Connection failed');
        
        if (!_engine.isNativeAvailable) {
          _addLog('⚠️ Native engine not available');
        }
        
        notifyListeners();
        await Future.delayed(const Duration(seconds: 2));
        _status = VpnStatus.disconnected;
      }
    } catch (e) {
      FlutterLog.e('Provider', '  Connect FAILED', e);
      _addLog('❌ Error: $e');
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
    _addLog('🔌 Disconnecting...');
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
    _addLog('✅ Disconnected');
    notifyListeners();
  }

  void toggleConnection() {
    FlutterLog.d('Provider', '>>> toggleConnection (${_status.name})');
    
    if (_status == VpnStatus.connected) {
      disconnect();
    } else if (_status == VpnStatus.disconnected || _status == VpnStatus.error) {
      connect();
    }
  }

  // ═══════════════════════════════════════════════════════════════
  // Battery & Data Saver
  // ═══════════════════════════════════════════════════════════════

  Future<void> setBatteryLevel(int level) async {
    _batteryLevel = level;
    await _prefs.setInt('battery_level', level);
    
    if (_status == VpnStatus.connected) {
      await _engine.setBatteryLevel(level);
      
      if (activeServer?.batteryAwareEnabled == true && level < 30) {
        _addLog('🔋 Low battery - reducing cover traffic');
      }
    }
    
    notifyListeners();
  }

  Future<void> toggleDataSaver() async {
    _dataSaverEnabled = !_dataSaverEnabled;
    await _prefs.setBool('data_saver', _dataSaverEnabled);
    
    if (_status == VpnStatus.connected) {
      await _engine.setDataSaverMode(_dataSaverEnabled);
      _addLog('💾 Data saver: ${_dataSaverEnabled ? "ON" : "OFF"}');
    }
    
    notifyListeners();
  }

  // ═══════════════════════════════════════════════════════════════
  // Ping
  // ═══════════════════════════════════════════════════════════════

  Future<int> pingServer(ServerConfig server) async {
    _addLog('📡 Pinging ${server.name}...');
    notifyListeners();
    
    final ping = await _engine.ping(server.address, server.port);
    
    if (ping > 0) {
      _addLog('${server.pingEmoji} ${server.name}: ${ping}ms');
    } else {
      _addLog('🔴 ${server.name}: unreachable');
    }
    
    notifyListeners();
    return ping;
  }

  Future<void> pingAllServers() async {
    for (var i = 0; i < _servers.length; i++) {
      final ping = await pingServer(_servers[i]);
      _servers[i] = _servers[i].copyWith(ping: ping);
      notifyListeners();
      
      // Small delay between pings
      if (i < _servers.length - 1) {
        await Future.delayed(const Duration(milliseconds: 500));
      }
    }
    _saveServers();
    _addLog('✅ Ping test complete');
    notifyListeners();
  }

  // ═══════════════════════════════════════════════════════════════
  // Import / Export
  // ═══════════════════════════════════════════════════════════════

  void importConfig(String data) {
    try {
      ServerConfig server;
      
      if (data.startsWith('guarch://') || 
          data.startsWith('grouk://') || 
          data.startsWith('zhip://')) {
        server = ServerConfig.fromShareString(data);
      } else if (data.trim().startsWith('{')) {
        final json = jsonDecode(data) as Map<String, dynamic>;
        json['id'] = DateTime.now().millisecondsSinceEpoch.toString();
        server = ServerConfig.fromJson(json);
      } else {
        throw Exception('Invalid config format');
      }
      
      if (server.address.isEmpty) {
        _addLog('❌ Import failed: empty address');
        notifyListeners();
        return;
      }
      
      addServer(server);
      _addLog('✅ Imported: ${server.name}');
    } catch (e) {
      _addLog('❌ Import failed: $e');
      FlutterLog.e('Provider', 'Import failed', e);
      notifyListeners();
    }
  }

  String exportConfig(ServerConfig server) => server.toShareString();

  String exportConfigJson(ServerConfig server) {
    const encoder = JsonEncoder.withIndent('  ');
    return encoder.convert(server.toJson());
  }

  // ═══════════════════════════════════════════════════════════════
  // Theme
  // ═══════════════════════════════════════════════════════════════

  void toggleTheme() {
    _isDarkMode = !_isDarkMode;
    _prefs.setBool('dark_mode', _isDarkMode);
    notifyListeners();
  }

  // ═══════════════════════════════════════════════════════════════
  // Statistics Timer
  // ═══════════════════════════════════════════════════════════════

  void _startStatsTimer() {
    _statsTimer?.cancel();
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

  // ═══════════════════════════════════════════════════════════════
  // Logging
  // ═══════════════════════════════════════════════════════════════

  void _addLog(String message) {
    final time = DateTime.now().toString().substring(11, 19);
    _logs.insert(0, '[$time] $message');
    if (_logs.length > 500) {
      _logs = _logs.sublist(0, 500);
    }
  }

  void clearLogs() {
    _logs.clear();
    _addLog('🗑️ Logs cleared');
    notifyListeners();
  }

  // ═══════════════════════════════════════════════════════════════
  // Dispose
  // ═══════════════════════════════════════════════════════════════

  @override
  void dispose() {
    _stopStatsTimer();
    _statusSub?.cancel();
    _statsSub?.cancel();
    _logSub?.cancel();
    _errorSub?.cancel();
    _sniSub?.cancel();
    _dnsSub?.cancel();
    _engine.dispose();
    super.dispose();
  }
}
