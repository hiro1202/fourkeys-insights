# Changelog

All notable changes to this project will be documented in this file.

## [0.1.0.0] - 2026-04-04

### Added
- Aggregation unit setting: choose weekly (Mon-Sun) or monthly periods for metrics and trends
- Trend charts: lead time, deploy frequency, CFR, and MTTR over 3/6/12 months (ECharts)
- Settings page with per-group configuration (aggregation unit, lead time start, MTTR start, incident rules)
- Separate MTTR start point setting, configurable independently from change lead time
- Deploy frequency now shows period total count (weekly/monthly) instead of per-day rate
- New Group button on dashboard for creating additional groups
- Last sync timestamp displayed next to the sync button
- Status bar showing period, total PRs, lead time start, and MTTR start
- Info icon tooltips on metrics cards (replaces ? text)
- Group trends API endpoint (`GET /api/v1/groups/:id/trends`)
- Group settings API endpoints (`GET/PUT /api/v1/groups/:id/settings`)
- MIT license for open source distribution
- GitHub Actions CI pipeline with frontend (TypeScript, i18n, Vite build) and backend (gofmt, vet, staticcheck, test with coverage) jobs
- CI posts check results and coverage reports as PR comments automatically
- Pre-commit hooks via lefthook (gofmt, go vet, go test, tsc, i18n validation)
- Custom JSON serialization types (`JSONNullString`, `JSONNullTime`, `JSONNullInt64`) for proper API responses

### Fixed
- Missing `/settings` and `/setup` routes causing blank pages
- Dashboard white screen on reload caused by `sql.NullString` serializing as objects instead of flat values
- Deprecated `Repositories.List` API call replaced with `ListByAuthenticatedUser` (go-github v62)
- `go test` exit code not propagated through pipe in CI (added `set -o pipefail`)
- lefthook glob patterns now recursive (`**/*.go` instead of `*.go`)
