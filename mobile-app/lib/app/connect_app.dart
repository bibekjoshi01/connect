import 'package:flutter/material.dart';

import '../screens/calls_screen.dart';
import '../screens/new_call_screen.dart';
import '../screens/profile_screen.dart';
import '../screens/profile_setup_screen.dart';
import '../screens/welcome_screen.dart';
import '../ui/app_theme.dart';
import '../ui/components.dart';

/// Presentation-only prototype. All state is local and discarded on restart.
class ConnectApp extends StatefulWidget {
  const ConnectApp({super.key});

  @override
  State<ConnectApp> createState() => _ConnectAppState();
}

class _ConnectAppState extends State<ConnectApp> {
  String? _username;
  ThemeMode _themeMode = ThemeMode.system;

  @override
  Widget build(BuildContext context) => MaterialApp(
    title: 'Connect',
    debugShowCheckedModeBanner: false,
    theme: AppTheme.light(),
    darkTheme: AppTheme.dark(),
    themeMode: _themeMode,
    home: Builder(
      builder: (context) => _username == null
          ? WelcomeScreen(onStart: () => _editProfile(context))
          : _AppHome(
              username: _username!,
              themeMode: _themeMode,
              onEditProfile: () => _editProfile(context),
              onThemeChanged: (mode) => setState(() => _themeMode = mode),
              onReset: () => setState(() {
                _username = null;
                _themeMode = ThemeMode.system;
              }),
            ),
    ),
  );

  Future<void> _editProfile(BuildContext context) async {
    final name = await Navigator.of(context).push<String>(
      MaterialPageRoute(
        builder: (_) => ProfileSetupScreen(initialName: _username ?? ''),
      ),
    );
    if (mounted && name != null) setState(() => _username = name);
  }
}

class _AppHome extends StatefulWidget {
  const _AppHome({
    required this.username,
    required this.themeMode,
    required this.onEditProfile,
    required this.onThemeChanged,
    required this.onReset,
  });

  final String username;
  final ThemeMode themeMode;
  final VoidCallback onEditProfile;
  final ValueChanged<ThemeMode> onThemeChanged;
  final VoidCallback onReset;

  @override
  State<_AppHome> createState() => _AppHomeState();
}

class _AppHomeState extends State<_AppHome> {
  int _selectedPage = 0;

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(
      toolbarHeight: 48 + MediaQuery.textScalerOf(context).scale(32),
      title: Text(
        _selectedPage == 0 ? 'Recents' : 'Settings',
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
        style: Theme.of(context).textTheme.headlineLarge,
      ),
      actions: [
        IconButton(
          tooltip: 'About this preview',
          icon: const Icon(Icons.info_outline_rounded),
          onPressed: () => showInfoSheet(
            context,
            title: 'A first look at Connect',
            message:
                'Explore the screens and try the controls. Account setup '
                'and calling are not available in this design preview.',
          ),
        ),
        const SizedBox(width: 8),
      ],
    ),
    body: SafeArea(
      child: IndexedStack(
        index: _selectedPage,
        children: [
          const CallsScreen(),
          ProfileScreen(
            username: widget.username,
            themeMode: widget.themeMode,
            onEditProfile: widget.onEditProfile,
            onThemeChanged: widget.onThemeChanged,
            onReset: widget.onReset,
          ),
        ],
      ),
    ),
    floatingActionButton: _selectedPage == 0
        ? FloatingActionButton(
            key: const ValueKey('new-call'),
            tooltip: 'New call',
            onPressed: () => Navigator.of(context).push<void>(
              MaterialPageRoute(builder: (_) => const NewCallScreen()),
            ),
            child: const Icon(Icons.add_call),
          )
        : null,
    bottomNavigationBar: NavigationBar(
      selectedIndex: _selectedPage,
      onDestinationSelected: (index) => setState(() => _selectedPage = index),
      destinations: const [
        NavigationDestination(
          key: ValueKey('calls-tab'),
          icon: Icon(Icons.call_outlined),
          selectedIcon: Icon(Icons.call_rounded),
          label: 'Recents',
        ),
        NavigationDestination(
          key: ValueKey('profile-tab'),
          icon: Icon(Icons.settings_outlined),
          selectedIcon: Icon(Icons.settings_rounded),
          label: 'Settings',
        ),
      ],
    ),
  );
}
