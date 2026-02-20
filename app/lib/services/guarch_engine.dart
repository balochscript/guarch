import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'package:flutter/services.dart';

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

  Future<void> init() async {
    if (_initialized) return;
    _initialized = true;

    _channel.setMethodCallHandler(_handleMethodCall);

    _eventChannel.receiveBroadcastStream().listen((event) {
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
            }
            break;
          case 'log':
            _logController.add(data as String);
            break;
        }
      }
    });
  }

  Future<dynamic> _handleMethodCall(MethodCall call) async {
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

  Future<bool> connect({
    required String serverAddr,
    int serverPort = 8443,
    String listenAddr = '127.0.0.1:1080',
    bool coverEnabled = true,
  }) async {
    try {
      final config = jsonEncode({
        'server_addr': serverAddr,
        'server_port': serverPort,
        'listen_addr': listenAddr,
        'cover_enabled': coverEnabled,
      });

      final result = await _channel.invokeMethod('connect', config);
      return result == true;
    } on PlatformException catch (e) {
      _logController.add('Connect error: ${e.message}');
      return false;
    }
  }

  Future<bool> disconnect() async {
    try {
      final result = await _channel.invokeMethod('disconnect');
      return result == true;
    } on PlatformException catch (e) {
      _logController.add('Disconnect error: ${e.message}');
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
      if (result is String) {
        return jsonDecode(result) as Map<String, dynamic>;
      }
      return {};
    } catch (_) {
      return {};
    }
  }

  Future<int> ping(String address, int port) async {
    try {
      final socket = await Socket.connect(
        address,
        port,
        timeout: const Duration(seconds: 5),
      );
      socket.destroy();

      final stopwatch = Stopwatch()..start();
      final socket2 = await Socket.connect(
        address,
        port,
        timeout: const Duration(seconds: 5),
      );
      stopwatch.stop();
      socket2.destroy();

      return stopwatch.elapsedMilliseconds;
    } catch (_) {
      return -1;
    }
  }

  void dispose() {
    _statusController.close();
    _statsController.close();
    _logController.close();
  }
}
