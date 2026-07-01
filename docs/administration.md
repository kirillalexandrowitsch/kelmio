# Administration

Kelmio organizes access on three levels — **site**, **organization**, and
**workspace** — so a single local installation can host several isolated
organizations, each with its own workspaces, members, groups, and reusable role
assignments. This guide describes how administration works today.

## Levels and roles

| Level | Role | Granted by | Can do |
| --- | --- | --- | --- |
| Site | Site administrator (`users.is_site_admin`) | Bootstrap / seed, or another site admin | Manage all organizations; create organizations and assign organization admins |
| Organization | `org_admin` / `org_member` | Site admin or organization admin | Manage the organization's workspaces, groups, directory, and role assignments |
| Workspace | `admin` / `member` | Organization admin or workspace admin | Manage workspace members and role assignments |

A user's **effective workspace role** is the maximum of their direct workspace
membership and any workspace role assignments granted to them directly or
through a group (`admin` outranks `member`). When no role assignments exist, the
effective role is exactly the direct membership, so single-workspace
installations behave as before.

## Organizations

An organization is the top isolation boundary: resources in one organization are
never visible from another. Each organization has a unique slug and an
`active` / `archived` status. Archived organizations are hidden but retain their
data.

Site administrators manage organizations from the **Administration** screen
(sidebar → Administration), which lists every organization and supports create,
rename, and archive/restore. The same actions are available through the
organizations API.

## Workspaces

Every workspace belongs to exactly one organization and has an `active` /
`archived` status and a slug that is unique within the organization. The active
workspace is stored per session; the shell exposes a **workspace switcher** that
lists the active workspaces a user can reach (directly or through a group) and
switches scope on the server for all subsequent requests. Archived workspaces
are excluded from the switcher.

Organization administrators manage workspaces from the **Workspaces** screen,
which lists every workspace in the active organization (including archived ones)
and supports create, rename, and archive/restore.

## Groups and directory

Groups are reusable, organization-scoped bundles of users. Organization
administrators manage them from the **Groups** screen: create, rename, delete,
and manage members. Members are chosen from the organization **directory** — the
active members of the current organization — so users from other organizations
are never exposed.

## Reusable role assignments

A role assignment maps a **subject** (a user or a group) to a role within a
**scope**. Workspace role assignments are managed from the **Roles** panel on the
Workspaces screen: assign a user or group the `admin` or `member` role, or
remove an assignment. Assignments feed the effective-role resolution above, so
adding a user to a group that holds a workspace role immediately grants that
access, and removing them revokes it.

## Bootstrapping the first site administrator

The seed and `bootstrap-admin` command create the first site administrator and
the default organization. Locally, `make setup-db` provisions:

- the **Default Organization** and its **Local Workspace**;
- `admin` / `admin12345` — a site administrator and organization admin;
- `demo_member` / `demo12345` — an organization member.

To exercise multi-organization behavior, the seed also provisions a second demo
organization, **Acme Corp**, with its own **Acme Product** workspace,
`acme_admin` / `acme12345` (organization admin) and `acme_member` / `acme12345`
(organization member), and an **Acme Engineers** group. That group is assigned
the workspace `admin` role, so `acme_member`'s effective role resolves to
`admin` through the group — a working demonstration of groups and reusable role
assignments. The seed is additive and idempotent.

## API surface

All endpoints live under `/api/v1` and resolve the organization and active
workspace from the session; access to another scope is rejected on the server.

- Organizations: `GET/POST /organizations`, `PATCH /organizations/{id}`,
  `GET/POST/DELETE /organizations/{id}/members`.
- Directory: `GET /directory`.
- Workspaces: `GET/POST /workspaces`, `PATCH /workspaces/{id}`,
  `GET /workspaces?scope=organization`, `POST /session/active-workspace`.
- Workspace role assignments:
  `GET/POST/DELETE /workspaces/{id}/role-assignments`.
- Groups: `GET/POST /groups`, `PATCH/DELETE /groups/{id}`,
  `GET/POST/DELETE /groups/{id}/members`.

Administrative actions on organizations are recorded in the `audit_log` table as
a minimal audit foundation.
