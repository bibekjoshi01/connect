# Connect mobile

Android-only Flutter/Kotlin project generated with Flutter 3.47.8, Dart 3.13.7.
See [toolchains](../docs/development/toolchains.md) for the pinned SDKs.
This folder owns the complete Android app, including Flutter and Kotlin.

This is the application installed on the user's phone. Entry point:
`lib/main.dart`; Android platform host: `android/`. It has its own `pubspec.yaml`,
lockfile, tests and Gradle build. It does not contain or deploy the Go backend.
The native RTC library will eventually be packaged into this app; that bridge
is not implemented yet.

From this directory:

```bash
make run      # Resolve dependencies and launch on a device/emulator
make check    # Lockfile, format, analyzer and widget tests
make apk      # Android debug build
```

The debug APK is `build/app/outputs/flutter-apk/app-debug.apk`. Future Google Play
distribution needs a signed release app bundle and a separate approved release
pipeline. See [deployment boundaries](../docs/architecture/deployment.md).

## UI preview

The app currently contains welcome, local profile setup, an empty calls screen,
Call ID entry, call controls preview, and profile/appearance settings. It uses
a shared light/dark theme and supports scrolling with enlarged text.
The layout follows a minimal phone app: Recents with a floating call action,
plain Settings rows, neutral surfaces and circular call controls.
Call ID entry uses an on-screen dial pad (0–9 and A–F), a large ID display,
paste, backspace and hold-to-clear. It does not open the system keyboard.

All state stays in memory and resets on app restart. Username and Call ID input
validate their format locally; names are not registered or checked for availability.
Call controls only change their visual state. No account requests, permissions,
audio, calling or native engine integration are implemented in this preview.

Widget tests cover navigation, validation, keypad limits, paste/delete/clear,
clipboard errors, local controls, profile editing/reset,
theme selection and small screens at 1×, 2× and 3× text sizes.

`dev.connect.connect_mobile` is a provisional identifier. Native RTC packaging
and OS facilities will be introduced in their approved stages. Release signing
is unconfigured; debug keys cannot stand in for production signing.

`lib/app/` contains navigation and local app state; `lib/screens/` contains the
screens; `lib/ui/` contains shared styling and widgets. `android/` contains the Kotlin host
and Android lifecycle/build resources; `test/` contains widget tests.
`.flutter-version`, `pubspec.yaml` and `pubspec.lock` belong to this component.
The backend lives in `../backend/` and is a separate process. Shared API schemas
live in `../contracts/`; portable media implementation belongs in
`../native-engine/`. The native bridge will be introduced in a later stage.
