import 'package:shared_preferences/shared_preferences.dart';
import 'package:guarch/models/server_config.dart';
import 'dart:convert';

class AppSettings {
  static const String _keySocksPort = 'socks_port';
  static const String _keyDialTimeout = 'dial_timeout';
  static const String _keyHandshakeTimeout = 'handshake_timeout';
  
  static const String _keyGlobalSniEnabled = 'global_sni_enabled';
  static const String _keyGlobalSniMode = 'global_sni_mode';
  static const String _keyGlobalSniRotationMinutes = 'global_sni_rotation_minutes';
  static const String _keyGlobalSniDomains = 'global_sni_domains';
  
  static const String _keyGlobalCoverEnabled = 'global_cover_enabled';
  static const String _keyGlobalCoverMode = 'global_cover_mode';
  static const String _keyGlobalBatteryAware = 'global_battery_aware';
  static const String _keyGlobalDataSaver = 'global_data_saver';
  static const String _keyGlobalCoverDomains = 'global_cover_domains';
  
  static const String _keyGlobalDnsEnabled = 'global_dns_enabled';
  static const String _keyGlobalDnsDomain = 'global_dns_domain';
  static const String _keyGlobalDnsServers = 'global_dns_servers';
  static const String _keyGlobalDnsSwitchThreshold = 'global_dns_switch_threshold';
  
  static const String _keyGlobalUtlsEnabled = 'global_utls_enabled';
  static const String _keyGlobalUtlsFingerprint = 'global_utls_fingerprint';
  
  static const String _keyGlobalFragmentEnabled = 'global_fragment_enabled';
  static const String _keyGlobalFragmentMinSize = 'global_fragment_min_size';
  static const String _keyGlobalFragmentMaxSize = 'global_fragment_max_size';

  static const int _defaultSocksPort = 7070;
  static const int _defaultDialTimeout = 30;
  static const int _defaultHandshakeTimeout = 15;

  final int socksPort;
  final int dialTimeout;
  final int handshakeTimeout;
  
  final bool globalSniEnabled;
  final String globalSniMode;
  final int globalSniRotationMinutes;
  final List<SNIDomain> globalSniDomains;
  
  final bool globalCoverEnabled;
  final String globalCoverMode;
  final bool globalBatteryAware;
  final bool globalDataSaver;
  final List<CoverDomain> globalCoverDomains;
  
  final bool globalDnsEnabled;
  final String globalDnsDomain;
  final List<String> globalDnsServers;
  final int globalDnsSwitchThreshold;
  
  final bool globalUtlsEnabled;
  final String globalUtlsFingerprint;
  
  final bool globalFragmentEnabled;
  final int globalFragmentMinSize;
  final int globalFragmentMaxSize;

  AppSettings({
    this.socksPort = _defaultSocksPort,
    this.dialTimeout = _defaultDialTimeout,
    this.handshakeTimeout = _defaultHandshakeTimeout,
    this.globalSniEnabled = true,
    this.globalSniMode = 'weighted',
    this.globalSniRotationMinutes = 5,
    List<SNIDomain>? globalSniDomains,
    this.globalCoverEnabled = true,
    this.globalCoverMode = 'balanced',
    this.globalBatteryAware = true,
    this.globalDataSaver = false,
    List<CoverDomain>? globalCoverDomains,
    this.globalDnsEnabled = false,
    this.globalDnsDomain = 'tunnel.example.com',
    List<String>? globalDnsServers,
    this.globalDnsSwitchThreshold = 3,
    this.globalUtlsEnabled = true,
    this.globalUtlsFingerprint = 'chrome_auto',
    this.globalFragmentEnabled = false,
    this.globalFragmentMinSize = 64,
    this.globalFragmentMaxSize = 256,
  })  : globalSniDomains = globalSniDomains ?? _defaultSniDomains(),
        globalCoverDomains = globalCoverDomains ?? _defaultCoverDomains(),
        globalDnsServers = globalDnsServers ?? ['8.8.8.8:53', '1.1.1.1:53'];

  static List<SNIDomain> _defaultSniDomains() => [
        SNIDomain(domain: 'google.com', weight: 20, checkHealth: true),
        SNIDomain(domain: 'cloudflare.com', weight: 20, checkHealth: true),
        SNIDomain(domain: 'microsoft.com', weight: 15, checkHealth: true),
        SNIDomain(domain: 'apple.com', weight: 10, fallback: true),
      ];

  static List<CoverDomain> _defaultCoverDomains() => [
        CoverDomain(domain: 'www.google.com', weight: 30, paths: ['/', '/search']),
        CoverDomain(domain: 'www.microsoft.com', weight: 20, paths: ['/', '/windows']),
        CoverDomain(domain: 'github.com', weight: 15, paths: ['/', '/explore']),
      ];

