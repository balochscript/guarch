import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'package:flutter/services.dart';
import 'package:http/http.dart' as http;
import 'package:guarch/models/server_config.dart';
import 'package:guarch/models/app_settings.dart';

class FlutterLog {
  static const _logChannel = MethodChannel('com.guarch.app/logs');
  static final List<String> entries = [];

  static void d(String tag, String msg) {
    final time = DateTime.now().toString().substring(11, 23);
    entries.add('[$time] $tag: $msg');
    if (entries.length > 1000) entries.removeAt(0);
    _writeToNative('$tag: $msg');
    print('[$tag] $msg');
  }

  static void e(String tag, String msg, [Object? error]) {
    final time = DateTime.now().toString().substring(11, 23);
    final errStr = error != null ? '\n  >> $error' : '';
    entries.add('[$time] E/$tag: $msg$errStr');
    if (entries.length > 1000) entries.removeAt(0);
    _writeToNative('E/$tag: $msg$errStr');
    print('[E/$tag] $msg $errStr');
  }

  static void w(String tag, String msg) {
    final time = DateTime.now().toString().substring(11, 23);
    entries.add('[$time] W/$tag: $msg');
    if (entries.length > 1000) entries.removeAt(0);
    _writeToNative('W/$tag: $msg');
    print('[W/$tag] $msg');
  }

  static void _writeToNative(String msg) {
    try {
      _logChannel.invokeMethod('writeFlutterLog', msg);
    } catch (_) {}
  }

  static String getAll() {
    return entries.isEmpty ? 'No Flutter logs' : entries.join('\n');
  }

  static void clear() {
    entries.clear();
  }
}

class GuarchEngine {
  static const _channel = MethodChannel('com.guarch.app/engine');
  static const _eventChannel = EventChannel('com.guarch.app/events');

  static final GuarchEngine _instance = GuarchEngine._internal();
  factory GuarchEngine() => _instance;
  GuarchEngine._internal();

  final _statusController = StreamController<String>.broadcast();
  final _statsController = StreamController<Map<String, dynamic>>.broadcast();
  final _logController = StreamController<String>.broadcast();
  final _errorController = StreamController<String>.broadcast();
  final _sniController = StreamController<String>.broadcast();
  final _dnsFallbackController = StreamController<bool>.broadcast();

  Stream<String> get statusStream => _statusController.stream;
  Stream<Map<String, dynamic>> get statsStream => _statsController.stream;
  Stream<String> get logStream => _logController.stream;
  Stream<String> get errorStream => _errorController.stream;
  Stream<String> get sniStream => _sniController.stream;
  Stream<bool> get dnsFallbackStream => _dnsFallbackController.stream;

  bool _initialized = false;
  bool _nativeAvailable = true;
  StreamSubscription? _eventSubscription;

  Timer? _healthCheckTimer;
  DateTime? _lastHeartbeat;
  int _missedHeartbeats = 0;
  String _currentStatus = 'disconnected';
  bool _autoRecoveryEnabled = true;
  int _recoveryAttempts = 0;
  bool _wasConnected = false;
  Map<String, dynamic>? _lastConnectionConfig;

  bool get isNativeAvailable => _nativeAvailable;

  static Future<String> getVersion() async {
    try {
      final result = await _channel.invokeMethod<String>('getVersion');
      return result ?? 'Unknown';
    } catch (e) {
      FlutterLog.e('Engine', 'getVersion failed', e);
      return 'Unknown';
    }
  }

  Future<void> init() async {
    FlutterLog.d('Engine', 'init() called, initialized=$_initialized');
    if (_initialized) return;
    _initialized = true;

    try {
      _channel.setMethodCallHandler(_handleMethodCall);

      _eventSubscription = _eventChannel.receiveBroadcastStream().listen(
        (event) {
          _handleEvent(event);
        },
        onError: (e) {
          FlutterLog.e('Engine', 'Event channel error', e);
          _nativeAvailable = false;
        },
      );

      FlutterLog.d('Engine', 'Event channel subscribed');

      try {
        final version = await getVersion();
        FlutterLog.d('Engine', 'Native engine version: $version');
        _nativeAvailable = true;
      } catch (e) {
        FlutterLog.e('Engine', 'Native engine not available', e);
        _nativeAvailable = false;
      }
    } catch (e) {
      FlutterLog.e('Engine', 'init FAILED', e);
      _nativeAvailable = false;
    }

    FlutterLog.d('Engine', 'init() done (native: $_nativeAvailable)');
  }

