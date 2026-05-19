import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'package:flutter/services.dart';
import 'package:guarch/models/server_config.dart';

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

      default:
        FlutterLog.w('Engine', 'Unknown event type: $type');
    }
  }

  Future<bool> connect({
    required String serverAddr,
    int serverPort = 8443,
    required String psk,
    String? certPin,
    String listenAddr = '127.0.0.1',
    int listenPort = 1080,
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
      final config = jsonEncode({
        'server_addr': serverAddr,
        'server_port': serverPort,
        'psk': psk,
        'cert_pin': certPin ?? '',
        'listen_addr': listenAddr,
        'listen_port': listenPort,
        'cover_enabled': coverEnabled,
        'protocol': protocol,
      });

      FlutterLog.d('Engine', '  invokeMethod("connect")...');
      final result = await _channel.invokeMethod('connect', config);
      FlutterLog.d('Engine', '  result: $result');
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

  Future<bool> connectWithConfig(String configJson) async {
    FlutterLog.d('Engine', '=== connectWithConfig() (v1.0.1) ===');

    if (!_nativeAvailable) {
      FlutterLog.e('Engine', '  Native not available');
      _errorController.add('Native engine not available');
      return false;
    }

    try {
      final config = jsonDecode(configJson) as Map<String, dynamic>;
      FlutterLog.d('Engine', '  Config version: ${config['version']}');
      FlutterLog.d('Engine', '  Server: ${config['server']?['address']}');
      FlutterLog.d('Engine', '  Protocol: ${config['server']?['protocol']}');

      final result = await _channel.invokeMethod('connectWithConfig', configJson);
      FlutterLog.d('Engine', '  result: $result');
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

  Future<bool> disconnect() async {
    FlutterLog.d('Engine', 'disconnect()');

    if (!_nativeAvailable) {
      _statusController.add('disconnected');
      return true;
    }

    try {
      final result = await _channel.invokeMethod('disconnect');
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

  Future<int> testRealDelay(ServerConfig server) async {
    FlutterLog.d('Engine', '=== testRealDelay ${server.address}:${server.port} ===');
    
    if (!_nativeAvailable) {
      FlutterLog.w('Engine', 'Native not available');
      return -1;
    }
    
    final startTime = DateTime.now();
    
    try {
      final testConfig = {
        'version': 1,
        'server': {
          'name': server.name,
          'address': '${server.address}:${server.port}',
          'protocol': server.protocol,
          'psk': server.psk,
          'cert_pin': server.certPin ?? '',
        },
        'socks_port': 1080,
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
        FlutterLog.d('Engine', 'Real delay: ${delay}ms ✅');
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
      final testConfig = {
        'version': 1,
        'server': {
          'name': 'Test',
          'address': '$address:$port',
          'protocol': 'guarch',
          'psk': psk,
        },
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

  void dispose() {
    FlutterLog.d('Engine', 'dispose()');
    _eventSubscription?.cancel();
    _statusController.close();
    _statsController.close();
    _logController.close();
    _errorController.close();
    _sniController.close();
    _dnsFallbackController.close();
  }
}
