import 'dart:convert';

class ServerConfig {
  String id;
  String name;
  String address;
  int port;
  String psk;              // ✅ اضافه شد
  String? certPin;         // ✅ اضافه شد
  int listenPort;          // ✅ اضافه شد
  bool coverEnabled;
  List<CoverDomain> coverDomains;
  String shapingPattern;
  int maxPadding;
  int? ping;
  bool isActive;
  DateTime createdAt;

  ServerConfig({
    required this.id,
    required this.name,
    required this.address,
    this.port = 8443,
    this.psk = '',           // ✅
    this.certPin,            // ✅
    this.listenPort = 1080,  // ✅
    this.coverEnabled = true,
    List<CoverDomain>? coverDomains,
    this.shapingPattern = 'web_browsing',
    this.maxPadding = 1024,
    this.ping,
    this.isActive = false,
    DateTime? createdAt,
  })  : coverDomains = coverDomains ?? defaultCoverDomains(),
        createdAt = createdAt ?? DateTime.now();

  String get fullAddress => '$address:$port';

  String get pingText {
    if (ping == null) return 'testing...';
    if (ping! < 0) return 'timeout';
    return '${ping}ms';
  }

  String get pingEmoji {
    if (ping == null) return '⏳';
    if (ping! < 0) return '🔴';
    if (ping! < 100) return '🟢';
    if (ping! < 300) return '🟡';
    return '🟠';
  }

  bool get isValid =>
      address.isNotEmpty && port > 0 && psk.isNotEmpty;  // ✅

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'name': name,
      'address': address,
      'port': port,
      'psk': psk,                    // ✅
      'cert_pin': certPin,           // ✅
      'listen_port': listenPort,     // ✅
      'cover_enabled': coverEnabled,
      'cover_domains': coverDomains.map((d) => d.toJson()).toList(),
      'shaping_pattern': shapingPattern,
      'max_padding': maxPadding,
      'is_active': isActive,
      'created_at': createdAt.toIso8601String(),
    };
  }

  factory ServerConfig.fromJson(Map<String, dynamic> json) {
    return ServerConfig(
      id: json['id'] ?? DateTime.now().millisecondsSinceEpoch.toString(),
      name: json['name'] ?? 'Server',
      address: json['address'] ?? json['server'] ?? '',
      port: json['port'] ?? 8443,
      psk: json['psk'] ?? '',                          // ✅
      certPin: json['cert_pin'] ?? json['pin'],        // ✅
      listenPort: json['listen_port'] ?? 1080,         // ✅
      coverEnabled: json['cover_enabled'] ?? json['cover']?['enabled'] ?? true,
      coverDomains: json['cover_domains'] != null
          ? (json['cover_domains'] as List)
              .map((d) => CoverDomain.fromJson(d))
              .toList()
          : defaultCoverDomains(),
      shapingPattern: json['shaping_pattern'] ?? 'web_browsing',
      maxPadding: json['max_padding'] ?? 1024,
      isActive: json['is_active'] ?? false,
      createdAt: json['created_at'] != null
          ? DateTime.parse(json['created_at'])
          : DateTime.now(),
    );
  }

  String toShareString() {
    final data = {
      'name': name,
      'address': address,
      'port': port,
      'psk': psk,                    // ✅
      'cert_pin': certPin,           // ✅
      'cover_enabled': coverEnabled,
      'cover_domains': coverDomains.map((d) => d.toJson()).toList(),
    };
    final jsonStr = jsonEncode(data);
    final encoded = base64Encode(utf8.encode(jsonStr));
    return 'guarch://$encoded';
  }

  factory ServerConfig.fromShareString(String shareStr) {
    String data = shareStr;
    if (data.startsWith('guarch://')) {
      data = data.substring(9);
    }
    final decoded = utf8.decode(base64Decode(data));
    final json = jsonDecode(decoded) as Map<String, dynamic>;
    json['id'] = DateTime.now().millisecondsSinceEpoch.toString();
    return ServerConfig.fromJson(json);
  }

  ServerConfig copyWith({
    String? name,
    String? address,
    int? port,
    String? psk,
    String? certPin,
    int? listenPort,
    bool? coverEnabled,
    List<CoverDomain>? coverDomains,
    bool? isActive,
    int? ping,
  }) {
    return ServerConfig(
      id: id,
      name: name ?? this.name,
      address: address ?? this.address,
      port: port ?? this.port,
      psk: psk ?? this.psk,
      certPin: certPin ?? this.certPin,
      listenPort: listenPort ?? this.listenPort,
      coverEnabled: coverEnabled ?? this.coverEnabled,
      coverDomains: coverDomains ?? this.coverDomains,
      shapingPattern: shapingPattern,
      maxPadding: maxPadding,
      ping: ping ?? this.ping,
      isActive: isActive ?? this.isActive,
      createdAt: createdAt,
    );
  }

  static List<CoverDomain> defaultCoverDomains() {
    return [
      CoverDomain(domain: 'www.google.com', weight: 30),
      CoverDomain(domain: 'www.microsoft.com', weight: 20),
      CoverDomain(domain: 'github.com', weight: 15),
      CoverDomain(domain: 'stackoverflow.com', weight: 15),
      CoverDomain(domain: 'www.cloudflare.com', weight: 10),     // ✅ بجای amazon
      CoverDomain(domain: 'learn.microsoft.com', weight: 10),    // ✅ بجای wikipedia
    ];
  }
}

class CoverDomain {
  String domain;
  int weight;

  CoverDomain({
    required this.domain,
    this.weight = 10,
  });

  Map<String, dynamic> toJson() => {
        'domain': domain,
        'weight': weight,
      };

  factory CoverDomain.fromJson(Map<String, dynamic> json) {
    return CoverDomain(
      domain: json['domain'] ?? '',
      weight: json['weight'] ?? 10,
    );
  }
}
