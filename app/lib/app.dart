import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:guarch/providers/app_provider.dart';
import 'package:guarch/screens/home_screen.dart';

// ═══ Brand Colors ═══
const Color kGold = Color(0xFFF8BC6C);
const Color kDarkBg = Color(0xFF1E3543);
const Color kDarkCard = Color(0xFF263F4F);
const Color kDarkSurface = Color(0xFF172B37);
const Color kGoldDim = Color(0xFFBF8A3E);
const Color kGoldLight = Color(0xFFFDD89B);

// ═══ Light Mode Colors ═══
const Color kLightBg = Color(0xFFF5E0B0);
const Color kLightCard = Color(0xFFFFF0D4);
const Color kLightSurface = Color(0xFFEDD5A0);

// ═══ Theme-aware color helpers ═══

Color textPrimary(BuildContext context) {
  return Theme.of(context).brightness == Brightness.dark ? kGold : kDarkBg;
}

Color textSecondary(BuildContext context) {
  return Theme.of(context).brightness == Brightness.dark
      ? kGoldLight
      : kDarkCard;
}

Color textMuted(BuildContext context) {
  return Theme.of(context).brightness == Brightness.dark
      ? kGold.withOpacity(0.5)
      : kDarkBg.withOpacity(0.5);
}

Color accentColor(BuildContext context) {
  return Theme.of(context).brightness == Brightness.dark ? kGold : kDarkBg;
}

Color buttonBg(BuildContext context) {
  return Theme.of(context).brightness == Brightness.dark ? kGold : kDarkBg;
}

Color buttonFg(BuildContext context) {
  return Theme.of(context).brightness == Brightness.dark ? kDarkBg : kGold;
}

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
        surface: kLightBg,
        onSurface: kDarkBg,
        primary: kDarkBg,
        onPrimary: kGold,
        secondary: kGoldDim,
      ),
      scaffoldBackgroundColor: kLightBg,
      cardTheme: CardTheme(
        color: kLightCard,
        elevation: 0,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(16),
          side: BorderSide(color: kDarkBg.withOpacity(0.08)),
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
        iconTheme: IconThemeData(color: kDarkBg),
      ),
      navigationBarTheme: NavigationBarThemeData(
        backgroundColor: kLightCard,
        indicatorColor: kDarkBg.withOpacity(0.15),
        labelBehavior: NavigationDestinationLabelBehavior.alwaysShow,
        iconTheme: WidgetStateProperty.resolveWith((states) {
          if (states.contains(WidgetState.selected)) {
            return IconThemeData(color: kDarkBg);
          }
          return IconThemeData(color: kDarkBg.withOpacity(0.35));
        }),
        labelTextStyle: WidgetStateProperty.resolveWith((states) {
          if (states.contains(WidgetState.selected)) {
            return TextStyle(color: kDarkBg, fontSize: 12, fontWeight: FontWeight.w600);
          }
          return TextStyle(color: kDarkBg.withOpacity(0.35), fontSize: 12);
        }),
      ),
      iconTheme: IconThemeData(color: kDarkBg),
      dividerColor: kDarkBg.withOpacity(0.08),
      switchTheme: SwitchThemeData(
        thumbColor: WidgetStateProperty.resolveWith((states) {
          if (states.contains(WidgetState.selected)) return kDarkBg;
          return kDarkBg.withOpacity(0.3);
        }),
        trackColor: WidgetStateProperty.resolveWith((states) {
          if (states.contains(WidgetState.selected)) {
            return kDarkBg.withOpacity(0.3);
          }
          return kDarkBg.withOpacity(0.1);
        }),
      ),
      inputDecorationTheme: InputDecorationTheme(
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: BorderSide(color: kDarkBg.withOpacity(0.2)),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: BorderSide(color: kDarkBg.withOpacity(0.2)),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: BorderSide(color: kDarkBg, width: 2),
        ),
        labelStyle: TextStyle(color: kDarkBg.withOpacity(0.6)),
        hintStyle: TextStyle(color: kDarkBg.withOpacity(0.3)),
        prefixIconColor: kDarkBg.withOpacity(0.5),
      ),
      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(
          backgroundColor: kDarkBg,
          foregroundColor: kGold,
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(12),
          ),
        ),
      ),
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          foregroundColor: kDarkBg,
          side: BorderSide(color: kDarkBg.withOpacity(0.3)),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(12),
          ),
        ),
      ),
      snackBarTheme: SnackBarThemeData(
        backgroundColor: kDarkBg,
        contentTextStyle: const TextStyle(color: kGold),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(12),
        ),
      ),
      chipTheme: ChipThemeData(
        backgroundColor: kLightCard,
        labelStyle: TextStyle(color: kDarkBg, fontSize: 12),
        side: BorderSide(color: kDarkBg.withOpacity(0.1)),
      ),
      dialogTheme: DialogTheme(
        backgroundColor: kLightCard,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(16),
        ),
      ),
      tabBarTheme: TabBarTheme(
        labelColor: kDarkBg,
        unselectedLabelColor: kDarkBg.withOpacity(0.35),
        indicatorColor: kDarkBg,
      ),
      floatingActionButtonTheme: FloatingActionButtonThemeData(
        backgroundColor: kDarkBg,
        foregroundColor: kGold,
      ),
    );
  }
}
