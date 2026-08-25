package store

import (
	"context"
	"fmt"
)

var connectionPragmas = []string{
	`PRAGMA foreign_keys = ON`,
	`PRAGMA journal_mode = WAL`,
	`PRAGMA busy_timeout = 5000`,
}

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS rigging_plans (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		venue TEXT NOT NULL,
		performance_date TEXT NOT NULL,
		state TEXT NOT NULL,
		current_revision INTEGER NOT NULL,
		version INTEGER NOT NULL,
		authorization_code TEXT NOT NULL DEFAULT '',
		frozen_digest TEXT NOT NULL DEFAULT '',
		snapshot_json BLOB NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_rigging_plans_authorization
		ON rigging_plans(authorization_code) WHERE authorization_code <> ''`,
	`CREATE INDEX IF NOT EXISTS idx_rigging_plans_updated ON rigging_plans(updated_at DESC)`,
	`CREATE TABLE IF NOT EXISTS plan_revisions (
		id TEXT PRIMARY KEY,
		plan_id TEXT NOT NULL REFERENCES rigging_plans(id) ON DELETE CASCADE,
		revision_no INTEGER NOT NULL,
		supersedes_id TEXT,
		submitted_by TEXT NOT NULL,
		submitted_at TEXT NOT NULL,
		revision_json BLOB NOT NULL,
		UNIQUE(plan_id, revision_no)
	)`,
	`CREATE TABLE IF NOT EXISTS audit_events (
		id TEXT PRIMARY KEY,
		plan_id TEXT NOT NULL REFERENCES rigging_plans(id) ON DELETE CASCADE,
		revision_id TEXT,
		event_type TEXT NOT NULL,
		actor TEXT NOT NULL,
		summary TEXT NOT NULL,
		state TEXT NOT NULL,
		occurred_at TEXT NOT NULL,
		event_json BLOB NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_audit_plan_time ON audit_events(plan_id, occurred_at, id)`,
	`CREATE TABLE IF NOT EXISTS request_results (
		request_key TEXT PRIMARY KEY,
		plan_id TEXT NOT NULL REFERENCES rigging_plans(id) ON DELETE CASCADE,
		result_json BLOB NOT NULL,
		created_at TEXT NOT NULL
	)`,
}

func (r *SQLiteRepository) migrate(ctx context.Context) error {
	for index, statement := range connectionPragmas {
		if _, err := r.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("配置 SQLite 连接 %d：%w", index+1, err)
		}
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始数据库迁移：%w", err)
	}
	defer tx.Rollback()
	for index, statement := range migrations {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("执行数据库迁移 %d：%w", index+1, err)
		}
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES(1, strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))`,
	); err != nil {
		return fmt.Errorf("记录数据库迁移：%w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交数据库迁移：%w", err)
	}
	return nil
}
