import 'package:connect_mobile/app/connect_app.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('shell launches and discloses unavailable features', (
    tester,
  ) async {
    await tester.pumpWidget(const ConnectApp());

    expect(find.text('Connect'), findsOneWidget);
    expect(find.text('Development foundation'), findsOneWidget);
    expect(
      find.text('Calling and account setup are not available in this build.'),
      findsOneWidget,
    );
    expect(find.byType(TextField), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets('shell fits a small screen with enlarged text', (tester) async {
    tester.view.physicalSize = const Size(320, 480);
    tester.view.devicePixelRatio = 1;
    tester.platformDispatcher.textScaleFactorTestValue = 3;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    addTearDown(tester.platformDispatcher.clearTextScaleFactorTestValue);

    await tester.pumpWidget(const ConnectApp());

    expect(tester.takeException(), isNull);
    await tester.drag(
      find.byType(SingleChildScrollView),
      const Offset(0, -1200),
    );
    await tester.pumpAndSettle();
    expect(
      tester
          .getBottomLeft(
            find.text(
              'Calling and account setup are not available in this build.',
            ),
          )
          .dy,
      lessThanOrEqualTo(480),
    );
  });
}
