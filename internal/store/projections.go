package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a/internal/domain"
)

func (r *SQLiteRepository) Get(ctx context.Context, id string) (domain.RiggingPlan, error) {
	var data []byte
	err := r.db.QueryRowContext(ctx, `SELECT snapshot_json FROM rigging_plans WHERE id=?`, id).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.RiggingPlan{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.RiggingPlan{}, fmt.Errorf("读取吊挂方案：%w", err)
	}
	plan, err := decodePlan(data)
	if err != nil {
		return domain.RiggingPlan{}, err
	}
	if err := r.hydrateRevisions(ctx, &plan); err != nil {
		return domain.RiggingPlan{}, err
	}
	if err := r.hydrateTimeline(ctx, &plan); err != nil {
		return domain.RiggingPlan{}, err
	}
	return plan, nil
}

func (r *SQLiteRepository) hydrateRevisions(ctx context.Context, plan *domain.RiggingPlan) error {
	rows, err := r.db.QueryContext(ctx, `SELECT revision_json FROM plan_revisions WHERE plan_id=? ORDER BY revision_no`, plan.ID)
	if err != nil {
		return fmt.Errorf("查询完整修订历史：%w", err)
	}
	defer rows.Close()
	revisions := make([]domain.PlanRevision, 0, len(plan.Revisions))
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return fmt.Errorf("扫描修订历史：%w", err)
		}
		var revision domain.PlanRevision
		if err := json.Unmarshal(data, &revision); err != nil {
			return fmt.Errorf("解析修订历史：%w", err)
		}
		revisions = append(revisions, revision)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("遍历修订历史：%w", err)
	}
	plan.Revisions = revisions
	return nil
}

func (r *SQLiteRepository) hydrateTimeline(ctx context.Context, plan *domain.RiggingPlan) error {
	rows, err := r.db.QueryContext(ctx, `SELECT event_json FROM audit_events WHERE plan_id=? ORDER BY occurred_at, id`, plan.ID)
	if err != nil {
		return fmt.Errorf("查询完整审计时间线：%w", err)
	}
	defer rows.Close()
	events := make([]domain.AuditEvent, 0, len(plan.Timeline))
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return fmt.Errorf("扫描审计时间线：%w", err)
		}
		var event domain.AuditEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return fmt.Errorf("解析审计时间线：%w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("遍历审计时间线：%w", err)
	}
	plan.Timeline = events
	return nil
}

func (r *SQLiteRepository) List(ctx context.Context) ([]domain.RiggingPlan, error) {
	plans, _, err := r.ListFiltered(ctx, domain.PlanListFilter{})
	return plans, err
}

func (r *SQLiteRepository) ListFiltered(ctx context.Context, filter domain.PlanListFilter) ([]domain.RiggingPlan, domain.PlanStateCounts, error) {
	where, arguments := planFilterSQL(filter)
	rows, err := r.db.QueryContext(ctx, `SELECT snapshot_json FROM rigging_plans`+where+` ORDER BY performance_date, updated_at, id`, arguments...)
	if err != nil {
		return nil, nil, fmt.Errorf("查询吊挂方案：%w", err)
	}
	defer rows.Close()
	plans := make([]domain.RiggingPlan, 0)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, nil, fmt.Errorf("扫描吊挂方案：%w", err)
		}
		plan, err := decodePlan(data)
		if err != nil {
			return nil, nil, err
		}
		plans = append(plans, plan)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("遍历吊挂方案：%w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, nil, fmt.Errorf("关闭方案查询：%w", err)
	}
	counts := allStateCounts()
	countRows, err := r.db.QueryContext(ctx, `SELECT state, COUNT(*) FROM rigging_plans`+where+` GROUP BY state ORDER BY state`, arguments...)
	if err != nil {
		return nil, nil, fmt.Errorf("统计方案状态：%w", err)
	}
	defer countRows.Close()
	for countRows.Next() {
		var state domain.PlanState
		var count int
		if err := countRows.Scan(&state, &count); err != nil {
			return nil, nil, fmt.Errorf("扫描方案状态统计：%w", err)
		}
		counts[state] = count
	}
	if err := countRows.Err(); err != nil {
		return nil, nil, fmt.Errorf("遍历方案状态统计：%w", err)
	}
	return plans, counts, nil
}

func planFilterSQL(filter domain.PlanListFilter) (string, []any) {
	clauses := make([]string, 0, 4)
	arguments := make([]any, 0, 4)
	if filter.State != "" {
		clauses = append(clauses, "state = ?")
		arguments = append(arguments, filter.State)
	}
	if filter.VenueKeyword != "" {
		clauses = append(clauses, "LOWER(venue) LIKE LOWER(?) ESCAPE '\\'")
		keyword := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(filter.VenueKeyword)
		arguments = append(arguments, "%"+keyword+"%")
	}
	if filter.PerformanceDateFrom != "" {
		clauses = append(clauses, "performance_date >= ?")
		arguments = append(arguments, filter.PerformanceDateFrom)
	}
	if filter.PerformanceDateTo != "" {
		clauses = append(clauses, "performance_date <= ?")
		arguments = append(arguments, filter.PerformanceDateTo)
	}
	if len(clauses) == 0 {
		return "", arguments
	}
	return " WHERE " + strings.Join(clauses, " AND "), arguments
}

func allStateCounts() domain.PlanStateCounts {
	return domain.PlanStateCounts{
		domain.StateDraft: 0, domain.StateCheckBlocked: 0, domain.StateRehearsalReady: 0,
		domain.StateRemediationRequired: 0, domain.StateReviewPending: 0, domain.StateAuthorized: 0,
	}
}

func (r *SQLiteRepository) FindByAuthorization(ctx context.Context, code string) (domain.RiggingPlan, error) {
	var data []byte
	err := r.db.QueryRowContext(ctx,
		`SELECT snapshot_json FROM rigging_plans WHERE authorization_code = ? COLLATE NOCASE`, strings.TrimSpace(code),
	).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.RiggingPlan{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.RiggingPlan{}, fmt.Errorf("查询授权码：%w", err)
	}
	plan, err := decodePlan(data)
	if err != nil {
		return domain.RiggingPlan{}, err
	}
	if err := r.hydrateRevisions(ctx, &plan); err != nil {
		return domain.RiggingPlan{}, err
	}
	if err := r.hydrateTimeline(ctx, &plan); err != nil {
		return domain.RiggingPlan{}, err
	}
	return plan, nil
}

func (r *SQLiteRepository) GetRevision(ctx context.Context, planID string, revisionNo int) (domain.PlanRevision, error) {
	var data []byte
	err := r.db.QueryRowContext(ctx, `SELECT revision_json FROM plan_revisions WHERE plan_id=? AND revision_no=?`,
		planID, revisionNo).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.PlanRevision{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.PlanRevision{}, fmt.Errorf("读取冻结修订：%w", err)
	}
	var revision domain.PlanRevision
	if err := json.Unmarshal(data, &revision); err != nil {
		return domain.PlanRevision{}, fmt.Errorf("解析冻结修订：%w", err)
	}
	return revision, nil
}
