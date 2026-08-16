# VNTU Timetable Go Monolith Rewrite Plan

## 1. Purpose

Rewrite the existing VNTU timetable website, API, and Telegram bot as one small Go application backed by one SQLite database.

The rewrite should reproduce the useful behavior of the current system, remove unnecessary infrastructure, improve reliability and visibility, and establish a safe base for future features. It is a rewrite of behavior, not a line-by-line port of the Python and React implementations.

The three existing repositories remain unchanged and serve as behavior references:

- `VntuTimetableAPI`
- `VntuTimetableSite`
- `Vntu_timetable_bot`

The new application will live in a separate repository. A suitable working name is `vntu-timetable`; the final repository name can be chosen when it is created.

## 2. Confirmed decisions

- Language: Go.
- Deployment target: Ubuntu 22.04.5 LTS, Linux `x86_64`, 1 CPU, 1 GB RAM.
- Runtime: one Go process managed by systemd.
- Storage: one SQLite database.
- HTTPS and reverse proxy: existing Nginx installation.
- Current production identity remains unchanged for the initial release:
  - Bot: `vntu_timetable_bot`
  - Domain: `vntu-timetable.zelefin.top`
- Website: server-side rendered with Go templates, usable both normally and as a Telegram Mini App.
- Visual direction: retain the current dark theme and lesson-card identity, but improve layout, accessibility, error handling, and desktop use. Do not reproduce React/Tailwind implementation details.
- API: a new, compatibility-breaking, public read-only API.
- Bot transport: Telegram webhooks instead of polling.
- Telegram implementation: a small handwritten client using Go's HTTP facilities rather than a bot framework.
- Keep:
  - registration and updating a user's group/subgroup;
  - `/timetable` access to the Mini App;
  - `/inline` timetable messages with week/day navigation buttons;
  - admin text mailing;
  - text-only mailing behavior;
  - one combined daily metadata and schedule refresh at 07:00 Europe/Kyiv.
- Remove:
  - Telegram inline-query sharing through `@bot group-name`;
  - sharing buttons;
  - public refresh/write endpoints;
  - Redis;
  - PostgreSQL after migration;
  - Docker from production and deployment;
  - the separate frontend build/runtime;
  - the bot's HTTP calls back into its own API;
  - the dedicated teacher discovery/synchronization process unless a future teacher feature requires it.
- Existing bot users will be migrated; JetIQ data will be downloaded fresh.
- CI runs automatically on pull requests and pushes. Production deployment is manually triggered.
- Brief deployment downtime is acceptable.
- SQLite backups rotate locally. Loss of the VPS remains an explicitly accepted risk.
- Proactive admin alerts are the primary operational signal; `/status` remains available for inspection without requiring routine server log access.

## 3. Scope boundaries

### Included in the initial rewrite

- Fresh JetIQ synchronization.
- Faculty and group browsing.
- Two-week group timetable display.
- Public JSON API.
- SSR website and Telegram Mini App behavior.
- Existing bot registration and timetable workflows.
- Admin refresh, week correction, mailing, and status commands.
- Persistent synchronization history and simple proactive alerts.
- A one-time, runbook-driven import from the old bot PostgreSQL database.
- SQLite migration and backup mechanisms.
- Automated tests.
- systemd, Nginx, CI, and manually triggered CD configuration.
- A documented cutover and rollback procedure.

### Explicitly excluded from the initial rewrite

- Product rebranding or domain/bot renaming.
- New timetable features unrelated to parity control and operational status.
- Grafana, Prometheus, or another monitoring stack.
- A web-based administration panel.
- Telegram inline-query sharing.
- Rich-media admin mailing.
- External/off-site backup storage.
- Kubernetes, Docker, Redis, PostgreSQL, or a distributed job queue.
- Premature internal service or repository abstractions.

## 4. Current behavior to preserve or intentionally replace

The existing API downloads faculties, groups, teachers, and schedules from JetIQ. It stores timetable data in PostgreSQL and caches API responses in Redis. The React site consumes the API. The Telegram bot has its own PostgreSQL user database and Redis state/cache, calls the public API, and formats the returned timetable for Telegram.

The rewrite must preserve user-visible behavior where listed above, but it should intentionally fix these weaknesses:

- maintenance `POST` endpoints are publicly accessible;
- cached/non-cached implementation details leak into API responses;
- lesson ordering depends on database insertion order;
- stale faculties and groups are never handled explicitly;
- teacher resolution causes a large number of upstream calls and can create invalid foreign-key references;
- per-group network failures can be swallowed;
- cache invalidation and retry behavior are spread across services;
- the bot calls the application's own public HTTP API;
- calendar week parity is hardcoded and sometimes requires a source-code change;
- there are no application tests;
- existing GitHub Actions only lint Python and do not deploy.

Before implementing each behavior, use representative current JetIQ responses and current UI/API outputs as references. Do not preserve known bugs merely for compatibility.

## 5. Target architecture

