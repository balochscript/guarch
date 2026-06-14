import 'dart:convert';
import 'package:flutter/material.dart';

class ServerConfig {
  final int version;
  String id;
  String name;
  String address;
  int port;
  String psk;
  String? certPin;
  String protocol;
  bool coverEnabled;
  List<CoverDomain> coverDomains;
  
  bool sniEnabled;
  String sniMode;
  List<SNIDomain> sniDomains;
  bool dnsFallbackEnabled;
  String dnsFallbackMode;
  bool batteryAwareEnabled;
  bool dataSaverEnabled;

  String dnsFallbackDomain;
  List<String> dnsFallbackServers;
  int dnsFallbackTimeout;
  int dnsFallbackSwitchThreshold;
  
  String shapingPattern;
  int maxPadding;

  bool paddingEnabled;
  String trafficPattern;
  int batteryThreshold;
  int hysteresisDelay;
  bool decoyEnabled;
  bool probeDetectionEnabled;
  int probeMaxRate;
  int probeWindow;
  
  TransportConfig? transport;
  GroukConfig? grouk;
  ZhipConfig? zhip;
  
  int? ping;
  int? realDelay;
  DateTime? lastTested;
  
  bool isActive;
  DateTime createdAt;
  
  Metadata? metadata;

  ServerConfig({
    this.version = 1,
    required this.id,
    required this.name,
    required this.address,
    this.port = 8443,
    this.psk = '',
    this.certPin,
    this.protocol = 'guarch',
    this.coverEnabled = false,
    List<CoverDomain>? coverDomains,
    this.sniEnabled = false,
    this.sniMode = 'weighted',
    List<SNIDomain>? sniDomains,
    this.dnsFallbackEnabled = false,
    this.dnsFallbackMode = 'auto',
    this.dnsFallbackDomain = 'tunnel.example.com',
    List<String>? dnsFallbackServers,
    this.dnsFallbackTimeout = 5,
    this.dnsFallbackSwitchThreshold = 3,
    this.batteryAwareEnabled = true,
    this.dataSaverEnabled = false,
    this.shapingPattern = 'web_browsing',
    this.maxPadding = 1024,
    this.paddingEnabled = false,
    this.trafficPattern = 'web',
    this.batteryThreshold = 20,
    this.hysteresisDelay = 30,
    this.decoyEnabled = true,
    this.probeDetectionEnabled = true,
    this.probeMaxRate = 10,
    this.probeWindow = 5,
    this.transport,
    this.grouk,
    this.zhip,
    this.ping,
    this.realDelay,
    this.lastTested,
    this.isActive = false,
    DateTime? createdAt,
    this.metadata,
  })  : coverDomains = coverDomains ?? [],
        sniDomains = sniDomains ?? [],
        dnsFallbackServers = dnsFallbackServers ?? ['8.8.8.8:53', '1.1.1.1:53'],
        createdAt = createdAt ?? DateTime.now();

  String get fullAddress => '$address:$port';

  String get pingText {
    if (ping == null) return 'testing...';
    if (ping! < 0) return 'timeout';
    return '${ping}ms';
  }

  String get realDelayText {
    if (realDelay == null) return 'not tested';
    if (realDelay! < 0) return 'timeout';
    return '${realDelay}ms';
  }

  String get pingEmoji {
    if (ping == null) return '⏳';
    if (ping! < 0) return '🔴';
    if (ping! < 100) return '🟢';
    if (ping! < 300) return '🟡';
    return '🟠';
  }

  String get realDelayEmoji {
    if (realDelay == null) return '⏳';
    if (realDelay! < 0) return '🔴';
    if (realDelay! < 200) return '🟢';
    if (realDelay! < 500) return '🟡';
    return '🟠';
  }

  String get lastTestedText {
    if (lastTested == null) return 'Never';
    final now = DateTime.now();
    final diff = now.difference(lastTested!);
    
    if (diff.inSeconds < 60) return '${diff.inSeconds}s ago';
    if (diff.inMinutes < 60) return '${diff.inMinutes}m ago';
    if (diff.inHours < 24) return '${diff.inHours}h ago';
    return '${diff.inDays}d ago';
  }

