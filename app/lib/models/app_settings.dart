import 'package:shared_preferences/shared_preferences.dart';

class AppSettings {
  static const String _keySocksPort = 'socks_port';
  static const int _defaultSocksPort = 7070;

  static Future<int> getSocksPort() async {
    final prefs = await SharedPreferences.getInstance();
    return prefs.getInt(_keySocksPort) ?? _defaultSocksPort;
  }

  static Future<void> setSocksPort(int port) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setInt(_keySocksPort, port);
  }

  static Future<void> resetSocksPort() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(_keySocksPort);
  }
}