```text
                         +--------------------+
                         |      Telegram      |
                         +----------+---------+
                                    |
                                  webhook
                                    |
+---------+        HTTPS       +----v----+        local       +----------+
| Browser +------------------->+  Nginx  +------------------->+ Go app   |
+---------+                     +---------+  127.0.0.1:8080    +----+-----+
                                                                    |
                    scheduled HTTPS                                 |
+---------+<---------------------------------------------------------+
| JetIQ   |                                                          |
+---------+                                                     +----v-----+
                                                                | SQLite  |
                                                                +----------+
```

One process owns:

- the public HTTP server;
- SSR rendering and embedded static files;
- the read-only API;
- Telegram webhook validation and update dispatch;
- outgoing Telegram Bot API requests;
- scheduled and admin-triggered JetIQ synchronization;
- migrations and backup commands;
- health checks, status calculation, and admin alerts.

No internal feature should call another feature through HTTP. Web, API, bot, and jobs call the same Go domain/database functions directly.

## 6. Simplicity rules

- Prefer the Go standard library where it is adequate.
- Use a supported Go release and pin it in `go.mod` and CI when implementation begins.
- Use one SQLite driver. Prefer a maintained pure-Go driver if it satisfies correctness, backup, and transaction requirements; otherwise document why CGO is accepted.
- Use `database/sql` and explicit SQL. Do not introduce an ORM.
- Use `net/http`, `html/template`, and embedded files. Do not introduce a frontend framework or Node build.
- Use a small handwritten Telegram client for only the Bot API methods the application needs.
- Keep initial application code in one package unless a concrete boundary makes another package useful.
- Do not add repositories, service layers, interfaces, or dependency injection solely to imitate an enterprise architecture.
- Introduce an interface when it materially enables testing or separates an external system, such as the JetIQ and Telegram HTTP clients.
- Optimize only after measurement. SQLite reads should initially have no additional cache.

## 7. Proposed repository layout

The precise split may evolve, but begin with a flat, discoverable structure:

```text
vntu-timetable/
├── .github/
│   └── workflows/
│       ├── ci.yml
│       └── deploy.yml
├── migrations/
├── static/
├── templates/
├── testdata/
│   └── jetiq/
├── main.go
├── config.go
├── database.go
├── migrations.go
├── models.go
├── jetiq.go
├── sync.go
├── schedule.go
├── api.go
├── web.go
├── telegram.go
├── bot.go
├── admin.go
├── status.go
├── backup.go
├── *_test.go
├── go.mod
├── go.sum
├── README.md
└── LICENSE
```

Files should be split when they become difficult to navigate, not according to a preselected architecture pattern. Templates, static assets, and migrations are embedded into the binary with `go:embed`.

The binary should expose subcommands while remaining one deployable artifact:

```text
vntu-timetable serve
vntu-timetable migrate
vntu-timetable sync
vntu-timetable backup
vntu-timetable version
```

## 8. Configuration and secrets

Read configuration from environment variables. In production, systemd loads them from a root-owned file such as `/etc/vntu-timetable/env`.

Required configuration should include:

- listen address, default `127.0.0.1:8080`;
- public base URL, initially `https://vntu-timetable.zelefin.top`;
- SQLite path;
- backup directory and retention count;
- Telegram bot token;
- independent Telegram webhook secret;
- Telegram bot username;
- Telegram administrator user ID;
- timezone, fixed to `Europe/Kyiv` by default;
- JetIQ base URL;
- daily synchronization time, ten-minute retry interval, and request pacing with safe defaults;
- log level;
- build version supplied at link time.

Rules:

- Never put the bot token in the webhook path.
- Generate a separate random webhook secret.
- Never commit production secrets, database files, dumps, or backups.
- Validate configuration at startup and fail with a precise message if required values are missing.
- Avoid a large configuration system; environment parsing and validation should remain explicit.

## 9. SQLite design

Enable and test these database settings on every connection:

- foreign keys;
- WAL journal mode;
- a bounded busy timeout;
- an appropriate synchronous mode;
- a deliberately small connection pool suitable for one process and one writer.

Use embedded, numbered SQL migrations and a schema-version table. Migrations must be transactional where SQLite permits it. Startup may verify the schema version, but production deployment should run migrations explicitly before starting the new process.

Prefer additive migrations during initial development and stabilization: new tables and new nullable columns rather than renaming, retyping, or dropping schema that the previous binary reads. This improves binary rollback but does not guarantee it; every migration must explicitly consider whether the previous binary understands data written by the new binary. Keep the verified pre-deployment backup and restore runbook as the fallback.

### Core tables

#### `faculties`

- JetIQ faculty ID as primary key.
- Name.
- Active flag.
- Created/updated timestamps.

#### `groups`

- JetIQ group ID as primary key.
- Faculty ID foreign key.
- Name.
- Active flag.
- Last successful schedule update timestamp.
- Created/updated timestamps.

Mark missing faculties/groups inactive instead of immediately deleting them. This preserves meaningful references for migrated users while excluding obsolete groups from normal selection.