  String get protocolEmoji {
    switch (protocol) {
      case 'grouk':
        return '🌩️';
      case 'zhip':
        return '⚡';
      default:
        return '🏹';
    }
  }

  String get protocolLabel {
    switch (protocol) {
      case 'grouk':
        return 'Grouk (UDP)';
      case 'zhip':
        return 'Zhip (QUIC)';
      default:
        return 'Guarch (TLS)';
    }
  }

  bool get groukFecEnabled => grouk?.enableFEC ?? false;
  int get groukFecDataShards => grouk?.fecDataShards ?? 10;
  int get groukFecParityShards => grouk?.fecParityShards ?? 3;

  int get zhipMaxIdleTimeout => zhip?.maxIdleTimeout ?? 60;
  int get zhipKeepAlivePeriod => zhip?.keepAlivePeriod ?? 25;
  int get zhipMaxStreams => zhip?.maxStreams ?? 256;

  bool get isValid =>
      address.isNotEmpty &&
      port > 0 &&
      psk.isNotEmpty &&
      ['guarch', 'grouk', 'zhip'].contains(protocol);

  Map<String, dynamic> toJson() {
    final json = {
      'version': version,
      'id': id,
      'server': {
        'name': name,
        'address': '$address:$port',
        'protocol': protocol,
        'psk': psk,
        if (certPin != null) 'cert_pin': certPin,
      },
      'is_active': isActive,
      if (ping != null) 'ping': ping,
      if (realDelay != null) 'real_delay': realDelay,
      if (lastTested != null) 'last_tested': lastTested!.toIso8601String(),
      'created_at': createdAt.toIso8601String(),
      'padding_enabled': paddingEnabled,
      'max_padding': maxPadding,
      'traffic_pattern': trafficPattern,
      'battery_threshold': batteryThreshold,
      'hysteresis_delay': hysteresisDelay,
      'decoy_enabled': decoyEnabled,
      'probe_detection_enabled': probeDetectionEnabled,
      'probe_max_rate': probeMaxRate,
      'probe_window': probeWindow,
    };

    if (transport != null) {
      json['transport'] = transport!.toJson();
    }

    if (grouk != null && protocol == 'grouk') {
      json['grouk'] = grouk!.toJson();
    }

    if (zhip != null && protocol == 'zhip') {
      json['zhip'] = zhip!.toJson();
    }

    if (sniEnabled && sniDomains.isNotEmpty && protocol != 'grouk' && protocol != 'zhip') {
      json['sni'] = {
        'enabled': true,
        'mode': sniMode,
        'rotation_interval': '5m',
        'domains': sniDomains.map((d) => d.toJson()).toList(),
      };
    }

    if (coverEnabled && coverDomains.isNotEmpty) {
      json['cover'] = {
        'enabled': true,
        'mode': shapingPattern,
        'domains': coverDomains.map((d) => d.toJson()).toList(),
        'adaptive': {
          'enabled': true,
          'battery_aware': batteryAwareEnabled,
          'data_saver_mode': dataSaverEnabled,
        },
      };
    }

    if (dnsFallbackEnabled && protocol != 'grouk' && protocol != 'zhip') {
      json['dns'] = {
        'enabled': true,
        'domain': dnsFallbackDomain,
        'servers': dnsFallbackServers,
        'auto_switch': dnsFallbackMode == 'auto',
        'switch_threshold': dnsFallbackSwitchThreshold,
        'timeout': '${dnsFallbackTimeout}s',
      };
    }

    if (metadata != null) {
      json['metadata'] = metadata!.toJson();
    }

    return json;
  }

