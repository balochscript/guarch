import 'package:shared_preferences/shared_preferences.dart';
import 'package:guarch/models/server_config.dart';
import 'dart:convert';

class AppSettings {
  static const String _keySocksPort = 'socks_port';
  static const String _keyDialTimeout = 'dial_timeout';
  static const String _keyHandshakeTimeout = 'handshake_timeout';
  static const String _keyPreferIPv6 = 'prefer_ipv6';
  static const String _keyReadTimeout = 'read_timeout';
  static const String _keyWriteTimeout = 'write_timeout';

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

  static const String _keyGlobalPaddingEnabled = 'global_padding_enabled';
  static const String _keyGlobalMaxPadding = 'global_max_padding';
  static const String _keyGlobalTrafficPattern = 'global_traffic_pattern';
  static const String _keyGlobalBatteryThreshold = 'global_battery_threshold';
  static const String _keyGlobalHysteresisDelay = 'global_hysteresis_delay';
  static const String _keyGlobalExpertMode = 'global_expert_mode';
  static const String _keyGlobalDecoyEnabled = 'global_decoy_enabled';
  static const String _keyGlobalProbeDetectionEnabled = 'global_probe_detection_enabled';
  static const String _keyGlobalProbeMaxRate = 'global_probe_max_rate';
  static const String _keyGlobalProbeWindow = 'global_probe_window';

  static const int _defaultSocksPort = 7070;
  static const int _defaultDialTimeout = 30;
  static const int _defaultHandshakeTimeout = 15;
  static const int _defaultReadTimeout = 90;
  static const int _defaultWriteTimeout = 180;

  final int socksPort;
  final int dialTimeout;
  final int handshakeTimeout;
  final bool preferIPv6;
  final int readTimeout;
  final int writeTimeout;

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

  final bool globalPaddingEnabled;
  final int globalMaxPadding;
  final String globalTrafficPattern;
  final int globalBatteryThreshold;
  final int globalHysteresisDelay;
  final bool globalExpertMode;
  final bool globalDecoyEnabled;
  final bool globalProbeDetectionEnabled;
  final int globalProbeMaxRate;
  final int globalProbeWindow;

  AppSettings({
    this.socksPort = _defaultSocksPort,
    this.dialTimeout = _defaultDialTimeout,
    this.handshakeTimeout = _defaultHandshakeTimeout,
    this.preferIPv6 = false,
    this.readTimeout = _defaultReadTimeout,
    this.writeTimeout = _defaultWriteTimeout,
    this.globalSniEnabled = false,
    this.globalSniMode = 'weighted',
    this.globalSniRotationMinutes = 5,
    List<SNIDomain>? globalSniDomains,
    this.globalCoverEnabled = false,
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
    this.globalPaddingEnabled = false,
    this.globalMaxPadding = 256,
    this.globalTrafficPattern = 'web',
    this.globalBatteryThreshold = 20,
    this.globalHysteresisDelay = 30,
    this.globalExpertMode = false,
    this.globalDecoyEnabled = true,
    this.globalProbeDetectionEnabled = true,
    this.globalProbeMaxRate = 10,
    this.globalProbeWindow = 5,
  })  : globalSniDomains = globalSniDomains ?? [],
        globalCoverDomains = globalCoverDomains ?? [],
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

  factory AppSettings.defaults() {
    return AppSettings(
      socksPort: _defaultSocksPort,
      dialTimeout: _defaultDialTimeout,
      handshakeTimeout: _defaultHandshakeTimeout,
      preferIPv6: false,
      readTimeout: _defaultReadTimeout,
      writeTimeout: _defaultWriteTimeout,
      globalSniEnabled: false,
      globalSniMode: 'weighted',
      globalSniRotationMinutes: 5,
      globalSniDomains: [],
      globalCoverEnabled: false,
      globalCoverMode: 'balanced',
      globalBatteryAware: true,
      globalDataSaver: false,
      globalCoverDomains: [],
      globalDnsEnabled: false,
      globalDnsDomain: 'tunnel.example.com',
      globalDnsServers: ['8.8.8.8:53', '1.1.1.1:53'],
      globalDnsSwitchThreshold: 3,
      globalUtlsEnabled: true,
      globalUtlsFingerprint: 'chrome_auto',
      globalFragmentEnabled: false,
      globalFragmentMinSize: 64,
      globalFragmentMaxSize: 256,
      globalPaddingEnabled: false,
      globalMaxPadding: 256,
      globalTrafficPattern: 'web',
      globalBatteryThreshold: 20,
      globalHysteresisDelay: 30,
      globalExpertMode: false,
      globalDecoyEnabled: true,
      globalProbeDetectionEnabled: true,
      globalProbeMaxRate: 10,
      globalProbeWindow: 5,
    );
  }

