# Native engine

Independent portable C++20/CMake static library. It currently implements build
metadata and a linkage smoke test only. It has no audio/network implementation,
WebRTC dependency, Flutter bridge, account storage or backend dependency.

- `include/connect/rtc/`: public library headers.
- `src/`: portable implementation.
- `tests/`: host smoke tests.
- `.clang-format` and `tooling/requirements.txt`: this component's formatting rules
  and pinned development formatter.

From this folder:

```bash
make build
make test
make check CLANG_FORMAT="$PWD/../.tools/format/bin/clang-format"
```

`make check` runs formatting and host tests. Use `CLANG_FORMAT=clang-format` when
the pinned formatter is available on PATH. Output is in `../.build/native-engine/`.

CMake also works independently of the repository Makefiles:

```bash
cmake -S . -B /tmp/connect-native-build -DBUILD_TESTING=ON
cmake --build /tmp/connect-native-build --parallel
ctest --test-dir /tmp/connect-native-build --output-on-failure
```

Later stages will package the library into `../mobile-app/` through a native
bridge. Android lifecycle, Keystore, Telecom and audio routing belong in that
app's Kotlin platform layer; they do not belong in this portable library.
The backend never links the media engine. Media/crypto work remains stage-gated.
