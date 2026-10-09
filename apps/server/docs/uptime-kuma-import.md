# Importing from Uptime Kuma

Pulsewatch includes a one-way importer for an Uptime Kuma SQLite database. It is a
standalone Go command and talks directly to the existing Pulsewatch SQLite schema;
it does not change the API, authentication, monitor executors, or notification
providers.

## Preview and apply

Stop Uptime Kuma or take a consistent copy of its database before importing. The
usual source path is `data/kuma.db`. Run the importer from `apps/server` and point
it at an already-migrated Pulsewatch SQLite database:

```sh
go run ./cmd/import-uptime-kuma \
  --source /path/to/uptime-kuma/data/kuma.db \
  --target /path/to/pulsewatch/data.db
```

The default is a read-only preview. The JSON report lists importable counts,
skipped records (source ID, name, and reason), and limitations, but never includes
monitor or notification configuration values. To apply the validated plan:

```sh
go run ./cmd/import-uptime-kuma \
  --source /path/to/uptime-kuma/data/kuma.db \
  --target /path/to/pulsewatch/data.db \
  --apply
```

The source opens in SQLite read-only mode. Before writing, the importer checks
that the destination has Pulsewatch's required tables and skips status-page slug
collisions. All inserted rows are written in a single SQLite transaction; a
write failure rolls the import back. Back up the destination first. The importer
does not overwrite or de-duplicate monitors and channels, so applying the same
source a second time can create duplicates.

## Data handling and supported records

* `monitor`: HTTP, keyword (as HTTP keyword check), port (as TCP), ping, and DNS
  records with compatible settings. Pulsewatch-required timing minimums are
  applied. Monitor creation starts disabled; `--activate` preserves the source
  active setting for monitors that were active.
* `notification` and `monitor_notification`: Discord, Slack, Telegram, Gotify,
  and generic webhook channels with recognized Uptime Kuma config keys. All
  imported channels stay disabled. Channel credentials are secrets and are
  skipped unless `--include-secrets` is explicitly supplied. Use that option only
  when you trust the database copy and destination filesystem. Secret config
  values are never printed by the tool. With secrets enabled, the channel
  associations of imported monitors are retained.
* `status_page`, `group`, and `monitor_group`: page content and public monitor
  memberships. Pages remain unpublished unless `--publish-pages` is supplied.
  Password-protected pages are always imported unpublished; the source password
  is not copied. Private groups are excluded from public-page memberships.
  Page auto-refresh interval is retained when present in the source schema.
  Uptime Kuma's monitor/group ordering is not retained because the Pulsewatch
  membership table has no order field; this limitation appears in the report.

Sensitive monitor request/authentication data is never imported by this version,
even with `--include-secrets`; affected monitors are reported as skipped. This
includes configured headers/body or credentials and URLs with user-info or query
parameters. Proxy-backed monitors are also skipped because their network route
cannot be retained. The importer intentionally does not preserve source push
tokens, tags, maintenance schedules, incidents, status-page domains, custom CSS,
analytics settings, or other customizations not represented by Pulsewatch.
Heartbeat history is not transferred. Unsupported monitor types, channel
providers, settings, and links to skipped records are listed individually in
the preview report.

Only SQLite-to-SQLite imports are supported. Run the destination's normal Bun
migrations before using the tool. Do not use a live destination database whose
schema is being changed concurrently.