  Future<dynamic> _handleMethodCall(MethodCall call) async {
    FlutterLog.d('Engine', 'Method call: ${call.method}');

    switch (call.method) {
      case 'onStatusChanged':
        final status = call.arguments as String;
        _currentStatus = status;
        if (status == 'connected') {
          _wasConnected = true;
        } else if (status == 'disconnected') {
          _wasConnected = false;
        }
        _statusController.add(status);
        break;

      case 'onStatsUpdate':
        try {
          final data = call.arguments;
          if (data is String) {
            _statsController.add(jsonDecode(data) as Map<String, dynamic>);
          } else if (data is Map) {
            _statsController.add(Map<String, dynamic>.from(data));
          }
        } catch (e) {
          FlutterLog.e('Engine', 'Stats parse error', e);
        }
        break;

      case 'onLog':
        final msg = call.arguments as String;
        _logController.add(msg);
        break;

      case 'onError':
        final error = call.arguments as String;
        _errorController.add(error);
        break;

      case 'onSNIRotation':
        final sni = call.arguments as String;
        _sniController.add(sni);
        break;

      case 'onDNSFallback':
        final enabled = call.arguments as bool;
        _dnsFallbackController.add(enabled);
        break;

      default:
        FlutterLog.w('Engine', 'Unknown method: ${call.method}');
    }
  }

  void _handleEvent(dynamic event) {
    if (event is! Map) return;

    final type = event['type'] as String?;
    final data = event['data'];

    FlutterLog.d('Engine', 'Event: $type');

    switch (type) {
      case 'status':
        if (data is String) {
          _currentStatus = data;
          if (data == 'connected') {
            _wasConnected = true;
          } else if (data == 'disconnected') {
            _wasConnected = false;
          }
          _statusController.add(data);
        }
        break;

      case 'stats':
        try {
          if (data is String) {
            _statsController.add(jsonDecode(data) as Map<String, dynamic>);
          } else if (data is Map) {
            _statsController.add(Map<String, dynamic>.from(data));
          }
        } catch (e) {
          FlutterLog.e('Engine', 'Stats event parse error', e);
        }
        break;

      case 'log':
        if (data is String) {
          _logController.add(data);
        }
        break;

      case 'error':
        if (data is String) {
          _errorController.add(data);
        }
        break;

      case 'sni':
        if (data is String) {
          _sniController.add(data);
        }
        break;

      case 'dns_fallback':
        if (data is bool) {
          _dnsFallbackController.add(data);
        }
        break;

      case 'heartbeat':
        _lastHeartbeat = DateTime.now();
        _missedHeartbeats = 0;
        FlutterLog.d('Engine', 'Heartbeat received');
        break;

      case 'vpn_service_event':
        if (data is String) {
          _handleVpnServiceEvent(data);
        }
        break;

      default:
        FlutterLog.w('Engine', 'Unknown event type: $type');
    }
  }

  void _handleVpnServiceEvent(String eventType) {
    FlutterLog.w('Engine', 'VPN service event: $eventType');

    switch (eventType) {
      case 'vpn_revoked':
        _currentStatus = 'disconnected';
        _statusController.add('disconnected');
        _errorController.add('VPN permission revoked by user');
        _stopHealthMonitoring();
        _lastConnectionConfig = null; // Clear so auto-recovery cannot run
        _wasConnected = false;
        break;

      case 'vpn_destroyed':
        _currentStatus = 'disconnected';
        _statusController.add('disconnected');
        _errorController.add('VPN service destroyed by system');
        _stopHealthMonitoring();
        _attemptAutoRecovery();
        break;

      case 'vpn_establish_failed':
        _currentStatus = 'disconnected';
        _statusController.add('disconnected');
        _errorController.add('VPN interface establishment failed');
        _stopHealthMonitoring();
        _lastConnectionConfig = null; // Clear so auto-recovery cannot run
        _wasConnected = false;
        break;

      case 'vpn_started':
        FlutterLog.d('Engine', 'VPN started successfully');
        break;

      case 'vpn_stopping':
        FlutterLog.d('Engine', 'VPN stopping...');
        _lastConnectionConfig = null; // Clear config to prevent auto-recovery on graceful notification disconnect
        _wasConnected = false;
        _currentStatus = 'disconnected';
        _statusController.add('disconnected');
        break;
    }
  }

