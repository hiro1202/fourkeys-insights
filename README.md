# Four Keys Insights

Local DORA Four Keys metrics dashboard powered by GitHub PR data. Single container, zero external infrastructure. Just `docker compose up`.

**[日本語版 README はこちら](README.ja.md)**

## What it does

Fetches merged pull requests from your GitHub repos, calculates the four DORA metrics, and displays them in a browser dashboard.

| Metric | How it's calculated |
|--------|-------------------|
| **Lead Time for Changes** | Median time from first commit (or issue/PR creation) to PR merge |
| **Deployment Frequency** | PR merge count per period (treats merge as deploy) |
| **Change Failure Rate** | % of PRs matching incident rules (title/branch/label keywords) |
| **Time to Restore Service** | Median lead time of incident PRs |

Each metric gets a DORA level: Elite, High, Medium, or Low.

## Quick Start

### Prerequisites

- Docker and Docker Compose
- GitHub Personal Access Token (fine-grained)
  - Required permissions: **Pull requests (read)**, **Contents (read)**
  - To include Organization repos: set the token's **Resource owner** to your Organization (not your personal account). You may need an org admin to approve the token request
  - Optional: **Organization Members (read)** permission enables future team-based filtering

### 1. Clone and configure

```bash
git clone https://github.com/hiro1202/fourkeys-insights.git
cd fourkeys-insights
cp .env.example .env
```

Edit `.env` and add your GitHub token:

```
GITHUB_TOKEN=github_pat_xxxxxxxxxxxx
```

### 2. Start

```bash
docker compose up
```

Open http://localhost:8080 in your browser.

### 3. Setup wizard

1. **Validate token** - Click "Validate" to verify your PAT
2. **Select repositories** - Search and check the repos you want to track. Only repos accessible with your PAT are listed. To see Organization repos, ensure the token's Resource owner is set to the Organization
3. **Create group** - Name your group and start syncing

The dashboard appears after sync completes.

## Features

- **Multi-repo grouping** - Track metrics across multiple repositories as a single team
- **URL-based group persistence** - Selected group is stored in the URL path, so bookmarks and page reloads keep your context
- **Aggregation unit** - Weekly (Mon-Sun) or monthly period for metrics cards and trend charts
- **Selectable lead time start point** - First commit, linked issue creation, or PR creation (configurable separately for lead time and MTTR)
- **Trend charts** - Lead time, deploy frequency, CFR, and MTTR trends over 3/6/12 months
- **Per-repo fallback markers** - Settings page shows which repos have PRs using fallback start points, with separate indicators for lead time and MTTR
- **Incident detection at query time** - Configurable rules (title/branch keywords, labels). Change rules without re-syncing
- **Issue-linked MTTR** - Parses `Closes #N` from PR body to use issue creation as MTTR start
- **ETag conditional requests** - Skips re-fetching unchanged PR lists on re-sync
- **CSV export** - ZIP with metrics summary and PR details (UTF-8 BOM for Excel)
- **DORA reference link** - Dashboard links to official Google Cloud DORA Four Keys article (localized per language)
- **DORA badge** - SVG badge at `/api/v1/groups/:id/badge`
- **PR size distribution** - Histogram chart (XS/S/M/L buckets)
- **Dark mode** - Toggle or follow OS preference
- **i18n** - English and Japanese

## Configuration

### Environment variables

| Variable | Description | Default |
|----------|------------|---------|
| `GITHUB_TOKEN` | GitHub PAT (required) | - |
| `APP_PORT` | Server port | `8080` |
| `APP_BIND` | Bind address | `localhost` |
| `LOG_LEVEL` | Log level (debug/info/warn/error) | `info` |

### Config file

Alternatively, edit `config/config.yaml`:

```yaml
app:
  port: 8080
  bind: "localhost"

github:
  token: ""
  api_base_url: "https://api.github.com"

log:
  level: "info"

fetch:
  concurrency: 1
```

Priority: environment variable > config.yaml > default.

### Per-group settings (via Settings page)

- **Aggregation unit** - Weekly or monthly
- **Change Lead Time start point** - First Commit (recommended), Issue Created, or PR Created
- **MTTR start point** - Same options, configurable independently from lead time. Combining with automated issue creation on incidents improves accuracy
- **Incident detection rules** - Title keywords, branch keywords, label matches

## API

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/auth/validate` | Validate PAT |
| GET | `/api/v1/repos` | List accessible repos |
| GET | `/api/v1/repos/:id/settings` | Get repo settings |
| PUT | `/api/v1/repos/:id/settings` | Update repo settings |
| GET | `/api/v1/groups` | List groups |
| POST | `/api/v1/groups` | Create group |
| PUT | `/api/v1/groups/:id` | Update group |
| DELETE | `/api/v1/groups/:id` | Delete group |
| GET | `/api/v1/groups/:id/metrics` | Get Four Keys metrics |
| GET | `/api/v1/groups/:id/trends` | Get trend data points |
| GET | `/api/v1/groups/:id/settings` | Get group settings |
| PUT | `/api/v1/groups/:id/settings` | Update group settings |
| GET | `/api/v1/groups/:id/pulls` | List PRs (paginated) |
| GET | `/api/v1/groups/:id/export` | CSV export (ZIP) |
| GET | `/api/v1/groups/:id/badge` | DORA level SVG badge |
| POST | `/api/v1/groups/:id/sync` | Start sync job |
| GET | `/api/v1/jobs/:id` | Get job status |
| POST | `/api/v1/jobs/:id/cancel` | Cancel job |

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go (chi, go-github, zap, viper) |
| Frontend | React, Vite, Tailwind CSS, ECharts, TanStack Query |
| Database | SQLite (WAL mode) |
| Container | Docker (multi-stage build, go:embed) |

## Development

### Backend

```bash
cd backend
go run ./cmd/server/
```

### Frontend (dev server with hot reload)

```bash
cd frontend
npm install
npm run dev
```

Vite proxies `/api` to `localhost:8080`.

### Tests

```bash
# Go tests (36 tests)
cd backend && CGO_ENABLED=1 go test ./...

# i18n key validation
cd frontend && node scripts/check-i18n.js

# E2E tests (5 tests)
cd e2e && npm install && npx playwright install chromium && npx playwright test
```

## Documentation

- [DESIGN.md](DESIGN.md) - Architecture, DB schema, API design, metrics definitions
- [TODOS.md](TODOS.md) - Deferred features and Phase 2 roadmap

## License

MIT
