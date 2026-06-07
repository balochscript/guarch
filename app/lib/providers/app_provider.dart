import 'dart:async';
import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:guarch/models/server_config.dart';
import 'package:guarch/models/connection_state.dart';
import 'package:guarch/models/app_settings.dart';
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

  int _batteryLevel = 100;
  bool _dataSaverEnabled = false;
  bool _debugModeEnabled = false;
  bool _vpnModeEnabled = true;

  int _connectionTimeout = 15;
  int _maxRetryAttempts = 3;
  int _retryDelay = 5;
  int _handshakeTimeout = 30;
  int _keepAliveInterval = 30;
  int _bufferSize = 32768;

  AppSettings? _appSettings;
  
  bool get debugModeEnabled => _debugModeEnabled;
  bool get vpnModeEnabled => _vpnModeEnabled;
  
  bool get globalSniEnabled => _appSettings?.globalSniEnabled ?? false;
  String get globalSniMode => _appSettings?.globalSniMode ?? 'weighted';
  int get globalSniRotationMinutes => _appSettings?.globalSniRotationMinutes ?? 5;
  List<SNIDomain> get globalSniDomains => _appSettings?.globalSniDomains ?? [];
  
  bool get globalCoverEnabled => _appSettings?.globalCoverEnabled ?? false;
  String get globalCoverMode => _appSettings?.globalCoverMode ?? 'balanced';
  bool get globalBatteryAware => _appSettings?.globalBatteryAware ?? true;
  bool get globalDataSaver => _appSettings?.globalDataSaver ?? false;
  List<CoverDomain> get globalCoverDomains => _appSettings?.globalCoverDomains ?? [];
  
  bool get globalDnsEnabled => _appSettings?.globalDnsEnabled ?? false;
  String get globalDnsDomain => _appSettings?.globalDnsDomain ?? 'tunnel.example.com';
  List<String> get globalDnsServers => _appSettings?.globalDnsServers ?? ['8.8.8.8:53', '1.1.1.1:53'];
  int get globalDnsSwitchThreshold => _appSettings?.globalDnsSwitchThreshold ?? 3;
  
  bool get globalUtlsEnabled => _appSettings?.globalUtlsEnabled ?? true;
  String get globalUtlsFingerprint => _appSettings?.globalUtlsFingerprint ?? 'chrome_auto';
  
  bool get globalFragmentEnabled => _appSettings?.globalFragmentEnabled ?? false;
  int get globalFragmentMinSize => _appSettings?.globalFragmentMinSize ?? 64;
  int get globalFragmentMaxSize => _appSettings?.globalFragmentMaxSize ?? 256;

  int get connectionTimeout => _connectionTimeout;
  int get maxRetryAttempts => _maxRetryAttempts;
  int get retryDelay => _retryDelay;
  int get handshakeTimeout => _handshakeTimeout;
  int get keepAliveInterval => _keepAliveInterval;
  int get bufferSize => _bufferSize;

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

  Future<void> init() async {
    FlutterLog.d('Provider', '=== init START (v1.0.1) ===');
    
    try {
      _prefs = await SharedPreferences.getInstance();
      _isDarkMode = _prefs.getBool('dark_mode') ?? true;
      _activeServerId = _prefs.getString('active_server');
      _batteryLevel = _prefs.getInt('battery_level') ?? 100;
      _dataSaverEnabled = _prefs.getBool('data_saver') ?? false;
      _vpnModeEnabled = _prefs.getBool('vpn_mode_enabled') ?? true;
      
      _debugModeEnabled = _prefs.getBool('debug_mode') ?? false;
      _connectionTimeout = _prefs.getInt('connection_timeout') ?? 15;
      _maxRetryAttempts = _prefs.getInt('max_retry_attempts') ?? 3;
      _retryDelay = _prefs.getInt('retry_delay') ?? 5;
      _handshakeTimeout = _prefs.getInt('handshake_timeout') ?? 30;
      _keepAliveInterval = _prefs.getInt('keep_alive_interval') ?? 30;
      _bufferSize = _prefs.getInt('buffer_size') ?? 32768;
      
      try {
        _appSettings = await AppSettings.load();
      } catch (e) {
        FlutterLog.e('Provider', 'Failed to load app settings', e);
        _appSettings = AppSettings.defaults();
        await _appSettings!.save();
      }
      
      _loadServers();
      FlutterLog.d('Provider', 'Prefs loaded. servers=${_servers.length} active=$_activeServerId vpnMode=$_vpnModeEnabled');
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

    _statsSub = _engine.statsStream.listen((data) {
      _stats = ConnectionStats.fromJson(data);
      
      if (_connectTime != null && _status == VpnStatus.connected) {
        _stats = _stats.copyWith(
          duration: DateTime.now().difference(_connectTime!),
        );
      }
      
      notifyListeners();
    });

    _logSub = _engine.logStream.listen((msg) {
      FlutterLog.d('EngineLog', msg);
      _addLog(msg);
      notifyListeners();
    });

    _errorSub = _engine.errorStream.listen((error) {
      FlutterLog.e('Engine', 'Error event: $error');
      _addLog('⚠️ Error: $error');
      
      if (_status == VpnStatus.connecting) {
        _status = VpnStatus.error;
        notifyListeners();
      }
    });

    _sniSub = _engine.sniStream.listen((newSNI) {
      FlutterLog.d('Provider', 'SNI rotated: $newSNI');
      _addLog('🔄 SNI: $newSNI');
      _stats = _stats.copyWith(
        currentSNI: newSNI,
        sniSwitches: _stats.sniSwitches + 1,
      );
      notifyListeners();
    });

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

    if (_servers.isNotEmpty) {
      _addLog('📡 Auto-pinging ${_servers.length} servers...');
      notifyListeners();
      pingAllServers();
    }

    FlutterLog.d('Provider', '=== init DONE ===');
  }

  Future<void> toggleVpnMode() async {
    _vpnModeEnabled = !_vpnModeEnabled;
    await _prefs.setBool('vpn_mode_enabled', _vpnModeEnabled);
    
    final mode = _vpnModeEnabled ? 'VPN' : 'Proxy';
    _addLog('🔄 Switched to $mode mode');
    
    notifyListeners();
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
    _addLog('➕ Server added: ${server.name} (${server.fullAddress})');
    notifyListeners();
    
    pingServer(server).then((ping) {
      final index = _servers.indexWhere((s) => s.id == server.id);
      if (index >= 0) {
        _servers[index] = _servers[index].copyWith(
          ping: ping,
          lastTested: DateTime.now(),
        );
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

    var server = activeServer!;
    
    if (_appSettings != null) {
      server = _appSettings!.applyToServer(server);
    }
    
    final mode = _vpnModeEnabled ? 'VPN' : 'Proxy';
    FlutterLog.d('Provider', '  Mode: $mode, ${server.protocol} → ${server.fullAddress}');

    if (server.psk.isEmpty) {
      _addLog('❌ Error: PSK is required');
      _status = VpnStatus.error;
      notifyListeners();
      await Future.delayed(const Duration(seconds: 2));
      _status = VpnStatus.disconnected;
      notifyListeners();
      return;
    }

    _status = VpnStatus.connecting;
    _addLog('🏹 Connecting to ${server.name} ($mode mode)...');
    _addLog('📡 Protocol: ${server.protocolLabel}');
    
    if (!_vpnModeEnabled) {
      _addLog('🔌 Proxy Mode: SOCKS5 on 127.0.0.1:7070');
    }
    
    if (_debugModeEnabled) {
      _addLog('🐛 Debug: Engine timeout=${_connectionTimeout}s, Retries=$_maxRetryAttempts');
    }
    
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
      final configJson = server.toJson();
      
      final success = await _engine.connectWithConfig(
        jsonEncode(configJson),
        _vpnModeEnabled,
      ).timeout(
        Duration(seconds: _connectionTimeout),
        onTimeout: () {
          FlutterLog.w('Provider', '  Connect timeout');
          _addLog('⏱️ Connection timeout (${_connectionTimeout}s)');
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
      }
    } catch (e) {
      FlutterLog.e('Provider', '  Connect FAILED', e);
      _addLog('❌ Error: $e');
      _status = VpnStatus.error;
    }

    notifyListeners();
  }

  Future<void> disconnect() async {
    FlutterLog.d('Provider', '=== disconnect() ===');
    
    if (_status == VpnStatus.disconnected) {
      return;
    }

    _status = VpnStatus.disconnecting;
    _addLog('🔌 Disconnecting...');
    notifyListeners();

    try {
      await _engine.disconnect(_vpnModeEnabled).timeout(
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

  Future<void> toggleDebugMode() async {
    _debugModeEnabled = !_debugModeEnabled;
    await _prefs.setBool('debug_mode', _debugModeEnabled);
    _addLog('🐛 Debug mode: ${_debugModeEnabled ? "ON" : "OFF"}');
    notifyListeners();
  }

  Future<void> setConnectionTimeout(int seconds) async {
    _connectionTimeout = seconds.clamp(5, 60);
    await _prefs.setInt('connection_timeout', _connectionTimeout);
    _addLog('⏱️ Connection timeout: ${_connectionTimeout}s');
    notifyListeners();
  }

  Future<void> setMaxRetryAttempts(int attempts) async {
    _maxRetryAttempts = attempts.clamp(1, 10);
    await _prefs.setInt('max_retry_attempts', _maxRetryAttempts);
    _addLog('🔄 Max retries: $_maxRetryAttempts');
    notifyListeners();
  }

  Future<void> setRetryDelay(int seconds) async {
    _retryDelay = seconds.clamp(1, 30);
    await _prefs.setInt('retry_delay', _retryDelay);
    _addLog('⏱️ Retry delay: ${_retryDelay}s');
    notifyListeners();
  }

  Future<void> setHandshakeTimeout(int seconds) async {
    _handshakeTimeout = seconds.clamp(10, 120);
    await _prefs.setInt('handshake_timeout', _handshakeTimeout);
    _addLog('🤝 Handshake timeout: ${_handshakeTimeout}s');
    notifyListeners();
  }

  Future<void> setKeepAliveInterval(int seconds) async {
    _keepAliveInterval = seconds.clamp(10, 300);
    await _prefs.setInt('keep_alive_interval', _keepAliveInterval);
    _addLog('💓 Keep-alive interval: ${_keepAliveInterval}s');
    notifyListeners();
  }

  Future<void> setBufferSize(int bytes) async {
    _bufferSize = [16384, 32768, 65536, 131072].contains(bytes) ? bytes : 32768;
    await _prefs.setInt('buffer_size', _bufferSize);
    _addLog('📦 Buffer size: ${_bufferSize ~/ 1024} KB');
    notifyListeners();
  }

  Future<void> toggleGlobalSni() async {
    if (_appSettings == null) return;
    
    _appSettings = _appSettings!.copyWith(
      globalSniEnabled: !_appSettings!.globalSniEnabled,
    );
    await _appSettings!.save();
    _addLog('🛡️ Global SNI: ${_appSettings!.globalSniEnabled ? "ON" : "OFF"}');
    notifyListeners();
  }

  Future<void> setGlobalSniMode(String mode) async {
    if (_appSettings == null) return;
    
    _appSettings = _appSettings!.copyWith(globalSniMode: mode);
    await _appSettings!.save();
    _addLog('🔄 SNI mode: $mode');
    notifyListeners();
  }

  Future<void> setGlobalSniRotationMinutes(int minutes) async {
    if (_appSettings == null) return;
    
    _appSettings = _appSettings!.copyWith(globalSniRotationMinutes: minutes);
    await _appSettings!.save();
    _addLog('⏱️ SNI rotation: ${minutes}m');
    notifyListeners();
  }

  Future<void> setGlobalSniDomains(List<SNIDomain> domains) async {
    if (_appSettings == null) return;
    
    _appSettings = _appSettings!.copyWith(globalSniDomains: domains);
    await _appSettings!.save();
    _addLog('📋 SNI domains updated (${domains.length} domains)');
    notifyListeners();
  }

  Future<void> addGlobalSniDomain(SNIDomain domain) async {
    if (_appSettings == null) return;
    
    final newDomains = List<SNIDomain>.from(_appSettings!.globalSniDomains)..add(domain);
    await setGlobalSniDomains(newDomains);
  }

  Future<void> updateGlobalSniDomain(int index, SNIDomain domain) async {
    if (_appSettings == null) return;
    
    final newDomains = List<SNIDomain>.from(_appSettings!.globalSniDomains);
    if (index >= 0 && index < newDomains.length) {
      newDomains[index] = domain;
      await setGlobalSniDomains(newDomains);
    }
  }

  Future<void> removeGlobalSniDomain(int index) async {
    if (_appSettings == null) return;
    
    final newDomains = List<SNIDomain>.from(_appSettings!.globalSniDomains);
    if (index >= 0 && index < newDomains.length) {
      newDomains.removeAt(index);
      await setGlobalSniDomains(newDomains);
    }
  }

  Future<void> resetSniToDefaults() async {
    if (_appSettings == null) return;
    
    _appSettings = _appSettings!.copyWith(
      globalSniMode: 'weighted',
      globalSniRotationMinutes: 5,
      globalSniDomains: [
        SNIDomain(domain: 'google.com', weight: 20, checkHealth: true),
        SNIDomain(domain: 'cloudflare.com', weight: 20, checkHealth: true),
        SNIDomain(domain: 'microsoft.com', weight: 15, checkHealth: true),
        SNIDomain(domain: 'apple.com', weight: 10, fallback: true),
      ],
    );
    await _appSettings!.save();
    _addLog('↺ SNI reset to defaults');
    notifyListeners();
  }

  Future<void> toggleGlobalCover() async {
    if (_appSettings == null) return;
    
    _appSettings = _appSettings!.copyWith(
      globalCoverEnabled: !_appSettings!.globalCoverEnabled,
    );
    await _appSettings!.save();
    _addLog('🎭 Global Cover: ${_appSettings!.globalCoverEnabled ? "ON" : "OFF"}');
    notifyListeners();
  }

  Future<void> setGlobalCoverMode(String mode) async {
    if (_appSettings == null) return;
    
    _appSettings = _appSettings!.copyWith(globalCoverMode: mode);
    await _appSettings!.save();
    _addLog('🎭 Cover mode: $mode');
    notifyListeners();
  }

  Future<void> toggleGlobalBatteryAware() async {
    if (_appSettings == null) return;
    
    _appSettings = _appSettings!.copyWith(
      globalBatteryAware: !_appSettings!.globalBatteryAware,
    );
    await _appSettings!.save();
    _addLog('🔋 Battery aware: ${_appSettings!.globalBatteryAware ? "ON" : "OFF"}');
    notifyListeners();
  }

  Future<void> setGlobalCoverDomains(List<CoverDomain> domains) async {
    if (_appSettings == null) return;
    
    _appSettings = _appSettings!.copyWith(globalCoverDomains: domains);
    await _appSettings!.save();
    _addLog('📋 Cover domains updated (${domains.length} domains)');
    notifyListeners();
  }

  Future<void> addGlobalCoverDomain(CoverDomain domain) async {
    if (_appSettings == null) return;
    
    final newDomains = List<CoverDomain>.from(_appSettings!.globalCoverDomains)..add(domain);
    await setGlobalCoverDomains(newDomains);
  }

  Future<void> updateGlobalCoverDomain(int index, CoverDomain domain) async {
    if (_appSettings == null) return;
    
    final newDomains = List<CoverDomain>.from(_appSettings!.globalCoverDomains);
    if (index >= 0 && index < newDomains.length) {
      newDomains[index] = domain;
      await setGlobalCoverDomains(newDomains);
    }
  }

  Future<void> removeGlobalCoverDomain(int index) async {
    if (_appSettings == null) return;
    
    final newDomains = List<CoverDomain>.from(_appSettings!.globalCoverDomains);
    if (index >= 0 && index < newDomains.length) {
      newDomains.removeAt(index);
      await setGlobalCoverDomains(newDomains);
    }
  }

  Future<void> resetCoverToDefaults() async {
    if (_appSettings == null) return;
    
    _appSettings = _appSettings!.copyWith(
      globalCoverMode: 'balanced',
      globalBatteryAware: true,
      globalCoverDomains: [
        CoverDomain(domain: 'www.google.com', weight: 30, paths: ['/', '/search']),
        CoverDomain(domain: 'www.microsoft.com', weight: 20, paths: ['/', '/windows']),
        CoverDomain(domain: 'github.com', weight: 15, paths: ['/', '/explore']),
      ],
    );
    await _appSettings!.save();
    _addLog('↺ Cover reset to defaults');
    notifyListeners();
  }

  Future<void> toggleGlobalDns() async {
    if (_appSettings == null) return;
    
    _appSettings = _appSettings!.copyWith(
      globalDnsEnabled: !_appSettings!.globalDnsEnabled,
    );
    await _appSettings!.save();
    _addLog('🌐 Global DNS: ${_appSettings!.globalDnsEnabled ? "ON" : "OFF"}');
    notifyListeners();
  }

  Future<void> setGlobalDnsDomain(String domain) async {
    if (_appSettings == null) return;
    
    _appSettings = _appSettings!.copyWith(globalDnsDomain: domain);
    await _appSettings!.save();
    _addLog('🌐 DNS domain: $domain');
    notifyListeners();
  }

  Future<void> setGlobalDnsServers(List<String> servers) async {
    if (_appSettings == null) return;
    
    _appSettings = _appSettings!.copyWith(globalDnsServers: servers);
    await _appSettings!.save();
    _addLog('📋 DNS servers updated (${servers.length} servers)');
    notifyListeners();
  }

  Future<void> addGlobalDnsServer(String server) async {
    if (_appSettings == null) return;
    
    final newServers = List<String>.from(_appSettings!.globalDnsServers)..add(server);
    await setGlobalDnsServers(newServers);
  }

  Future<void> removeGlobalDnsServer(int index) async {
    if (_appSettings == null) return;
    
    final newServers = List<String>.from(_appSettings!.globalDnsServers);
    if (index >= 0 && index < newServers.length) {
      newServers.removeAt(index);
      await setGlobalDnsServers(newServers);
    }
  }

  Future<void> setGlobalDnsSwitchThreshold(int threshold) async {
    if (_appSettings == null) return;
    
    _appSettings = _appSettings!.copyWith(globalDnsSwitchThreshold: threshold);
    await _appSettings!.save();
    _addLog('🔄 DNS switch threshold: $threshold');
    notifyListeners();
  }

  Future<void> resetDnsToDefaults() async {
    if (_appSettings == null) return;
    
    _appSettings = _appSettings!.copyWith(
      globalDnsDomain: 'tunnel.example.com',
      globalDnsServers: ['8.8.8.8:53', '1.1.1.1:53'],
      globalDnsSwitchThreshold: 3,
    );
    await _appSettings!.save();
    _addLog('↺ DNS reset to defaults');
    notifyListeners();
  }

  Future<void> toggleGlobalUtls() async {
    if (_appSettings == null) return;
    
    _appSettings = _appSettings!.copyWith(
      globalUtlsEnabled: !_appSettings!.globalUtlsEnabled,
    );
    await _appSettings!.save();
    _addLog('🔐 Global UTLS: ${_appSettings!.globalUtlsEnabled ? "ON" : "OFF"}');
    notifyListeners();
  }

  Future<void> setGlobalUtlsFingerprint(String fingerprint) async {
    if (_appSettings == null) return;
    
    _appSettings = _appSettings!.copyWith(globalUtlsFingerprint: fingerprint);
    await _appSettings!.save();
    _addLog('🔐 UTLS fingerprint: $fingerprint');
    notifyListeners();
  }

  Future<void> toggleGlobalFragment() async {
    if (_appSettings == null) return;
    
    _appSettings = _appSettings!.copyWith(
      globalFragmentEnabled: !_appSettings!.globalFragmentEnabled,
    );
    await _appSettings!.save();
    _addLog('✂️ Global Fragment: ${_appSettings!.globalFragmentEnabled ? "ON" : "OFF"}');
    notifyListeners();
  }

  Future<void> setGlobalFragmentSizes(int minSize, int maxSize) async {
    if (_appSettings == null) return;
    
    _appSettings = _appSettings!.copyWith(
      globalFragmentMinSize: minSize,
      globalFragmentMaxSize: maxSize,
    );
    await _appSettings!.save();
    _addLog('✂️ Fragment sizes: $minSize-$maxSize bytes');
    notifyListeners();
  }

  Future<void> resetSecurityToDefaults() async {
    if (_appSettings == null) return;
    
    _appSettings = _appSettings!.copyWith(
      globalUtlsEnabled: true,
      globalUtlsFingerprint: 'chrome_auto',
      globalFragmentEnabled: false,
      globalFragmentMinSize: 64,
      globalFragmentMaxSize: 256,
    );
    await _appSettings!.save();
    _addLog('↺ Security reset to defaults');
    notifyListeners();
  }

  Future<int> pingServer(ServerConfig server) async {
    _addLog('📡 TCP Ping: ${server.name}...');
    notifyListeners();
    
    final ping = await _engine.ping(server.address, server.port);
    
    if (ping > 0) {
      _addLog('${server.pingEmoji} ${server.name}: ${ping}ms (TCP)');
    } else {
      _addLog('🔴 ${server.name}: unreachable');
    }
    
    notifyListeners();
    return ping;
  }

  Future<int> testRealDelay(ServerConfig server) async {
    _addLog('⏱️ Real Delay Test: ${server.name}...');
    notifyListeners();
    
    final delay = await _engine.testRealDelay(server);
    
    if (delay > 0) {
      _addLog('${server.realDelayEmoji} ${server.name}: ${delay}ms (real)');
    } else {
      _addLog('🔴 ${server.name}: handshake failed');
    }
    
    notifyListeners();
    return delay;
  }

  Future<void> pingAllServers({bool includeRealDelay = false}) async {
    final testType = includeRealDelay ? 'TCP + Real Delay' : 'TCP Ping';
    _addLog('📡 Testing ${_servers.length} servers ($testType)...');
    notifyListeners();
    
    for (var i = 0; i < _servers.length; i++) {
      final ping = await pingServer(_servers[i]);
      
      int? realDelay;
      if (includeRealDelay && ping > 0) {
        realDelay = await testRealDelay(_servers[i]);
      }
      
      _servers[i] = _servers[i].copyWith(
        ping: ping,
        realDelay: realDelay,
        lastTested: DateTime.now(),
      );
      notifyListeners();
      
      if (i < _servers.length - 1) {
        await Future.delayed(const Duration(milliseconds: 500));
      }
    }
    
    _saveServers();
    _addLog('✅ Test complete! (${_servers.where((s) => (s.ping ?? 0) > 0).length}/${_servers.length} reachable)');
    notifyListeners();
  }

  Future<void> testServer(ServerConfig server) async {
    _addLog('🔍 Full test: ${server.name}...');
    notifyListeners();
    
    final ping = await pingServer(server);
    
    int? realDelay;
    if (ping > 0) {
      realDelay = await testRealDelay(server);
    }
    
    final index = _servers.indexWhere((s) => s.id == server.id);
    if (index >= 0) {
      _servers[index] = _servers[index].copyWith(
        ping: ping,
        realDelay: realDelay,
        lastTested: DateTime.now(),
      );
      _saveServers();
      notifyListeners();
    }
    
    if (ping > 0 && realDelay != null && realDelay > 0) {
      _addLog('✅ ${server.name}: TCP=${ping}ms, Real=${realDelay}ms');
    } else {
      _addLog('❌ ${server.name}: Test failed');
    }
  }

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

  void toggleTheme() {
    _isDarkMode = !_isDarkMode;
    _prefs.setBool('dark_mode', _isDarkMode);
    notifyListeners();
  }

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
