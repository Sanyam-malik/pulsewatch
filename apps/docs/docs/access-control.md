# Groups and access levels

Pulsewatch supports multiple accounts and named groups (workspaces). A user may belong to multiple groups and has a separate role in each group. The active group is sent with authenticated requests as `X-Group-ID`; requests without that header use the group in the JWT (or the user's first membership for older tokens).

## Roles

| Role | Access |
| --- | --- |
| Owner | Full access, including member and role management. The last owner cannot be demoted or removed. |
| Admin | Manage members and regular workspace resources. Only an owner can grant admin access or change another owner's role. |
| Member | Read and modify workspace resources; cannot manage membership. |
| Viewer | Read-only access to workspace resources. |

Role checks are enforced by the API on every request, not only by hiding controls in the web UI. Membership and account-active status are reloaded for requests, so removed/deactivated users and changed roles take effect without waiting for their access token to expire.

## First-owner bootstrap and existing installations

Public registration remains available only while there are no users. The first successful registration atomically creates the first group and its owner membership. Subsequent accounts are provisioned by a group owner/admin from **Groups and access**; a new account requires a strong initial password. An existing account can be added to another group without resetting its password.

For SQL installations, apply the normal Bun migration before starting the updated API:

```sh
cd apps/server
go run ./cmd/bun db migrate
```

The migration creates the group and membership tables and preserves every existing account in a `Default group`. The oldest account is the owner; any additional legacy accounts are viewers. It does not alter existing monitoring or notification data. MongoDB deployments retain existing user documents; membership is backfilled lazily into a default group at login, granting owner to the oldest account and viewer to other pre-existing accounts.

## Management API

Authenticated endpoints:

- `GET /api/v1/groups` — list the caller's groups and roles.
- `POST /api/v1/groups` — create a group (creator becomes owner).
- `GET /api/v1/groups/{groupId}/members` — list members (owner/admin).
- `POST /api/v1/groups/{groupId}/members` — provision a user or add an existing account.
- `PATCH /api/v1/groups/{groupId}/members/{userId}` — set `admin`, `member`, or `viewer`.
- `DELETE /api/v1/groups/{groupId}/members/{userId}` — remove a membership.

When provisioning a new account, supply `email`, a strong `password`, and `role`. For an existing account, omit `password`; the existing credentials are retained. Owner assignment is not available through the member API, and every group must retain at least one owner.

## Data scope

Monitor, proxy, tag, notification-channel, maintenance, status-page, and API-key reads and writes are scoped to the selected group. API keys are bound to the group in which they are created; a caller cannot switch groups by sending `X-Group-ID` with an API key. Monitor associations with proxies, notification channels, tags, maintenance schedules, and status pages are checked against the same group. Heartbeat/stat access and monitor deletion first verify that the monitor belongs to the active group.

The SQL migration assigns existing records to the default group without deleting data. MongoDB group assignments are backfilled lazily for existing records and accounts. Group membership is an application-level boundary; the underlying SQL/Mongo database remains shared, and deployment-wide settings are not group-specific. Public status pages remain public by design. Take normal backups before migration and do not expose the database directly to untrusted users.