  static Future<AppSettings> load() async {
    final prefs = await SharedPreferences.getInstance();

    List<SNIDomain> sniDomains = [];
    final sniDomainsJson = prefs.getStringList(_keyGlobalSniDomains);
    if (sniDomainsJson != null && sniDomainsJson.isNotEmpty) {
      try {
        sniDomains = sniDomainsJson.map((d) => SNIDomain.fromJson(jsonDecode(d))).toList();
      } catch (_) {
        sniDomains = [];
      }
    }

    List<CoverDomain> coverDomains = [];
    final coverDomainsJson = prefs.getStringList(_keyGlobalCoverDomains);
    if (coverDomainsJson != null && coverDomainsJson.isNotEmpty) {
      try {
        coverDomains = coverDomainsJson.map((d) => CoverDomain.fromJson(jsonDecode(d))).toList();
      } catch (_) {
        coverDomains = [];
      }
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
      preferIPv6: prefs.getBool(_keyPreferIPv6) ?? false,
      readTimeout: prefs.getInt(_keyReadTimeout) ?? _defaultReadTimeout,
      writeTimeout: prefs.getInt(_keyWriteTimeout) ?? _defaultWriteTimeout,
      globalSniEnabled: prefs.getBool(_keyGlobalSniEnabled) ?? false,
      globalSniMode: prefs.getString(_keyGlobalSniMode) ?? 'weighted',
      globalSniRotationMinutes: prefs.getInt(_keyGlobalSniRotationMinutes) ?? 5,
      globalSniDomains: sniDomains,
      globalCoverEnabled: prefs.getBool(_keyGlobalCoverEnabled) ?? false,
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
      globalPaddingEnabled: prefs.getBool(_keyGlobalPaddingEnabled) ?? false,
      globalMaxPadding: prefs.getInt(_keyGlobalMaxPadding) ?? 256,
      globalTrafficPattern: prefs.getString(_keyGlobalTrafficPattern) ?? 'web',
      globalBatteryThreshold: prefs.getInt(_keyGlobalBatteryThreshold) ?? 20,
      globalHysteresisDelay: prefs.getInt(_keyGlobalHysteresisDelay) ?? 30,
      globalExpertMode: prefs.getBool(_keyGlobalExpertMode) ?? false,
      globalDecoyEnabled: prefs.getBool(_keyGlobalDecoyEnabled) ?? true,
      globalProbeDetectionEnabled: prefs.getBool(_keyGlobalProbeDetectionEnabled) ?? true,
      globalProbeMaxRate: prefs.getInt(_keyGlobalProbeMaxRate) ?? 10,
      globalProbeWindow: prefs.getInt(_keyGlobalProbeWindow) ?? 5,
    );
  }

  Future<void> save() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setInt(_keySocksPort, socksPort);
    await prefs.setInt(_keyDialTimeout, dialTimeout);
    await prefs.setInt(_keyHandshakeTimeout, handshakeTimeout);
    await prefs.setBool(_keyPreferIPv6, preferIPv6);
    await prefs.setInt(_keyReadTimeout, readTimeout);
    await prefs.setInt(_keyWriteTimeout, writeTimeout);

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

