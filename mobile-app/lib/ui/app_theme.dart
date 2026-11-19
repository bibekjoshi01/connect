import 'package:flutter/material.dart';

abstract final class AppTheme {
  static const callColor = Color(0xFF19834C);

  static ThemeData light() => _build(Brightness.light);
  static ThemeData dark() => _build(Brightness.dark);

  static ThemeData _build(Brightness brightness) {
    final dark = brightness == Brightness.dark;
    final scheme =
        ColorScheme.fromSeed(
          seedColor: const Color(0xFF1668DC),
          brightness: brightness,
        ).copyWith(
          primary: dark ? const Color(0xFF8CB8FF) : const Color(0xFF1668DC),
          onPrimary: dark ? const Color(0xFF002E6A) : Colors.white,
          surface: dark ? const Color(0xFF101010) : Colors.white,
          onSurface: dark ? const Color(0xFFF1F1F1) : const Color(0xFF181818),
          onSurfaceVariant: dark
              ? const Color(0xFFA6A6A6)
              : const Color(0xFF6A6A6A),
          surfaceContainerLowest: dark
              ? const Color(0xFF181818)
              : const Color(0xFFF5F5F5),
          surfaceContainerLow: dark
              ? const Color(0xFF242424)
              : const Color(0xFFF3F3F3),
          outlineVariant: dark
              ? const Color(0xFF303030)
              : const Color(0xFFEAEAEA),
        );
    final base = ThemeData(useMaterial3: true, colorScheme: scheme);
    return base.copyWith(
      scaffoldBackgroundColor: scheme.surface,
      textTheme: base.textTheme.copyWith(
        headlineLarge: base.textTheme.headlineLarge?.copyWith(
          fontSize: 38,
          height: 1.18,
          letterSpacing: -1,
          fontWeight: FontWeight.w400,
        ),
        headlineMedium: base.textTheme.headlineMedium?.copyWith(
          fontSize: 30,
          height: 1.2,
          letterSpacing: -0.5,
          fontWeight: FontWeight.w400,
        ),
        titleLarge: base.textTheme.titleLarge?.copyWith(
          fontSize: 21,
          fontWeight: FontWeight.w500,
        ),
        bodyLarge: base.textTheme.bodyLarge?.copyWith(height: 1.45),
        bodyMedium: base.textTheme.bodyMedium?.copyWith(height: 1.45),
      ),
      appBarTheme: AppBarTheme(
        backgroundColor: scheme.surface,
        foregroundColor: scheme.onSurface,
        elevation: 0,
        scrolledUnderElevation: 0,
        centerTitle: false,
        titleSpacing: 24,
        titleTextStyle: base.textTheme.titleLarge?.copyWith(
          color: scheme.onSurface,
          fontSize: 20,
          fontWeight: FontWeight.w400,
        ),
      ),
      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(
          minimumSize: const Size(0, 52),
          padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 14),
          textStyle: const TextStyle(fontSize: 16, fontWeight: FontWeight.w500),
          shape: const StadiumBorder(),
        ),
      ),
      floatingActionButtonTheme: const FloatingActionButtonThemeData(
        backgroundColor: callColor,
        foregroundColor: Colors.white,
        elevation: 0,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.all(Radius.circular(20)),
        ),
      ),
      navigationBarTheme: NavigationBarThemeData(
        height: 72,
        backgroundColor: scheme.surface,
        indicatorColor: Colors.transparent,
        surfaceTintColor: Colors.transparent,
        elevation: 0,
        iconTheme: WidgetStateProperty.resolveWith(
          (states) => IconThemeData(
            size: 24,
            color: states.contains(WidgetState.selected)
                ? scheme.primary
                : scheme.onSurfaceVariant,
          ),
        ),
        labelTextStyle: WidgetStateProperty.resolveWith(
          (states) => TextStyle(
            color: states.contains(WidgetState.selected)
                ? scheme.primary
                : scheme.onSurfaceVariant,
            fontSize: 12,
            fontWeight: FontWeight.w500,
          ),
        ),
      ),
      listTileTheme: const ListTileThemeData(
        contentPadding: EdgeInsets.symmetric(horizontal: 0, vertical: 6),
      ),
      dividerTheme: DividerThemeData(
        color: scheme.outlineVariant,
        thickness: 0.5,
      ),
    );
  }
}
