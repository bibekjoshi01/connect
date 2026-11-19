import 'package:flutter/material.dart';

import '../ui/components.dart';

class ProfileSetupScreen extends StatefulWidget {
  const ProfileSetupScreen({super.key, this.initialName = ''});
  final String initialName;

  @override
  State<ProfileSetupScreen> createState() => _ProfileSetupScreenState();
}

class _ProfileSetupScreenState extends State<ProfileSetupScreen> {
  final _formKey = GlobalKey<FormState>();
  late final TextEditingController _name;

  @override
  void initState() {
    super.initState();
    _name = TextEditingController(text: widget.initialName);
  }

  @override
  void dispose() {
    _name.dispose();
    super.dispose();
  }

  void _continue() {
    if (_formKey.currentState!.validate()) {
      Navigator.pop(context, _name.text.toLowerCase());
    }
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(title: const Text('Your profile')),
    body: SafeArea(
      child: PageContent(
        children: [
          const Align(alignment: Alignment.centerLeft, child: PreviewLabel()),
          const SizedBox(height: 28),
          Text(
            'Your username',
            style: Theme.of(context).textTheme.headlineMedium,
          ),
          const SizedBox(height: 12),
          const SupportingText('Choose a name for this preview.'),
          const SizedBox(height: 32),
          Form(
            key: _formKey,
            child: TextFormField(
              key: const ValueKey('username-field'),
              controller: _name,
              autocorrect: false,
              enableSuggestions: false,
              textCapitalization: TextCapitalization.none,
              textInputAction: TextInputAction.done,
              maxLength: 32,
              decoration: textFieldDecoration(
                context,
                label: 'Username',
                hint: 'your_name',
                prefix: '@',
                helper: '3–32 letters, numbers or underscores. Start with a letter.',
              ),
              onFieldSubmitted: (_) => _continue(),
              validator: (value) {
                if (!RegExp(r'^[a-zA-Z][a-zA-Z0-9_]{2,31}$')
                    .hasMatch(value ?? '')) {
                  return 'Use 3–32 letters, numbers or underscores, starting with a letter.';
                }
                return null;
              },
            ),
          ),
          const SizedBox(height: 28),
          FilledButton(
            key: const ValueKey('continue-profile'),
            onPressed: _continue,
            child: Text(
              widget.initialName.isEmpty ? 'Continue' : 'Save changes',
            ),
          ),
          const SizedBox(height: 16),
          Text(
            'This is a local preview. Your username is not registered or saved '
            'across app restarts.',
            style: Theme.of(context).textTheme.bodyMedium?.copyWith(
              color: Theme.of(context).colorScheme.onSurfaceVariant,
            ),
          ),
        ],
      ),
    ),
  );
}