  void _attemptAutoRecovery() {
    if (!_autoRecoveryEnabled) {
      FlutterLog.d('Engine', 'Auto-recovery disabled');
      return;
    }

    if (!_wasConnected) {
      FlutterLog.d('Engine', 'Skipping auto-recovery: VPN was not successfully established before disconnection');
      return;
    }

    if (_recoveryAttempts >= 3) {
      FlutterLog.e('Engine', 'Max recovery attempts reached');
      _errorController.add('Connection lost - manual reconnection required');
      _recoveryAttempts = 0;
      return;
    }

    if (_lastConnectionConfig == null) {
      FlutterLog.w('Engine', 'No config for recovery');
      return;
    }

    _recoveryAttempts++;
    final delay = Duration(seconds: 5 * _recoveryAttempts);
    FlutterLog.w('Engine', 'Auto-recovery attempt $_recoveryAttempts/3 in ${delay.inSeconds}s');
    _errorController.add('Connection lost, attempting recovery in ${delay.inSeconds}s...');

    Future.delayed(delay, () async {
      if (_currentStatus == 'disconnected') {
        FlutterLog.d('Engine', 'Executing auto-recovery...');
        final config = _lastConnectionConfig!;
        await connectWithConfig(
          config['configJson'] as String,
          config['vpnModeEnabled'] as bool,
          preferIPv6: config['preferIPv6'] as bool? ?? false,
          allowedApps: List<String>.from(config['allowedApps'] ?? []),
          disallowedApps: List<String>.from(config['disallowedApps'] ?? []),
        );
      }
    });
  }

  void _startHealthMonitoring() {
    _stopHealthMonitoring();
    _lastHeartbeat = DateTime.now();
    _missedHeartbeats = 0;
    _recoveryAttempts = 0;

    _healthCheckTimer = Timer.periodic(const Duration(seconds: 5), (timer) async {
      await _performHealthCheck();
    });

    FlutterLog.d('Engine', 'Health monitoring started');
  }

  void _stopHealthMonitoring() {
    _healthCheckTimer?.cancel();
    _healthCheckTimer = null;
  }

  Future<void> _performHealthCheck() async {
    try {
      if (_lastHeartbeat != null) {
        final diff = DateTime.now().difference(_lastHeartbeat!);
        if (diff.inSeconds > 15) {
          _missedHeartbeats++;
          FlutterLog.w('Engine', 'No heartbeat for ${diff.inSeconds}s (missed: $_missedHeartbeats)');

          if (_missedHeartbeats >= 3) {
            FlutterLog.e('Engine', 'Connection lost (no heartbeat)');
            _currentStatus = 'disconnected';
            _statusController.add('disconnected');
            _errorController.add('Connection lost - no heartbeat from engine');
            _stopHealthMonitoring();
            _attemptAutoRecovery();
            return;
          }
        } else {
          _missedHeartbeats = 0;
        }
      }

      final nativeStatus = await getStatus();

      if (nativeStatus != _currentStatus) {
        FlutterLog.w('Engine', 'State mismatch: native=$nativeStatus, UI=$_currentStatus');
        _currentStatus = nativeStatus;
        _statusController.add(nativeStatus);

        if (nativeStatus == 'disconnected') {
          _stopHealthMonitoring();
          _attemptAutoRecovery();
        }
      }

      if (_nativeAvailable && _currentStatus == 'connected') {
        try {
          final tunStats = await getTUNStats();
          if (tunStats.isEmpty) {
            FlutterLog.w('Engine', 'TUN stats empty, possible VPN issue');
          }
        } catch (e) {
          FlutterLog.w('Engine', 'TUN stats check failed: $e');
        }
      }

      FlutterLog.d('Engine', 'Health check OK (native: $nativeStatus, heartbeat: ${_missedHeartbeats})');

    } catch (e) {
      FlutterLog.e('Engine', 'Health check failed', e);
    }
  }

