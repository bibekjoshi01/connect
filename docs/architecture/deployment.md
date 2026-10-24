# Separate app and server deployment

Status: explanation and proposed future deployment flow. Stage 0 builds local
components; it does not implement or authorize production deployment.

## Projects and runtime locations

`apps/mobile` is the Android application. Flutter/Dart and Kotlin run on the
phone. `engine/rtc` is a portable native library intended for inclusion in that
application later; it is not a remotely hosted RTC processing service.

`services/api` is a separate Go server. It will own account operations and
authenticated signaling. PostgreSQL stores server data, Redis is optional and
needs a demonstrated use, and coturn provides standard STUN/TURN infrastructure.
Server components do not require a Flutter installation in their release image.

`contracts` is shared source documentation/schema material, not a runtime service.
One repository supports coordinated changes while preserving separate builds,
configuration, deployments and version compatibility.

## Android release flow — future

Build the app using the Android SDK/NDK and package its native libraries inside
the application. For development, `make mobile-apk` builds a debug APK that can be
installed directly. For Google Play, configure reviewed release signing and
produce a signed Android App Bundle, then use an approved Play release process.
The release build is not configured in Stage 0.

The app will receive the appropriate environment's public HTTPS/WSS endpoint as
application configuration. A server hostname is not a secret; database passwords,
FCM server credentials and TURN shared secrets must never be packaged in the app.

Reference: [Flutter Android release guide](https://docs.flutter.dev/deployment/android).

## Backend and infrastructure release flow — proposal

Build only the Go module into a Linux executable and a minimal runtime container.
Deploy it on separately approved server infrastructure behind HTTPS/WSS ingress.
When containerized, configure the API's internal listening address explicitly;
its current loopback default is for host development.

Keep PostgreSQL, optional Redis and operational endpoints on private networks;
supply runtime secrets through the approved server configuration mechanism.
Define migrations, readiness checks, backups, monitoring and rollback procedures
before staging. The current Compose file is a local fixture, contains no API
service, and must not be published as a production stack.

coturn needs a separately configured reachable address and UDP relay ports.
It may share an initial approved host or run on another host, subject to capacity,
network and security review. An ordinary HTTP reverse proxy does not provide UDP
TURN relay connectivity. The current container-local TURN fixture is insufficient
for physical-device or public NAT traversal tests.

The intended control path is phone → HTTPS/WSS ingress → Go API → private storage.
The intended media path is encrypted phone-to-phone UDP, or encrypted packets
through TURN. The Go API does not process plaintext audio. These are future
capabilities, not Stage 0 behavior.

Reference: [Docker environment-specific production configuration](https://docs.docker.com/compose/how-tos/production/).

## Independent updates

An Android release updates the installed app; an API release updates the server.
A compatible server fix need not require a new Android release, and a UI-only
change need not redeploy the backend. Version contracts and preserve compatibility
for installed app versions; coordinated breaking changes need an explicit plan.

Existing CI jobs already validate components separately. Artifact signing,
environment-specific deployment pipelines, hosting selection, DNS/TLS and public
TURN belong to approved later work. Evaluate costs before selecting or provisioning
hosting or distribution accounts. No paid provider is selected by this proposal.