  static Future<AppSettings> load() async {
    final prefs = await SharedPreferences.getInstance();
    
    List<SNIDomain> sniDomains = _defaultSniDomains();
    final sniDomainsJson = prefs.getStringList(_keyGlobalSniDomains);
    if (sniDomainsJson != null && sniDomainsJson.isNotEmpty) {
      try {
        sniDomains = sniDomainsJson.map((d) => SNIDomain.fromJson(jsonDecode(d))).toList();
      } catch (_) {}
    }
    
    List<CoverDomain> coverDomains = _defaultCoverDomains();
    final coverDomainsJson = prefs.getStringList(_keyGlobalCoverDomains);
    if (coverDomainsJson != null && coverDomainsJson.isNotEmpty) {
      try {
        coverDomains = coverDomainsJson.map((d) => CoverDomain.fromJson(jsonDecode(d))).toList();
      } catch (_) {}
    }
    
    List<String> dnsServers = ['8.8.8.8:53', '1.1.1.1:53'];
    final dnsServersStored = prefs.getStringList(_keyGlobalDnsServers);
    if (dnsServersStored != null && dnsServersStored.isNotEmpty) {
      dnsServers = dnsServersStored;
    }
    
    return AppSettings(
      socksPort: prefs.getInt(_keySocksPort) ?? _defaultSocksPort,
      dialTimeout: prefs.getInt(_keyDialTimeout) ?? _defaultDialTimeout,
      handshakeTimeout: prefs.getInt(_keyHandshakeTimeout) ?? _defaultHandshakeTimeout,
      globalSniEnabled: prefs.getBool(_keyGlobalSniEnabled) ?? true,
      globalSniMode: prefs.getString(_keyGlobalSniMode) ?? 'weighted',
      globalSniRotationMinutes: prefs.getInt(_keyGlobalSniRotationMinutes) ?? 5,
      globalSniDomains: sniDomains,
      globalCoverEnabled: prefs.getBool(_keyGlobalCoverEnabled) ?? true,
      globalCoverMode: prefs.getString(_keyGlobalCoverMode) ?? 'balanced',
      globalBatteryAware: prefs.getBool(_keyGlobalBatteryAware) ?? true,
      globalDataSaver: prefs.getBool(_keyGlobalDataSaver) ?? false,
      globalCoverDomains: coverDomains,
      globalDnsEnabled: prefs.getBool(_keyGlobalDnsEnabled) ?? false,
      globalDnsDomain: prefs.getString(_keyGlobalDnsDomain) ?? 'tunnel.example.com',
      globalDnsServers: dnsServers,
      globalDnsSwitchThreshold: prefs.getInt(_keyGlobalDnsSwitchThreshold) ?? 3,
      globalUtlsEnabled: prefs.getBool(_keyGlobalUtlsEnabled) ?? true,
      globalUtlsFingerprint: prefs.getString(_keyGlobalUtlsFingerprint) ?? 'chrome_auto',
      globalFragmentEnabled: prefs.getBool(_keyGlobalFragmentEnabled) ?? false,
      globalFragmentMinSize: prefs.getInt(_keyGlobalFragmentMinSize) ?? 64,
      globalFragmentMaxSize: prefs.getInt(_keyGlobalFragmentMaxSize) ?? 256,
    );
  }

  Future<void> save() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setInt(_keySocksPort, socksPort);
    await prefs.setInt(_keyDialTimeout, dialTimeout);
    await prefs.setInt(_keyHandshakeTimeout, handshakeTimeout);
    
    await prefs.setBool(_keyGlobalSniEnabled, globalSniEnabled);
    await prefs.setString(_keyGlobalSniMode, globalSniMode);
    await prefs.setInt(_keyGlobalSniRotationMinutes, globalSniRotationMinutes);
    final sniDomainsJson = globalSniDomains.map((d) => jsonEncode(d.toJson())).toList();
    await prefs.setStringList(_keyGlobalSniDomains, sniDomainsJson);
    
    await prefs.setBool(_keyGlobalCoverEnabled, globalCoverEnabled);
    await prefs.setString(_keyGlobalCoverMode, globalCoverMode);
    await prefs.setBool(_keyGlobalBatteryAware, globalBatteryAware);
    await prefs.setBool(_keyGlobalDataSaver, globalDataSaver);
    final coverDomainsJson = globalCoverDomains.map((d) => jsonEncode(d.toJson())).toList();
    await prefs.setStringList(_keyGlobalCoverDomains, coverDomainsJson);
    
    await prefs.setBool(_keyGlobalDnsEnabled, globalDnsEnabled);
    await prefs.setString(_keyGlobalDnsDomain, globalDnsDomain);
    await prefs.setStringList(_keyGlobalDnsServers, globalDnsServers);
    await prefs.setInt(_keyGlobalDnsSwitchThreshold, globalDnsSwitchThreshold);
    
    await prefs.setBool(_keyGlobalUtlsEnabled, globalUtlsEnabled);
    await prefs.setString(_keyGlobalUtlsFingerprint, globalUtlsFingerprint);
    
