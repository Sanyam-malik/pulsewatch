---
sidebar_position: 2
---

# Game and browser monitors

These monitor types use the standard monitor API fields (`type`, `config`, interval,
timeout, retries, notifications, and tags). The `config` value is a JSON string.

## Steam

Steam monitors send the Source Engine `A2S_INFO` UDP query to the server's query
port. The monitor is up when the server returns a valid info response. Optional
`expected_app_id` and `expected_name` fields turn the response into an assertion.

```json
{
  "host": "game.example.com",
  "port": 27015,
  "expected_app_id": 730,
  "expected_name": "Production server"
}
```

## GameDig

The current GameDig monitor uses the same `A2S_INFO` query, covering Source
Query-compatible servers (including many Steam-hosted games). It is not a
replacement for GameDig's full protocol catalog; protocols other than A2S are not
currently supported. `game` is the configured game identifier; it does not choose
another wire protocol. The server response can optionally be checked by name,
map, and player bounds.

```json
{
  "game": "cs2",
  "host": "game.example.com",
  "port": 27015,
  "expected_map": "de_dust2",
  "min_players": 1,
  "max_players": 32
}
```

## Playwright browser check

This monitor connects to a remote Chromium DevTools Protocol WebSocket endpoint,
opens a fresh page, and waits for document load. Configure a CSS selector, page
text keyword, or both as assertions. A Chromium browser is not bundled in the
worker; the endpoint must be reachable from each worker. Do not expose a
DevTools endpoint to untrusted networks because it grants control of the browser.

```json
{
  "browser_ws_endpoint": "ws://browser:9222/devtools/browser/<browser-id>",
  "url": "https://example.com",
  "selector": "main h1",
  "keyword": "Welcome"
}
```

The executor speaks Chromium DevTools Protocol directly. It does not install or
run Playwright scripts, interact with authenticated sessions, or support
non-Chromium browser protocols.
