package domain

import "testing"

func TestFrozenRevisionDigestIsCanonical(t *testing.T) {
	plan := RiggingPlan{ID: "plan", Title: "飞景", Venue: "主舞台", PerformanceDate: "2030-01-02"}
	revision := PlanRevision{
		ID: "revision", RevisionNo: 2,
		LoadPoints: []LoadPoint{
			{ID: "b", Name: "RX", RatedCapacityKg: 1000, PlannedLoadKg: 200, SafetyFactor: 1.5},
			{ID: "a", Name: "LX", RatedCapacityKg: 900, PlannedLoadKg: 180, SafetyFactor: 1.5},
		},
		Cues: []MovementCue{
			{ID: "q2", CueNo: 2, Label: "落下", MovingPoints: []string{"RX", "LX"}, DurationMs: 1000},
			{ID: "q1", CueNo: 1, Label: "升起", MovingPoints: []string{"LX"}, DurationMs: 1000},
		},
	}
	first, err := FrozenRevisionDigest(plan, revision)
	if err != nil {
		t.Fatal(err)
	}
	revision.LoadPoints[0], revision.LoadPoints[1] = revision.LoadPoints[1], revision.LoadPoints[0]
	revision.Cues[0], revision.Cues[1] = revision.Cues[1], revision.Cues[0]
	second, err := FrozenRevisionDigest(plan, revision)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("规范化摘要受切片顺序影响：%s != %s", first, second)
	}
}

func TestAuthorizationCodeVerificationRejectsMutation(t *testing.T) {
	revision := PlanRevision{ID: "r1", RevisionNo: 1, LoadPoints: []LoadPoint{{ID: "p", Name: "LX", RatedCapacityKg: 1000}}, Cues: []MovementCue{{ID: "q", CueNo: 1}}}
	plan := RiggingPlan{ID: "plan", Title: "测试", Venue: "主舞台", PerformanceDate: "2030-01-01", State: StateAuthorized, CurrentRevision: 1, Revisions: []PlanRevision{revision}}
	digest, err := FrozenRevisionDigest(plan, revision)
	if err != nil {
		t.Fatal(err)
	}
	plan.FrozenDigest = digest
	plan.AuthorizationCode = BuildAuthorizationCode(digest, "review")
	if !VerifyAuthorization(plan, plan.AuthorizationCode) {
		t.Fatal("原始冻结方案应通过验证")
	}
	plan.Revisions[0].LoadPoints[0].RatedCapacityKg = 999
	if VerifyAuthorization(plan, plan.AuthorizationCode) {
		t.Fatal("冻结数据变更后不应通过验证")
	}
}
