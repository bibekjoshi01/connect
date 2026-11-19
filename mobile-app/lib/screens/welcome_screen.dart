import 'package:flutter/material.dart';

import '../ui/components.dart';

class WelcomeScreen extends StatelessWidget {
  const WelcomeScreen({super.key, required this.onStart});
  final VoidCallback onStart;

  @override
  Widget build(BuildContext context) => Scaffold(
    body: SafeArea(
      child: PageContent(
        children: [
          const SizedBox(height: 96),
          const Align(
            alignment: Alignment.centerLeft,
            child: VoiceMark(size: 64),
          ),
          const SizedBox(height: 28),
          Text('Connect', style: Theme.of(context).textTheme.headlineLarge),
          const SizedBox(height: 12),
          const SupportingText('Voice calls, with a Call ID.'),
          const SizedBox(height: 64),
          FilledButton(
            key: const ValueKey('get-started'),
            onPressed: onStart,
            child: const Text('Get started'),
          ),
          const SizedBox(height: 20),
          Text(
            'Explore the design. No account or calls will be created.',
            style: Theme.of(context).textTheme.bodyMedium?.copyWith(
              color: Theme.of(context).colorScheme.onSurfaceVariant,
            ),
          ),
        ],
      ),
    ),
  );
}