    await prefs.setBool(_keyGlobalFragmentEnabled, globalFragmentEnabled);
    await prefs.setInt(_keyGlobalFragmentMinSize, globalFragmentMinSize);
    await prefs.setInt(_keyGlobalFragmentMaxSize, globalFragmentMaxSize);
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
    final keys = [
      _keySocksPort, _keyDialTimeout, _keyHandshakeTimeout,
      _keyGlobalSniEnabled, _keyGlobalSniMode, _keyGlobalSniRotationMinutes, _keyGlobalSniDomains,
      _keyGlobalCoverEnabled, _keyGlobalCoverMode, _keyGlobalBatteryAware, _keyGlobalDataSaver, _keyGlobalCoverDomains,
      _keyGlobalDnsEnabled, _keyGlobalDnsDomain, _keyGlobalDnsServers, _keyGlobalDnsSwitchThreshold,
      _keyGlobalUtlsEnabled, _keyGlobalUtlsFingerprint,
      _keyGlobalFragmentEnabled, _keyGlobalFragmentMinSize, _keyGlobalFragmentMaxSize,
    ];
    for (var key in keys) {
      await prefs.remove(key);
    }
  }

  static Future<void> resetSocksPort() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(_keySocksPort);
  }

  AppSettings copyWith({
    int? socksPort,
    int? dialTimeout,
    int? handshakeTimeout,
    bool? globalSniEnabled,
    String? globalSniMode,
    int? globalSniRotationMinutes,
    List<SNIDomain>? globalSniDomains,
    bool? globalCoverEnabled,
    String? globalCoverMode,
    bool? globalBatteryAware,
    bool? globalDataSaver,
    List<CoverDomain>? globalCoverDomains,
    bool? globalDnsEnabled,
    String? globalDnsDomain,
    List<String>? globalDnsServers,
    int? globalDnsSwitchThreshold,
    bool? globalUtlsEnabled,
    String? globalUtlsFingerprint,
    bool? globalFragmentEnabled,
    int? globalFragmentMinSize,
    int? globalFragmentMaxSize,
  }) {
    return AppSettings(
      socksPort: socksPort ?? this.socksPort,
      dialTimeout: dialTimeout ?? this.dialTimeout,
      handshakeTimeout: handshakeTimeout ?? this.handshakeTimeout,
      globalSniEnabled: globalSniEnabled ?? this.globalSniEnabled,
      globalSniMode: globalSniMode ?? this.globalSniMode,
      globalSniRotationMinutes: globalSniRotationMinutes ?? this.globalSniRotationMinutes,
      globalSniDomains: globalSniDomains ?? this.globalSniDomains,
      globalCoverEnabled: globalCoverEnabled ?? this.globalCoverEnabled,
      globalCoverMode: globalCoverMode ?? this.globalCoverMode,
      globalBatteryAware: globalBatteryAware ?? this.globalBatteryAware,
      globalDataSaver: globalDataSaver ?? this.globalDataSaver,
      globalCoverDomains: globalCoverDomains ?? this.globalCoverDomains,
      globalDnsEnabled: globalDnsEnabled ?? this.globalDnsEnabled,
      globalDnsDomain: globalDnsDomain ?? this.globalDnsDomain,
      globalDnsServers: globalDnsServers ?? this.globalDnsServers,
      globalDnsSwitchThreshold: globalDnsSwitchThreshold ?? this.globalDnsSwitchThreshold,
      globalUtlsEnabled: globalUtlsEnabled ?? this.globalUtlsEnabled,
      globalUtlsFingerprint: globalUtlsFingerprint ?? this.globalUtlsFingerprint,
      globalFragmentEnabled: globalFragmentEnabled ?? this.globalFragmentEnabled,
      globalFragmentMinSize: globalFragmentMinSize ?? this.globalFragmentMinSize,
      globalFragmentMaxSize: globalFragmentMaxSize ?? this.globalFragmentMaxSize,
    );
  }

  ServerConfig applyToServer(ServerConfig server) {
    if (server.sniDomains.isEmpty && globalSniEnabled) {
      server = server.copyWith(
        sniEnabled: true,
        sniMode: globalSniMode,
        sniDomains: globalSniDomains,
      );
    }
    
    if (server.coverDomains.isEmpty && globalCoverEnabled) {
      server = server.copyWith(
        coverEnabled: true,
        coverDomains: globalCoverDomains,
        batteryAwareEnabled: globalBatteryAware,
        dataSaverEnabled: globalDataSaver,
        shapingPattern: globalCoverMode,
      );
    }
    
    if (!server.dnsFallbackEnabled && globalDnsEnabled) {
      server = server.copyWith(
        dnsFallbackEnabled: true,
        dnsFallbackDomain: globalDnsDomain,
        dnsFallbackServers: globalDnsServers,
        dnsFallbackSwitchThreshold: globalDnsSwitchThreshold,
      );
    }
    
    return server;
  }
}