#### `lessons`

- Local integer primary key.
- Group ID foreign key.
- Week number: 1 or 2.
- Day number: 0 through 6.
- Lesson number.
- Subgroup: 0, 1, or 2 unless JetIQ demonstrates more values.
- Subject name.
- Lesson type.
- Teacher name stored directly.
- Room/auditory.
- Start and end time stored consistently as minutes after midnight or validated `HH:MM` text.
- Created/updated timestamps only if they provide operational value.

Add an index that supports reads ordered by group, week, day, lesson number, and subgroup. Always use an explicit `ORDER BY`.

A separate teachers table is unnecessary for the initial feature set because schedule data already contains the displayed teacher name.

#### `users`

- Telegram user ID as primary key.
- Group ID, nullable.
- Subgroup, nullable.
- Created timestamp.

Derive faculty, group name, username, and full name when needed instead of persisting duplicated or fresh Telegram data. A user whose group is null is unregistered. Validate subgroup as null, 0, 1, or 2 unless JetIQ demonstrates more values.

Do not track an `active` flag. A mailing may retry users who previously blocked the bot; this is an accepted tradeoff for approximately 100 users.

### In-memory bot state

Keep short registration and admin-mailing conversations in a `map[userID]state` protected by a mutex. `/start` or a cancel action replaces/removes abandoned state. No persistence or expiration job is required at the expected scale.

Accepted tradeoff: users partway through registration, and an administrator partway through mailing confirmation, must restart the flow after any process restart, including a normal deployment.

#### `settings`

- Key as primary key.
- Value.
- Updated timestamp.
- Telegram administrator ID responsible for a manual change, when applicable.

Store the university week offset here.

#### `job_runs`

- Job ID.
- Job name: daily sync, backup, or another bounded maintenance job.
- Trigger: scheduled, admin, startup, CLI, or deployment.
- Started and finished timestamps.
- State: running, succeeded, partially succeeded, failed, or interrupted.
- Total, successful, unchanged, and failed item counts.
- Primary error category, nullable: `network`, `upstream_contract`, or `internal`.
- Short error summary.
- Application version.

Retain a bounded history, such as the latest 30 runs per job. On startup, mark an old `running` row as interrupted.

A run may encounter more than one failure category. Store the most important category as the primary category using `internal > upstream_contract > network`, and include secondary category counts in the short summary. Treat the category as a Go-validated text value rather than a rigid database enum that is difficult to extend.

## 10. JetIQ synchronization

### Parsing and external-client behavior

- Capture representative raw JetIQ responses in `testdata/jetiq` before implementing parsing.
- Treat JetIQ data as untrusted external input.
- Use an HTTP client with explicit connection and request timeouts.
- Retry transient network and server errors with bounded exponential backoff and jitter.
- Do not retry permanent parsing or validation errors indefinitely.
- Use a descriptive User-Agent.
- Pace per-group requests. Begin with the existing approximate 2.5-second delay unless testing shows a safer documented limit.
- Never erase good stored data because JetIQ returned an error or malformed response.

### Combined daily synchronization

Run one job daily at 07:00 Europe/Kyiv. The same job implementation serves scheduled, admin, startup-recovery, and CLI triggers:

1. Fetch faculties and every faculty's group list.
2. Validate that the metadata snapshot completed successfully.
3. Upsert current metadata and, only for a complete snapshot, mark missing faculties/groups inactive.
4. Read the active group list.
5. Fetch schedules in a plain sequential loop: one group request at a time, followed by the existing approximate 2.5-second pacing delay. Do not implement a worker pool or per-group goroutines.
6. Replace one group's lessons in one short transaction only after its response parses and validates successfully.
7. Record one complete `job_runs` result and send any required alert.

If an individual group schedule fails:

- continue processing other groups;
- preserve that group's previous schedule;
- leave its `schedule_updated_at` unchanged;
- keep the failed group ID in the running process for a retry after ten minutes;
- retry only failed groups, not the successful batch.

If the process restarts and loses an in-memory retry list, detect the interrupted or stale daily job at startup and schedule a fresh run.

Metadata safety rules:

- Never mark records inactive from a partial metadata response.
- If one faculty's group request fails, preserve all existing active flags, upsert only data known to be safe, use existing groups where possible, and classify the run as partial.
- If the initial faculties request shows that JetIQ is unreachable, stop early rather than making hundreds of predictably failing schedule requests.
- Do not delete groups referenced by users.

Allow only one daily synchronization at a time, regardless of its trigger. Retry failed work after ten minutes until it succeeds. Keep this retry behavior because current-day data freshness is a primary requirement.

### Initial data load

The first production cutover must run an explicit full synchronization before Nginx directs users to the new application. `/readyz` remains unsuccessful until the database contains usable faculty/group data and at least one usable schedule, according to a clearly documented readiness rule.

That readiness threshold is sufficient for normal operation and later deployments, but it is not the initial cutover acceptance signal. Before the first Nginx switch, inspect the initial `job_runs` result and confirm that the failed-group count is zero or that every failure is explicitly reviewed and accepted. This is a manual runbook gate, not additional application code.

