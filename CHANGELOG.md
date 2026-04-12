# Changelog

All notable changes to this project will be documented in this file.

## [0.0.0.1] - 2026-04-13

### Added
- URL-based group persistence: group selection now saved in URL path (`/dashboard/groups/:id`, `/settings/groups/:id`) so bookmarks and shared links restore the correct group
- Per-repo fallback markers on settings page showing which repositories use fallback start points for Lead Time and MTTR calculations
- SVG favicon with DORA metric color bars (blue, green, yellow, red)
- DORA reference link ("What is Four Keys?") in header
- PAT permission guidance for Organization repos in README

### Changed
- Unified DORA metric terminology: "Deploy Frequency" to "Deployment Frequency", "Mean Time to Restore" to "Time to Restore Service"
- Removed "(recommended)" label from first commit start point option
- Settings page navigation switched from query params (`?groupId=`) to URL path segments
- Removed PR Size Distribution feature from scope (DESIGN.md and dead code cleaned up)

### Fixed
- Invalid group IDs in URL now redirect to first available group instead of showing empty state
- Missing `</svg>` closing tag in favicon for Firefox/Safari compatibility
- Fallback tooltip text shortened to prevent line wrapping
