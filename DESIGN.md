# DESIGN.md - Four Keys Insights v1

## Overview

GitHub PRデータからDORA Four Keysメトリクスを算出するローカルツール。
`docker compose up` だけで起動。外部インフラ不要。

## Architecture

```
[Browser] --HTTP--> [Single Container: fourkeys-insights]
                      |-- Static Frontend (React/Vite build, go:embed)
                      |-- Backend API (Go / chi router)
                      |     |-- api/       HTTP handlers
                      |     |-- github/    GitHub API client
                      |     |-- metrics/   Four Keys calculation
                      |     |-- db/        SQLite (interface-based)
                      |     |-- jobs/      Sync job queue
                      |     |-- config/    viper config
                      |-- SQLite (docker volume: ./data/)
                      `-- --> GitHub REST API
```

### Tech Stack

| Layer | Choice | Reason |
|-------|--------|--------|
| Backend | Go (chi, go-github, zap, viper) | Single binary, go:embed, goroutine job queue |
| Frontend | React + Vite + Tailwind CSS + ECharts | Lightweight, fast build, chart expressiveness |
| Data Fetching | TanStack Query | Server state management, cache, sync |
| DB | SQLite (WAL mode) | Zero-config, single file, concurrent read/write |
| Auth | Fine-grained PAT | Minimal permissions (Pull requests: read, Contents: read) |
| Container | Docker (multi-stage build) | Single container, one-command startup |

### Directory Structure

```
repo-root/
  README.md
  LICENSE
  docker-compose.yml
  Dockerfile
  .env.example
  config/
    config.example.yaml
  backend/
    go.mod
    cmd/server/main.go
    internal/
      api/        # router.go, handlers.go, middleware.go
      github/     # client.go, pulls.go, commits.go, repos.go
      metrics/    # fourkeys.go, incident.go, leadtime.go
      jobs/       # queue.go, types.go
      config/     # config.go
      db/         # interface.go, sqlite.go, migrations.go
  frontend/
    package.json
    vite.config.ts
    tailwind.config.js
    src/
      app/
      pages/        # SetupPage, DashboardPage, SettingsPage
      components/   # MetricsCard, Chart, PRTable, Wizard
      api/          # TanStack Query hooks
      charts/       # ECharts configs
      i18n/         # en.json, ja.json, context.tsx
  data/             # docker volume mount (SQLite DB)