## 11. University week calculation

Centralize week calculation in one tested function used by the web, API, and bot.

Store a `week_offset` value of `0` or `1` in `settings`. Calculate the displayed university week from ISO week parity plus this offset.

The admin `/week` command should show:

- today's date in Europe/Kyiv;
- current ISO week;
- calculated university week;
- whether the current offset is the default or manually changed;
- who last changed it and when.

Provide explicit buttons:

- `Set current week to First`
- `Set current week to Second`

Do not expose only a blind “toggle” action. Setting an explicit current week calculates and stores the necessary offset, reducing operator mistakes.

Test year boundaries, daylight-saving transitions where relevant, odd/even parity, and both offset values.

## 12. Public read-only API

Create a new API without compatibility requirements. A minimal initial contract is:

- `GET /api/v1/faculties`
  - returns active faculties and their active groups;
- `GET /api/v1/groups/{groupID}/schedule`
  - returns the group's ordered two-week schedule and freshness information;
- optionally `GET /api/v1/groups/{groupID}` if group metadata is useful independently.

API requirements:

- stable JSON field names;
- proper `200`, `400`, `404`, `405`, and `500` responses;
- one consistent error envelope;
- explicit deterministic lesson ordering;
- no `cached` field;
- no public mutation or refresh routes;
- bounded response sizes;
- `Content-Type: application/json`;
- GET-only cross-origin access may use `Access-Control-Allow-Origin: *` because the API is intentionally public and read-only;
- API handlers call domain/database functions directly;
- API schema examples documented in the repository README.

Do not add pagination unless the actual response size makes it necessary.

## 13. SSR website and Mini App

### Routes

- `GET /`: faculty/group selection and useful landing content.
- `GET /groups/{groupID}`: selected group timetable.
- Query parameters such as `week=1` and `day=0` select the displayed week/day and remain linkable.
- Invalid groups or parameters return useful HTML errors rather than a spinner that never resolves.

Faculty ID does not need to be part of the group timetable URL because the group already belongs to a faculty.

### Rendering and interaction

- Render the initial page fully on the server.
- Use semantic HTML and ordinary links/forms.
- Use a small amount of plain JavaScript only where it materially improves interaction, such as automatic selector submission and Telegram Mini App setup.
- Use the Telegram Mini App SDK only as an enhancement: expand the view and apply relevant Telegram theme information when available.
- The same pages must remain fully usable outside Telegram.
- Prefer a direct group URL in the bot's Web App button. The root page remains a functional fallback and group browser.
- Do not require user authentication to view schedules.

### Visual goals

- Preserve the dark theme and recognizable schedule cards.
- Improve responsive behavior for both narrow Telegram screens and desktop browsers.
- Make faculty/group, week, and day controls clear and keyboard accessible.
- Use visible focus styles, proper labels, and sufficient contrast.
- Display explicit loading only for actual progressive operations; SSR should normally provide immediate useful content.
- Keep three user-facing states: normal (including a legitimately empty day), data may be slightly stale, and not-found/unavailable. The HTTP/API layer must still preserve correct distinctions such as `404` versus `500`; they may share visual treatment without becoming the same technical response.
- Clearly show the selected group, calendar date, university week, lesson time, type, subject, teacher, room, and subgroup.
- Handle more than two lessons at the same time without assuming pairs.
- Use modest handcrafted CSS embedded in the binary; no Tailwind or Node toolchain.

## 14. Telegram integration

### Webhook

- Endpoint: `POST /telegram/webhook`.
- Register it with Telegram using an independent random `secret_token`.
- Require an exact `X-Telegram-Bot-Api-Secret-Token` header match before decoding JSON.
- Do not derive the secret from or expose the bot API token.
- A plain exact string comparison is sufficient; authorization of admin operations still independently requires the configured Telegram administrator user ID.
- Accept only POST and enforce a small request body limit.
- Configure only needed update types: primarily `message` and `callback_query`.
- Return a successful response promptly; long refreshes and mailings run as managed background jobs.
- Use panic recovery at the HTTP boundary, log the stack, classify it as `internal`, and attempt a deduplicated admin alert through the same notification path as synchronization failures.
- Wrap application-owned background goroutines, including synchronization, mailing, backup, and scheduler workers, with equivalent panic recovery. An alert failure must be logged without recursively raising another alert.

Do not persist processed Telegram update IDs. Registration, navigation, and explicit week-setting actions are naturally idempotent. Protect the few non-idempotent operations with narrow single-flight and one-time-state guards.

### Minimal outgoing Bot API client

Implement only required methods, likely including:

- `setWebhook` and `getWebhookInfo`;
- `sendMessage`;
- `editMessageText`;
- `answerCallbackQuery`;
- the Web App button/markup structures;
- `sendPhoto` only if the current welcome image is intentionally preserved;
- any command-registration method chosen during setup.

