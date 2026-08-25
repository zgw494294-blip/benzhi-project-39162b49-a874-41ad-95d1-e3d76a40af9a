package domain

import (
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const EvidenceRefMaxLength = 256

func (p *RiggingPlan) RecordCheck(run CheckRun, actor string, now time.Time) error {
	if p.State != StateDraft {
		return fmt.Errorf("%w：只有草拟状态可执行校核", ErrInvalidState)
	}
	current, err := p.Current()
	if err != nil {
		return err
	}
	if run.RevisionID != current.ID {
		return fmt.Errorf("%w：校核结果不属于当前修订", ErrValidation)
	}
	p.CheckRuns = append(p.CheckRuns, run)
	next := StateRehearsalReady
	summary := "自动校核通过，可进入技术联排"
	if !run.Passed {
		next = StateCheckBlocked
		summary = fmt.Sprintf("自动校核发现 %d 个阻断项", countBlockers(run.Findings))
	}
	if err := p.Transition(next); err != nil {
		return err
	}
	p.UpdatedAt = now
	p.AppendEvent(AuditEvent{
		ID: run.ID + "-checked", PlanID: p.ID, RevisionID: current.ID,
		Type: "CHECK_COMPLETED", Actor: actor, Summary: summary,
		State: p.State, OccurredAt: now,
	})
	return nil
}

func (p *RiggingPlan) RecordRehearsal(record RehearsalRecord, now time.Time) error {
	if p.State != StateRehearsalReady {
		return fmt.Errorf("%w：只有联排就绪状态可记录联排", ErrInvalidState)
	}
	current, err := p.Current()
	if err != nil {
		return err
	}
	var fields []FieldError
	if record.RevisionID != current.ID {
		fields = AddFieldError(fields, "revisionId", "必须是当前修订")
	}
	if strings.TrimSpace(record.Observer) == "" {
		fields = AddFieldError(fields, "observer", "不能为空")
	}
	if record.Outcome != RehearsalPassed && record.Outcome != RehearsalBlocked {
		fields = AddFieldError(fields, "outcome", "必须为 PASSED 或 BLOCKED")
	}
	if record.CompletedAt.Before(record.StartedAt) {
		fields = AddFieldError(fields, "completedAt", "不能早于开始时间")
	}
	normalizedEvidence, evidenceFields := NormalizeEvidenceRefs(record.EvidenceRefs)
	fields = append(fields, evidenceFields...)
	if record.Outcome == RehearsalPassed && len(normalizedEvidence) == 0 {
		fields = AddFieldError(fields, "evidenceRefs", "PASSED 联排至少需要一条证据引用")
	}
	if record.Outcome == RehearsalBlocked && strings.TrimSpace(record.Observations) == "" {
		fields = AddFieldError(fields, "observations", "阻断联排必须填写观察记录")
	}
	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	record.EvidenceRefs = normalizedEvidence
	p.Rehearsals = append(p.Rehearsals, record)
	next := StateReviewPending
	summary := "技术联排通过，进入独立评审"
	if record.Outcome == RehearsalBlocked {
		next = StateRemediationRequired
		summary = "技术联排阻断，需要提交整改修订"
	}
	if err := p.Transition(next); err != nil {
		return err
	}
	p.UpdatedAt = now
	p.AppendEvent(AuditEvent{
		ID: record.ID + "-rehearsed", PlanID: p.ID, RevisionID: current.ID,
		Type: "REHEARSAL_RECORDED", Actor: record.Observer, Summary: summary,
		State: p.State, OccurredAt: now,
	})
	return nil
}

func NormalizeEvidenceRefs(values []string) ([]string, []FieldError) {
	result := make([]string, 0, len(values))
	fields := make([]FieldError, 0)
	seen := make(map[string]bool, len(values))
	for index, value := range values {
		field := fmt.Sprintf("evidenceRefs[%d]", index)
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			fields = AddFieldError(fields, field, "不能为空")
			continue
		}
		if utf8.RuneCountInString(trimmed) > EvidenceRefMaxLength {
			fields = AddFieldError(fields, field, fmt.Sprintf("长度不能超过 %d 个字符", EvidenceRefMaxLength))
			continue
		}
		invalidControl := false
		for _, character := range trimmed {
			if unicode.IsControl(character) {
				invalidControl = true
				break
			}
		}
		if invalidControl {
			fields = AddFieldError(fields, field, "不能包含控制字符")
			continue
		}
		if !seen[trimmed] {
			seen[trimmed] = true
			result = append(result, trimmed)
		}
	}
	return result, fields
}

func (p *RiggingPlan) RecordReview(review SafetyReview, now time.Time) error {
	if p.State != StateReviewPending {
		return fmt.Errorf("%w：只有待评审状态可执行安全评审", ErrInvalidState)
	}
	if err := p.RequireReviewReadiness(); err != nil {
		return err
	}
	current, err := p.Current()
	if err != nil {
		return err
	}
	if strings.EqualFold(strings.TrimSpace(review.Reviewer), strings.TrimSpace(current.SubmittedBy)) {
		return ErrRoleSeparation
	}
	if review.RevisionID != current.ID || strings.TrimSpace(review.Reviewer) == "" {
		return &ValidationError{Fields: []FieldError{{Field: "reviewer", Message: "评审人不能为空且修订必须为当前修订"}}}
	}
	if review.Decision != ReviewApproved && review.Decision != ReviewRejected {
		return &ValidationError{Fields: []FieldError{{Field: "decision", Message: "必须为 APPROVED 或 REJECTED"}}}
	}
	if review.Decision == ReviewRejected && strings.TrimSpace(review.Comment) == "" {
		return &ValidationError{Fields: []FieldError{{Field: "comment", Message: "退回评审必须说明原因"}}}
	}
	p.Reviews = append(p.Reviews, review)
	next := StateRemediationRequired
	summary := "独立评审退回，需要整改"
	if review.Decision == ReviewApproved {
		next = StateAuthorized
		summary = "独立评审通过，演出启用单已冻结"
		p.FrozenDigest = review.FrozenDigest
		p.AuthorizationCode = review.AuthorizationCode
	}
	if err := p.Transition(next); err != nil {
		return err
	}
	p.UpdatedAt = now
	p.AppendEvent(AuditEvent{
		ID: review.ID + "-reviewed", PlanID: p.ID, RevisionID: current.ID,
		Type: "REVIEW_DECIDED", Actor: review.Reviewer, Summary: summary,
		State: p.State, OccurredAt: now,
	})
	return nil
}

func (p *RiggingPlan) AppendEvent(event AuditEvent) {
	p.Timeline = append(p.Timeline, event)
}

func countBlockers(findings []CheckFinding) int {
	count := 0
	for _, finding := range findings {
		if finding.Severity == SeverityBlocker {
			count++
		}
	}
	return count
}
