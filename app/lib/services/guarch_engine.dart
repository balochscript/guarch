import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'package:flutter/services.dart';

/// لاگ ساده Flutter-side
class FlutterLog {
  static final List<String> entries = [];

  static void d(String tag, String msg) {
    final time = DateTime.now().toString().substring(11, 23);
    entries.add('[$time] $tag: $msg');
    if (entries.length > 500) entries.removeAt(0);
    // ignore: avoid_print
    print('[$tag] $msg');
  }

  static void e(String tag, String msg, [Object? error]) {
    final time = DateTime.now().toString().substring(11, 23);
    final errStr = error != null ? '\n  >> $error' : '';
    entries.add('[$time] E/$tag: $msg$errStr');
    if (entries.length > 500) entries.removeAt(0);
    // ignore: avoid_print
    print('[E/$tag] $msg $errStr');
  }

  static String getAll() {
    return entries.isEmpty ? 'No Flutter logs' : entries.join('\n');
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

  Stream<String> get statusStream => _statusController.stream;
  Stream<Map<String, dynamic>> get statsStream => _statsController.stream;
  Stream<String> get logStream => _logController.stream;

  bool _initialized = false;
  bool _nativeAvailable = true;

  Future<void> init() async {
    FlutterLog.d('Engine', 'init() called, initialized=$_initialized');
    if (_initialized) return;
    _initialized = true;

    try {
      FlutterLog.d('Engine', 'Setting method call handler...');
      _channel.setMethodCallHandler(_handleMethodCall);
      FlutterLog.d('Engine', 'Method call handler set');

      FlutterLog.d('Engine', 'Setting up event channel...');
      _eventChannel.receiveBroadcastStream().listen(
        (event) {
          FlutterLog.d('Engine', 'Event received: ${event.runtimeType}');
          if (event is Map) {
            final type = event['type'] as String?;
            final data = event['data'];
            switch (type) {
              case 'status':
                _statusController.add(data as String);
                break;
              case 'stats':
                if (data is String) {
                  try {
                    final map = jsonDecode(data) as Map<String, dynamic>;
                    _statsController.add(map);
                  } catch (_) {}
                } else if (data is Map) {
                  _statsController.add(Map<String, dynamic>.from(data));
                }
                break;
              case 'log':
                _logController.add(data as String);
                break;
            }
          }
        },
        onError: (e) {
          FlutterLog.e('Engine', 'Event channel error', e);
        },
      );
      FlutterLog.d('Engine', 'Event channel setup done');
    } catch (e) {
      FlutterLog.e('Engine', 'init FAILED', e);
    }
    FlutterLog.d('Engine', 'init() completed');
  }

  Future<dynamic> _handleMethodCall(MethodCall call) async {
    FlutterLog.d('Engine', 'Incoming call from native: ${call.method}');
    switch (call.method) {
      case 'onStatusChanged':
        _statusController.add(call.arguments as String);
        break;
      case 'onStatsUpdate':
        try {
          final map = jsonDecode(call.arguments as String) as Map<String, dynamic>;
          _statsController.add(map);
        } catch (_) {}
        break;
      case 'onLog':
        _logController.add(call.arguments as String);
        break;
    }
  }

  Future<bool> requestVpnPermission() async {
    FlutterLog.d('Engine', 'requestVpnPermission...');
    try {
      final result = await _channel.invokeMethod('requestVpnPermission');
      FlutterLog.d('Engine', 'VPN permission result: $result');
      return result == true;
    } catch (e) {
      FlutterLog.e('Engine', 'VPN permission error', e);
      return false;
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
    FlutterLog.d('Engine', '=== connect() START ===');
    FlutterLog.d('Engine', '  addr=$serverAddr:$serverPort proto=$protocol');
    FlutterLog.d('Engine', '  psk=${psk.isNotEmpty ? "${psk.length} chars" : "EMPTY"}');
    FlutterLog.d('Engine', '  certPin=${certPin ?? "null"} listenPort=$listenPort');
    FlutterLog.d('Engine', '  cover=$coverEnabled');

    if (serverAddr.isEmpty) {
      FlutterLog.e('Engine', '  Server address is EMPTY');
      _logController.add('Error: server address is empty');
      return false;
    }
    if (psk.isEmpty) {
      FlutterLog.e('Engine', '  PSK is EMPTY');
      _logController.add('Error: PSK is required');
      return false;
    }

    try {
      final configMap = {
        'server_addr': serverAddr,
        'server_port': serverPort,
        'psk': psk,
        'cert_pin': certPin ?? '',
        'listen_addr': listenAddr,
        'listen_port': listenPort,
        'cover_enabled': coverEnabled,
        'protocol': protocol,
      };
      FlutterLog.d('Engine', '  Config map built');

      final config = jsonEncode(configMap);
      FlutterLog.d('Engine', '  JSON encoded, length=${config.length}');
      FlutterLog.d('Engine', '  JSON: ${config.substring(0, config.length.clamp(0, 200))}');

      _logController.add('Connecting via $protocol to $serverAddr:$serverPort...');

      FlutterLog.d('Engine', '  Calling invokeMethod("connect")...');
      final result = await _channel.invokeMethod('connect', config);
      FlutterLog.d('Engine', '  invokeMethod returned: $result (${result.runtimeType})');

      return result == true;
    } on PlatformException catch (e) {
      FlutterLog.e('Engine', '  PlatformException: ${e.code} - ${e.message}', e);
      _logController.add('Connect error: ${e.message}');
      _statusController.add('disconnected');
      return false;
    } on MissingPluginException catch (e) {
      FlutterLog.e('Engine', '  MissingPluginException', e);
      _logController.add('⚠️ Native engine not available');
      _nativeAvailable = false;
      _statusController.add('disconnected');
      return false;
    } catch (e) {
      FlutterLog.e('Engine', '  UNEXPECTED ERROR', e);
      _logController.add('Unexpected error: $e');
      _statusController.add('disconnected');
      return false;
    }
  }

  Future<bool> disconnect() async {
    FlutterLog.d('Engine', '=== disconnect() ===');
    try {
      final result = await _channel.invokeMethod('disconnect');
      FlutterLog.d('Engine', '  Result: $result');
      return result == true;
    } on PlatformException catch (e) {
      FlutterLog.e('Engine', '  PlatformException', e);
      return false;
    } on MissingPluginException {
      FlutterLog.d('Engine', '  MissingPlugin - returning true');
      _statusController.add('disconnected');
      return true;
    } catch (e) {
      FlutterLog.e('Engine', '  UNEXPECTED', e);
      return false;
    }
  }

  Future<String> getStatus() async {
    try {
      final result = await _channel.invokeMethod('getStatus');
      return result as String? ?? 'disconnected';
    } catch (_) {
      return 'disconnected';
    }
  }

  Future<Map<String, dynamic>> getStats() async {
    try {
      final result = await _channel.invokeMethod('getStats');
      if (result is String) return jsonDecode(result) as Map<String, dynamic>;
      if (result is Map) return Map<String, dynamic>.from(result);
      return {};
    } catch (_) {
      return {};
    }
  }

  bool get isNativeAvailable => _nativeAvailable;

  Future<int> ping(String address, int port) async {
    FlutterLog.d('Engine', 'ping $address:$port');
    try {
      final addresses = await InternetAddress.lookup(address)
          .timeout(const Duration(seconds: 5));
      if (addresses.isEmpty) return -1;

      final stopwatch = Stopwatch()..start();
      final socket = await Socket.connect(
        addresses.first.address, port,
        timeout: const Duration(seconds: 5),
      );
      stopwatch.stop();
      socket.destroy();
      return stopwatch.elapsedMilliseconds;
    } catch (e) {
      FlutterLog.e('Engine', 'Ping failed', e);
      return -1;
    }
  }

  void dispose() {
    _statusController.close();
    _statsController.close();
    _logController.close();
  }
}