The client must support Telegram error responses and `retry_after` flood-control handling. Tests use a fake HTTP server; they never call Telegram.

### User commands and flows

#### `/start`

- Ensure the minimal user row exists and use identity information directly from the current Telegram update without persisting it.
- Show current registration if present.
- Offer registration/update.
- Preserve the welcoming behavior without requiring the old exact message formatting.

#### Registration

- Ask for a group name.
- Match normalized group names case-insensitively against active groups.
- Ask for subgroup 1, subgroup 2, or no subgroup.
- Persist the user's choice.
- Store the short-lived conversational state in the mutex-protected in-memory state map.
- Provide clear retry/cancel behavior.

#### `/timetable`

- For registered users, provide a Web App button opening their group URL.
- For unregistered users, direct them to `/start`.
- Do not include sharing controls.

#### `/inline`

- Keep the existing meaning: render the registered user's schedule directly as a Telegram text message.
- Keep weekday, today, tomorrow, and current/next-week navigation through callback buttons.
- Apply subgroup filtering.
- Fix Friday/weekend navigation edge cases.
- Use the central week calculation and Europe/Kyiv timezone.
- Return a clear no-lessons state.

Despite the command name, do not register or process Telegram `inline_query` updates. Remove `@bot group-name` sharing behavior entirely.

### Admin authorization

- Compare the Telegram sender ID to the configured administrator ID for every admin operation.
- Do not rely on usernames for authorization.
- Reject unauthorized callbacks as well as commands.

### `/mailing`

- Keep text/Telegram-HTML mailing only.
- Show a confirmation before sending.
- Atomically consume the pending in-memory confirmation before starting, so a retried callback cannot resend a completed mailing.
- Send at a conservative rate and honor Telegram `retry_after` responses.
- Count blocked or unavailable recipients as failures for that run; do not persist active/inactive status.
- Run as a background job so the webhook responds quickly.
- Prevent two mailings from running simultaneously with a mutex-protected in-memory guard.
- Report success/failure totals to the administrator.
- Do not add photos, forwarding, or arbitrary media in the initial rewrite.

### `/refresh`

Offer a full daily refresh, retry of current failures when applicable, and current refresh status. Enqueue the selected work, acknowledge immediately, enforce the shared daily-sync single-flight rule, and notify the administrator when it finishes. The same synchronization code must serve scheduled, CLI, and admin triggers.

## 15. Simple operational visibility

### Admin `/status`

Proactive alerts are the primary operational interface. `/status` is the secondary, on-demand view for inspecting current and historical behavior. It must be fast and must not make a live JetIQ call merely to render status.

Show at least:

- overall healthy, degraded, or unhealthy state;
- application version/commit and uptime;
- current memory usage and goroutine count;
- SQLite database size and available filesystem space;
- last successful backup time and size;
- total users and registered users;
- last received Telegram update time;
- current university week and parity offset details;
- last and next combined daily synchronization;
- duration and counts from the last run;
- count of groups with stale or missing schedule data;
- current job and progress if one is running;
- consecutive failure count;
- primary failure category: network, upstream contract, or internal;
- short last-error summary and next retry time.

Keep output within Telegram message limits and make it understandable without reading server logs. A more compact summary followed by detail buttons is acceptable if one message becomes too large.

### Proactive admin alerts

Send deduplicated notifications when:

- a daily synchronization fails or is materially partial;
- schedule data passes a stale threshold;
- synchronization recovers after an outage;
- an HTTP handler or application-owned background worker panics;
- backup or integrity verification fails;
- free disk space passes a low-space threshold;
- the application starts after a restart.

Do not notify on every successful daily job by default. Avoid repeated identical alerts during each ten-minute retry; send one incident notification and one recovery notification.

### Health endpoints

- `GET /healthz`: process is alive; return minimal information.
- `GET /readyz`: SQLite is usable, migrations are current, and minimum initial data exists.

Do not expose detailed operational status or user counts publicly. CI/CD uses `/readyz` after deployment.

### Logs

- Write structured, human-readable logs to stdout/stderr for journald.
- Include job IDs, group IDs, request paths, durations, and concise external errors where useful.
- Never log secrets, Telegram webhook headers, bot tokens, or full sensitive Telegram updates.
- Use logs for detailed diagnosis; normal operation should be understandable through `/status` and alerts.

## 16. Local SQLite backups

Use a SQLite-safe online backup operation or another driver-supported consistent mechanism. Never copy only the main database file while a live WAL database may have uncheckpointed data.

Initial default policy:

- create one backup daily during a low-traffic hour;
- create an additional backup immediately before a deployment migration;
- keep the most recent four daily backups;
- store them under a dedicated directory such as `/var/lib/vntu-timetable/backups`;
- keep backups uncompressed because the database is expected to be small and direct verification/restoration is simpler;
- name files with an unambiguous UTC timestamp, for example `vntu-timetable-20260815T031500Z.db`;
- verify that a produced backup opens and passes an appropriate quick integrity check;
- record the result in `job_runs` and show it in `/status`;
- notify the administrator on failure;
- document and test a minimal restore procedure before production cutover: stop the service, move the failed database aside, copy the selected verified backup into place, correct ownership, start the intended binary, and verify readiness.

