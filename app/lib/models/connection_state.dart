enum VpnStatus {
  disconnected,
  connecting,
  connected,
  disconnecting,
  error,
}

extension VpnStatusExtension on VpnStatus {
  String get displayName {
    switch (this) {
      case VpnStatus.disconnected:
        return 'Disconnected';
      case VpnStatus.connecting:
        return 'Connecting...';
      case VpnStatus.connected:
        return 'Connected';
      case VpnStatus.disconnecting:
        return 'Disconnecting...';
      case VpnStatus.error:
        return 'Error';
    }
  }

  String get emoji {
    switch (this) {
      case VpnStatus.disconnected:
        return '⚫';
      case VpnStatus.connecting:
        return '🟡';
      case VpnStatus.connected:
        return '🟢';
      case VpnStatus.disconnecting:
        return '🟠';
      case VpnStatus.error:
        return '🔴';
    }
  }
}

class ConnectionStats {
  final int uploadSpeed;
  final int downloadSpeed;
  final int totalUpload;
  final int totalDownload;
  final Duration duration;
  final int coverRequests;
  
  // Enhanced stats (v1.0.1)
  final String currentSNI;
  final int sniSwitches;
  final String activityLevel;
  final bool dnsFallbackUsed;
  final int activeStreams;
  final int totalConnections;

  const ConnectionStats({
    this.uploadSpeed = 0,
    this.downloadSpeed = 0,
    this.totalUpload = 0,
    this.totalDownload = 0,
    this.duration = Duration.zero,
    this.coverRequests = 0,
    this.currentSNI = '',
    this.sniSwitches = 0,
    this.activityLevel = 'idle',
    this.dnsFallbackUsed = false,
    this.activeStreams = 0,
    this.totalConnections = 0,
  });

  // Formatting methods
  String get uploadSpeedText => _formatSpeed(uploadSpeed);
  String get downloadSpeedText => _formatSpeed(downloadSpeed);
  String get totalUploadText => _formatBytes(totalUpload);
  String get totalDownloadText => _formatBytes(totalDownload);
  String get durationText => _formatDuration(duration);

  String get activityEmoji {
    switch (activityLevel.toLowerCase()) {
      case 'idle':
        return '🟢';
      case 'light':
        return '🟡';
      case 'medium':
        return '🟠';
      case 'heavy':
        return '🔴';
      default:
        return '⚪';
    }
  }

  String get activityText {
    switch (activityLevel.toLowerCase()) {
      case 'idle':
        return 'Idle';
      case 'light':
        return 'Light';
      case 'medium':
        return 'Medium';
      case 'heavy':
        return 'Heavy';
      default:
        return 'Unknown';
    }
  }

  String _formatSpeed(int bytesPerSecond) {
    if (bytesPerSecond < 1024) return '$bytesPerSecond B/s';
    if (bytesPerSecond < 1024 * 1024) {
      return '${(bytesPerSecond / 1024).toStringAsFixed(1)} KB/s';
    }
    return '${(bytesPerSecond / 1024 / 1024).toStringAsFixed(2)} MB/s';
  }

  String _formatBytes(int bytes) {
    if (bytes < 1024) return '$bytes B';
    if (bytes < 1024 * 1024) {
      return '${(bytes / 1024).toStringAsFixed(1)} KB';
    }
    if (bytes < 1024 * 1024 * 1024) {
      return '${(bytes / 1024 / 1024).toStringAsFixed(2)} MB';
    }
    return '${(bytes / 1024 / 1024 / 1024).toStringAsFixed(2)} GB';
  }

  String _formatDuration(Duration d) {
    final hours = d.inHours.toString().padLeft(2, '0');
    final minutes = (d.inMinutes % 60).toString().padLeft(2, '0');
    final seconds = (d.inSeconds % 60).toString().padLeft(2, '0');
    return '$hours:$minutes:$seconds';
  }

  ConnectionStats copyWith({
    int? uploadSpeed,
    int? downloadSpeed,
    int? totalUpload,
    int? totalDownload,
    Duration? duration,
    int? coverRequests,
    String? currentSNI,
    int? sniSwitches,
    String? activityLevel,
    bool? dnsFallbackUsed,
    int? activeStreams,
    int? totalConnections,
  }) {
    return ConnectionStats(
      uploadSpeed: uploadSpeed ?? this.uploadSpeed,
      downloadSpeed: downloadSpeed ?? this.downloadSpeed,
      totalUpload: totalUpload ?? this.totalUpload,
      totalDownload: totalDownload ?? this.totalDownload,
      duration: duration ?? this.duration,
      coverRequests: coverRequests ?? this.coverRequests,
      currentSNI: currentSNI ?? this.currentSNI,
      sniSwitches: sniSwitches ?? this.sniSwitches,
      activityLevel: activityLevel ?? this.activityLevel,
      dnsFallbackUsed: dnsFallbackUsed ?? this.dnsFallbackUsed,
      activeStreams: activeStreams ?? this.activeStreams,
      totalConnections: totalConnections ?? this.totalConnections,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'upload_speed': uploadSpeed,
      'download_speed': downloadSpeed,
      'total_upload': totalUpload,
      'total_download': totalDownload,
      'duration_seconds': duration.inSeconds,
      'cover_requests': coverRequests,
      'current_sni': currentSNI,
      'sni_switches': sniSwitches,
      'activity_level': activityLevel,
      'dns_fallback_used': dnsFallbackUsed,
      'active_streams': activeStreams,
      'total_connections': totalConnections,
    };
  }

  factory ConnectionStats.fromJson(Map<String, dynamic> json) {
    return ConnectionStats(
      uploadSpeed: json['upload_speed'] as int? ?? 0,
      downloadSpeed: json['download_speed'] as int? ?? 0,
      totalUpload: json['total_upload'] as int? ?? 0,
      totalDownload: json['total_download'] as int? ?? 0,
      duration: Duration(seconds: json['duration_seconds'] as int? ?? 0),
      coverRequests: json['cover_requests'] as int? ?? 0,
      currentSNI: json['current_sni'] as String? ?? '',
      sniSwitches: json['sni_switches'] as int? ?? 0,
      activityLevel: json['activity_level'] as String? ?? 'idle',
      dnsFallbackUsed: json['dns_fallback'] as bool? ?? false,
      activeStreams: json['active_streams'] as int? ?? 0,
      totalConnections: json['total_connections'] as int? ?? 0,
    );
  }
}
