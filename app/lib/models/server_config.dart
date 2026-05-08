import 'dart:convert';

class ServerConfig {
  String id;
  String name;
  String address;
  int port;
  String psk;
  String? certPin;
  int listenPort;
  String protocol;
  bool coverEnabled;
  List<CoverDomain> coverDomains;
  
  // Enhanced v1.0.1
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
  
  // Ping & Delay Testing
  int? ping;
  int? realDelay;
  DateTime? lastTested;
  
  bool isActive;
  DateTime createdAt;
  
  // Metadata
  Metadata? metadata;

    ServerConfig({
    required this.id,
    required this.name,
    required this.address,
    this.port = 8443,
    this.psk = '',
    this.certPin,
    this.listenPort = 1080,
    this.protocol = 'guarch',
    this.coverEnabled = true,
    List<CoverDomain>? coverDomains,
    this.sniEnabled = true,
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
    this.ping,
    this.realDelay,
    this.lastTested,
    this.isActive = false,
    DateTime? createdAt,
    this.metadata,
  })  : coverDomains = coverDomains ?? defaultCoverDomains(),
        sniDomains = sniDomains ?? defaultSNIDomains(),
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

  bool get isValid =>
      address.isNotEmpty &&
      port > 0 &&
      psk.isNotEmpty &&
      ['guarch', 'grouk', 'zhip'].contains(protocol);

    Map<String, dynamic> toJson() {
    return {
      'version': 1,
      'id': id,
      'server': {
        'name': name,
        'address': address,
        'port': port,
        'protocol': protocol,
        'psk': psk,
        'cert_pin': certPin,
      },
      'sni': {
        'enabled': sniEnabled,
        'mode': sniMode,
        'rotation_interval': '5m',
        'domains': sniDomains.map((d) => d.toJson()).toList(),
      },
      'cover_traffic': {
        'enabled': coverEnabled,
        'mode': shapingPattern,
        'domains': coverDomains.map((d) => d.toJson()).toList(),
        'battery_aware': {
          'enabled': batteryAwareEnabled,
          'low_battery_threshold': 30,
        },
        'data_saver': {
          'enabled': dataSaverEnabled,
        },
      },
      'dns_fallback': {
        'enabled': dnsFallbackEnabled,
        'mode': dnsFallbackMode,
        'domain': dnsFallbackDomain,
        'servers': dnsFallbackServers,
        'timeout': '${dnsFallbackTimeout}s',
        'switch_threshold': dnsFallbackSwitchThreshold,
      },
      if (metadata != null) 'metadata': metadata!.toJson(),
      'listen_port': listenPort,
      'is_active': isActive,
      'ping': ping,
      'real_delay': realDelay,
      'last_tested': lastTested?.toIso8601String(),
      'created_at': createdAt.toIso8601String(),
    };
  }

