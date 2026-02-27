import 'dart:async';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:guarch/app.dart';
import 'package:guarch/providers/app_provider.dart';
import 'package:guarch/services/guarch_engine.dart';

void main() {
  runZonedGuarded(() async {
    WidgetsFlutterBinding.ensureInitialized();

    FlutterLog.d('Main', '====== GUARCH APP STARTING ======');

    // گرفتن خطاهای Flutter framework
    FlutterError.onError = (details) {
      FlutterLog.e('FlutterError', details.exceptionAsString());
      if (details.stack != null) {
        FlutterLog.e('FlutterError', details.stack.toString());
      }
    };

    // ساخت Provider
    final provider = AppProvider();

    FlutterLog.d('Main', 'Calling provider.init()...');
    try {
      await provider.init();
      FlutterLog.d('Main', 'provider.init() completed ✅');
    } catch (e, stack) {
      FlutterLog.e('Main', 'provider.init() FAILED: $e\n$stack');
    }

    FlutterLog.d('Main', 'Starting app...');
    runApp(
      ChangeNotifierProvider.value(
        value: provider,
        child: const GuarchApp(),
      ),
    );
    FlutterLog.d('Main', 'App running ✅');
  }, (error, stackTrace) {
    // هر exception که catch نشه اینجا گیر میافته
    FlutterLog.e('UNCAUGHT', '$error');
    FlutterLog.e('UNCAUGHT', '$stackTrace');
  });
}