  Future<bool> connect({
    required String serverAddr,
    int serverPort = 8443,
    required String psk,
    String? certPin,
    String listenAddr = '127.0.0.1',
    int? listenPort,
    bool coverEnabled = true,
    String protocol = 'guarch',
  }) async {
    FlutterLog.d('Engine', '=== connect() (legacy) ===');
    FlutterLog.d('Engine', '  $protocol → $serverAddr:$serverPort');

    if (serverAddr.isEmpty || psk.isEmpty) {
      FlutterLog.e('Engine', '  Empty addr or psk!');
      _errorController.add('Empty server address or PSK');
      return false;
    }

    try {
      final socksPort = listenPort ?? await AppSettings.getSocksPort();
      FlutterLog.d('Engine', '  SOCKS port: $socksPort');

      final config = jsonEncode({
        'server_addr': serverAddr,
        'server_port': serverPort,
        'psk': psk,
        'cert_pin': certPin ?? '',
        'listen_addr': listenAddr,
        'listen_port': socksPort,
        'cover_enabled': coverEnabled,
        'protocol': protocol,
      });

      FlutterLog.d('Engine', '  invokeMethod("connect")...');
      final result = await _channel.invokeMethod('connect', config);
      FlutterLog.d('Engine', '  result: $result');
      
      if (result == true) {
        _startHealthMonitoring();
      }
      
      return result == true;
    } on PlatformException catch (e) {
      FlutterLog.e('Engine', '  PlatformException: ${e.code} ${e.message}', e);
      _errorController.add('Platform error: ${e.message}');
      _statusController.add('disconnected');
      return false;
    } on MissingPluginException {
      FlutterLog.e('Engine', '  MissingPlugin!');
      _nativeAvailable = false;
      _errorController.add('Native engine not available');
      _statusController.add('disconnected');
      return false;
    } catch (e) {
      FlutterLog.e('Engine', '  UNEXPECTED', e);
      _errorController.add('Connection failed: $e');
      _statusController.add('disconnected');
      return false;
    }
  }

  Future<bool> connectWithConfig(
    String configJson, 
    bool vpnModeEnabled, {
    bool preferIPv6 = false,
    List<String> allowedApps = const [],
    List<String> disallowedApps = const [],
  }) async {
    FlutterLog.d('Engine', '=== connectWithConfig() (v1.0.1) ===');
    FlutterLog.d('Engine', '  VPN Mode: $vpnModeEnabled, Prefer IPv6: $preferIPv6');

    if (!_nativeAvailable) {
      FlutterLog.e('Engine', '  Native not available');
      _errorController.add('Native engine not available');
      return false;
    }

    _lastConnectionConfig = {
      'configJson': configJson,
      'vpnModeEnabled': vpnModeEnabled,
      'preferIPv6': preferIPv6,
      'allowedApps': allowedApps,
      'disallowedApps': disallowedApps,
    };

    try {
      final settings = await AppSettings.load();

      FlutterLog.d('Engine', '  User settings: socks=${settings.socksPort}, dial=${settings.dialTimeout}s, handshake=${settings.handshakeTimeout}s');

      final config = jsonDecode(configJson) as Map<String, dynamic>;
      config['socks_port'] = settings.socksPort;
      config['vpn_mode'] = vpnModeEnabled;

      final userSettingsJson = jsonEncode({
        'socks_port': settings.socksPort,
        'dial_timeout': settings.dialTimeout,
        'handshake_timeout': settings.handshakeTimeout,
      });

      FlutterLog.d('Engine', '  Config version: ${config['version']}');
      FlutterLog.d('Engine', '  Server: ${config['server']?['address']}');
      FlutterLog.d('Engine', '  Protocol: ${config['server']?['protocol']}');

      final updatedConfigJson = jsonEncode(config);

      await _channel.invokeMethod('setUserSettings', userSettingsJson);
      FlutterLog.d('Engine', '  User settings sent to Go');

      final params = {
        'config': updatedConfigJson,
        'vpnMode': vpnModeEnabled,
        'preferIPv6': preferIPv6,
        'allowedApps': allowedApps,
        'disallowedApps': disallowedApps,
      };

      final result = await _channel.invokeMethod('connectWithConfig', params);
      FlutterLog.d('Engine', '  result: $result');

      if (result == true) {
        _startHealthMonitoring();
        _recoveryAttempts = 0;
      }

      return result == true;
    } on PlatformException catch (e) {
      FlutterLog.e('Engine', '  PlatformException: ${e.code} ${e.message}', e);
      _errorController.add('Platform error: ${e.message}');
      _statusController.add('disconnected');
      return false;
    } on MissingPluginException {
      FlutterLog.e('Engine', '  MissingPlugin!');
      _nativeAvailable = false;
      _errorController.add('Native engine not available');
      _statusController.add('disconnected');
      return false;
    } catch (e) {
      FlutterLog.e('Engine', '  Config connect FAILED', e);
      _errorController.add('Connection failed: $e');
      _statusController.add('disconnected');
      return false;
    }
  }

