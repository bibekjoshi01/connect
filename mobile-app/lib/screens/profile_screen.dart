import 'package:flutter/material.dart';

import '../ui/components.dart';

class ProfileScreen extends StatelessWidget {
  const ProfileScreen({
    super.key,
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

  String get _appearance => switch (themeMode) {
    ThemeMode.system => 'Use device setting',
    ThemeMode.light => 'Light',
    ThemeMode.dark => 'Dark',
  };

  Future<void> _chooseAppearance(BuildContext context) =>
      showModalBottomSheet<void>(
        context: context,
        showDragHandle: true,
        useSafeArea: true,
        isScrollControlled: true,
        builder: (sheetContext) => SingleChildScrollView(
          padding: const EdgeInsets.fromLTRB(12, 8, 12, 24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Padding(
                padding: const EdgeInsets.fromLTRB(12, 0, 12, 12),
                child: Text(
                  'Appearance',
                  style: Theme.of(context).textTheme.titleLarge,
                ),
              ),
              for (final option in const [
                (
                  ThemeMode.system,
                  'Use device setting',
                  Icons.brightness_auto_outlined,
                ),
                (ThemeMode.light, 'Light', Icons.light_mode_outlined),
                (ThemeMode.dark, 'Dark', Icons.dark_mode_outlined),
              ])
                ListTile(
                  key: ValueKey('theme-${option.$1.name}'),
                  leading: Icon(option.$3),
                  title: Text(option.$2),
                  trailing: option.$1 == themeMode
                      ? Icon(
                          Icons.check_rounded,
                          color: Theme.of(context).colorScheme.primary,
                        )
                      : null,
                  selected: option.$1 == themeMode,
                  onTap: () {
                    onThemeChanged(option.$1);
                    Navigator.pop(sheetContext);
                  },
                ),
            ],
          ),
        ),
      );

  Future<void> _reset(BuildContext context) async {
    final reset = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        scrollable: true,
        title: const Text('Start fresh?'),
        content: const Text(
          'This clears your preview name and appearance. No real account is affected.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(dialogContext, false),
            child: const Text('Cancel'),
          ),
          TextButton(
            onPressed: () => Navigator.pop(dialogContext, true),
            child: const Text('Reset preview'),
          ),
        ],
      ),
    );
    if (context.mounted && reset == true) onReset();
  }

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    return PageContent(
      children: [
        const SizedBox(height: 12),
        Center(
          child: CircleAvatar(
            radius: 32,
            backgroundColor: colors.surfaceContainerLow,
            foregroundColor: colors.onSurfaceVariant,
            child: Text(
              username[0].toUpperCase(),
              style: const TextStyle(fontSize: 26),
            ),
          ),
        ),
        const SizedBox(height: 16),
        Text(
          '@$username',
          key: const ValueKey('preview-username'),
          textAlign: TextAlign.center,
          style: Theme.of(context).textTheme.titleLarge,
        ),
        const SizedBox(height: 4),
        Text(
          'Preview profile',
          textAlign: TextAlign.center,
          style: Theme.of(context).textTheme.bodyMedium
              ?.copyWith(color: colors.onSurfaceVariant),
        ),
        const SizedBox(height: 40),
        Text('Call ID', style: Theme.of(context).textTheme.titleSmall),
        const SizedBox(height: 8),
        Text(
          'Your Call ID will appear here when account setup is available.',
          style: Theme.of(context).textTheme.bodyMedium
              ?.copyWith(color: colors.onSurfaceVariant),
        ),
        const SizedBox(height: 24),
        const Divider(),
        ListTile(
          key: const ValueKey('edit-profile'),
          leading: const Icon(Icons.person_outline_rounded),
          title: const Text('Edit profile'),
          trailing: Icon(
            Icons.chevron_right_rounded,
            color: colors.onSurfaceVariant,
          ),
          onTap: onEditProfile,
        ),
        ListTile(
          key: const ValueKey('appearance'),
          leading: const Icon(Icons.contrast_rounded),
          title: const Text('Appearance'),
          subtitle: Text(_appearance),
          trailing: Icon(
            Icons.chevron_right_rounded,
            color: colors.onSurfaceVariant,
          ),
          onTap: () => _chooseAppearance(context),
        ),
        const Divider(),
        const SizedBox(height: 12),
        Align(
          alignment: Alignment.centerLeft,
          child: TextButton(
            key: const ValueKey('reset-preview'),
            onPressed: () => _reset(context),
            child: const Text('Reset preview'),
          ),
        ),
      ],
    );
  }
}
