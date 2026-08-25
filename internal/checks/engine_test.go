package checks

import (
	"testing"
	"time"

	"benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a/internal/domain"
)

func TestEvaluateProducesStableBlockingFindings(t *testing.T) {
	revision := domain.PlanRevision{
		ID: "revision-1",
		LoadPoints: []domain.LoadPoint{
			{Name: "LX-01", RatedCapacityKg: 500, PlannedLoadKg: 450, AngleDeg: 30, SafetyFactor: 1.5},
			{Name: "RX-01", RatedCapacityKg: 1000, PlannedLoadKg: 200, AngleDeg: 0, SafetyFactor: 1.5},
		},
		Cues: []domain.MovementCue{
			{CueNo: 1, Label: "升起", StartOffsetMs: 0, DurationMs: 2000, MovingPoints: []string{"LX-01"}, ClearanceCm: 20, Operator: "甲"},
			{CueNo: 2, Label: "落下", StartOffsetMs: 1000, DurationMs: 2000, MovingPoints: []string{"LX-01"}, ClearanceCm: 50, Operator: "乙"},
		},
	}
	run := New(DefaultConfig()).Evaluate(revision, "run-1", time.Unix(100, 0))
	if run.Passed {
		t.Fatal("存在超载、净空和共享吊点冲突时不应通过")
	}
	wantCodes := []string{"LOAD-001", "LOAD-002", "MOVE-001", "MOVE-002"}
	if len(run.Findings) != len(wantCodes) {
		t.Fatalf("findings=%v, want %d", run.Findings, len(wantCodes))
	}
	for index, code := range wantCodes {
		if run.Findings[index].Code != code {
			t.Fatalf("finding[%d].Code=%s, want %s", index, run.Findings[index].Code, code)
		}
	}
}

func TestEvaluateAllowsIndependentNonOverlappingCues(t *testing.T) {
	revision := domain.PlanRevision{
		ID:         "revision-safe",
		LoadPoints: []domain.LoadPoint{{Name: "LX", RatedCapacityKg: 1000, PlannedLoadKg: 300, SafetyFactor: 1.5}},
		Cues: []domain.MovementCue{
			{CueNo: 1, Label: "上升", DurationMs: 1000, MovingPoints: []string{"LX"}, ClearanceCm: 40, Operator: "甲"},
			{CueNo: 2, Label: "下降", StartOffsetMs: 1000, DurationMs: 1000, MovingPoints: []string{"LX"}, ClearanceCm: 40, Operator: "甲"},
		},
	}
	run := New(DefaultConfig()).Evaluate(revision, "run-safe", time.Now())
	if !run.Passed || len(run.Findings) != 0 {
		t.Fatalf("安全动作应通过，得到 %+v", run.Findings)
	}
}