  Future<bool> disconnect(bool vpnModeEnabled) async {
    FlutterLog.d('Engine', 'disconnect() - VPN mode: $vpnModeEnabled');

    _stopHealthMonitoring();
    _lastConnectionConfig = null;
    _recoveryAttempts = 0;
    _wasConnected = false;

    if (!_nativeAvailable) {
      _statusController.add('disconnected');
      return true;
    }

    try {
      final result = await _channel.invokeMethod('disconnect', vpnModeEnabled);
      return result == true;
    } on MissingPluginException {
      _statusController.add('disconnected');
      return true;
    } catch (e) {
      FlutterLog.e('Engine', 'disconnect error', e);
      return false;
    }
  }

  Future<String> getStatus() async {
    if (!_nativeAvailable) return 'disconnected';

    try {
      return await _channel.invokeMethod<String>('getStatus') ?? 'disconnected';
    } catch (_) {
      return 'disconnected';
    }
  }

  Future<Map<String, dynamic>> getStats() async {
    if (!_nativeAvailable) return {};

    try {
      final r = await _channel.invokeMethod('getStats');
      if (r is String) return jsonDecode(r) as Map<String, dynamic>;
      if (r is Map) return Map<String, dynamic>.from(r);
      return {};
    } catch (_) {
      return {};
    }
  }

  Future<bool> setBatteryLevel(int level) async {
    if (!_nativeAvailable) return false;

    try {
      final result = await _channel.invokeMethod('setBatteryLevel', level);
      FlutterLog.d('Engine', 'Battery level set: $level%');
      return result == true;
    } catch (e) {
      FlutterLog.e('Engine', 'setBatteryLevel failed', e);
      return false;
    }
  }

  Future<bool> setDataSaverMode(bool enabled) async {
    if (!_nativeAvailable) return false;

    try {
      final result = await _channel.invokeMethod('setDataSaverMode', enabled);
      FlutterLog.d('Engine', 'Data saver mode: $enabled');
      return result == true;
    } catch (e) {
      FlutterLog.e('Engine', 'setDataSaverMode failed', e);
      return false;
    }
  }

  Future<bool> loadConfigJSON(String jsonStr) async {
    if (!_nativeAvailable) return false;

    try {
      final result = await _channel.invokeMethod('loadConfigJSON', jsonStr);
      FlutterLog.d('Engine', 'Config loaded from JSON');
      return result == true;
    } catch (e) {
      FlutterLog.e('Engine', 'loadConfigJSON failed', e);
      return false;
    }
  }

  Future<bool> loadConfigURI(String uri) async {
    if (!_nativeAvailable) return false;

    try {
      final result = await _channel.invokeMethod('loadConfigURI', uri);
      FlutterLog.d('Engine', 'Config loaded from URI');
      return result == true;
    } catch (e) {
      FlutterLog.e('Engine', 'loadConfigURI failed', e);
      return false;
    }
  }

  Future<bool> loadPreset(String presetName) async {
    if (!_nativeAvailable) return false;

    try {
      final result = await _channel.invokeMethod('loadPreset', presetName);
      FlutterLog.d('Engine', 'Preset loaded: $presetName');
      return result == true;
    } catch (e) {
      FlutterLog.e('Engine', 'loadPreset failed', e);
      return false;
    }
  }

