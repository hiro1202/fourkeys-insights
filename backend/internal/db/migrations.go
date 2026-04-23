package db

import "database/sql"

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS repos (
		id              INTEGER PRIMARY KEY AUTOINCREMENT,
		owner           TEXT NOT NULL,
		name            TEXT NOT NULL,
		full_name       TEXT NOT NULL UNIQUE,
		default_branch  TEXT NOT NULL DEFAULT 'main',
		etag            TEXT,
		incident_rules  TEXT,
		lead_time_start TEXT NOT NULL DEFAULT 'first_commit_at',
		mttr_start      TEXT,
		created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,

	`CREATE TABLE IF NOT EXISTS repo_groups (
		id               INTEGER PRIMARY KEY AUTOINCREMENT,
		name             TEXT NOT NULL,
		aggregation_unit TEXT NOT NULL DEFAULT 'weekly',
		lead_time_start  TEXT NOT NULL DEFAULT 'first_commit_at',
		mttr_start       TEXT NOT NULL DEFAULT 'first_commit_at',
		incident_rules   TEXT,
		created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,

	`CREATE TABLE IF NOT EXISTS repo_group_members (
		group_id INTEGER NOT NULL REFERENCES repo_groups(id) ON DELETE CASCADE,
		repo_id  INTEGER NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
		PRIMARY KEY (group_id, repo_id)
	)`,

	`CREATE TABLE IF NOT EXISTS pull_requests (
		id                      INTEGER PRIMARY KEY AUTOINCREMENT,
		repo_id                 INTEGER NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
		pr_number               INTEGER NOT NULL,
		title                   TEXT NOT NULL,
		branch_name             TEXT,
		labels                  TEXT,
		body                    TEXT,
		additions               INTEGER DEFAULT 0,
		deletions               INTEGER DEFAULT 0,
		first_commit_at         DATETIME,
		created_at              DATETIME NOT NULL,
		merged_at               DATETIME NOT NULL,
		linked_issue_number     INTEGER,
		linked_issue_created_at DATETIME,
		UNIQUE(repo_id, pr_number)
	)`,

	`CREATE TABLE IF NOT EXISTS jobs (
		id           INTEGER PRIMARY KEY AUTOINCREMENT,
		group_id     INTEGER REFERENCES repo_groups(id) ON DELETE CASCADE,
		status       TEXT NOT NULL DEFAULT 'idle',
		progress     TEXT,
		error        TEXT,
		started_at   DATETIME,
		completed_at DATETIME
	)`,
}

func runMigrations(db *sql.DB) error {
	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return err
		}
	}
	return nil
}
