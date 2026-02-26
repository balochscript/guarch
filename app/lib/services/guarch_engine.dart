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
  bool _nativeAvailable = true;

  Future<void> init() async {
    if (_initialized) return;
    _initialized = true;

    try {
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
              } else if (data is Map) {
                _statsController.add(Map<String, dynamic>.from(data));
              }
              break;
            case 'log':
              _logController.add(data as String);
              break;
          }
        }
      }, onError: (e) {
        _logController.add('Event channel error: $e');
      });
    } catch (e) {
      _logController.add('Engine init error: $e');
    }
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
    required String psk,
    String? certPin,
    String listenAddr = '127.0.0.1',
    int listenPort = 1080,
    bool coverEnabled = true,
    String protocol = 'guarch', 
  }) async {
    if (serverAddr.isEmpty) {
      _logController.add('Error: server address is empty');
      return false;
    }
    if (psk.isEmpty) {
      _logController.add('Error: PSK is required');
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

      _logController.add('Connecting via $protocol to $serverAddr:$serverPort...');

      final result = await _channel.invokeMethod('connect', config);
      return result == true;
    } on PlatformException catch (e) {
      _logController.add('Connect error: ${e.message}');
      _statusController.add('disconnected');
      return false;
    } on MissingPluginException {
      _logController.add('⚠️ Native engine not available');
      _nativeAvailable = false;
      _statusController.add('disconnected');
      return false;
    }
  }

  Future<bool> disconnect() async {
    try {
      _logController.add('Disconnecting...');
      final result = await _channel.invokeMethod('disconnect');
      return result == true;
    } on PlatformException catch (e) {
      _logController.add('Disconnect error: ${e.message}');
      return false;
    } on MissingPluginException {
      _statusController.add('disconnected');
      return true;
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
      if (result is Map) {
        return Map<String, dynamic>.from(result);
      }
      return {};
    } catch (_) {
      return {};
    }
  }

  bool get isNativeAvailable => _nativeAvailable;

  Future<int> ping(String address, int port) async {
    try {
      final List<InternetAddress> addresses;
      try {
        addresses = await InternetAddress.lookup(address)
            .timeout(const Duration(seconds: 5));
      } catch (e) {
        _logController.add('DNS lookup failed for $address');
        return -1;
      }

      if (addresses.isEmpty) {
        _logController.add('DNS: no addresses for $address');
        return -1;
      }

      final ip = addresses.first.address;
      final stopwatch = Stopwatch()..start();

      final socket = await Socket.connect(
        ip,
        port,
        timeout: const Duration(seconds: 5),
      );

      stopwatch.stop();
      socket.destroy();

      return stopwatch.elapsedMilliseconds;
    } on SocketException catch (e) {
      _logController.add('Ping failed ($address:$port): ${e.message}');
      return -1;
    } on TimeoutException {
      _logController.add('Ping timeout ($address:$port)');
      return -1;
    } catch (e) {
      _logController.add('Ping error: $e');
      return -1;
    }
  }

  void dispose() {
    _statusController.close();
    _statsController.close();
    _logController.close();
  }
}
