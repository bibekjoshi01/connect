# Connect backend service

This is the independent Go server project. Entry point: `cmd/api/main.go`.
It owns its Go module, configuration, HTTP transport and tests. Building it does
not require Flutter, Android, or the C++ RTC engine.

Stage 1 provides PostgreSQL connectivity, explicit transactional migrations,
`GET /healthz`, `GET /readyz`, and private `GET /metrics`. Account/authentication
and WebSocket signaling are future stages. The backend is not a plaintext voice
processor and must never receive private device keys or media decryption keys.

From the repository root:

```bash
make api-lint api-test api-build
make env infra-up
make api-migrate
make api-run
make api-integration
```

Or from this directory, with writable Go build/module caches:

```bash
go test -race ./...
go vet ./...
go build -o /tmp/connect-api ./cmd/api
# Supply DATABASE_URL securely in the process environment before launching:
API_LISTEN_ADDR=127.0.0.1:8080 /tmp/connect-api
```

Direct execution also requires `DATABASE_URL` in the process environment. The
root build produces `.build/api/connect-api` and `.build/api/connect-migrate`.
The executables read process environment only; the local Make targets privately
load `.env` and derive the loopback database URL. Local HTTP binds to loopback by
default. This is not a public production server.

Read [Stage 1 configuration and migration behavior](../../docs/development/stage-1.md).

The proposed server release is a Linux executable/container behind HTTPS/WSS
ingress, with private database connectivity. No API Dockerfile, production
Compose configuration, remote deployment or release automation exists yet.
See [deployment boundaries](../../docs/architecture/deployment.md).