    factory ServerConfig.fromJson(Map<String, dynamic> json) {
    final server = json['server'] as Map<String, dynamic>? ?? json;
    final sni = json['sni'] as Map<String, dynamic>? ?? {};
    final cover = json['cover_traffic'] as Map<String, dynamic>? ?? json['cover'] ?? {};
    final dnsFallback = json['dns_fallback'] as Map<String, dynamic>? ?? {};

    return ServerConfig(
      id: json['id'] ?? DateTime.now().millisecondsSinceEpoch.toString(),
      name: server['name'] ?? 'Server',
      address: server['address'] ?? json['address'] ?? '',
      port: server['port'] ?? json['port'] ?? 8443,
      psk: server['psk'] ?? json['psk'] ?? '',
      certPin: server['cert_pin'] ?? json['cert_pin'],
      listenPort: json['listen_port'] ?? 1080,
      protocol: server['protocol'] ?? json['protocol'] ?? 'guarch',
      coverEnabled: cover['enabled'] ?? json['cover_enabled'] ?? true,
      coverDomains: _parseCoverDomains(cover['domains'] ?? json['cover_domains']),
      sniEnabled: sni['enabled'] ?? true,
      sniMode: sni['mode'] ?? 'weighted',
      sniDomains: _parseSNIDomains(sni['domains']),
      dnsFallbackEnabled: dnsFallback['enabled'] ?? false,
      dnsFallbackMode: dnsFallback['mode'] ?? 'auto',
      dnsFallbackDomain: dnsFallback['domain'] ?? 'tunnel.example.com',
      dnsFallbackServers: (dnsFallback['servers'] as List?)?.cast<String>() ?? ['8.8.8.8:53', '1.1.1.1:53'],
      dnsFallbackTimeout: dnsFallback['timeout'] != null ? int.tryParse(dnsFallback['timeout'].toString().replaceAll('s', '')) ?? 5 : 5,
      dnsFallbackSwitchThreshold: dnsFallback['switch_threshold'] ?? 3,
      batteryAwareEnabled: cover['battery_aware']?['enabled'] ?? true,
      dataSaverEnabled: cover['data_saver']?['enabled'] ?? false,
      shapingPattern: cover['mode'] ?? json['shaping_pattern'] ?? 'web_browsing',
      maxPadding: json['max_padding'] ?? 1024,
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
    if (data == null) return defaultCoverDomains();
    if (data is! List) return defaultCoverDomains();
    return data.map((d) => CoverDomain.fromJson(d)).toList();
  }

  static List<SNIDomain> _parseSNIDomains(dynamic data) {
    if (data == null) return defaultSNIDomains();
    if (data is! List) return defaultSNIDomains();
    return data.map((d) => SNIDomain.fromJson(d)).toList();
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
    int? listenPort,
    String? protocol,
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
    bool? isActive,
    int? ping,
    int? realDelay,
    DateTime? lastTested,
    Metadata? metadata,
  }) {
    return ServerConfig(
      id: id,
      name: name ?? this.name,
      address: address ?? this.address,
      port: port ?? this.port,
      psk: psk ?? this.psk,
      certPin: certPin ?? this.certPin,
      listenPort: listenPort ?? this.listenPort,
      protocol: protocol ?? this.protocol,
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
      ping: ping ?? this.ping,
      realDelay: realDelay ?? this.realDelay,
      lastTested: lastTested ?? this.lastTested,
      isActive: isActive ?? this.isActive,
      createdAt: createdAt,
      metadata: metadata ?? this.metadata,
    );
  }

  // ✅ متدهای static داخل کلاس
  static List<CoverDomain> defaultCoverDomains() {
    return [
      CoverDomain(domain: 'www.google.com', weight: 30, paths: ['/', '/search']),
      CoverDomain(domain: 'www.microsoft.com', weight: 20, paths: ['/', '/en-us']),
      CoverDomain(domain: 'github.com', weight: 15, paths: ['/', '/explore']),
      CoverDomain(domain: 'stackoverflow.com', weight: 15, paths: ['/', '/questions']),
      CoverDomain(domain: 'www.cloudflare.com', weight: 10, paths: ['/', '/learning']),
      CoverDomain(domain: 'learn.microsoft.com', weight: 10, paths: ['/', '/en-us/docs']),
    ];
  }

  static List<SNIDomain> defaultSNIDomains() {
    return [
      SNIDomain(
        domain: 'www.google.com',
        weight: 30,
        checkHealth: true,
        fallback: false,
      ),
      SNIDomain(
        domain: 'www.microsoft.com',
        weight: 20,
        checkHealth: true,
        fallback: false,
      ),
      SNIDomain(
        domain: 'github.com',
        weight: 15,
        checkHealth: true,
        fallback: false,
      ),
      SNIDomain(
        domain: 'www.cloudflare.com',
        weight: 15,
        checkHealth: false,
        fallback: true,
      ),
      SNIDomain(
        domain: 'stackoverflow.com',
        weight: 10,
        checkHealth: true,
        fallback: false,
      ),
      SNIDomain(
        domain: 'learn.microsoft.com',
        weight: 10,
        checkHealth: false,
        fallback: true,
      ),
    ];
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
        'min_interval': '2s',
        'max_interval': '8s',
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