  Future<String?> exportConfigURI() async {
    if (!_nativeAvailable) return null;

    try {
      return await _channel.invokeMethod<String>('exportConfigURI');
    } catch (e) {
      FlutterLog.e('Engine', 'exportConfigURI failed', e);
      return null;
    }
  }

  Future<String?> exportConfigJSON() async {
    if (!_nativeAvailable) return null;

    try {
      return await _channel.invokeMethod<String>('exportConfigJSON');
    } catch (e) {
      FlutterLog.e('Engine', 'exportConfigJSON failed', e);
      return null;
    }
  }

  Future<bool> setSplitTunnelMode(String mode) async {
    if (!_nativeAvailable) return false;

    try {
      final result = await _channel.invokeMethod('setSplitTunnelMode', mode);
      FlutterLog.d('Engine', 'Split tunnel mode: $mode');
      return result == true;
    } catch (e) {
      FlutterLog.e('Engine', 'setSplitTunnelMode failed', e);
      return false;
    }
  }

  Future<bool> addSplitTunnelDomain(String domain, bool isWhitelist) async {
    if (!_nativeAvailable) return false;

    try {
      final result = await _channel.invokeMethod('addSplitTunnelDomain', {
        'domain': domain,
        'isWhitelist': isWhitelist,
      });
      FlutterLog.d('Engine', 'Added domain to split tunnel: $domain');
      return result == true;
    } catch (e) {
      FlutterLog.e('Engine', 'addSplitTunnelDomain failed', e);
      return false;
    }
  }

  Future<int> ping(String address, int port) async {
    FlutterLog.d('Engine', 'TCP ping $address:$port');

    try {
      final addrs = await InternetAddress.lookup(address).timeout(
        const Duration(seconds: 5),
      );

      if (addrs.isEmpty) {
        FlutterLog.w('Engine', 'DNS lookup failed');
        return -1;
      }

      final sw = Stopwatch()..start();
      final socket = await Socket.connect(
        addrs.first.address,
        port,
        timeout: const Duration(seconds: 5),
      );
      sw.stop();

      socket.destroy();

      final ms = sw.elapsedMilliseconds;
      FlutterLog.d('Engine', 'TCP ping: ${ms}ms');
      return ms;
    } catch (e) {
      FlutterLog.e('Engine', 'TCP ping failed', e);
      return -1;
    }
  }

  Future<Map<String, dynamic>?> pingWithPolicy(String address, int port, String psk) async {
    FlutterLog.d('Engine', 'PingWithPolicy $address:$port');

    try {
      final result = await _channel.invokeMethod('pingWithPolicy', {
        'address': '$address:$port',
        'psk': psk,
      });

      if (result == null || result.isEmpty) {
        FlutterLog.w('Engine', 'PingWithPolicy returned empty');
        return null;
      }

      final decoded = jsonDecode(result) as Map<String, dynamic>;
      FlutterLog.d('Engine', 'PingWithPolicy result: $decoded');
      return decoded;
    } catch (e) {
      FlutterLog.e('Engine', 'PingWithPolicy failed', e);
      return null;
    }
  }

  Future<int> testRealDelayViaHTTP(ServerConfig server) async {
    FlutterLog.d('Engine', '=== testRealDelayViaHTTP ===');

    final startTime = DateTime.now();

    try {
      final response = await http.get(
        Uri.parse('https://www.google.com/gen_204'),
      ).timeout(const Duration(seconds: 5));

      if (response.statusCode == 204) {
        final delay = DateTime.now().difference(startTime).inMilliseconds;
        FlutterLog.d('Engine', 'HTTP 204 OK: ${delay}ms');
        return delay;
      } else {
        FlutterLog.w('Engine', 'Unexpected status: ${response.statusCode}');
        return -1;
      }
    } on TimeoutException {
      FlutterLog.w('Engine', 'HTTP timeout');
      return -1;
    } catch (e) {
      FlutterLog.e('Engine', 'HTTP test failed', e);
      return -1;
    }
  }