  factory ServerConfig.fromJson(Map<String, dynamic> json) {
    int version = json['version'] ?? 1;
    if (version != 1) {
      throw Exception('Unsupported config version: $version (expected 1)');
    }

    final server = json['server'] as Map<String, dynamic>? ?? json;
    final sni = json['sni'] as Map<String, dynamic>? ?? {};
    final cover = json['cover'] ?? json['cover_traffic'] ?? {};
    final dns = json['dns'] ?? json['dns_fallback'] ?? {};
    final transportData = json['transport'] as Map<String, dynamic>?;
    final groukData = json['grouk'] as Map<String, dynamic>?;
    final zhipData = json['zhip'] as Map<String, dynamic>?;

    String address = server['address'] ?? json['address'] ?? '';
    int port = server['port'] ?? json['port'] ?? 8443;
    
    if (address.contains(':')) {
      final parts = address.split(':');
      address = parts[0];
      port = int.tryParse(parts[1]) ?? port;
    }

    return ServerConfig(
      version: version,
      id: json['id'] ?? DateTime.now().millisecondsSinceEpoch.toString(),
      name: server['name'] ?? 'Server',
      address: address,
      port: port,
      psk: server['psk'] ?? json['psk'] ?? '',
      certPin: server['cert_pin'] ?? json['cert_pin'],
      protocol: server['protocol'] ?? json['protocol'] ?? 'guarch',
      
      transport: transportData != null 
          ? TransportConfig.fromJson(transportData)
          : null,
      
      grouk: groukData != null ? GroukConfig.fromJson(groukData) : null,
      zhip: zhipData != null ? ZhipConfig.fromJson(zhipData) : null,
      
      coverEnabled: cover['enabled'] == true,
      coverDomains: cover['enabled'] == true 
          ? _parseCoverDomains(cover['domains'] ?? json['cover_domains'])
          : [],
      
      sniEnabled: sni['enabled'] == true,
      sniMode: sni['mode'] ?? 'weighted',
      sniDomains: sni['enabled'] == true
          ? _parseSNIDomains(sni['domains'])
          : [],
      
      dnsFallbackEnabled: dns['enabled'] == true,
      dnsFallbackMode: dns['mode'] ?? 'auto',
      dnsFallbackDomain: dns['domain'] ?? 'tunnel.example.com',
      dnsFallbackServers: (dns['servers'] as List?)?.cast<String>() ?? ['8.8.8.8:53', '1.1.1.1:53'],
      dnsFallbackTimeout: dns['timeout'] != null 
          ? int.tryParse(dns['timeout'].toString().replaceAll('s', '')) ?? 5 
          : 5,
      dnsFallbackSwitchThreshold: dns['switch_threshold'] ?? 3,
      
      batteryAwareEnabled: cover['adaptive']?['battery_aware'] ?? true,
      dataSaverEnabled: cover['adaptive']?['data_saver_mode'] ?? false,
      shapingPattern: cover['mode'] ?? json['shaping_pattern'] ?? 'web_browsing',
      maxPadding: json['max_padding'] ?? 1024,

      paddingEnabled: json['padding_enabled'] ?? false,
      trafficPattern: json['traffic_pattern'] ?? 'web',
      batteryThreshold: json['battery_threshold'] ?? 20,
      hysteresisDelay: json['hysteresis_delay'] ?? 30,
      decoyEnabled: json['decoy_enabled'] ?? true,
      probeDetectionEnabled: json['probe_detection_enabled'] ?? true,
      probeMaxRate: json['probe_max_rate'] ?? 10,
      probeWindow: json['probe_window'] ?? 5,
      
      ping: json['ping'],
      realDelay: json['real_delay'],
      lastTested: json['last_tested'] != null
          ? DateTime.parse(json['last_tested'])
          : null,
      isActive: json['is_active'] ?? false,
      createdAt: json['created_at'] != null
          ? DateTime.parse(json['created_at'])
          : DateTime.now(),
      metadata: json['metadata'] != null 
          ? Metadata.fromJson(json['metadata']) 
          : null,
    );
  }

  static List<CoverDomain> _parseCoverDomains(dynamic data) {
    if (data == null) return [];
    if (data is! List) return [];
    try {
      return data.map((d) => CoverDomain.fromJson(d)).toList();
    } catch (e) {
      return [];
    }
  }

  static List<SNIDomain> _parseSNIDomains(dynamic data) {
    if (data == null) return [];
    if (data is! List) return [];
    try {
      return data.map((d) => SNIDomain.fromJson(d)).toList();
    } catch (e) {
      return [];
    }
  }

  String toShareString() {
    final data = toJson();
    final jsonStr = jsonEncode(data);
    final encoded = base64Encode(utf8.encode(jsonStr));
    return '$protocol://$encoded';
  }

