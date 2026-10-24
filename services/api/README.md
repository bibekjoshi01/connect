# Connect backend service

This is the independent Go server project. Entry point: `cmd/api/main.go`.
It owns its Go module, configuration, HTTP transport and tests. Building it does
not require Flutter, Android, or the C++ RTC engine.

Stage 0 serves `GET /healthz` only. Database integration, account/authentication
and WebSocket signaling are future stages. The backend is not a plaintext voice
processor and must never receive private device keys or media decryption keys.

From the repository root:

```bash
make api-lint api-test api-build
make api-run
```

Or from this directory, with writable Go build/module caches:

```bash
go test -race ./...
go vet ./...
go build -o /tmp/connect-api ./cmd/api
API_LISTEN_ADDR=127.0.0.1:8080 /tmp/connect-api
```

The root build produces `.build/api/connect-api`. The executable currently reads
only `API_LISTEN_ADDR` from its process environment; it does not load `.env`.
Local HTTP binds to loopback by default. This is not a public production server.

The proposed server release is a Linux executable/container behind HTTPS/WSS
ingress, with private database connectivity. No API Dockerfile, production
Compose configuration, remote deployment or release automation exists yet.
See [deployment boundaries](../../docs/architecture/deployment.md).
