import 'package:flutter/material.dart';

import '../ui/components.dart';

class CallsScreen extends StatelessWidget {
  const CallsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return PageContent(
      children: [
        Text(
          'Call history',
          style: theme.textTheme.labelLarge?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
        const SizedBox(height: 100),
        Icon(
          Icons.history_rounded,
          size: 40,
          color: theme.colorScheme.onSurfaceVariant,
        ),
        const SizedBox(height: 20),
        Text(
          'No calls yet',
          textAlign: TextAlign.center,
          style: theme.textTheme.titleMedium,
        ),
        const SizedBox(height: 8),
        Text(
          'Your recent calls will appear here.',
          textAlign: TextAlign.center,
          style: theme.textTheme.bodyMedium?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
        const SizedBox(height: 20),
        Center(
          child: TextButton(
            onPressed: () => showInfoSheet(
              context,
              title: 'Call ID',
              message:
                  'Use a shared Call ID to find someone on Connect. '
                  'It has 32 characters, using 0–9 and a–f. '
                  'Calling is unavailable in this preview.',
            ),
            child: const Text('What is a Call ID?'),
          ),
        ),
      ],
    );
  }
}