  factory ServerConfig.fromShareString(String shareStr) {
    String data = shareStr;
    String detectedProtocol = 'guarch';

    for (final proto in ['guarch', 'grouk', 'zhip']) {
      if (data.startsWith('$proto://')) {
        detectedProtocol = proto;
        data = data.substring(proto.length + 3);
        break;
      }
    }

    final decoded = utf8.decode(base64Decode(data));
    final json = jsonDecode(decoded) as Map<String, dynamic>;
    json['id'] = DateTime.now().millisecondsSinceEpoch.toString();
    
    if (json['server'] != null) {
      json['server']['protocol'] = detectedProtocol;
    } else {
      json['protocol'] = detectedProtocol;
    }
    
    return ServerConfig.fromJson(json);
  }

  ServerConfig copyWith({
    String? name,
    String? address,
    int? port,
    String? psk,
    String? certPin,
    String? protocol,
    TransportConfig? transport,
    GroukConfig? grouk,
    ZhipConfig? zhip,
    bool? coverEnabled,
    List<CoverDomain>? coverDomains,
    bool? sniEnabled,
    String? sniMode,
    List<SNIDomain>? sniDomains,
    bool? dnsFallbackEnabled,
    String? dnsFallbackMode,
    String? dnsFallbackDomain,
    List<String>? dnsFallbackServers,
    int? dnsFallbackTimeout,
    int? dnsFallbackSwitchThreshold,
    bool? batteryAwareEnabled,
    bool? dataSaverEnabled,
    String? shapingPattern,
    int? maxPadding,
    bool? paddingEnabled,
    String? trafficPattern,
    int? batteryThreshold,
    int? hysteresisDelay,
    bool? decoyEnabled,
    bool? probeDetectionEnabled,
    int? probeMaxRate,
    int? probeWindow,
    bool? isActive,
    int? ping,
    int? realDelay,
    DateTime? lastTested,
    Metadata? metadata,
  }) {
    return ServerConfig(
      version: version,
      id: id,
      name: name ?? this.name,
      address: address ?? this.address,
      port: port ?? this.port,
      psk: psk ?? this.psk,
      certPin: certPin ?? this.certPin,
      protocol: protocol ?? this.protocol,
      transport: transport ?? this.transport,
      grouk: grouk ?? this.grouk,
      zhip: zhip ?? this.zhip,
      coverEnabled: coverEnabled ?? this.coverEnabled,
      coverDomains: coverDomains ?? this.coverDomains,
      sniEnabled: sniEnabled ?? this.sniEnabled,
      sniMode: sniMode ?? this.sniMode,
      sniDomains: sniDomains ?? this.sniDomains,
      dnsFallbackEnabled: dnsFallbackEnabled ?? this.dnsFallbackEnabled,
      dnsFallbackMode: dnsFallbackMode ?? this.dnsFallbackMode,
      dnsFallbackDomain: dnsFallbackDomain ?? this.dnsFallbackDomain,
      dnsFallbackServers: dnsFallbackServers ?? this.dnsFallbackServers,
      dnsFallbackTimeout: dnsFallbackTimeout ?? this.dnsFallbackTimeout,
      dnsFallbackSwitchThreshold: dnsFallbackSwitchThreshold ?? this.dnsFallbackSwitchThreshold,
      batteryAwareEnabled: batteryAwareEnabled ?? this.batteryAwareEnabled,
      dataSaverEnabled: dataSaverEnabled ?? this.dataSaverEnabled,
      shapingPattern: shapingPattern ?? this.shapingPattern,
      maxPadding: maxPadding ?? this.maxPadding,
      paddingEnabled: paddingEnabled ?? this.paddingEnabled,
      trafficPattern: trafficPattern ?? this.trafficPattern,
      batteryThreshold: batteryThreshold ?? this.batteryThreshold,
      hysteresisDelay: hysteresisDelay ?? this.hysteresisDelay,
      decoyEnabled: decoyEnabled ?? this.decoyEnabled,
      probeDetectionEnabled: probeDetectionEnabled ?? this.probeDetectionEnabled,
      probeMaxRate: probeMaxRate ?? this.probeMaxRate,
      probeWindow: probeWindow ?? this.probeWindow,
      ping: ping ?? this.ping,
      realDelay: realDelay ?? this.realDelay,
      lastTested: lastTested ?? this.lastTested,
      isActive: isActive ?? this.isActive,
      createdAt: createdAt,
      metadata: metadata ?? this.metadata,
    );
  }
}