Because backups remain on the same VPS, they protect against application mistakes and a damaged active database but not total VPS loss. That residual risk is accepted for the initial version.

## 17. Nginx and systemd deployment

### Filesystem layout

A suggested production layout is:

```text
/opt/vntu-timetable/releases/<version>/vntu-timetable
/opt/vntu-timetable/current -> releases/<version>
/var/lib/vntu-timetable/vntu-timetable.db
/var/lib/vntu-timetable/backups/
/etc/vntu-timetable/env
/etc/systemd/system/vntu-timetable.service
```

Run the service as a dedicated unprivileged user. That user owns only the application data and backup directories. Secrets remain readable only by the service account/root.

### systemd

The unit should:

- start `vntu-timetable serve`;
- use the production environment file;
- set the working/data directory explicitly;
- restart on unexpected failure with a reasonable delay;
- stop gracefully with SIGTERM;
- apply practical hardening that does not block the required database and network access;
- send logs to journald;
- use sensible startup/shutdown timeouts.

### Nginx

- Terminate TLS for `vntu-timetable.zelefin.top`.
- Proxy the site, API, health checks, and Telegram webhook to `127.0.0.1:8080`.
- Preserve the real client/proxy headers needed by the application.
- Limit request body size, especially for the webhook.
- Set reasonable connect/read/write timeouts.
- Do not cache dynamic timetable responses initially.
- Do not log secrets; the webhook secret is in a header, not the URL.
- Keep the Go listener inaccessible directly from the public network.

## 18. CI and manually triggered deployment

### Continuous integration

`ci.yml` runs automatically for:

- every pull request;
- every push to `main`;
- optionally other development branches if useful.

Required checks:

- verify formatting (`gofmt` produces no diff);
- `go vet ./...`;
- `go test ./...`;
- `go test -race ./...`; choose a SQLite driver/build setup that supports the race-enabled CI run unless a concrete blocker is documented and reviewed;
- build the Linux `amd64` binary;
- optionally run a maintained vulnerability/static-analysis tool after its cost and stability are evaluated;
- ensure embedded templates, assets, and migrations are present.

CI never modifies production.

### Manual deployment workflow

Use a separate `deploy.yml` with GitHub Actions' `workflow_dispatch` trigger.

Practical behavior:

1. Code is merged to `main`.
2. Automatic CI completes.
3. Nothing is deployed yet.
4. The maintainer opens the repository's **Actions** tab.
5. The maintainer selects **Deploy production** and clicks **Run workflow**, choosing the intended branch/commit.
6. The workflow reruns required checks, builds the artifact, and deploys that exact revision.

This is the initial meaning of “manual approval.” A GitHub protected Environment can later add a second pause requiring an approval click after the workflow starts, but that is unnecessary overhead for a single maintainer. Version tags can also be introduced later for formal releases; they are not required initially.

### Deployment job

- Build for Linux `amd64` in CI, not on the 1 GB VPS.
- Embed version, commit, and build time into the binary.
- Produce and verify a checksum.
- Use a dedicated SSH deployment user and pinned host key.
- Store SSH credentials/host details in GitHub Actions secrets.
- Never disable SSH host-key checking.
- Upload to a new versioned release directory.
- Stop the service, accepting a brief interruption.
- Create and verify a pre-migration SQLite backup.
- Run database migrations using the uploaded binary.
- Switch the `current` symlink or install the new binary atomically.
- Start the service.
- Poll `/readyz` with a bounded timeout.
- Report the deployed version.
- If startup/readiness fails, restore the prior binary first when the migration and data changes are backward compatible.
- If the previous binary cannot safely use the migrated database, follow the tested database-restore procedure using the verified pre-deployment backup.
- Retain a small number of prior binaries for rollback.

The deployment user should have only the narrowly required sudo permission to manage this one systemd service and its release directory.

## 19. Test strategy

Tests should emphasize behavior and failure handling rather than chasing an arbitrary coverage percentage.

### Unit tests

- JetIQ parsing from checked-in sanitized fixtures.
- Missing and malformed JetIQ fields.
- Lesson normalization and deterministic ordering.
- ISO week plus manual offset calculation.
- Today/tomorrow navigation across Friday, weekends, year boundaries, and parity changes.
- Subgroup filtering.
- Telegram HTML escaping and timetable message formatting.
- Group-name normalization.
- Status health/staleness decisions.
- Backup retention selection.

### Database integration tests

- Run against temporary SQLite databases.
- Apply every migration from an empty database.
- Upgrade representative older schemas when migrations are later added.
- Enforce foreign keys and expected indexes.
- Replace a group's schedule atomically.
- Preserve old data after a failed parse/sync.
- Mark missing metadata inactive without breaking users.
- Verify the minimal user foreign key and subgroup constraints.

