# Pulsewatch

Pulsewatch is a self-hosted uptime monitoring platform and a community hard fork of [Peekaping](https://github.com/0xfurai/peekaping), built with Go and React. It monitors websites, APIs, networks, databases, and game servers, with status pages, incident timelines, and notification integrations.

## Features

- HTTP/HTTPS checks with status, latency, keyword, and JSON response conditions; TCP, ICMP, DNS, gRPC, SNMP, database, message-queue, Docker, Steam, GameDig-compatible, and remote-browser checks
- Status pages, badges, maintenance windows, incidents, and Homepage widget integration
- Multi-user workspaces with owner, admin, member, and viewer roles; separate monitor groups
- Notification channels including email, chat, SMS, and incident-management integrations
- SQLite, PostgreSQL, and MongoDB storage
- Uptime Kuma SQLite import utility

## Quick start

Build the SQLite bundle from this source tree to run the Pulsewatch fork:

```bash
docker build -f deploy/docker/Dockerfile.bundle.sqlite -t pulsewatch-bundle-sqlite:local .
docker run -d --restart=always \
  -p 8383:8383 \
  -e DB_NAME=/app/data/peekaping.db \
  -v "$(pwd)/.data/sqlite:/app/data" \
  --name pulsewatch \
  pulsewatch-bundle-sqlite:local
```

The SQLite file keeps its legacy `peekaping.db` name for compatibility. The published `0xfurai/peekaping-*` image coordinates are also retained for existing deployments, but those upstream-maintained images may not include Pulsewatch features.

Docker Compose files, bundle Dockerfiles, and the Helm chart are under [`deploy/`](deploy/). Compose targets run from the repository root and read `.env` there; sample environment files are in `deploy/docker/`. Run `make help` to see the supported Compose targets. Deployment examples using prebuilt upstream images are for compatibility; build images from this repository to use Pulsewatch-specific changes.

## Development

```bash
pnpm install
make dev
```

Run the server tests with `cd apps/server && go test ./...`; build the web app with `pnpm --dir apps/web build`. The API's Swagger UI is available at `http://localhost:8034/swagger/index.html` when the server is running. See the [server README](apps/server/README.md) and [Uptime Kuma importer guide](apps/server/docs/uptime-kuma-import.md) for server-specific details.

## Repository layout

- `apps/server/` — Go API, monitoring services, workers, migrations, and importer
- `apps/web/` — React and TypeScript application
- `deploy/docker/` — Docker Compose configurations and bundle deployment assets
- `deploy/helm/` — Helm chart

## Upstream and license

Pulsewatch retains the upstream project's MIT license and acknowledges Peekaping contributors. This project is not affiliated with the upstream maintainer. See [LICENSE](LICENSE) for license terms.
