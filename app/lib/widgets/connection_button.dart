import 'package:flutter/material.dart';
import 'package:guarch/models/connection_state.dart';
import 'package:guarch/app.dart';

class ConnectionButton extends StatefulWidget {
  final VpnStatus status;
  final VoidCallback onTap;

  const ConnectionButton({super.key, required this.status, required this.onTap});

  @override
  State<ConnectionButton> createState() => _ConnectionButtonState();
}

class _ConnectionButtonState extends State<ConnectionButton> with SingleTickerProviderStateMixin {
  late AnimationController _controller;
  late Animation<double> _scaleAnimation;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(duration: const Duration(milliseconds: 150), vsync: this);
    _scaleAnimation = Tween<double>(begin: 1.0, end: 0.92).animate(CurvedAnimation(parent: _controller, curve: Curves.easeInOut));
  }

  @override
  void dispose() { _controller.dispose(); super.dispose(); }

  Color _mainColor(BuildContext context) {
    switch (widget.status) {
      case VpnStatus.connected: return Colors.green;
      case VpnStatus.connecting:
      case VpnStatus.disconnecting: return accentColor(context);
      case VpnStatus.error: return Colors.red;
      case VpnStatus.disconnected: return accentColor(context);
    }
  }

  IconData get _icon {
    switch (widget.status) {
      case VpnStatus.connected: return Icons.power_settings_new;
      case VpnStatus.connecting:
      case VpnStatus.disconnecting: return Icons.hourglass_top;
      case VpnStatus.error: return Icons.error_outline;
      case VpnStatus.disconnected: return Icons.power_settings_new;
    }
  }

  bool get _isLoading => widget.status == VpnStatus.connecting || widget.status == VpnStatus.disconnecting;

  @override
  Widget build(BuildContext context) {
    final color = _mainColor(context);
    final bgColor = Theme.of(context).scaffoldBackgroundColor;

    return GestureDetector(
      onTapDown: (_) => _controller.forward(),
      onTapUp: (_) { _controller.reverse(); widget.onTap(); },
      onTapCancel: () => _controller.reverse(),
      child: ScaleTransition(
        scale: _scaleAnimation,
        child: Container(
          width: 160, height: 160,
          decoration: BoxDecoration(
            shape: BoxShape.circle,
            gradient: RadialGradient(colors: [color.withOpacity(0.3), color.withOpacity(0.05)]),
          ),
          child: Center(
            child: Container(
              width: 120, height: 120,
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                gradient: LinearGradient(begin: Alignment.topLeft, end: Alignment.bottomRight, colors: [color, color.withOpacity(0.7)]),
                boxShadow: [BoxShadow(color: color.withOpacity(0.4), blurRadius: 20, spreadRadius: 2)],
              ),
              child: _isLoading
                  ? Padding(padding: const EdgeInsets.all(35), child: CircularProgressIndicator(color: bgColor, strokeWidth: 3))
                  : Icon(_icon, color: bgColor, size: 48),
            ),
          ),
        ),
      ),
    );
  }
}