```

## DB Schema

```sql
CREATE TABLE repos (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  owner         TEXT NOT NULL,
  name          TEXT NOT NULL,
  full_name     TEXT NOT NULL UNIQUE,
  default_branch TEXT NOT NULL DEFAULT 'main',
  etag          TEXT,           -- GitHub API ETag for conditional requests
  incident_rules TEXT,          -- JSON: {"title_keywords":["revert","hotfix"],...}
  lead_time_start TEXT NOT NULL DEFAULT 'first_commit_at',
  period_days   INTEGER NOT NULL DEFAULT 30,
  created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE repo_groups (
  id               INTEGER PRIMARY KEY AUTOINCREMENT,
  name             TEXT NOT NULL,
  aggregation_unit TEXT NOT NULL DEFAULT 'weekly',
  lead_time_start  TEXT NOT NULL DEFAULT 'first_commit_at',
  mttr_start       TEXT NOT NULL DEFAULT 'first_commit_at',
  incident_rules   TEXT,
  period_days      INTEGER NOT NULL DEFAULT 30,
  created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE repo_group_members (
  group_id INTEGER NOT NULL REFERENCES repo_groups(id) ON DELETE CASCADE,
  repo_id  INTEGER NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
  PRIMARY KEY (group_id, repo_id)
);

CREATE TABLE pull_requests (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
  repo_id         INTEGER NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
  pr_number       INTEGER NOT NULL,
  title           TEXT NOT NULL,
  branch_name     TEXT,
  labels          TEXT,          -- JSON array: ["bug","feature"]
  body            TEXT,          -- PR body for issue linking
  additions       INTEGER DEFAULT 0,
  deletions       INTEGER DEFAULT 0,
  first_commit_at DATETIME,
  created_at      DATETIME NOT NULL,
  merged_at       DATETIME NOT NULL,
  linked_issue_number INTEGER,  -- Parsed from PR body (Closes #N)
  linked_issue_created_at DATETIME,  -- Fetched from GitHub Issues API
  UNIQUE(repo_id, pr_number)
);

CREATE TABLE jobs (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  group_id   INTEGER REFERENCES repo_groups(id),
  status     TEXT NOT NULL DEFAULT 'idle',  -- idle, fetching, computing, complete, failed, cancelled
  progress   TEXT,          -- JSON: {"fetched":42,"total":120,"current_repo":"owner/name"}
  error      TEXT,
  started_at DATETIME,
  completed_at DATETIME
);
```

### SQLite Configuration

```sql
PRAGMA journal_mode=WAL;     -- Concurrent read/write
PRAGMA foreign_keys=ON;
```

### Startup Recovery

On application startup: `UPDATE jobs SET status='failed', error='Process restarted' WHERE status IN ('fetching', 'computing')`.
UI displays banner: "Previous sync was interrupted. Please re-run."

## API Endpoints

```
Authentication:
  POST   /api/v1/auth/validate        -- Validate PAT, return user info

Repositories:
  GET    /api/v1/repos                 -- List repos (auto-discovered from PAT)
  GET    /api/v1/repos/:id/settings    -- Get repo settings (incident rules, lead time start)
  PUT    /api/v1/repos/:id/settings    -- Update repo settings

Groups:
  GET    /api/v1/groups                -- List groups
  POST   /api/v1/groups               -- Create group
  PUT    /api/v1/groups/:id           -- Update group (name, period_days)
  DELETE /api/v1/groups/:id           -- Delete group
  GET    /api/v1/groups/:id/metrics   -- Get group Four Keys metrics
  GET    /api/v1/groups/:id/trends    -- Get trend data points (since, until, unit)
  GET    /api/v1/groups/:id/settings  -- Get group settings
  PUT    /api/v1/groups/:id/settings  -- Update group settings
  GET    /api/v1/groups/:id/export    -- CSV export (ZIP)
  GET    /api/v1/groups/:id/badge     -- DORA level SVG badge

Sync:
  POST   /api/v1/groups/:id/sync      -- Start sync job
  GET    /api/v1/jobs/:id              -- Get job status/progress
  POST   /api/v1/jobs/:id/cancel      -- Cancel running job

Pull Requests:
  GET    /api/v1/groups/:id/pulls     -- List PRs with pagination
```

## Metrics Definitions

### Lead Time for Changes
- Definition: Time from selected start point to PR merge
- Calculation: `merged_at - start_point` (median across group)
- Start point options (per-group setting, configurable separately for lead time and MTTR):
  - `first_commit_at` (default): First commit timestamp in the PR
  - `issue.created_at`: Linked issue's creation time (if issue linked)
  - `pr_created_at`: PR creation timestamp

### Fallback Chain (per start point selection)
| Selected | Fallback 1 | Fallback 2 | Always available |
|----------|-----------|-----------|-----------------|
| first_commit_at | - | - | pr_created_at |
| issue.created_at | first_commit_at | - | pr_created_at |
| pr_created_at | - | - | (always available) |

When fallback occurs, UI displays warning icon next to the affected PR row.

The `GET /groups/:id/settings` endpoint returns a `fallback_stats` array with per-repo fallback counts:
```json
{
  "fallback_stats": [
    { "repo_id": 7, "total_prs": 4, "lead_time_fallbacks": 4, "mttr_fallbacks": 4 }
  ]
}
```
Settings page displays a warning icon (⚠) next to repos with fallback usage, with tooltip showing lead time and MTTR fallback counts separately.

### Deployment Frequency
- Definition: PR merge count to base branch per period
- Calculation: `merged_pr_count` (period total). DORA level classification still uses `merged_pr_count / period_days` internally
- Note: Assumes PR merge = deploy

### Change Failure Rate
- Definition: Percentage of incident PRs
- Calculation: `incident_pr_count / total_pr_count * 100`
- Incident detection: evaluated at query time (not persisted)

### Time to Restore Service (MTTR)
- Definition: Median lead time of incident PRs
- Calculation: `median(merged_at - start_point)` for incident PRs
- MTTR start point is configurable independently from change lead time start point
- Issue-linked MTTR: When a PR body contains `Closes #N` etc., use `issue.created_at` as start point
- Combining with automated issue creation on incidents makes `issue.created_at` a close approximation of detection time, improving MTTR accuracy

### DORA Level Thresholds

| Metric | Elite | High | Medium | Low |
|--------|-------|------|--------|-----|
| Deployment Frequency | Multiple/day | Weekly-daily | Monthly-weekly | <Monthly |
| Lead Time | <1 day | 1 day-1 week | 1 week-1 month | >1 month |
| Change Failure Rate | 0-5% | 5-10% | 10-15% | >15% |
| MTTR | <1 hour | <1 day | <1 week | >1 week |

These thresholds follow DORA's 2023 State of DevOps Report classifications.
Displayed in metrics card tooltips.

## Incident Detection

Evaluated at query time using per-repo rules stored in `repos.incident_rules` (JSON).

Default rules:
```json
{
  "title_keywords": ["revert", "hotfix"],
  "branch_keywords": ["hotfix"],
  "labels": ["incident", "bug"]
}
```

Logic: `is_incident = title matches ANY keyword OR branch matches ANY keyword OR labels contain ANY label`

Rules are mutable. Dashboard always evaluates with current rules.
CSV export includes rule snapshot in header row for reproducibility context.

## Issue Linking

Parse PR body for GitHub closing keywords:
- Pattern: `(closes|close|closed|fix|fixes|fixed|resolve|resolves|resolved)\s+#(\d+)`
- Case-insensitive matching
- First match wins (if multiple issue references)
- Fetch matched issue's `created_at` via `GET /repos/:owner/:repo/issues/:number`
- Store in `pull_requests.linked_issue_number` and `linked_issue_created_at`

Limitations (displayed in UI):
- Only parses PR body (not comments)
- Assumes closing keywords accurately represent incident-related issues
- REST API only (GraphQL `closingIssuesReferences` would be more reliable)

## GitHub API Strategy

### Pagination
- All list endpoints use `per_page=100`
- Loop through pages until empty response or `Link` header has no `next`
- Applies to: repos, pulls, commits

### Rate Limiting
- Concurrency: 1 (sequential requests, default)
- Primary rate limit (5000/hour): Check `X-RateLimit-Remaining` header
- Secondary rate limit (403/429): Respect `Retry-After` header, exponential backoff
- UI: Display "Rate limit reached. Retrying in N seconds..." with progress bar

### ETag Conditional Requests
- Target: `GET /repos/:owner/:repo/pulls` (PR list endpoint)
- Store ETag in `repos.etag` column
- On re-sync: send `If-None-Match` header
- 304 response: skip PR re-fetch for that repo
- Note: Does not apply to commit fetching (always per-PR)

### Repo Auto-Discovery
- `GET /user/repos?per_page=100&type=all` with pagination
- Returns all repos accessible with the PAT
- UI: Searchable list with checkbox selection

## UI Flow

### Setup Wizard (3 steps)

```
Step 1: PAT Input
  [Text input: GitHub Personal Access Token]
  [Validate button] --> calls POST /api/v1/auth/validate
  Success: proceed to Step 2
  Failure: inline error message
  [Back: disabled (first step)]

Step 2: Repository Selection
  [Search filter input]
  [Checkbox list of repos with pagination (50/page)]
  [Select All / Deselect All buttons]
  Selected count badge
  [Back] [Next: enabled when >= 1 repo selected]

Step 3: Group Creation
  [Group name input]
  [Selected repos summary (collapsible)]
  [Back] [Create & Start Sync]
```

### Dashboard

```
+--------------------------------------------------+
| URL: /dashboard/groups/:id                              |
| [Group dropdown] [Dark/Light toggle] [EN/JP] [Gear] |
+--------------------------------------------------+
| [Lead Time] [Deploy Freq] [CFR]    [MTTR]        |
| (card)      (card)        (card)   (card+proxy)  |
+--------------------------------------------------+
| [Lead Time trend] | [Deploy Freq trend]           |
| [CFR trend]       | [MTTR trend]                  |
+--------------------------------------------------+
| [PR Table: paginated, sortable]                   |
| #  Title  Branch  Merged  Lead Time  Incident     |
+--------------------------------------------------+
| [CSV Export] [Re-sync] | Status bar               |
+--------------------------------------------------+
```

Header includes "What is Four Keys?" link next to the app title, linking to Google Cloud's official DORA Four Keys article. URL is localized per i18n language (EN → English blog, JA → Japanese blog).

### Settings (full page at /settings/groups/:id)
- Per-group aggregation unit (weekly/monthly)
- Per-group lead time start point selector
- Per-group MTTR start point selector (independent from lead time)
- Per-group incident detection rules (title keywords, branch keywords, labels)
- Repository list with per-repo fallback markers (warning icon when PRs use fallback start points, separate indicators for lead time and MTTR)
- Group deletion

### Sync States
| State | UI | User Action |
|-------|------|-------------|
| idle | "Start Sync" button enabled | Click to start |
| fetching | Progress bar ("Fetching PRs: 42/120") | Cancel available |
| computing | Spinner "Computing metrics..." | Wait |
| complete | Dashboard displayed | "Re-sync" button |
| failed | Error message + "Retry" button | Retry or edit settings |
| cancelled | "Sync cancelled" -> idle | Start again |

### Edge Cases
- 0 repos from PAT: Empty state with guidance
- 0 merged PRs: Empty state message
- 0 incident PRs: MTTR shows "N/A", CFR shows "0%"
- 1000+ repos: Paginated list (50/page) with search filter
- Sync in progress: Export button disabled
- Browser closed during sync: Job continues, results shown on next visit

## i18n

Self-built JSON + React Context. No react-i18next.

### File Structure
```
frontend/src/i18n/
  en.json      -- English (default)
  ja.json      -- Japanese
  context.tsx  -- I18nProvider, useI18n hook
```

### Key Structure
Flat keys (no nesting):
```json
{
  "setup.title": "Setup",
  "setup.pat_label": "GitHub Personal Access Token",
  "dashboard.lead_time": "Lead Time for Changes",
  "error.pat_invalid": "Invalid token or insufficient permissions"
}
```

### Build-time Validation
Script checks EN and JP JSON files have identical key sets. CI fails on mismatch.

### Language Toggle
Header dropdown: EN | JP. Stored in localStorage.

## Dark Mode

- Tailwind `dark:` variant classes on all components
- Toggle switch in header (sun/moon icon)
- Default: follow OS `prefers-color-scheme`
- Persist choice in `localStorage`
- Implementation: `<html class="dark">` toggle

## DORA Badge API

`GET /api/v1/groups/:id/badge` returns SVG image.

Colors: Elite=green(#22c55e), High=blue(#3b82f6), Medium=yellow(#eab308), Low=red(#ef4444)

Format: shield.io style badge. "DORA | Elite" with colored background.

Cache-Control: `no-cache` (always fresh from DB query).

## CSV Export

`GET /api/v1/groups/:id/export` returns `Content-Type: application/zip`.

### ZIP Contents
1. `metrics_summary.csv`
   ```
   # Exported: 2026-04-04T09:00:00Z
   # Group: backend-team
   # Period: 30 days
   # Lead Time Start: first_commit_at
   # Incident Rules: title_keywords=revert,hotfix; branch_keywords=hotfix; labels=incident,bug
   Period,Lead Time (hours),Deployment Frequency (/day),Change Failure Rate (%),MTTR (hours),DORA Level
   2026-03-05 - 2026-04-04,24.5,2.3,8.5,4.2,High
   ```

2. `pull_requests.csv`
   ```
   PR Number,Title,Repository,Branch,Merged At,Lead Time (hours),Is Incident,Linked Issue
   123,Fix login bug,owner/repo,hotfix/login,2026-04-01T10:00:00Z,12.5,Yes,#45
   ```

Encoding: UTF-8 with BOM (Excel compatibility).

## PAT Security

- **Preferred**: Environment variable `GITHUB_TOKEN` (or config.yaml `github.token`)
- **Alternative**: UI input, stored plaintext in SQLite
- **Priority**: env var > config.yaml > DB
- **Logging**: PAT is NEVER logged. zap fields filter token values
- **Network**: Default bind to `localhost:8080` only. `0.0.0.0` requires explicit config

This is a local-only tool. If an attacker has access to the SQLite file, the machine itself is compromised. Encryption would be security theater (key on same machine).

## Error Handling

See CEO Plan for full error/rescue map. Key patterns:

- GitHub API errors: Specific error types (AuthError, RateLimitError, etc.), not catch-all
- DB errors: WAL mode prevents most locking issues. DiskFull caught and displayed
- Job errors: Failed jobs marked in DB, UI shows error + retry button
- Network errors: 3x retry with exponential backoff

## Testing Strategy

### Backend (go test + httptest + in-memory SQLite)
- Unit: metrics calculation, incident detection, issue parsing, fallback chain
- Integration: API handlers with httptest, GitHub API mock responses
- Key tests: metric correctness (the highest-risk area), sync idempotency, pagination boundaries, rate limit recovery

### Frontend (Vitest + React Testing Library)
- Component rendering with mock data
- i18n key coverage validation

### E2E (Playwright)
- Happy path: Setup wizard -> repo selection -> group creation -> sync -> dashboard
- Error cases: Invalid PAT, zero repos, zero merged PRs
- Mock: MSW (Mock Service Worker) for GitHub API, in-memory SQLite for backend

## Config

```yaml
app:
  port: 8080
  bind: "localhost"    # Use "0.0.0.0" for network access

github:
  token: ""            # Or use GITHUB_TOKEN env var
  api_base_url: "https://api.github.com"

log:
  level: "info"        # debug, info, warn, error

fetch:
  concurrency: 1       # 1 recommended. 2-3 faster but secondary rate limit risk
```

Per-repo settings (incident rules, lead time start, period_days) are stored in DB, not config.yaml.

## Phase 2 Roadmap

1. **PostgreSQL support** (P1): Add PostgreSQL implementation of db/ interface. SQLite remains default
2. **Slack/Webhook notification** (P2): POST to webhook URL on sync completion
3. See TODOS.md for deferred v1 items (SSO, drilldown, incident tool integration)
