import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:guarch/providers/app_provider.dart';
import 'package:guarch/screens/home_screen.dart';

const Color kGold = Color(0xFFF8BC6C);
const Color kDarkBg = Color(0xFF1E3543);
const Color kDarkCard = Color(0xFF263F4F);
const Color kDarkSurface = Color(0xFF172B37);
const Color kGoldDim = Color(0xFFBF8A3E);
const Color kGoldLight = Color(0xFFFDD89B);

class GuarchApp extends StatelessWidget {
  const GuarchApp({super.key});

  @override
  Widget build(BuildContext context) {
    return Consumer<AppProvider>(
      builder: (context, provider, _) {
        return MaterialApp(
          title: 'Guarch',
          debugShowCheckedModeBanner: false,
          themeMode: provider.isDarkMode ? ThemeMode.dark : ThemeMode.light,
          theme: _lightTheme(),
          darkTheme: _darkTheme(),
          home: const HomeScreen(),
        );
      },
    );
  }

  ThemeData _darkTheme() {
    return ThemeData(
      useMaterial3: true,
      brightness: Brightness.dark,
      textTheme: GoogleFonts.interTextTheme(ThemeData.dark().textTheme),
      colorScheme: ColorScheme.fromSeed(
        seedColor: kGold,
        brightness: Brightness.dark,
        surface: kDarkBg,
        onSurface: Colors.white,
        primary: kGold,
        onPrimary: kDarkBg,
        secondary: kGoldDim,
      ),
      scaffoldBackgroundColor: kDarkSurface,
      cardTheme: CardTheme(
        color: kDarkCard,
        elevation: 0,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(16),
        ),
      ),
      appBarTheme: const AppBarTheme(
        backgroundColor: Colors.transparent,
        elevation: 0,
        centerTitle: true,
        foregroundColor: kGold,
        titleTextStyle: TextStyle(
          color: kGold,
          fontSize: 20,
          fontWeight: FontWeight.w600,
        ),
        iconTheme: IconThemeData(color: kGold),
      ),
      navigationBarTheme: NavigationBarThemeData(
        backgroundColor: kDarkBg,
        indicatorColor: kGold.withOpacity(0.2),
        labelBehavior: NavigationDestinationLabelBehavior.alwaysShow,
        iconTheme: WidgetStateProperty.resolveWith((states) {
          if (states.contains(WidgetState.selected)) {
            return const IconThemeData(color: kGold);
          }
          return IconThemeData(color: Colors.grey.shade500);
        }),
        labelTextStyle: WidgetStateProperty.resolveWith((states) {
          if (states.contains(WidgetState.selected)) {
            return const TextStyle(color: kGold, fontSize: 12);
          }
          return TextStyle(color: Colors.grey.shade500, fontSize: 12);
        }),
      ),
      iconTheme: const IconThemeData(color: kGold),
      dividerColor: kGold.withOpacity(0.1),
      switchTheme: SwitchThemeData(
        thumbColor: WidgetStateProperty.resolveWith((states) {
          if (states.contains(WidgetState.selected)) return kGold;
          return Colors.grey;
        }),
        trackColor: WidgetStateProperty.resolveWith((states) {
          if (states.contains(WidgetState.selected)) {
            return kGold.withOpacity(0.3);
          }
          return Colors.grey.withOpacity(0.3);
        }),
      ),
      inputDecorationTheme: InputDecorationTheme(
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: BorderSide(color: kGold.withOpacity(0.3)),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: BorderSide(color: kGold.withOpacity(0.3)),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: const BorderSide(color: kGold, width: 2),
        ),
        labelStyle: TextStyle(color: kGold.withOpacity(0.7)),
        hintStyle: TextStyle(color: Colors.grey.shade600),
        prefixIconColor: kGold.withOpacity(0.7),
      ),
      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(
          backgroundColor: kGold,
          foregroundColor: kDarkBg,
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(12),
          ),
        ),
      ),
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          foregroundColor: kGold,
          side: BorderSide(color: kGold.withOpacity(0.5)),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(12),
          ),
        ),
      ),
      snackBarTheme: SnackBarThemeData(
        backgroundColor: kDarkCard,
        contentTextStyle: const TextStyle(color: kGoldLight),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(12),
        ),
      ),
      chipTheme: ChipThemeData(
        backgroundColor: kDarkCard,
        labelStyle: const TextStyle(color: kGoldLight, fontSize: 12),
        side: BorderSide(color: kGold.withOpacity(0.2)),
      ),
      dialogTheme: DialogTheme(
        backgroundColor: kDarkBg,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(16),
        ),
      ),
      tabBarTheme: TabBarTheme(
        labelColor: kGold,
        unselectedLabelColor: Colors.grey.shade500,
        indicatorColor: kGold,
      ),
      floatingActionButtonTheme: const FloatingActionButtonThemeData(
        backgroundColor: kGold,
        foregroundColor: kDarkBg,
      ),
    );
  }

  ThemeData _lightTheme() {
    return ThemeData(
      useMaterial3: true,
      brightness: Brightness.light,
      textTheme: GoogleFonts.interTextTheme(ThemeData.light().textTheme),
      colorScheme: ColorScheme.fromSeed(
        seedColor: kGold,
        brightness: Brightness.light,
        primary: kDarkBg,
        secondary: kGoldDim,
      ),
      cardTheme: CardTheme(
        elevation: 0,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(16),
        ),
      ),
      appBarTheme: AppBarTheme(
        backgroundColor: Colors.transparent,
        elevation: 0,
        centerTitle: true,
        foregroundColor: kDarkBg,
        titleTextStyle: TextStyle(
          color: kDarkBg,
          fontSize: 20,
          fontWeight: FontWeight.w600,
        ),
      ),
      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(
          backgroundColor: kDarkBg,
          foregroundColor: kGold,
        ),
      ),
    );
  }
}
