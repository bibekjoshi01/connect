import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../ui/app_theme.dart';
import '../ui/components.dart';
import 'call_preview_screen.dart';

class NewCallScreen extends StatefulWidget {
  const NewCallScreen({super.key});

  @override
  State<NewCallScreen> createState() => _NewCallScreenState();
}

class _NewCallScreenState extends State<NewCallScreen> {
  String _callId = '';
  String? _error;

  void _append(String character) {
    if (_callId.length == 32) return;
    setState(() {
      _callId += character;
      _error = null;
    });
  }

  void _delete() => setState(() {
    if (_callId.isNotEmpty) {
      _callId = _callId.substring(0, _callId.length - 1);
    }
    _error = null;
  });

  void _clear() => setState(() {
    _callId = '';
    _error = null;
  });

  Future<void> _paste() async {
    try {
      final clipboard = await Clipboard.getData(Clipboard.kTextPlain);
      if (!mounted) return;
      final value = clipboard?.text?.trim().toLowerCase() ?? '';
      setState(() {
        if (RegExp(r'^[0-9a-f]{32}$').hasMatch(value)) {
          _callId = value;
          _error = null;
        } else {
          _error = 'Paste a 32-character Call ID using 0–9 and A–F.';
        }
      });
    } on PlatformException {
      if (mounted) {
        setState(
          () => _error = 'Unable to paste. Use the keypad to enter a Call ID.',
        );
      }
    }
  }

  Future<void> _preview() async {
    if (_callId.length != 32) {
      setState(() => _error = 'Enter a 32-character Call ID.');
      return;
    }
    final close = await Navigator.of(context).push<bool>(
      MaterialPageRoute(builder: (_) => CallPreviewScreen(callId: _callId)),
    );
    if (mounted && close == true) Navigator.pop(context);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;
    final groupedId = [
      for (var i = 0; i < _callId.length; i += 8)
        _callId.substring(i, math.min(i + 8, _callId.length)),
    ].join(' ');
    return Scaffold(
      appBar: AppBar(title: const Text('New call')),
      body: SafeArea(
        child: PageContent(
          children: [
            const SizedBox(height: 12),
            ConstrainedBox(
              constraints: const BoxConstraints(minHeight: 88),
              child: Center(
                child: Text(
                  _callId.isEmpty ? 'Call ID' : groupedId.toUpperCase(),
                  key: const ValueKey('call-id-display'),
                  textAlign: TextAlign.center,
                  style: theme.textTheme.headlineMedium?.copyWith(
                    fontSize: 28,
                    fontFamily: _callId.isEmpty ? null : 'monospace',
                    color: _callId.isEmpty
                        ? colors.onSurfaceVariant
                        : colors.onSurface,
                  ),
                ),
              ),
            ),
            Text(
              '${_callId.length}/32',
              key: const ValueKey('call-id-count'),
              textAlign: TextAlign.center,
              style: theme.textTheme.bodySmall?.copyWith(
                color: colors.onSurfaceVariant,
              ),
            ),
            if (_error != null) ...[
              const SizedBox(height: 12),
              Semantics(
                liveRegion: true,
                child: Text(
                  _error!,
                  key: const ValueKey('call-id-error'),
                  textAlign: TextAlign.center,
                  style: theme.textTheme.bodyMedium?.copyWith(
                    color: colors.error,
                  ),
                ),
              ),
            ],
            const SizedBox(height: 24),
            // Call IDs are hexadecimal. Keep every valid character available
            // without a system keyboard or changing the shared ID format.
            LayoutBuilder(
              builder: (context, constraints) {
                final letterWidth = math.max(
                  48.0,
                  MediaQuery.textScalerOf(context).scale(18) + 24,
                );
                final columns = constraints.maxWidth >= letterWidth * 6 ? 6 : 3;
                return Wrap(
                  children: [
                    for (final letter in ['a', 'b', 'c', 'd', 'e', 'f'])
                      SizedBox(
                        width: constraints.maxWidth / columns,
                        child: _DialKey(
                          key: ValueKey('dial-$letter'),
                          label: letter.toUpperCase(),
                          fontSize: 18,
                          height: 48,
                          onPressed: () => _append(letter),
                        ),
                      ),
                  ],
                );
              },
            ),
            const SizedBox(height: 8),
            for (final row in ['123', '456', '789'])
              Row(
                children: [
                  for (final digit in row.split(''))
                    Expanded(
                      child: _DialKey(
                        key: ValueKey('dial-$digit'),
                        label: digit,
                        onPressed: () => _append(digit),
                      ),
                    ),
                ],
              ),
            Row(
              children: [
                Expanded(
                  child: _DialKey(
                    key: const ValueKey('paste-call-id'),
                    icon: Icons.content_paste_rounded,
                    tooltip: 'Paste Call ID',
                    onPressed: _paste,
                  ),
                ),
                Expanded(
                  child: _DialKey(
                    key: const ValueKey('dial-0'),
                    label: '0',
                    onPressed: () => _append('0'),
                  ),
                ),
                Expanded(
                  child: _DialKey(
                    key: const ValueKey('delete-call-id'),
                    icon: Icons.backspace_outlined,
                    tooltip: 'Delete last character. Hold to clear.',
                    onPressed: _callId.isEmpty ? null : _delete,
                    onLongPress: _callId.isEmpty ? null : _clear,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 20),
            Center(
              child: IconButton.filled(
                key: const ValueKey('preview-call'),
                tooltip: 'Preview call',
                onPressed: _preview,
                style: IconButton.styleFrom(
                  backgroundColor: AppTheme.callColor,
                  foregroundColor: Colors.white,
                  padding: const EdgeInsets.all(22),
                  shape: const CircleBorder(),
                ),
                iconSize: 28,
                icon: const Icon(Icons.call_rounded),
              ),
            ),
            const SizedBox(height: 12),
            Text(
              'Preview call',
              textAlign: TextAlign.center,
              style: theme.textTheme.labelLarge,
            ),
            const SizedBox(height: 16),
            Text(
              'This opens a preview. It won’t contact anyone or place a call.',
              textAlign: TextAlign.center,
              style: theme.textTheme.bodySmall?.copyWith(
                color: colors.onSurfaceVariant,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _DialKey extends StatelessWidget {
  const _DialKey({
    super.key,
    this.label,
    this.icon,
    this.tooltip,
    this.fontSize = 32,
    this.height = 64,
    required this.onPressed,
    this.onLongPress,
  });

  final String? label;
  final IconData? icon;
  final String? tooltip;
  final double fontSize;
  final double height;
  final VoidCallback? onPressed;
  final VoidCallback? onLongPress;

  @override
  Widget build(BuildContext context) {
    final button = TextButton(
      onPressed: onPressed,
      onLongPress: onLongPress,
      style: TextButton.styleFrom(
        foregroundColor: Theme.of(context).colorScheme.onSurface,
        minimumSize: Size(48, height),
        padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 10),
        shape: const StadiumBorder(),
      ),
      child: icon != null
          ? Icon(icon, size: 24)
          : Text(
              label!,
              style: TextStyle(fontSize: fontSize, fontWeight: FontWeight.w400),
            ),
    );
    return tooltip == null ? button : Tooltip(message: tooltip!, child: button);
  }
}
