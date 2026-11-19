import 'package:connect_mobile/app/connect_app.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

const sampleCallId = '0123456789abcdef0123456789abcdef';

Future<void> tapVisible(WidgetTester tester, Finder finder) async {
  await tester.ensureVisible(finder);
  await tester.pumpAndSettle();
  await tester.tap(finder);
  await tester.pumpAndSettle();
}

Future<void> enterPreview(WidgetTester tester, {String name = 'alice'}) async {
  await tapVisible(tester, find.byKey(const ValueKey('get-started')));
  await tester.enterText(find.byKey(const ValueKey('username-field')), name);
  await tapVisible(tester, find.byKey(const ValueKey('continue-profile')));
}

Future<void> dial(WidgetTester tester, String value) async {
  for (final character in value.split('')) {
    await tapVisible(tester, find.byKey(ValueKey('dial-$character')));
  }
}

void main() {
  testWidgets('welcome explains the preview and validates the local profile', (
    tester,
  ) async {
    await tester.pumpWidget(const ConnectApp());
    expect(find.text('Voice calls, with a Call ID.'), findsOneWidget);
    expect(
      find.text('Explore the design. No account or calls will be created.'),
      findsOneWidget,
    );
    await tapVisible(tester, find.byKey(const ValueKey('get-started')));
    await tapVisible(tester, find.byKey(const ValueKey('continue-profile')));
    expect(find.textContaining('Use 3–32'), findsOneWidget);
    await tester.enterText(
      find.byKey(const ValueKey('username-field')),
      'bad name',
    );
    await tapVisible(tester, find.byKey(const ValueKey('continue-profile')));
    expect(find.textContaining('Use 3–32'), findsOneWidget);
    await tester.enterText(
      find.byKey(const ValueKey('username-field')),
      'Alice_7',
    );
    await tapVisible(tester, find.byKey(const ValueKey('continue-profile')));
    expect(find.text('No calls yet'), findsOneWidget);
    await tapVisible(tester, find.byKey(const ValueKey('profile-tab')));
    expect(find.text('@alice_7'), findsOneWidget);
    expect(find.text('Preview profile'), findsOneWidget);
    expect(
      find.text(
        'Your Call ID will appear here when account setup is available.',
      ),
      findsOneWidget,
    );
    expect(tester.takeException(), isNull);
  });

  testWidgets(
    'Call ID validation opens local controls without creating call history',
    (tester) async {
      await tester.pumpWidget(const ConnectApp());
      await enterPreview(tester);
      await tapVisible(tester, find.byKey(const ValueKey('new-call')));
      expect(find.byType(TextField), findsNothing);
      await dial(tester, '12345');
      await tapVisible(tester, find.byKey(const ValueKey('preview-call')));
      expect(find.text('Enter a 32-character Call ID.'), findsOneWidget);
      var clipboard = 'g' * 32;
      tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
        SystemChannels.platform,
        (call) async =>
            call.method == 'Clipboard.getData' ? {'text': clipboard} : null,
      );
      addTearDown(
        () => tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
          SystemChannels.platform,
          null,
        ),
      );
      await tapVisible(tester, find.byKey(const ValueKey('paste-call-id')));
      expect(
        find.text('Paste a 32-character Call ID using 0–9 and A–F.'),
        findsOneWidget,
      );
      expect(find.text('5/32'), findsOneWidget);
      clipboard = '${sampleCallId}0';
      await tapVisible(tester, find.byKey(const ValueKey('paste-call-id')));
      expect(find.text('5/32'), findsOneWidget);
      clipboard = '  ${sampleCallId.toUpperCase()}  ';
      await tapVisible(tester, find.byKey(const ValueKey('paste-call-id')));
      expect(find.text('32/32'), findsOneWidget);
      await tapVisible(tester, find.byKey(const ValueKey('preview-call')));
      expect(find.text('Call preview'), findsOneWidget);
      expect(find.text('Not connected'), findsOneWidget);
      expect(
        find.text(
          'Design preview only. No call is placed and no audio is used.',
        ),
        findsOneWidget,
      );
      await tapVisible(tester, find.byKey(const ValueKey('mute-control')));
      expect(find.text('Unmute'), findsOneWidget);
      await tapVisible(tester, find.byKey(const ValueKey('speaker-control')));
      expect(
        tester
            .widget<IconButton>(find.byKey(const ValueKey('speaker-control')))
            .isSelected,
        isTrue,
      );
      await tapVisible(
        tester,
        find.byKey(const ValueKey('close-call-preview')),
      );
      expect(find.text('No calls yet'), findsOneWidget);
      expect(find.text('Call preview'), findsNothing);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets(
    'dial pad supports every character, caps length and deletes or clears',
    (tester) async {
      await tester.pumpWidget(const ConnectApp());
      await enterPreview(tester);
      await tapVisible(tester, find.byKey(const ValueKey('new-call')));
      await dial(tester, sampleCallId);
      expect(find.text('32/32'), findsOneWidget);
      expect(
        tester.widget<Text>(find.byKey(const ValueKey('call-id-display'))).data,
        '01234567 89ABCDEF 01234567 89ABCDEF',
      );
      await dial(tester, '0');
      expect(find.text('32/32'), findsOneWidget);
      await tapVisible(tester, find.byKey(const ValueKey('delete-call-id')));
      expect(find.text('31/32'), findsOneWidget);
      await dial(tester, 'f');
      expect(find.text('32/32'), findsOneWidget);
      await tester.ensureVisible(find.byKey(const ValueKey('delete-call-id')));
      await tester.pumpAndSettle();
      await tester.longPress(find.byKey(const ValueKey('delete-call-id')));
      await tester.pumpAndSettle();
      expect(find.text('0/32'), findsOneWidget);
      expect(
        tester.widget<Text>(find.byKey(const ValueKey('call-id-display'))).data,
        'Call ID',
      );
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets('clipboard failure keeps the keypad usable', (tester) async {
    await tester.pumpWidget(const ConnectApp());
    await enterPreview(tester);
    await tapVisible(tester, find.byKey(const ValueKey('new-call')));
    tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
      SystemChannels.platform,
      (call) async {
        if (call.method == 'Clipboard.getData') {
          throw PlatformException(code: 'unavailable');
        }
        return null;
      },
    );
    addTearDown(
      () => tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
        SystemChannels.platform,
        null,
      ),
    );
    await tapVisible(tester, find.byKey(const ValueKey('paste-call-id')));
    expect(
      find.text('Unable to paste. Use the keypad to enter a Call ID.'),
      findsOneWidget,
    );
    await dial(tester, 'a');
    expect(find.text('1/32'), findsOneWidget);
    expect(find.byKey(const ValueKey('call-id-error')), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets(
    'appearance, profile editing and reset work only in preview state',
    (tester) async {
      await tester.pumpWidget(const ConnectApp());
      await enterPreview(tester);
      await tapVisible(tester, find.byKey(const ValueKey('profile-tab')));
      await tapVisible(tester, find.byKey(const ValueKey('appearance')));
      await tapVisible(tester, find.byKey(const ValueKey('theme-dark')));
      expect(
        tester.widget<MaterialApp>(find.byType(MaterialApp)).themeMode,
        ThemeMode.dark,
      );
      await tapVisible(tester, find.byKey(const ValueKey('edit-profile')));
      await tester.enterText(
        find.byKey(const ValueKey('username-field')),
        'new_name',
      );
      await tapVisible(tester, find.byKey(const ValueKey('continue-profile')));
      expect(find.text('@new_name'), findsOneWidget);
      await tapVisible(tester, find.byKey(const ValueKey('reset-preview')));
      await tapVisible(tester, find.text('Cancel'));
      expect(find.text('@new_name'), findsOneWidget);
      await tapVisible(tester, find.byKey(const ValueKey('reset-preview')));
      await tapVisible(
        tester,
        find.widgetWithText(TextButton, 'Reset preview').last,
      );
      expect(find.byKey(const ValueKey('get-started')), findsOneWidget);
      expect(
        tester.widget<MaterialApp>(find.byType(MaterialApp)).themeMode,
        ThemeMode.system,
      );
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets(
    'back navigation cancels edits and preserves the existing profile',
    (tester) async {
      await tester.pumpWidget(const ConnectApp());
      await enterPreview(tester);
      await tapVisible(tester, find.byKey(const ValueKey('profile-tab')));
      await tapVisible(tester, find.byKey(const ValueKey('edit-profile')));
      await tester.enterText(
        find.byKey(const ValueKey('username-field')),
        'unsaved',
      );
      await tester.pageBack();
      await tester.pumpAndSettle();
      expect(find.text('@alice'), findsOneWidget);
      await tapVisible(tester, find.byKey(const ValueKey('calls-tab')));
      await tapVisible(tester, find.byKey(const ValueKey('new-call')));
      await tester.pageBack();
      await tester.pumpAndSettle();
      expect(find.text('No calls yet'), findsOneWidget);
    },
  );

  for (final scale in [1.0, 2.0, 3.0]) {
    testWidgets('main flows fit a small screen at ${scale}x text', (
      tester,
    ) async {
      tester.view.physicalSize = const Size(320, 568);
      tester.view.devicePixelRatio = 1;
      tester.platformDispatcher.textScaleFactorTestValue = scale;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      addTearDown(tester.platformDispatcher.clearTextScaleFactorTestValue);
      await tester.pumpWidget(const ConnectApp());
      await enterPreview(tester);
      expect(tester.takeException(), isNull);
      await tapVisible(tester, find.byKey(const ValueKey('new-call')));
      await dial(tester, sampleCallId);
      await tapVisible(tester, find.byKey(const ValueKey('preview-call')));
      await tapVisible(
        tester,
        find.byKey(const ValueKey('close-call-preview')),
      );
      expect(tester.takeException(), isNull);
      await tapVisible(tester, find.byKey(const ValueKey('profile-tab')));
      await tapVisible(tester, find.byKey(const ValueKey('appearance')));
      await tapVisible(tester, find.byKey(const ValueKey('theme-dark')));
      expect(tester.takeException(), isNull);
    });
  }
}
