import 'package:flutter/material.dart';

/// Shared presentation only: no network, device credential or audio behavior.
class PageContent extends StatelessWidget {
  const PageContent({super.key, required this.children});
  final List<Widget> children;

  @override
  Widget build(BuildContext context) => Align(
    alignment: Alignment.topCenter,
    child: ConstrainedBox(
      constraints: const BoxConstraints(maxWidth: 540),
      child: SingleChildScrollView(
        padding: const EdgeInsets.fromLTRB(24, 20, 24, 32),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: children,
        ),
      ),
    ),
  );
}

class VoiceMark extends StatelessWidget {
  const VoiceMark({super.key, this.size = 80});
  final double size;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    return ExcludeSemantics(
      child: Container(
        width: size,
        height: size,
        decoration: BoxDecoration(
          color: colors.surfaceContainerLow,
          shape: BoxShape.circle,
        ),
        child: Icon(
          Icons.call_outlined,
          size: size * 0.43,
          color: colors.onSurfaceVariant,
        ),
      ),
    );
  }
}

class PreviewLabel extends StatelessWidget {
  const PreviewLabel({super.key});

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    return Text(
      'Design preview',
      style: Theme.of(context).textTheme.labelMedium?.copyWith(
        color: colors.onSurfaceVariant,
        fontWeight: FontWeight.w400,
      ),
    );
  }
}

class SupportingText extends StatelessWidget {
  const SupportingText(this.text, {super.key, this.center = false});
  final String text;
  final bool center;

  @override
  Widget build(BuildContext context) => Text(
    text,
    textAlign: center ? TextAlign.center : TextAlign.start,
    style: Theme.of(context).textTheme.bodyLarge
        ?.copyWith(color: Theme.of(context).colorScheme.onSurfaceVariant),
  );
}

InputDecoration textFieldDecoration(
  BuildContext context, {
  required String label,
  required String hint,
  String? helper,
  String? prefix,
}) {
  final colors = Theme.of(context).colorScheme;
  final border = OutlineInputBorder(
    borderRadius: BorderRadius.circular(12),
    borderSide: BorderSide.none,
  );
  return InputDecoration(
    labelText: label,
    hintText: hint,
    helperText: helper,
    helperMaxLines: 4,
    errorMaxLines: 4,
    prefixText: prefix,
    filled: true,
    fillColor: colors.surfaceContainerLow,
    contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 18),
    border: border,
    enabledBorder: border,
    focusedBorder: border.copyWith(
      borderSide: BorderSide(color: colors.primary, width: 2),
    ),
  );
}

Future<void> showInfoSheet(
  BuildContext context, {
  required String title,
  required String message,
}) => showModalBottomSheet<void>(
  context: context,
  showDragHandle: true,
  isScrollControlled: true,
  useSafeArea: true,
  builder: (sheetContext) => SingleChildScrollView(
    padding: const EdgeInsets.fromLTRB(24, 8, 24, 24),
    child: Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        Text(title, style: Theme.of(sheetContext).textTheme.titleLarge),
        const SizedBox(height: 12),
        SupportingText(message),
        const SizedBox(height: 24),
        SizedBox(
          width: double.infinity,
          child: FilledButton(
            onPressed: () => Navigator.pop(sheetContext),
            child: const Text('Got it'),
          ),
        ),
      ],
    ),
  ),
);
