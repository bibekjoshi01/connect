import 'package:flutter/material.dart';

import '../ui/components.dart';

class CallPreviewScreen extends StatefulWidget {
  const CallPreviewScreen({super.key, required this.callId});
  final String callId;

  @override
  State<CallPreviewScreen> createState() => _CallPreviewScreenState();
}

class _CallPreviewScreenState extends State<CallPreviewScreen> {
  bool _muted = false;
  bool _speaker = false;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    final groupedId = [
      for (var i = 0; i < widget.callId.length; i += 8)
        widget.callId.substring(i, i + 8),
    ].join(' ');
    return Scaffold(
      appBar: AppBar(),
      body: SafeArea(
        child: PageContent(
          children: [
            const Align(alignment: Alignment.centerLeft, child: PreviewLabel()),
            const SizedBox(height: 44),
            const Center(child: VoiceMark(size: 88)),
            const SizedBox(height: 32),
            Text(
              'Call preview',
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.headlineMedium,
            ),
            const SizedBox(height: 16),
            Text(
              'Call ID',
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.labelLarge
                  ?.copyWith(color: colors.onSurfaceVariant),
            ),
            const SizedBox(height: 8),
            SelectableText(
              groupedId,
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodyLarge
                  ?.copyWith(fontFamily: 'monospace'),
            ),
            const SizedBox(height: 20),
            const SupportingText('Not connected', center: true),
            const SizedBox(height: 36),
            Wrap(
              alignment: WrapAlignment.center,
              spacing: 36,
              runSpacing: 24,
              children: [
                _PreviewControl(
                  controlKey: const ValueKey('mute-control'),
                  selected: _muted,
                  icon: _muted
                      ? Icons.mic_off_outlined
                      : Icons.mic_none_outlined,
                  label: _muted ? 'Unmute' : 'Mute',
                  tooltip: _muted ? 'Unmute preview' : 'Mute preview',
                  onTap: () => setState(() => _muted = !_muted),
                ),
                _PreviewControl(
                  controlKey: const ValueKey('speaker-control'),
                  selected: _speaker,
                  icon: _speaker
                      ? Icons.volume_up_outlined
                      : Icons.volume_down_outlined,
                  label: 'Speaker',
                  tooltip: _speaker
                      ? 'Speaker preview on'
                      : 'Speaker preview off',
                  onTap: () => setState(() => _speaker = !_speaker),
                ),
              ],
            ),
            const SizedBox(height: 40),
            Center(
              child: IconButton.filled(
                key: const ValueKey('close-call-preview'),
                tooltip: 'Back to calls',
                style: IconButton.styleFrom(
                  backgroundColor: const Color(0xFFD9363E),
                  foregroundColor: Colors.white,
                  padding: const EdgeInsets.all(22),
                  shape: const CircleBorder(),
                ),
                iconSize: 28,
                onPressed: () => Navigator.pop(context, true),
                icon: const Icon(Icons.call_end_rounded),
              ),
            ),
            const SizedBox(height: 10),
            Text(
              'Back to calls',
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.labelLarge,
            ),
            const SizedBox(height: 16),
            Text(
              'Design preview only. No call is placed and no audio is used.',
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodyMedium
                  ?.copyWith(color: colors.onSurfaceVariant),
            ),
          ],
        ),
      ),
    );
  }
}

class _PreviewControl extends StatelessWidget {
  const _PreviewControl({
    required this.controlKey,
    required this.selected,
    required this.icon,
    required this.label,
    required this.tooltip,
    required this.onTap,
  });
  final Key controlKey;
  final bool selected;
  final IconData icon;
  final String label;
  final String tooltip;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        IconButton.filledTonal(
          key: controlKey,
          isSelected: selected,
          onPressed: onTap,
          tooltip: tooltip,
          style: IconButton.styleFrom(
            backgroundColor: selected
                ? colors.primary
                : colors.surfaceContainerLow,
            foregroundColor: selected ? colors.onPrimary : colors.onSurface,
            shape: const CircleBorder(),
          ),
          iconSize: 28,
          padding: const EdgeInsets.all(20),
          icon: Icon(icon),
        ),
        const SizedBox(height: 10),
        Text(label, style: Theme.of(context).textTheme.labelLarge),
      ],
    );
  }
}
