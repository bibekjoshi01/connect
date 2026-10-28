# Connect mobile

Android-only Flutter/Kotlin project generated with Flutter 3.47.8, Dart 3.13.7.
See the root README for toolchain setup and exact build/check commands.

This is the application installed on the user's phone. Entry point:
`lib/main.dart`; Android platform host: `android/`. It has its own `pubspec.yaml`,
lockfile, tests and Gradle build. It does not contain or deploy the Go backend.
The native RTC library will eventually be packaged into this app; that bridge
is not implemented yet.

From this directory:

```bash
flutter pub get --enforce-lockfile
flutter analyze --no-pub --fatal-infos
flutter test --no-pub
flutter build apk --debug --no-pub
```

The debug APK is `build/app/outputs/flutter-apk/app-debug.apk`. Future Google Play
distribution needs a signed release app bundle and a separate approved release
pipeline. See [deployment boundaries](../../docs/architecture/deployment.md).

Stage 0 launches a development screen. It has no account, calling, permissions,
backend integration, native engine bridge or E2EE functionality. Product screens
belong to Stage 4. `dev.connect.connect_mobile` is a provisional identifier.
Native RTC packaging and OS facilities will be introduced in their approved stages.
Release signing is unconfigured; debug keys cannot stand in for production signing.
