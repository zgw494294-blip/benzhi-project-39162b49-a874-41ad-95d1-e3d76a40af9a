package application_test

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a/internal/application"
	"benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a/internal/checks"
	"benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a/internal/domain"
	"benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a/internal/store"
)

func TestCompleteRemediationAndAuthorizationFlow(t *testing.T) {
	ctx := context.Background()
	repository, err := store.Open(ctx, filepath.Join(t.TempDir(), "flow.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repository.Close()
	service := application.NewService(repository, checks.New(checks.DefaultConfig()))
	plan, err := service.CreatePlan(ctx, application.CreatePlanCommand{
		RequestKey: "create", Title: "主景片吊挂", Venue: "大剧场", PerformanceDate: "2030-05-01",
		Owner: "技术负责人", ChangeReason: "初版", SubmittedBy: "方案提交者",
		LoadPoints: []domain.LoadPoint{{Name: "LX", Position: "左", RatedCapacityKg: 500, PlannedLoadKg: 480, SafetyFactor: 1.5}},
		Cues:       []domain.MovementCue{{CueNo: 1, Label: "升起", DurationMs: 2000, MovingPoints: []string{"LX"}, ClearanceCm: 20, Operator: "甲"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := service.CreatePlan(ctx, application.CreatePlanCommand{RequestKey: "create"})
	if err != nil || replayed.ID != plan.ID {
		t.Fatalf("请求键应返回首次建档结果：plan=%s replay=%s err=%v", plan.ID, replayed.ID, err)
	}
	plan, err = service.RunChecks(ctx, application.RunChecksCommand{RequestKey: "check-1", PlanID: plan.ID, Version: plan.Version, Actor: "负责人"})
	if err != nil || plan.State != domain.StateCheckBlocked {
		t.Fatalf("不安全首版应被阻断：state=%s err=%v", plan.State, err)
	}
	plan, err = service.SubmitRevision(ctx, application.SubmitRevisionCommand{
		RequestKey: "revision-2", PlanID: plan.ID, Version: plan.Version, ChangeReason: "降低载荷并增加净空", SubmittedBy: "方案提交者",
		LoadPoints: []domain.LoadPoint{{Name: "LX", Position: "左", RatedCapacityKg: 1000, PlannedLoadKg: 300, SafetyFactor: 1.5}},
		Cues:       []domain.MovementCue{{CueNo: 1, Label: "升起", DurationMs: 2000, MovingPoints: []string{"LX"}, ClearanceCm: 50, Operator: "甲"}},
	})
	if err != nil || plan.CurrentRevision != 2 || plan.State != domain.StateDraft {
		t.Fatalf("整改修订失败：revision=%d state=%s err=%v", plan.CurrentRevision, plan.State, err)
	}
	plan, err = service.RunChecks(ctx, application.RunChecksCommand{RequestKey: "check-2", PlanID: plan.ID, Version: plan.Version, Actor: "负责人"})
	if err != nil || plan.State != domain.StateRehearsalReady {
		t.Fatalf("整改修订应通过校核：state=%s err=%v", plan.State, err)
	}
	plan, err = service.RecordRehearsal(ctx, application.RecordRehearsalCommand{
		RequestKey: "rehearsal", PlanID: plan.ID, Version: plan.Version,
		Observer: "联排监督员", Outcome: domain.RehearsalPassed, Observations: "全行程稳定", EvidenceRefs: []string{"VID-001"},
	})
	if err != nil || plan.State != domain.StateReviewPending {
		t.Fatalf("通过联排应进入待评审：state=%s err=%v", plan.State, err)
	}
	_, err = service.DecideReview(ctx, application.DecideReviewCommand{
		RequestKey: "bad-review", PlanID: plan.ID, Version: plan.Version,
		Reviewer: "方案提交者", Decision: domain.ReviewApproved,
	})
	if !errors.Is(err, domain.ErrRoleSeparation) {
		t.Fatalf("提交者评审应被拒绝，得到 %v", err)
	}
	plan, err = service.DecideReview(ctx, application.DecideReviewCommand{
		RequestKey: "review", PlanID: plan.ID, Version: plan.Version,
		Reviewer: "独立评审员", Decision: domain.ReviewApproved, Comment: "资料和证据完整",
	})
	if err != nil || plan.State != domain.StateAuthorized || plan.AuthorizationCode == "" {
		t.Fatalf("独立评审启用失败：state=%s code=%s err=%v", plan.State, plan.AuthorizationCode, err)
	}
	verification, err := service.VerifyAuthorization(ctx, plan.AuthorizationCode)
	if err != nil || !verification.Valid {
		t.Fatalf("授权验证失败：result=%+v err=%v", verification, err)
	}
	stored, err := service.GetPlan(ctx, plan.ID)
	if err != nil || len(stored.Revisions) != 2 || len(stored.Timeline) < 7 {
		t.Fatalf("完整历史投影不正确：revisions=%d timeline=%d err=%v", len(stored.Revisions), len(stored.Timeline), err)
	}
}

func TestPlanFilteringStatisticsAndValidation(t *testing.T) {
	ctx := context.Background()
	repository, err := store.Open(ctx, filepath.Join(t.TempDir(), "filter.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repository.Close()
	service := application.NewService(repository, checks.New(checks.DefaultConfig()))
	for index, fixture := range []struct{ title, venue, date string }{
		{"早场", "人民大剧场", "2030-04-01"},
		{"晚场", "人民实验剧场", "2030-05-01"},
		{"巡演", "滨海剧院", "2030-06-01"},
	} {
		_, err := service.CreatePlan(ctx, application.CreatePlanCommand{
			RequestKey: fixture.title, Title: fixture.title, Venue: fixture.venue, PerformanceDate: fixture.date,
			Owner: "负责人", ChangeReason: "首版", SubmittedBy: "提交者",
			LoadPoints: []domain.LoadPoint{{Name: "P", RatedCapacityKg: 1000, PlannedLoadKg: float64(100 + index), SafetyFactor: 1.5}},
			Cues:       []domain.MovementCue{{CueNo: 1, Label: "动作", DurationMs: 1000, MovingPoints: []string{"P"}, ClearanceCm: 40}},
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	result, err := service.ListPlansFiltered(ctx, application.PlanListQuery{Venue: "人民", PerformanceDateFrom: "2030-04-15", PerformanceDateTo: "2030-05-15"})
	if err != nil || len(result.Plans) != 1 || result.Plans[0].Title != "晚场" || result.Plans[0].CurrentRevision != 1 {
		t.Fatalf("组合筛选不正确：result=%+v err=%v", result, err)
	}
	if result.StateCounts[domain.StateDraft] != 1 || result.StateCounts[domain.StateAuthorized] != 0 {
		t.Fatalf("状态统计不正确：%+v", result.StateCounts)
	}
	empty, err := service.ListPlansFiltered(ctx, application.PlanListQuery{Venue: "不存在"})
	if err != nil || empty.Plans == nil || len(empty.Plans) != 0 || len(empty.StateCounts) != 6 {
		t.Fatalf("空结果语义不正确：result=%+v err=%v", empty, err)
	}
	_, err = service.ListPlansFiltered(ctx, application.PlanListQuery{State: "UNKNOWN", PerformanceDateFrom: "2030-05-02", PerformanceDateTo: "2030-05-01"})
	var validation *domain.ValidationError
	if !errors.As(err, &validation) || len(validation.Fields) != 2 {
		t.Fatalf("筛选校验应返回字段错误：%v", err)
	}
}

func TestRehearsalEvidenceValidationIsAtomic(t *testing.T) {
	ctx := context.Background()
	repository, err := store.Open(ctx, filepath.Join(t.TempDir(), "evidence.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repository.Close()
	service := application.NewService(repository, checks.New(checks.DefaultConfig()))
	plan, err := service.CreatePlan(ctx, application.CreatePlanCommand{
		Title: "证据测试", Venue: "测试剧场", PerformanceDate: "2030-05-01", Owner: "负责人", ChangeReason: "首版", SubmittedBy: "提交者",
		LoadPoints: []domain.LoadPoint{{Name: "P", RatedCapacityKg: 1000, PlannedLoadKg: 100, SafetyFactor: 1.5}},
		Cues:       []domain.MovementCue{{CueNo: 1, Label: "动作", DurationMs: 1000, MovingPoints: []string{"P"}, ClearanceCm: 40}},
	})
	if err != nil {
		t.Fatal(err)
	}
	plan, err = service.RunChecks(ctx, application.RunChecksCommand{PlanID: plan.ID, Version: plan.Version})
	if err != nil {
		t.Fatal(err)
	}
	beforeVersion, beforeEvents := plan.Version, len(plan.Timeline)
	_, err = service.RecordRehearsal(ctx, application.RecordRehearsalCommand{
		PlanID: plan.ID, RevisionID: "stale", Version: plan.Version, Observer: "监督员", Outcome: domain.RehearsalPassed,
		EvidenceRefs: []string{" ", "bad\nref", strings.Repeat("X", domain.EvidenceRefMaxLength+1)},
	})
	var validation *domain.ValidationError
	if !errors.As(err, &validation) || len(validation.Fields) < 4 {
		t.Fatalf("联排校验应返回逐字段错误：%v", err)
	}
	stored, err := service.GetPlan(ctx, plan.ID)
	if err != nil || stored.Version != beforeVersion || len(stored.Rehearsals) != 0 || len(stored.Timeline) != beforeEvents {
		t.Fatalf("失败联排不应写入：plan=%+v err=%v", stored, err)
	}
	current, _ := stored.Current()
	stored, err = service.RecordRehearsal(ctx, application.RecordRehearsalCommand{
		PlanID: stored.ID, RevisionID: current.ID, Version: stored.Version, Observer: "监督员", Outcome: domain.RehearsalPassed,
		EvidenceRefs: []string{" VID-1 ", "VID-1", "CHECK-2"},
	})
	if err != nil || len(stored.Rehearsals) != 1 || len(stored.Rehearsals[0].EvidenceRefs) != 2 || stored.Rehearsals[0].EvidenceRefs[0] != "VID-1" {
		t.Fatalf("成功联排未保存规范证据：plan=%+v err=%v", stored, err)
	}
}

func TestAuthorizationUsesPersistedRevisionAndStableReasons(t *testing.T) {
	ctx := context.Background()
	repository, err := store.Open(ctx, filepath.Join(t.TempDir(), "authorization.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repository.Close()
	service := application.NewService(repository, checks.New(checks.DefaultConfig()))
	invalid, err := service.VerifyAuthorization(ctx, "bad-code")
	if err != nil || invalid.Reason != "INVALID_FORMAT" {
		t.Fatalf("格式错误 reason 不稳定：%+v err=%v", invalid, err)
	}
	plan, err := service.CreatePlan(ctx, application.CreatePlanCommand{
		Title: "授权复核", Venue: "测试剧场", PerformanceDate: "2030-05-01", Owner: "负责人", ChangeReason: "首版", SubmittedBy: "提交者",
		LoadPoints: []domain.LoadPoint{{Name: "P", RatedCapacityKg: 1000, PlannedLoadKg: 100, SafetyFactor: 1.5}},
		Cues:       []domain.MovementCue{{CueNo: 1, Label: "动作", DurationMs: 1000, MovingPoints: []string{"P"}, ClearanceCm: 40}},
	})
	if err != nil {
		t.Fatal(err)
	}
	plan, _ = service.RunChecks(ctx, application.RunChecksCommand{PlanID: plan.ID, Version: plan.Version})
	plan, _ = service.RecordRehearsal(ctx, application.RecordRehearsalCommand{PlanID: plan.ID, Version: plan.Version, Observer: "监督员", Outcome: domain.RehearsalPassed, EvidenceRefs: []string{"VID-1"}})
	plan, err = service.DecideReview(ctx, application.DecideReviewCommand{PlanID: plan.ID, Version: plan.Version, Reviewer: "独立评审员", Decision: domain.ReviewApproved})
	if err != nil {
		t.Fatal(err)
	}
	valid, err := service.VerifyAuthorization(ctx, strings.ToLower(plan.AuthorizationCode))
	if err != nil || !valid.Valid || valid.Reason != "VALID" || valid.FrozenRevisionDigest == "" {
		t.Fatalf("大小写不敏感复核失败：%+v err=%v", valid, err)
	}
	revision, err := repository.GetRevision(ctx, plan.ID, plan.CurrentRevision)
	if err != nil {
		t.Fatal(err)
	}
	revision.LoadPoints[0].PlannedLoadKg++
	data, err := json.Marshal(revision)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.DB().ExecContext(ctx, `UPDATE plan_revisions SET revision_json=? WHERE id=?`, data, revision.ID); err != nil {
		t.Fatal(err)
	}
	mismatch, err := service.VerifyAuthorization(ctx, plan.AuthorizationCode)
	if err != nil || mismatch.Valid || mismatch.Reason != "DIGEST_MISMATCH" {
		t.Fatalf("持久化修订变化应被识别：%+v err=%v", mismatch, err)
	}
}

func TestOptimisticVersionConflict(t *testing.T) {
	ctx := context.Background()
	repository, err := store.Open(ctx, filepath.Join(t.TempDir(), "conflict.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repository.Close()
	service := application.NewService(repository, checks.New(checks.DefaultConfig()))
	plan, err := service.CreatePlan(ctx, application.CreatePlanCommand{
		Title: "冲突测试", Venue: "测试场", PerformanceDate: "2030-01-01", Owner: "甲", ChangeReason: "首版", SubmittedBy: "甲",
		LoadPoints: []domain.LoadPoint{{Name: "P", RatedCapacityKg: 1000, PlannedLoadKg: 100, SafetyFactor: 1.5}},
		Cues:       []domain.MovementCue{{CueNo: 1, Label: "动作", DurationMs: 1000, MovingPoints: []string{"P"}, ClearanceCm: 40}},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.RunChecks(ctx, application.RunChecksCommand{PlanID: plan.ID, Version: plan.Version - 1})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("旧版本应冲突，得到 %v", err)
	}
}