    await prefs.setBool(_keyGlobalPaddingEnabled, globalPaddingEnabled);
    await prefs.setInt(_keyGlobalMaxPadding, globalMaxPadding);
    await prefs.setString(_keyGlobalTrafficPattern, globalTrafficPattern);
    await prefs.setInt(_keyGlobalBatteryThreshold, globalBatteryThreshold);
    await prefs.setInt(_keyGlobalHysteresisDelay, globalHysteresisDelay);
    await prefs.setBool(_keyGlobalExpertMode, globalExpertMode);
    await prefs.setBool(_keyGlobalDecoyEnabled, globalDecoyEnabled);
    await prefs.setBool(_keyGlobalProbeDetectionEnabled, globalProbeDetectionEnabled);
    await prefs.setInt(_keyGlobalProbeMaxRate, globalProbeMaxRate);
    await prefs.setInt(_keyGlobalProbeWindow, globalProbeWindow);
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
      _keyPreferIPv6, _keyReadTimeout, _keyWriteTimeout,
      _keyGlobalSniEnabled, _keyGlobalSniMode, _keyGlobalSniRotationMinutes, _keyGlobalSniDomains,
      _keyGlobalCoverEnabled, _keyGlobalCoverMode, _keyGlobalBatteryAware, _keyGlobalDataSaver, _keyGlobalCoverDomains,
      _keyGlobalDnsEnabled, _keyGlobalDnsDomain, _keyGlobalDnsServers, _keyGlobalDnsSwitchThreshold,
      _keyGlobalUtlsEnabled, _keyGlobalUtlsFingerprint,
      _keyGlobalFragmentEnabled, _keyGlobalFragmentMinSize, _keyGlobalFragmentMaxSize,
      _keyGlobalPaddingEnabled, _keyGlobalMaxPadding, _keyGlobalTrafficPattern,
      _keyGlobalBatteryThreshold, _keyGlobalHysteresisDelay, _keyGlobalExpertMode,
      _keyGlobalDecoyEnabled, _keyGlobalProbeDetectionEnabled, _keyGlobalProbeMaxRate, _keyGlobalProbeWindow,
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
    bool? preferIPv6,
    int? readTimeout,
    int? writeTimeout,
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
    bool? globalPaddingEnabled,
    int? globalMaxPadding,
    String? globalTrafficPattern,
    int? globalBatteryThreshold,
    int? globalHysteresisDelay,
    bool? globalExpertMode,
    bool? globalDecoyEnabled,
    bool? globalProbeDetectionEnabled,
    int? globalProbeMaxRate,
    int? globalProbeWindow,
  }) {
    return AppSettings(
      socksPort: socksPort ?? this.socksPort,
      dialTimeout: dialTimeout ?? this.dialTimeout,
      handshakeTimeout: handshakeTimeout ?? this.handshakeTimeout,
      preferIPv6: preferIPv6 ?? this.preferIPv6,
      readTimeout: readTimeout ?? this.readTimeout,
      writeTimeout: writeTimeout ?? this.writeTimeout,
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
      globalPaddingEnabled: globalPaddingEnabled ?? this.globalPaddingEnabled,
      globalMaxPadding: globalMaxPadding ?? this.globalMaxPadding,
      globalTrafficPattern: globalTrafficPattern ?? this.globalTrafficPattern,
      globalBatteryThreshold: globalBatteryThreshold ?? this.globalBatteryThreshold,
      globalHysteresisDelay: globalHysteresisDelay ?? this.globalHysteresisDelay,
      globalExpertMode: globalExpertMode ?? this.globalExpertMode,
      globalDecoyEnabled: globalDecoyEnabled ?? this.globalDecoyEnabled,
      globalProbeDetectionEnabled: globalProbeDetectionEnabled ?? this.globalProbeDetectionEnabled,
      globalProbeMaxRate: globalProbeMaxRate ?? this.globalProbeMaxRate,
      globalProbeWindow: globalProbeWindow ?? this.globalProbeWindow,
    );
  }

  ServerConfig applyToServer(ServerConfig server) {
    if (server.sniDomains.isEmpty && globalSniEnabled && globalSniDomains.isNotEmpty) {
      server = server.copyWith(
        sniEnabled: true,
        sniMode: globalSniMode,
        sniDomains: globalSniDomains,
      );
    }

    if (server.coverDomains.isEmpty && globalCoverEnabled && globalCoverDomains.isNotEmpty) {
      server = server.copyWith(
        coverEnabled: true,
        coverDomains: globalCoverDomains,
        batteryAwareEnabled: globalBatteryAware,
        dataSaverEnabled: globalDataSaver,
        shapingPattern: globalCoverMode,
      );
    }

    if (!server.paddingEnabled && globalPaddingEnabled) {
      server = server.copyWith(
        paddingEnabled: true,
        maxPadding: globalMaxPadding,
      );
    }

    if (server.trafficPattern == 'web' && globalTrafficPattern != 'web') {
      server = server.copyWith(
        trafficPattern: globalTrafficPattern,
      );
    }

    server = server.copyWith(
      batteryThreshold: globalBatteryThreshold,
      hysteresisDelay: globalHysteresisDelay,
    );

    if (!server.dnsFallbackEnabled && globalDnsEnabled) {
      server = server.copyWith(
        dnsFallbackEnabled: true,
        dnsFallbackDomain: globalDnsDomain,
        dnsFallbackServers: globalDnsServers,
        dnsFallbackSwitchThreshold: globalDnsSwitchThreshold,
      );
    }

    server = server.copyWith(
      decoyEnabled: globalDecoyEnabled,
      probeDetectionEnabled: globalProbeDetectionEnabled,
      probeMaxRate: globalProbeMaxRate,
      probeWindow: globalProbeWindow,
    );

    return server;
  }
}
