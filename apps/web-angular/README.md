# Pulsewatch Angular web client

This is the Angular candidate for the React-to-Angular web migration. The existing
`apps/web` React client remains the deployed app until feature parity is complete.

From the repository root, install workspace dependencies with `pnpm install`, then
run the Angular client with `pnpm --filter web-angular dev` or build it with
`pnpm --filter web-angular build`.

The client reads its API endpoint from `window.__CONFIG__.API_URL`, which is supplied
by `public/env.js` in development and by the container entrypoint in deployment.
