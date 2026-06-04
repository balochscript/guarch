import 'package:shared_preferences/shared_preferences.dart';

class AppSettings {
  static const String _keySocksPort = 'socks_port';
  static const String _keyDialTimeout = 'dial_timeout';
  static const String _keyHandshakeTimeout = 'handshake_timeout';

  static const int _defaultSocksPort = 7070;
  static const int _defaultDialTimeout = 30;
  static const int _defaultHandshakeTimeout = 15;

  final int socksPort;
  final int dialTimeout;
  final int handshakeTimeout;

  AppSettings({
    this.socksPort = _defaultSocksPort,
    this.dialTimeout = _defaultDialTimeout,
    this.handshakeTimeout = _defaultHandshakeTimeout,
  });

  static Future<AppSettings> load() async {
    final prefs = await SharedPreferences.getInstance();
    return AppSettings(
      socksPort: prefs.getInt(_keySocksPort) ?? _defaultSocksPort,
      dialTimeout: prefs.getInt(_keyDialTimeout) ?? _defaultDialTimeout,
      handshakeTimeout: prefs.getInt(_keyHandshakeTimeout) ?? _defaultHandshakeTimeout,
    );
  }

  Future<void> save() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setInt(_keySocksPort, socksPort);
    await prefs.setInt(_keyDialTimeout, dialTimeout);
    await prefs.setInt(_keyHandshakeTimeout, handshakeTimeout);
  }

  static Future<int> getSocksPort() async {
    final prefs = await SharedPreferences.getInstance();
    return prefs.getInt(_keySocksPort) ?? _defaultSocksPort;
  }

  static Future<void> setSocksPort(int port) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setInt(_keySocksPort, port);
  }

  static Future<int> getDialTimeout() async {
    final prefs = await SharedPreferences.getInstance();
    return prefs.getInt(_keyDialTimeout) ?? _defaultDialTimeout;
  }

  static Future<void> setDialTimeout(int timeout) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setInt(_keyDialTimeout, timeout);
  }

  static Future<int> getHandshakeTimeout() async {
    final prefs = await SharedPreferences.getInstance();
    return prefs.getInt(_keyHandshakeTimeout) ?? _defaultHandshakeTimeout;
  }

  static Future<void> setHandshakeTimeout(int timeout) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setInt(_keyHandshakeTimeout, timeout);
  }

  static Future<void> reset() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(_keySocksPort);
    await prefs.remove(_keyDialTimeout);
    await prefs.remove(_keyHandshakeTimeout);
  }

  static Future<void> resetSocksPort() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(_keySocksPort);
  }

  Map<String, dynamic> toJson() => {
    'socks_port': socksPort,
    'dial_timeout': dialTimeout,
    'handshake_timeout': handshakeTimeout,
  };

  factory AppSettings.fromJson(Map<String, dynamic> json) {
    return AppSettings(
      socksPort: json['socks_port'] ?? _defaultSocksPort,
      dialTimeout: json['dial_timeout'] ?? _defaultDialTimeout,
      handshakeTimeout: json['handshake_timeout'] ?? _defaultHandshakeTimeout,
    );
  }

  AppSettings copyWith({
    int? socksPort,
    int? dialTimeout,
    int? handshakeTimeout,
  }) {
    return AppSettings(
      socksPort: socksPort ?? this.socksPort,
      dialTimeout: dialTimeout ?? this.dialTimeout,
      handshakeTimeout: handshakeTimeout ?? this.handshakeTimeout,
    );
  }
}