### HTTP tests

- Use `httptest` for API, SSR, health, and webhook handlers.
- Assert status codes, content types, response schema, and method restrictions.
- Test unknown groups and invalid week/day parameters.
- Test webhook header rejection, body limits, malformed JSON, and safe behavior when idempotent updates or one-time confirmation callbacks are duplicated.
- Test normal browser rendering without Telegram-specific state.
- Use HTML assertions or focused golden files for important pages without making tests fragile to harmless whitespace.

### External-client tests

- Use fake JetIQ and Telegram HTTP servers.
- Test timeouts, retryable failures, permanent failures, invalid JSON, and Telegram flood control.
- Never call live JetIQ or Telegram in the normal test suite.
- Optionally provide an explicitly invoked diagnostic command for live JetIQ contract checking.

### End-to-end checks

- Empty database to initial sync and ready service.
- Register a migrated/new user, choose subgroup, open Mini App, and use `/inline` navigation.
- Admin refresh with progress and completion report.
- Week correction reflected consistently in web, API, and bot.
- Text mailing with successful, blocked, and rate-limited recipients.
- Backup, integrity verification, and restore.
- Deployment health check and binary rollback rehearsal.

## 20. One-time manual user migration

Only the bot's user IDs and group/subgroup selections are migrated. Timetable data is downloaded again from JetIQ. Do not build or maintain an `import-users` Go subcommand for a process that runs once for approximately 100 users.

Perform the migration as a controlled cutover runbook:

1. Run the fresh JetIQ synchronization first so referenced `groups` rows exist; this long operation can happen while the old bot still serves users.
2. Stop the old bot to freeze user writes.
3. Export the final old PostgreSQL users to UTF-8 CSV, retaining only the fields needed by the new schema: Telegram user ID, group ID, subgroup, and creation time.
4. Load the CSV into a temporary SQLite staging table inside a transaction, using the `sqlite3` CLI or a throwaway one-off script that is not part of the maintained application.
5. Query every distinct imported `group_id` that does not resolve to a new `groups` row. A handful of spot checks is not sufficient for foreign-key validation.
6. Resolve every missing group explicitly. If no mapping exists, preserve the user ID with a null group/subgroup so that user can register again; never silently discard a user.
7. Validate subgroup/null values, then insert the staged rows into `users`.
8. Compare source, staged, inserted, registered, and unresolved counts before committing.
9. Confirm that representative imported users with subgroup 0, 1, and 2 can use `/start`, `/timetable`, and `/inline` without registering again.
10. Drop the staging table after validation and keep the old PostgreSQL dump untouched throughout the stabilization period.

Document the exact commands used in the private production cutover notes. The maintained repository only needs the target schema and this runbook, not reusable migration code or tests.

## 21. Implementation sequence and gates

Each phase ends with passing tests and a reviewable result. Avoid a long-lived branch containing the entire rewrite.

### Phase 0: behavior fixtures and project bootstrap

- Create the new repository.
- Add README, Go module, formatting/test commands, and CI.
- Record sanitized representative JetIQ responses.
- Record examples of current API output, timetable messages, and important UI states.
- Decide and document the SQLite driver after a minimal backup/concurrency spike.

Gate: a trivial Linux `amd64` binary builds in CI, and external behavior fixtures exist.

### Phase 1: database, migrations, and core schedule model

- Implement configuration validation.
- Implement SQLite opening, pragmas, migrations, and core tables.
- Implement central lesson, date, and university-week logic.
- Implement repository functions as explicit SQL without a generic repository abstraction.

Gate: migrations and domain/database tests pass from a fresh temporary database.

### Phase 2: JetIQ synchronization

- Implement the JetIQ client and fixture-driven parser.
- Implement the combined daily metadata/schedule job and per-group schedule transactions.
- Implement sequential request pacing, job locking, progress, failed-group retry state, and history.
- Implement CLI synchronization.

Gate: a local fresh database can be populated, failures preserve old data, and job results are inspectable.

### Phase 3: read-only API and SSR website

- Define and document `/api/v1` responses.
- Implement API handlers and HTTP error behavior.
- Build embedded templates and CSS.
- Implement normal website navigation and Mini App enhancement.

Gate: the complete timetable is usable in a normal browser with JavaScript disabled except for optional enhancements, and API tests pass.

### Phase 4: Telegram webhook and user features

- Implement the minimal Telegram client.
- Implement webhook authentication, narrow idempotency guards, and panic alerting.
- Implement minimal user persistence and mutex-protected in-memory conversation state.
- Implement `/start`, registration, `/timetable`, and `/inline` callbacks.
- Explicitly omit inline-query sharing.

Gate: bot flows pass against a fake Telegram server and a staging/test bot works through a webhook.

### Phase 5: administration and visibility

- Implement `/refresh`, `/week`, `/mailing`, and `/status`.
- Add the daily scheduler, ten-minute retry behavior, progress reporting, and alert-first operational visibility.
- Add health/readiness endpoints and structured logs.
- Implement backup, retention, integrity check, and restore documentation.

