# Changelog

All notable changes to this project will be documented in this file.

## [0.1.0.0] - 2026-04-04

### Added
- MIT license for open source distribution
- GitHub Actions CI pipeline with frontend (TypeScript, i18n, Vite build) and backend (gofmt, vet, staticcheck, test with coverage) jobs
- CI posts check results and coverage reports as PR comments automatically
- Pre-commit hooks via lefthook (gofmt, go vet, go test, tsc, i18n validation)
- Custom JSON serialization types (`JSONNullString`, `JSONNullTime`, `JSONNullInt64`) for proper API responses

### Fixed
- Dashboard white screen on reload caused by `sql.NullString` serializing as objects instead of flat values
- Deprecated `Repositories.List` API call replaced with `ListByAuthenticatedUser` (go-github v62)
- `go test` exit code not propagated through pipe in CI (added `set -o pipefail`)
- lefthook glob patterns now recursive (`**/*.go` instead of `*.go`)
