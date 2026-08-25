package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type AuthorizationVerification struct {
	Valid          bool
	Reason         string
	Message        string
	RevisionNo     int
	ComputedDigest string
}

type canonicalRevision struct {
	PlanID          string           `json:"planId"`
	Title           string           `json:"title"`
	Venue           string           `json:"venue"`
	PerformanceDate string           `json:"performanceDate"`
	RevisionNo      int              `json:"revisionNo"`
	RevisionID      string           `json:"revisionId"`
	LoadPoints      []canonicalPoint `json:"loadPoints"`
	Cues            []canonicalCue   `json:"cues"`
}

type canonicalPoint struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	RatedCapacityKg float64 `json:"ratedCapacityKg"`
	PlannedLoadKg   float64 `json:"plannedLoadKg"`
	AngleDeg        float64 `json:"angleDeg"`
	SafetyFactor    float64 `json:"safetyFactor"`
	Position        string  `json:"position"`
}

type canonicalCue struct {
	ID            string   `json:"id"`
	CueNo         int      `json:"cueNo"`
	Label         string   `json:"label"`
	StartOffsetMs int64    `json:"startOffsetMs"`
	DurationMs    int64    `json:"durationMs"`
	MovingPoints  []string `json:"movingPoints"`
	ClearanceCm   float64  `json:"clearanceCm"`
	Operator      string   `json:"operator"`
}

func FrozenRevisionDigest(plan RiggingPlan, revision PlanRevision) (string, error) {
	points := make([]canonicalPoint, 0, len(revision.LoadPoints))
	for _, point := range revision.LoadPoints {
		points = append(points, canonicalPoint{
			ID: point.ID, Name: point.Name, RatedCapacityKg: point.RatedCapacityKg,
			PlannedLoadKg: point.PlannedLoadKg, AngleDeg: point.AngleDeg,
			SafetyFactor: point.SafetyFactor, Position: point.Position,
		})
	}
	sort.Slice(points, func(i, j int) bool {
		if points[i].Name == points[j].Name {
			return points[i].ID < points[j].ID
		}
		return points[i].Name < points[j].Name
	})
	cues := make([]canonicalCue, 0, len(revision.Cues))
	for _, cue := range revision.Cues {
		moving := append([]string(nil), cue.MovingPoints...)
		sort.Strings(moving)
		cues = append(cues, canonicalCue{
			ID: cue.ID, CueNo: cue.CueNo, Label: cue.Label,
			StartOffsetMs: cue.StartOffsetMs, DurationMs: cue.DurationMs,
			MovingPoints: moving, ClearanceCm: cue.ClearanceCm, Operator: cue.Operator,
		})
	}
	sort.Slice(cues, func(i, j int) bool {
		if cues[i].CueNo == cues[j].CueNo {
			return cues[i].ID < cues[j].ID
		}
		return cues[i].CueNo < cues[j].CueNo
	})
	payload := canonicalRevision{
		PlanID: plan.ID, Title: plan.Title, Venue: plan.Venue,
		PerformanceDate: plan.PerformanceDate, RevisionNo: revision.RevisionNo,
		RevisionID: revision.ID, LoadPoints: points, Cues: cues,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("序列化冻结修订：%w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func BuildAuthorizationCode(digest, reviewID string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(digest) + ":" + reviewID))
	value := strings.ToUpper(hex.EncodeToString(sum[:10]))
	return fmt.Sprintf("RIG-%s-%s-%s-%s", value[0:5], value[5:10], value[10:15], value[15:20])
}

func VerifyAuthorization(plan RiggingPlan, code string) bool {
	if plan.State != StateAuthorized || !strings.EqualFold(plan.AuthorizationCode, strings.TrimSpace(code)) {
		return false
	}
	revision, err := plan.Current()
	if err != nil {
		return false
	}
	digest, err := FrozenRevisionDigest(plan, revision)
	return err == nil && strings.EqualFold(digest, plan.FrozenDigest)
}

func VerifyAuthorizationDetailed(plan RiggingPlan, code string) AuthorizationVerification {
	result := AuthorizationVerification{}
	if plan.State != StateAuthorized {
		result.Reason = "NOT_AUTHORIZED"
		result.Message = "方案尚未处于 AUTHORIZED 状态"
		return result
	}
	if !strings.EqualFold(plan.AuthorizationCode, strings.TrimSpace(code)) {
		result.Reason = "AUTHORIZATION_MISMATCH"
		result.Message = "授权码不属于当前方案"
		return result
	}
	revision, err := plan.Current()
	if err != nil {
		result.Reason = "REVISION_MISSING"
		result.Message = "冻结修订不存在"
		return result
	}
	result.RevisionNo = revision.RevisionNo
	digest, err := FrozenRevisionDigest(plan, revision)
	if err != nil {
		result.Reason = "DIGEST_MISMATCH"
		result.Message = "无法重新计算冻结修订摘要"
		return result
	}
	result.ComputedDigest = digest
	if !strings.EqualFold(digest, plan.FrozenDigest) {
		result.Reason = "DIGEST_MISMATCH"
		result.Message = "当前修订摘要与冻结摘要不一致"
		return result
	}
	foundDerivedCode := false
	for _, review := range plan.Reviews {
		if review.Decision != ReviewApproved || review.RevisionID != revision.ID {
			continue
		}
		if strings.EqualFold(BuildAuthorizationCode(digest, review.ID), strings.TrimSpace(code)) {
			foundDerivedCode = true
			break
		}
	}
	if !foundDerivedCode {
		result.Reason = "AUTHORIZATION_MISMATCH"
		result.Message = "授权码与当前冻结修订的派生关系不一致"
		return result
	}
	result.Valid = true
	result.Reason = "VALID"
	result.Message = "授权码、授权状态与冻结修订摘要一致"
	return result
}
