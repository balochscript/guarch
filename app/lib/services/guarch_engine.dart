import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'package:flutter/services.dart';

class FlutterLog {
  static const _logChannel = MethodChannel('com.guarch.app/logs');
  static final List<String> entries = [];

  static void d(String tag, String msg) {
    final time = DateTime.now().toString().substring(11, 23);
    entries.add('[$time] $tag: $msg');
    if (entries.length > 500) entries.removeAt(0);
    // همزمان به native هم بنویس (توی فایل ذخیره بشه)
    _writeToNative('$tag: $msg');
    // ignore: avoid_print
    print('[$tag] $msg');
  }

  static void e(String tag, String msg, [Object? error]) {
    final time = DateTime.now().toString().substring(11, 23);
    final errStr = error != null ? '\n  >> $error' : '';
    entries.add('[$time] E/$tag: $msg$errStr');
    if (entries.length > 500) entries.removeAt(0);
    _writeToNative('E/$tag: $msg$errStr');
    // ignore: avoid_print
    print('[E/$tag] $msg $errStr');
  }

  static void w(String tag, String msg) {
    final time = DateTime.now().toString().substring(11, 23);
    entries.add('[$time] W/$tag: $msg');
    if (entries.length > 500) entries.removeAt(0);
    _writeToNative('W/$tag: $msg');
    // ignore: avoid_print
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
      _channel.setMethodCallHandler(_handleMethodCall);

      _eventChannel.receiveBroadcastStream().listen(
        (event) {
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
                    _statsController.add(jsonDecode(data) as Map<String, dynamic>);
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
    } catch (e) {
      FlutterLog.e('Engine', 'init FAILED', e);
    }
    FlutterLog.d('Engine', 'init() done');
  }

  Future<dynamic> _handleMethodCall(MethodCall call) async {
    switch (call.method) {
      case 'onStatusChanged':
        _statusController.add(call.arguments as String);
        break;
      case 'onStatsUpdate':
        try {
          _statsController.add(jsonDecode(call.arguments as String) as Map<String, dynamic>);
        } catch (_) {}
        break;
      case 'onLog':
        _logController.add(call.arguments as String);
        break;
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
    FlutterLog.d('Engine', '=== connect() ===');
    FlutterLog.d('Engine', '  $protocol → $serverAddr:$serverPort');

    if (serverAddr.isEmpty || psk.isEmpty) {
      FlutterLog.e('Engine', '  Empty addr or psk!');
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
      _statusController.add('disconnected');
      return false;
    } on MissingPluginException {
      FlutterLog.e('Engine', '  MissingPlugin!');
      _nativeAvailable = false;
      _statusController.add('disconnected');
      return false;
    } catch (e) {
      FlutterLog.e('Engine', '  UNEXPECTED', e);
      _statusController.add('disconnected');
      return false;
    }
  }

  Future<bool> disconnect() async {
    FlutterLog.d('Engine', 'disconnect()');
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
    try {
      return await _channel.invokeMethod('getStatus') as String? ?? 'disconnected';
    } catch (_) {
      return 'disconnected';
    }
  }

  Future<Map<String, dynamic>> getStats() async {
    try {
      final r = await _channel.invokeMethod('getStats');
      if (r is String) return jsonDecode(r) as Map<String, dynamic>;
      if (r is Map) return Map<String, dynamic>.from(r);
      return {};
    } catch (_) {
      return {};
    }
  }

  bool get isNativeAvailable => _nativeAvailable;

  Future<int> ping(String address, int port) async {
    FlutterLog.d('Engine', 'ping $address:$port');
    try {
      final addrs = await InternetAddress.lookup(address).timeout(const Duration(seconds: 5));
      if (addrs.isEmpty) return -1;
      final sw = Stopwatch()..start();
      final socket = await Socket.connect(addrs.first.address, port, timeout: const Duration(seconds: 5));
      sw.stop();
      socket.destroy();
      return sw.elapsedMilliseconds;
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
