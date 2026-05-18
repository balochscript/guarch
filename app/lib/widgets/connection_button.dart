import 'package:flutter/material.dart';
import 'package:guarch/app.dart';
import 'package:guarch/models/connection_state.dart';

class ConnectionButton extends StatefulWidget {
  final VpnStatus status;
  final VoidCallback onTap;

  const ConnectionButton({
    super.key,
    required this.status,
    required this.onTap,
  });

  @override
  State<ConnectionButton> createState() => _ConnectionButtonState();
}

class _ConnectionButtonState extends State<ConnectionButton>
    with SingleTickerProviderStateMixin {
  late AnimationController _controller;
  late Animation<double> _scaleAnimation;
  late Animation<double> _pulseAnimation;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 2000),
    );

    _scaleAnimation = Tween<double>(begin: 0.95, end: 1.0).animate(
      CurvedAnimation(parent: _controller, curve: Curves.easeInOut),
    );

    _pulseAnimation = Tween<double>(begin: 1.0, end: 1.2).animate(
      CurvedAnimation(parent: _controller, curve: Curves.easeInOut),
    );

    if (widget.status == VpnStatus.connecting) {
      _controller.repeat(reverse: true);
    }
  }

  @override
  void didUpdateWidget(ConnectionButton oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.status == VpnStatus.connecting) {
      _controller.repeat(reverse: true);
    } else {
      _controller.stop();
      _controller.value = 0;
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final isConnected = widget.status == VpnStatus.connected;
    final isConnecting = widget.status == VpnStatus.connecting;
    final isDisconnecting = widget.status == VpnStatus.disconnecting;
    final isError = widget.status == VpnStatus.error;

    Color buttonColor;
    Color shadowColor;
    IconData icon;
    double size = 120;

    if (isConnected) {
      buttonColor = Colors.green;
      shadowColor = Colors.green.withOpacity(0.5);
      icon = Icons.shield_outlined;
    } else if (isConnecting || isDisconnecting) {
      buttonColor = accentColor(context);
      shadowColor = accentColor(context).withOpacity(0.5);
      icon = Icons.sync;
    } else if (isError) {
      buttonColor = Colors.red;
      shadowColor = Colors.red.withOpacity(0.5);
      icon = Icons.error_outline;
    } else {
      buttonColor = Colors.grey.shade700;
      shadowColor = Colors.grey.withOpacity(0.3);
      icon = Icons.power_settings_new;
    }

    return GestureDetector(
      onTap: (isConnecting || isDisconnecting) ? null : widget.onTap,
      child: AnimatedBuilder(
        animation: _controller,
        builder: (context, child) {
          return Stack(
            alignment: Alignment.center,
            children: [
              if (isConnecting)
                Container(
                  width: size * _pulseAnimation.value,
                  height: size * _pulseAnimation.value,
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    color: shadowColor.withOpacity(0.2),
                  ),
                ),

              ScaleTransition(
                scale: isConnecting ? _scaleAnimation : const AlwaysStoppedAnimation(1.0),
                child: Container(
                  width: size,
                  height: size,
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    gradient: LinearGradient(
                      begin: Alignment.topLeft,
                      end: Alignment.bottomRight,
                      colors: [
                        buttonColor,
                        buttonColor.withOpacity(0.7),
                      ],
                    ),
                    boxShadow: [
                      BoxShadow(
                        color: shadowColor,
                        blurRadius: 20,
                        spreadRadius: 5,
                      ),
                    ],
                  ),
                  child: isConnecting || isDisconnecting
                      ? const Center(
                          child: CircularProgressIndicator(
                            color: Colors.white,
                            strokeWidth: 3,
                          ),
                        )
                      : Icon(
                          icon,
                          size: 50,
                          color: Colors.white,
                        ),
                ),
              ),

              if (isConnected)
                Positioned(
                  bottom: 0,
                  right: 0,
                  child: Container(
                    padding: const EdgeInsets.all(8),
                    decoration: BoxDecoration(
                      color: Colors.green,
                      shape: BoxShape.circle,
                      border: Border.all(
                        color: Colors.white,
                        width: 3,
                      ),
                    ),
                    child: const Text(
                      '🎯',
                      style: TextStyle(fontSize: 20),
                    ),
                  ),
                ),

              if (isError)
                Positioned(
                  bottom: 0,
                  right: 0,
                  child: Container(
                    padding: const EdgeInsets.all(8),
                    decoration: BoxDecoration(
                      color: Colors.red,
                      shape: BoxShape.circle,
                      border: Border.all(
                        color: Colors.white,
                        width: 3,
                      ),
                    ),
                    child: const Text(
                      '❌',
                      style: TextStyle(fontSize: 20),
                    ),
                  ),
                ),
            ],
          );
        },
      ),
    );
  }
}