class TransportConfig {
  final String type;
  final String? host;
  final int? port;
  final String? path;
  final bool useTls;
  final Map<String, String>? headers;
  final List<String>? fallbackOrder;
  final int? dialTimeout;
  final int? handshakeTimeout;

  TransportConfig({
    this.type = 'direct',
    this.host,
    this.port,
    this.path,
    this.useTls = true,
    this.headers,
    this.fallbackOrder,
    this.dialTimeout,
    this.handshakeTimeout,
  });

  Map<String, dynamic> toJson() => {
    'type': type,
    if (host != null && host!.isNotEmpty) 'host': host,
    if (port != null && port! > 0) 'port': port,
    if (path != null && path!.isNotEmpty) 'path': path,
    'use_tls': useTls,
    if (headers != null && headers!.isNotEmpty) 'headers': headers,
    if (fallbackOrder != null && fallbackOrder!.isNotEmpty) 
      'fallback_order': fallbackOrder,
    if (dialTimeout != null) 'dial_timeout': dialTimeout,
    if (handshakeTimeout != null) 'handshake_timeout': handshakeTimeout,
  };

  factory TransportConfig.fromJson(Map<String, dynamic> json) {
    return TransportConfig(
      type: json['type'] ?? 'direct',
      host: json['host'],
      port: json['port'],
      path: json['path'],
      useTls: json['use_tls'] ?? true,
      headers: json['headers'] != null 
          ? Map<String, String>.from(json['headers']) 
          : null,
      fallbackOrder: json['fallback_order'] != null
          ? List<String>.from(json['fallback_order'])
          : null,
      dialTimeout: json['dial_timeout'],
      handshakeTimeout: json['handshake_timeout'],
    );
  }

  TransportConfig copyWith({
    String? type,
    String? host,
    int? port,
    String? path,
    bool? useTls,
    Map<String, String>? headers,
    List<String>? fallbackOrder,
    int? dialTimeout,
    int? handshakeTimeout,
  }) {
    return TransportConfig(
      type: type ?? this.type,
      host: host ?? this.host,
      port: port ?? this.port,
      path: path ?? this.path,
      useTls: useTls ?? this.useTls,
      headers: headers ?? this.headers,
      fallbackOrder: fallbackOrder ?? this.fallbackOrder,
      dialTimeout: dialTimeout ?? this.dialTimeout,
      handshakeTimeout: handshakeTimeout ?? this.handshakeTimeout,
    );
  }

  String get displayText {
    switch (type) {
      case 'websocket':
        return 'WebSocket${host != null ? " → $host" : ""}';
      case 'http2':
        return 'HTTP/2${host != null ? " → $host" : ""}';
      case 'dns':
        return 'DNS Tunnel';
      default:
        return 'Direct TCP';
    }
  }
}

class GroukConfig {
  final bool enableFEC;
  final int fecDataShards;
  final int fecParityShards;

  GroukConfig({
    this.enableFEC = false,
    this.fecDataShards = 10,
    this.fecParityShards = 3,
  });

  Map<String, dynamic> toJson() => {
    'enable_fec': enableFEC,
    'fec_data_shards': fecDataShards,
    'fec_parity_shards': fecParityShards,
  };

  factory GroukConfig.fromJson(Map<String, dynamic> json) {
    return GroukConfig(
      enableFEC: json['enable_fec'] ?? false,
      fecDataShards: json['fec_data_shards'] ?? 10,
      fecParityShards: json['fec_parity_shards'] ?? 3,
    );
  }

  GroukConfig copyWith({
    bool? enableFEC,
    int? fecDataShards,
    int? fecParityShards,
  }) {
    return GroukConfig(
      enableFEC: enableFEC ?? this.enableFEC,
      fecDataShards: fecDataShards ?? this.fecDataShards,
      fecParityShards: fecParityShards ?? this.fecParityShards,
    );
  }
}