Gate: an administrator can diagnose sync freshness and trigger recovery without logging into the VPS.

### Phase 6: production packaging and deployment

- Add systemd and Nginx example configurations.
- Add the manual GitHub deployment workflow.
- Provision the unprivileged service/deployment users and production paths.
- Rehearse backup, migration, health check, and rollback on a staging copy or alternate port.

Gate: a manually triggered workflow deploys a known commit and successfully verifies `/readyz`.

### Phase 7: migration and cutover

- Back up both old PostgreSQL databases and existing service configuration.
- Run the initial full JetIQ sync before importing users so group foreign keys exist.
- Inspect the initial `job_runs` result and require zero failed groups or an explicitly reviewed and accepted failure list before switching Nginx.
- Stop the old bot to prevent further user-data changes.
- Export the final approximately 100 existing users to the cutover CSV and validate its counts.
- Load users through the temporary staging table, validate every unresolved group ID, and compare counts before commit.
- Deploy the Go service and update Nginx to serve the website/API from it.
- Update the existing bot's Mini App/menu URL as required.
- Register the Telegram webhook with its independent secret without dropping pending updates unless explicitly chosen.
- Verify the site, API, admin status, registration, `/timetable`, `/inline`, refresh, and a controlled mailing test.

Gate: production behavior is verified, migrated users retain preferences, and rollback remains possible.

### Phase 8: stabilization and retirement

- Monitor `/status`, alerts, journald, memory, DB growth, and sync duration.
- Fix migration-only or upstream-contract issues before adding new product features.
- Keep old services and data available but stopped during a defined stabilization period.
- After confidence is established, remove old containers/services from the VPS while retaining dumps and repositories.
- Rebranding, if still desired, becomes a separate later project.

Gate: stable operation through multiple combined daily syncs, backup restore verification, and one deployment rollback rehearsal.

## 22. Production cutover checklist

- [ ] CI is green for the exact release commit.
- [ ] Production binary checksum and embedded version are recorded.
- [ ] Nginx and systemd configurations have been reviewed and syntax-checked.
- [ ] Telegram webhook secret is independent of the bot token.
- [ ] Old PostgreSQL databases and service configuration are backed up.
- [ ] New SQLite database has a verified backup.
- [ ] Initial combined JetIQ sync ran while the old bot remained available and before user import.
- [ ] Initial `job_runs` result has zero failed groups or an explicitly reviewed and accepted failure list.
- [ ] Old bot polling process is stopped so user writes are frozen.
- [ ] Final old-user CSV was exported after the old bot stopped.
- [ ] Every imported group ID was validated against the fresh `groups` table.
- [ ] Source, staging, inserted, registered, and unresolved user counts reconcile.
- [ ] Week offset reflects the actual university week.
- [ ] New service is started and `/healthz` and `/readyz` pass.
- [ ] Nginx serves `/`, `/api/v1`, and the webhook route correctly.
- [ ] Telegram webhook information reports the expected URL and no pending error.
- [ ] `/start`, existing-user recognition, registration, `/timetable`, and `/inline` work.
- [ ] Admin `/status`, `/refresh`, `/week`, and `/mailing` work.
- [ ] A normal browser and Telegram Mini App both render the website correctly.
- [ ] Rollback commands and the prior binary/service configuration are immediately available.

## 23. Initial release acceptance criteria

The rewrite is ready when:

- one Go process and one SQLite database replace the three application repositories' production runtimes;
- Redis, PostgreSQL, Docker, Python, Node, and frontend build tooling are not required in production;
- the website works normally and as a Telegram Mini App;
- the public API is read-only and documented;
- migrated users retain their group/subgroup information;
- `/inline` schedule navigation works across week and weekend boundaries;
- Telegram inline-query sharing and sharing buttons are absent;
- the combined daily job retains good data during JetIQ failures and retries failed groups safely every ten minutes;
- parity can be corrected through Telegram without editing code;
- proactive alerts report important failures, categories, panics, and recoveries without alert spam;
- `/status` provides useful on-demand detail about freshness, failures, progress, resource use, and backups;
- tests cover external parsing, core domain rules, database transactions, HTTP handlers, bot flows, and migrations;
- a manual GitHub Actions deployment can back up, migrate, deploy, health-check, and roll back the service;
- after the first production deployment, memory and CPU usage are manually inspected through `/status` and host tools; usage is stable over normal operation and leaves adequate headroom on the 1 GB RAM, 1 CPU VPS without requiring a formal benchmark suite.

## 24. Deferred decisions and future work

Do not block the initial rewrite on these items:

- rebranding to `VntuScheduleBot` or a new domain;
- off-site backups;
- teacher search or teacher-specific schedules;
- additional timetable personalization;
- a web administration interface;
- public API authentication or rate-limit products;
- formal tag-based releases;
- zero-downtime deployment;
- additional monitoring systems.

Revisit them only after the monolith is stable and measured in production.
