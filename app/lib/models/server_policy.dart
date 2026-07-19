class ServerPolicy {
  final bool coverEnabled;
  final String? coverMode;
  final int? coverDomainsCount;
  final bool sniEnabled;
  final String? sniMode;
  final int? sniDomainsCount;
  final bool dnsEnabled;
  final bool utlsEnabled;
  final bool fragmentEnabled;
  final bool paddingEnabled;
  final int? maxPadding;

  ServerPolicy({
    this.coverEnabled = false,
    this.coverMode,
    this.coverDomainsCount,
    this.sniEnabled = false,
    this.sniMode,
    this.sniDomainsCount,
    this.dnsEnabled = false,
    this.utlsEnabled = false,
    this.fragmentEnabled = false,
    this.paddingEnabled = false,
    this.maxPadding,
  });

  factory ServerPolicy.fromJson(Map<String, dynamic> json) {
    return ServerPolicy(
      coverEnabled: json['cover_enabled'] as bool? ?? false,
      coverMode: json['cover_mode'] as String?,
      coverDomainsCount: json['cover_domains_count'] as int?,
      sniEnabled: json['sni_enabled'] as bool? ?? false,
      sniMode: json['sni_mode'] as String?,
      sniDomainsCount: json['sni_domains_count'] as int?,
      dnsEnabled: json['dns_enabled'] as bool? ?? false,
      utlsEnabled: json['utls_enabled'] as bool? ?? false,
      fragmentEnabled: json['fragment_enabled'] as bool? ?? false,
      paddingEnabled: json['padding_enabled'] as bool? ?? false,
      maxPadding: json['max_padding'] as int?,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'cover_enabled': coverEnabled,
      'cover_mode': coverMode,
      'cover_domains_count': coverDomainsCount,
      'sni_enabled': sniEnabled,
      'sni_mode': sniMode,
      'sni_domains_count': sniDomainsCount,
      'dns_enabled': dnsEnabled,
      'utls_enabled': utlsEnabled,
      'fragment_enabled': fragmentEnabled,
      'padding_enabled': paddingEnabled,
      'max_padding': maxPadding,
    };
  }

  bool get hasLockedSettings {
    return coverEnabled || sniEnabled || dnsEnabled || utlsEnabled || 
           fragmentEnabled || paddingEnabled;
  }

  int get lockedSettingsCount {
    int count = 0;
    if (coverEnabled) count++;
    if (sniEnabled) count++;
    if (dnsEnabled) count++;
    if (utlsEnabled) count++;
    if (fragmentEnabled) count++;
    if (paddingEnabled) count++;
    return count;
  }
}