class ZhipConfig {
  final int maxIdleTimeout;
  final int keepAlivePeriod;
  final int maxStreams;

  ZhipConfig({
    this.maxIdleTimeout = 60,
    this.keepAlivePeriod = 25,
    this.maxStreams = 256,
  });

  Map<String, dynamic> toJson() => {
    'max_idle_timeout': maxIdleTimeout,
    'keepalive_period': keepAlivePeriod,
    'max_streams': maxStreams,
  };

  factory ZhipConfig.fromJson(Map<String, dynamic> json) {
    return ZhipConfig(
      maxIdleTimeout: json['max_idle_timeout'] ?? 60,
      keepAlivePeriod: json['keepalive_period'] ?? 25,
      maxStreams: json['max_streams'] ?? 256,
    );
  }

  ZhipConfig copyWith({
    int? maxIdleTimeout,
    int? keepAlivePeriod,
    int? maxStreams,
  }) {
    return ZhipConfig(
      maxIdleTimeout: maxIdleTimeout ?? this.maxIdleTimeout,
      keepAlivePeriod: keepAlivePeriod ?? this.keepAlivePeriod,
      maxStreams: maxStreams ?? this.maxStreams,
    );
  }
}

class CoverDomain {
  String domain;
  int weight;
  List<String> paths;

  CoverDomain({
    required this.domain,
    this.weight = 10,
    List<String>? paths,
  }) : paths = paths ?? ['/'];

  Map<String, dynamic> toJson() => {
        'domain': domain,
        'weight': weight,
        'paths': paths,
        'interval_min': '2s',
        'interval_max': '8s',
      };

  factory CoverDomain.fromJson(Map<String, dynamic> json) {
    return CoverDomain(
      domain: json['domain'] ?? '',
      weight: json['weight'] ?? 10,
      paths: (json['paths'] as List?)?.cast<String>() ?? ['/'],
    );
  }
}

class SNIDomain {
  String domain;
  int weight;
  bool checkHealth;
  bool fallback;
  int priority;

  SNIDomain({
    required this.domain,
    this.weight = 10,
    this.checkHealth = true,
    this.fallback = false,
    this.priority = 0,
  });

  Map<String, dynamic> toJson() => {
        'domain': domain,
        'weight': weight,
        'check_health': checkHealth,
        'fallback': fallback,
        'priority': priority,
      };

  factory SNIDomain.fromJson(Map<String, dynamic> json) {
    return SNIDomain(
      domain: json['domain'] ?? '',
      weight: json['weight'] ?? 10,
      checkHealth: json['check_health'] ?? json['enabled'] ?? true,
      fallback: json['fallback'] ?? false,
      priority: json['priority'] ?? 0,
    );
  }
}

class Metadata {
  final String? createdAt;
  final String? updatedAt;
  final String? expiresAt;
  final String? country;
  final String? ispHint;
  final String? notes;
  final List<String>? tags;
  final Map<String, String>? custom;
  final QuotaInfo? quota;
  final AnnouncementConfig? announcement;

  Metadata({
    this.createdAt,
    this.updatedAt,
    this.expiresAt,
    this.country,
    this.ispHint,
    this.notes,
    this.tags,
    this.custom,
    this.quota,
    this.announcement,
  });

  factory Metadata.fromJson(Map<String, dynamic> json) {
    return Metadata(
      createdAt: json['created_at'],
      updatedAt: json['updated_at'],
      expiresAt: json['expires_at'],
      country: json['country'],
      ispHint: json['isp_hint'],
      notes: json['notes'],
      tags: json['tags'] != null ? List<String>.from(json['tags']) : null,
      custom: json['custom'] != null
          ? Map<String, String>.from(json['custom'])
          : null,
      quota: json['quota'] != null ? QuotaInfo.fromJson(json['quota']) : null,
      announcement: json['announcement'] != null
          ? AnnouncementConfig.fromJson(json['announcement'])
          : null,
    );
  }

