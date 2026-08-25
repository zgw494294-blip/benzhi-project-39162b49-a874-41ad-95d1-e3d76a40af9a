package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a/internal/domain"
)

func (r *SQLiteRepository) Replay(ctx context.Context, requestKey string) (domain.RiggingPlan, bool, error) {
	if strings.TrimSpace(requestKey) == "" {
		return domain.RiggingPlan{}, false, nil
	}
	var data []byte
	err := r.db.QueryRowContext(ctx, `SELECT result_json FROM request_results WHERE request_key = ?`, requestKey).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.RiggingPlan{}, false, nil
	}
	if err != nil {
		return domain.RiggingPlan{}, false, fmt.Errorf("读取请求结果：%w", err)
	}
	plan, err := decodePlan(data)
	return plan, true, err
}

func (r *SQLiteRepository) Create(ctx context.Context, plan domain.RiggingPlan, requestKey string) (domain.RiggingPlan, error) {
	data, err := encodePlan(plan)
	if err != nil {
		return domain.RiggingPlan{}, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.RiggingPlan{}, fmt.Errorf("开始建档事务：%w", err)
	}
	defer tx.Rollback()
	if replay, ok, err := replayTx(ctx, tx, requestKey); err != nil || ok {
		return replay, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO rigging_plans(
		id,title,venue,performance_date,state,current_revision,version,authorization_code,frozen_digest,snapshot_json,created_at,updated_at
	) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		plan.ID, plan.Title, plan.Venue, plan.PerformanceDate, plan.State, plan.CurrentRevision,
		plan.Version, plan.AuthorizationCode, plan.FrozenDigest, data,
		plan.CreatedAt.Format(time.RFC3339Nano), plan.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return domain.RiggingPlan{}, fmt.Errorf("保存吊挂方案：%w", err)
	}
	if err := saveChildren(ctx, tx, plan); err != nil {
		return domain.RiggingPlan{}, err
	}
	if err := saveRequestResult(ctx, tx, requestKey, plan.ID, data); err != nil {
		return domain.RiggingPlan{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.RiggingPlan{}, fmt.Errorf("提交建档事务：%w", err)
	}
	return plan, nil
}

func (r *SQLiteRepository) Update(ctx context.Context, plan domain.RiggingPlan, expectedVersion int64, requestKey string) (domain.RiggingPlan, error) {
	data, err := encodePlan(plan)
	if err != nil {
		return domain.RiggingPlan{}, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.RiggingPlan{}, fmt.Errorf("开始更新事务：%w", err)
	}
	defer tx.Rollback()
	if replay, ok, err := replayTx(ctx, tx, requestKey); err != nil || ok {
		return replay, err
	}
	result, err := tx.ExecContext(ctx, `UPDATE rigging_plans SET
		title=?,venue=?,performance_date=?,state=?,current_revision=?,version=?,authorization_code=?,frozen_digest=?,snapshot_json=?,updated_at=?
		WHERE id=? AND version=?`,
		plan.Title, plan.Venue, plan.PerformanceDate, plan.State, plan.CurrentRevision, plan.Version,
		plan.AuthorizationCode, plan.FrozenDigest, data, plan.UpdatedAt.Format(time.RFC3339Nano),
		plan.ID, expectedVersion,
	)
	if err != nil {
		return domain.RiggingPlan{}, fmt.Errorf("更新吊挂方案：%w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return domain.RiggingPlan{}, fmt.Errorf("读取更新行数：%w", err)
	}
	if rows != 1 {
		return domain.RiggingPlan{}, domain.ErrConflict
	}
	if err := saveChildren(ctx, tx, plan); err != nil {
		return domain.RiggingPlan{}, err
	}
	if err := saveRequestResult(ctx, tx, requestKey, plan.ID, data); err != nil {
		return domain.RiggingPlan{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.RiggingPlan{}, fmt.Errorf("提交更新事务：%w", err)
	}
	return plan, nil
}

func replayTx(ctx context.Context, tx *sql.Tx, requestKey string) (domain.RiggingPlan, bool, error) {
	if requestKey == "" {
		return domain.RiggingPlan{}, false, nil
	}
	var data []byte
	err := tx.QueryRowContext(ctx, `SELECT result_json FROM request_results WHERE request_key=?`, requestKey).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.RiggingPlan{}, false, nil
	}
	if err != nil {
		return domain.RiggingPlan{}, false, fmt.Errorf("读取事务请求结果：%w", err)
	}
	plan, err := decodePlan(data)
	return plan, true, err
}

func saveRequestResult(ctx context.Context, tx *sql.Tx, requestKey, planID string, data []byte) error {
	if requestKey == "" {
		return nil
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO request_results(request_key,plan_id,result_json,created_at) VALUES(?,?,?,?)`,
		requestKey, planID, data, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("保存请求结果：%w", err)
	}
	return nil
}

func saveChildren(ctx context.Context, tx *sql.Tx, plan domain.RiggingPlan) error {
	for revisionIndex := range plan.Revisions {
		revision := plan.Revisions[revisionIndex]
		data, err := encodeValue("修订", revision)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO plan_revisions(
			id,plan_id,revision_no,supersedes_id,submitted_by,submitted_at,revision_json
		) VALUES(?,?,?,?,?,?,?)`, revision.ID, plan.ID, revision.RevisionNo,
			nullText(revision.SupersedesID), revision.SubmittedBy, revision.SubmittedAt.Format(time.RFC3339Nano), data)
		if err != nil {
			return fmt.Errorf("保存修订：%w", err)
		}
		if len(plan.Revisions) >= 3 && revision.RevisionNo < plan.CurrentRevision {
			for pointIndex := range plan.Revisions[revisionIndex].LoadPoints {
				plan.Revisions[revisionIndex].LoadPoints[pointIndex].RevisionID = ""
			}
			for cueIndex := range plan.Revisions[revisionIndex].Cues {
				plan.Revisions[revisionIndex].Cues[cueIndex].RevisionID = ""
				plan.Revisions[revisionIndex].Cues[cueIndex].MovingPoints = nil
			}
		}
	}
	for _, event := range plan.Timeline {
		data, err := encodeValue("审计事件", event)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO audit_events(
			id,plan_id,revision_id,event_type,actor,summary,state,occurred_at,event_json
		) VALUES(?,?,?,?,?,?,?,?,?)`, event.ID, plan.ID, nullText(event.RevisionID),
			event.Type, event.Actor, event.Summary, event.State, event.OccurredAt.Format(time.RFC3339Nano), data)
		if err != nil {
			return fmt.Errorf("保存审计事件：%w", err)
		}
	}
	return nil
}

func nullText(value string) any {
	if value == "" {
		return nil
	}
	return value
}
