# Toolchain baseline

Flutter 3.47.8 (official tag `737ba8f885dc774b53b442759999a312c9461e4e`),
bundled Dart 3.13.7. `.flutter-version`, the generated metadata and CI pin the
baseline; update them together with `pubspec.lock` and validate again.
Flutter's stable channel is the recommended baseline for new applications:
https://docs.flutter.dev/install/archive

Go language floor: 1.26.0. The module toolchain directive and CI pin patched Go
1.26.10. The preinstalled system Go is 1.26.2; validation uses the downloaded
1.26.10 toolchain in the ignored project cache. The backend currently has no
external Go dependencies.
Check upstream release/security notes before each toolchain update:
https://go.dev/doc/devel/release

Native: C++20 compiler (Apple Clang 21 locally; GCC on Linux CI), CMake >=3.22,
CTest, optional Ninja. clang-format 21.1.8 is pinned as a dev-only pip wheel.
No Opus/crypto/audio/network dependencies have been added.

Android baseline from the generated Flutter template: Gradle 9.3.1, Android
Gradle Plugin 9.1.0, Kotlin plugin 2.4.0, compile/target API 36, minimum API 24,
NDK 28.2.13676358. Use JDK 21 for Gradle; Java/Kotlin source targets remain 17.
The generated template disables AGP's built-in Kotlin/new DSL for Flutter
compatibility; Flutter's Gradle plugin applies the declared Kotlin plugin.
Review supported SDK/device requirements before changing the minimum version. JDK 25 was found
locally; a compatible Android toolchain has not been installed by this bootstrap.

Docker Engine plus Compose v2+; Python >=3.10 for local secret generation and
TURN smoke tests; Git; Bash; curl for the manual HTTP smoke check. Android Studio
or its command-line SDK tools are needed for APK builds. Follow the official
setup and review Android licenses yourself:
https://docs.flutter.dev/platform-integration/android/setup

Container images are pinned by multi-platform digest in Compose. The tags
identify the tested bootstrap versions, not an assertion that they are the latest
security releases. Review patch updates and advisories before broader use. Redis
is optional; review its selected version's license before any commercial service
deployment. No production dependency/licensing approval is implied by Stage 0.

GitHub Actions are pinned by commit SHA. GitHub execution requires pushing this
repository to an approved remote; no remote is configured by Stage 0.
