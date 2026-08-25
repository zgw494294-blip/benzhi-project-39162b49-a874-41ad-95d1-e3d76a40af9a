package domain

import (
	"strings"
	"testing"
)

func TestBuildRevisionDiffsAndRemediationClosure(t *testing.T) {
	first := PlanRevision{
		ID: "r1", RevisionNo: 1,
		LoadPoints: []LoadPoint{{Name: "LX", PlannedLoadKg: 480}, {Name: "RX", PlannedLoadKg: 200}},
		Cues:       []MovementCue{{CueNo: 2, Label: "旧动作"}},
	}
	second := PlanRevision{
		ID: "r2", RevisionNo: 2, SupersedesID: "r1",
		LoadPoints: []LoadPoint{{Name: "LX", PlannedLoadKg: 300}, {Name: "CX", PlannedLoadKg: 100}},
		Cues:       []MovementCue{{CueNo: 2, Label: "新动作"}, {CueNo: 10, Label: "新增动作"}},
	}
	plan := RiggingPlan{
		CurrentRevision: 2, Revisions: []PlanRevision{first, second},
		CheckRuns: []CheckRun{
			{RevisionID: "r1", Findings: []CheckFinding{{Code: "OVERLOAD", Severity: SeverityBlocker}}},
			{RevisionID: "r2", Passed: true},
		},
		Rehearsals: []RehearsalRecord{
			{RevisionID: "r1", Outcome: RehearsalBlocked, Observations: "抖动", EvidenceRefs: []string{"VID-1"}},
			{RevisionID: "r2", Outcome: RehearsalPassed, EvidenceRefs: []string{"VID-2"}},
		},
	}

	diffs := plan.BuildRevisionDiffs()
	if len(diffs) != 1 || len(diffs[0].Entries) != 5 {
		t.Fatalf("差异数量不正确：%+v", diffs)
	}
	closure := diffs[0].Closure
	if len(closure.BlockingFindings) != 1 || closure.OldObservations != "抖动" || !closure.CurrentRechecked || !closure.CurrentRehearsalPassed {
		t.Fatalf("整改闭环关联不正确：%+v", closure)
	}
	if diffs[0].Entries[0].Subject != "CUE" || diffs[0].Entries[0].Identifier != "2" {
		t.Fatalf("差异排序不稳定：%+v", diffs[0].Entries)
	}
}

func TestNormalizeEvidenceRefsRejectsInvalidAndPreservesFirstOccurrence(t *testing.T) {
	values, fields := NormalizeEvidenceRefs([]string{" VID-1 ", "VID-1", "CHECK-2"})
	if len(fields) != 0 || len(values) != 2 || values[0] != "VID-1" || values[1] != "CHECK-2" {
		t.Fatalf("证据规范化不正确：values=%v fields=%v", values, fields)
	}
	_, fields = NormalizeEvidenceRefs([]string{" ", "bad\nref", strings.Repeat("界", EvidenceRefMaxLength+1)})
	if len(fields) != 3 {
		t.Fatalf("应逐项返回字段错误：%+v", fields)
	}
}