  Map<String, dynamic> toJson() => {
        if (createdAt != null) 'created_at': createdAt,
        if (updatedAt != null) 'updated_at': updatedAt,
        if (expiresAt != null) 'expires_at': expiresAt,
        if (country != null) 'country': country,
        if (ispHint != null) 'isp_hint': ispHint,
        if (notes != null) 'notes': notes,
        if (tags != null) 'tags': tags,
        if (custom != null) 'custom': custom,
        if (quota != null) 'quota': quota!.toJson(),
        if (announcement != null) 'announcement': announcement!.toJson(),
      };

  bool get isExpired {
    if (expiresAt == null) return false;
    try {
      final expiry = DateTime.parse(expiresAt!);
      return DateTime.now().isAfter(expiry);
    } catch (_) {
      return false;
    }
  }

  String get expiryText {
    if (expiresAt == null) return 'Never';
    if (isExpired) return 'Expired';

    try {
      final expiry = DateTime.parse(expiresAt!);
      final diff = expiry.difference(DateTime.now()).inDays;

      if (diff == 0) return 'Today';
      if (diff == 1) return 'Tomorrow';
      if (diff < 30) return '$diff days';

      return '${expiry.day}/${expiry.month}/${expiry.year}';
    } catch (_) {
      return expiresAt!;
    }
  }
}

class QuotaInfo {
  final int? totalBytes;
  final int? usedBytes;
  final int? remainingBytes;
  final String? resetDate;
  final bool unlimited;

  QuotaInfo({
    this.totalBytes,
    this.usedBytes,
    this.remainingBytes,
    this.resetDate,
    this.unlimited = false,
  });

  factory QuotaInfo.fromJson(Map<String, dynamic> json) {
    return QuotaInfo(
      totalBytes: json['total_bytes'],
      usedBytes: json['used_bytes'],
      remainingBytes: json['remaining_bytes'],
      resetDate: json['reset_date'],
      unlimited: json['unlimited'] ?? false,
    );
  }

  Map<String, dynamic> toJson() => {
        if (totalBytes != null) 'total_bytes': totalBytes,
        if (usedBytes != null) 'used_bytes': usedBytes,
        if (remainingBytes != null) 'remaining_bytes': remainingBytes,
        if (resetDate != null) 'reset_date': resetDate,
        'unlimited': unlimited,
      };

  String get totalFormatted => _formatBytes(totalBytes ?? 0);
  String get usedFormatted => _formatBytes(usedBytes ?? 0);
  String get remainingFormatted => _formatBytes(remainingBytes ?? 0);

  double get usagePercent {
    if (unlimited || totalBytes == null || totalBytes == 0) return 0.0;
    return ((usedBytes ?? 0) / totalBytes!) * 100;
  }

  Color get progressColor {
    if (usagePercent > 90) return Colors.red;
    if (usagePercent > 70) return Colors.orange;
    return Colors.green;
  }

  String _formatBytes(int bytes) {
    if (bytes < 1024) return '$bytes B';
    if (bytes < 1048576) return '${(bytes / 1024).toStringAsFixed(1)} KB';
    if (bytes < 1073741824) {
      return '${(bytes / 1048576).toStringAsFixed(1)} MB';
    }
    return '${(bytes / 1073741824).toStringAsFixed(1)} GB';
  }
}

class AnnouncementConfig {
  final bool enabled;
  final String? url;
  final String? text;
  final String? interval;
  final String? priority;

  AnnouncementConfig({
    this.enabled = false,
    this.url,
    this.text,
    this.interval,
    this.priority,
  });

  factory AnnouncementConfig.fromJson(Map<String, dynamic> json) {
    return AnnouncementConfig(
      enabled: json['enabled'] ?? false,
      url: json['url'],
      text: json['text'],
      interval: json['interval'],
      priority: json['priority'] ?? 'info',
    );
  }

  Map<String, dynamic> toJson() => {
        'enabled': enabled,
        if (url != null) 'url': url,
        if (text != null) 'text': text,
        if (interval != null) 'interval': interval,
        if (priority != null) 'priority': priority,
      };

  String get icon {
    switch (priority) {
      case 'critical':
        return '🚨';
      case 'warning':
        return '⚠️';
      default:
        return 'ℹ️';
    }
  }

  Color get color {
    switch (priority) {
      case 'critical':
        return Colors.red;
      case 'warning':
        return Colors.orange;
      default:
        return Colors.blue;
    }
  }
}