  Future<int> testRealDelay(ServerConfig server) async {
    FlutterLog.d('Engine', '=== testRealDelay ${server.address}:${server.port} ===');

    if (!_nativeAvailable) {
      FlutterLog.w('Engine', 'Native not available');
      return -1;
    }

    final startTime = DateTime.now();

    try {
      final socksPort = await AppSettings.getSocksPort();

      final testConfig = {
        'version': 1,
        'server': {
          'name': server.name,
          'address': '${server.address}:${server.port}',
          'protocol': server.protocol,
          'psk': server.psk,
          'cert_pin': server.certPin ?? '',
        },
        'socks_port': socksPort,
        'sni': {
          'enabled': false,
        },
        'cover': {
          'enabled': false,
          'mode': 'balanced',
        },
        'dns': {
          'enabled': false,
        },
      };

      final configJson = jsonEncode(testConfig);

      FlutterLog.d('Engine', 'Calling testRealDelay with minimal config...');

      final result = await _channel.invokeMethod(
        'testRealDelay',
        configJson,
      ).timeout(
        const Duration(seconds: 10),
        onTimeout: () {
          FlutterLog.w('Engine', 'Real delay test timeout');
          return false;
        },
      );

      if (result == true) {
        final delay = DateTime.now().difference(startTime).inMilliseconds;
        FlutterLog.d('Engine', 'Real delay: ${delay}ms');
        return delay;
      } else {
        FlutterLog.w('Engine', 'Real delay test failed (handshake failed)');
        return -1;
      }
    } on PlatformException catch (e) {
      FlutterLog.e('Engine', 'Real delay test PlatformException: ${e.message}', e);
      return -1;
    } catch (e) {
      FlutterLog.e('Engine', 'Real delay test error', e);
      return -1;
    }
  }

  Future<bool> testConnection(String address, int port, String psk) async {
    FlutterLog.d('Engine', 'testConnection $address:$port');

    if (!_nativeAvailable) return false;

    try {
      final socksPort = await AppSettings.getSocksPort();

      final testConfig = {
        'version': 1,
        'server': {
          'name': 'Test',
          'address': '$address:$port',
          'protocol': 'guarch',
          'psk': psk,
        },
        'socks_port': socksPort,
        'sni': {'enabled': false},
        'cover': {
          'enabled': false,
          'mode': 'balanced',
        },
        'dns': {'enabled': false},
      };

      final result = await _channel.invokeMethod(
        'testConnection',
        jsonEncode(testConfig),
      ).timeout(
        const Duration(seconds: 8),
        onTimeout: () => false,
      );

      return result == true;
    } catch (e) {
      FlutterLog.e('Engine', 'testConnection failed', e);
      return false;
    }
  }

  Future<Map<String, dynamic>> getTUNStats() async {
    if (!_nativeAvailable) return {};

    try {
      final r = await _channel.invokeMethod('getTUNStats');
      if (r is String) return jsonDecode(r) as Map<String, dynamic>;
      if (r is Map) return Map<String, dynamic>.from(r);
      return {};
    } catch (e) {
      FlutterLog.e('Engine', 'getTUNStats failed', e);
      return {};
    }
  }

  Future<String> readGoLog() async {
    try {
      return await _channel.invokeMethod<String>('readGoLog') ?? 'No Go log';
    } catch (_) {
      return 'No Go log';
    }
  }

  Future<void> clearGoLog() async {
    try {
      await _channel.invokeMethod('clearGoLog');
    } catch (_) {}
  }

  Future<List<Map<String, String>>> getInstalledApps() async {
    if (!_nativeAvailable) return [];
    try {
      final List? result = await _channel.invokeMethod('getInstalledApps');
      if (result == null) return [];
      return result.map((item) {
        final map = Map<String, dynamic>.from(item as Map);
        return {
          'name': map['name'] as String,
          'packageName': map['packageName'] as String,
        };
      }).toList();
    } catch (e) {
      FlutterLog.e('Engine', 'getInstalledApps failed', e);
      return [];
    }
  }

  void setAutoRecovery(bool enabled) {
    _autoRecoveryEnabled = enabled;
    FlutterLog.d('Engine', 'Auto-recovery: $enabled');
  }

  void dispose() {
    FlutterLog.d('Engine', 'dispose()');
    _stopHealthMonitoring();
    _eventSubscription?.cancel();
    _statusController.close();
    _statsController.close();
    _logController.close();
    _errorController.close();
    _sniController.close();
    _dnsFallbackController.close();
  }
}
