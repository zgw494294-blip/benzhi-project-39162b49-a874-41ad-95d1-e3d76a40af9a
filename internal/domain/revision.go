package domain

import (
	"fmt"
	"strings"
	"time"
)

func (p *RiggingPlan) AddRevision(revision PlanRevision, now time.Time) error {
	if p.State == StateAuthorized {
		return fmt.Errorf("%w：已启用方案不能再修订", ErrInvalidState)
	}
	if p.State != StateDraft && p.State != StateCheckBlocked && p.State != StateRemediationRequired {
		return fmt.Errorf("%w：状态 %s 不能提交修订", ErrInvalidState, p.State)
	}
	if err := ValidateRevision(revision); err != nil {
		return err
	}
	expected := p.CurrentRevision + 1
	if revision.RevisionNo != expected {
		return fmt.Errorf("%w：修订号应为 %d", ErrValidation, expected)
	}
	if p.CurrentRevision == 0 && revision.SupersedesID != "" {
		return fmt.Errorf("%w：首个修订不能替代其他修订", ErrValidation)
	}
	if p.CurrentRevision > 0 {
		current, err := p.Current()
		if err != nil {
			return err
		}
		if revision.SupersedesID != current.ID {
			return fmt.Errorf("%w：新修订必须替代当前修订", ErrValidation)
		}
	}
	for i := range revision.LoadPoints {
		revision.LoadPoints[i].RevisionID = revision.ID
	}
	for i := range revision.Cues {
		revision.Cues[i].RevisionID = revision.ID
	}
	p.Revisions = append(p.Revisions, revision)
	p.CurrentRevision = revision.RevisionNo
	p.UpdatedAt = now
	if p.State != StateDraft {
		if err := p.Transition(StateDraft); err != nil {
			return err
		}
	}
	p.AppendEvent(AuditEvent{
		ID: revision.ID + "-submitted", PlanID: p.ID, RevisionID: revision.ID,
		Type: "REVISION_SUBMITTED", Actor: revision.SubmittedBy,
		Summary: fmt.Sprintf("提交第 %d 版：%s", revision.RevisionNo, revision.ChangeReason),
		State:   p.State, OccurredAt: now,
	})
	return nil
}

func ValidateRevision(revision PlanRevision) error {
	var fields []FieldError
	if strings.TrimSpace(revision.ID) == "" {
		fields = AddFieldError(fields, "id", "不能为空")
	}
	if strings.TrimSpace(revision.ChangeReason) == "" {
		fields = AddFieldError(fields, "changeReason", "不能为空")
	}
	if strings.TrimSpace(revision.SubmittedBy) == "" {
		fields = AddFieldError(fields, "submittedBy", "不能为空")
	}
	if len(revision.LoadPoints) == 0 {
		fields = AddFieldError(fields, "loadPoints", "至少需要一个吊点")
	}
	if len(revision.Cues) == 0 {
		fields = AddFieldError(fields, "cues", "至少需要一个动作")
	}
	pointNames := map[string]bool{}
	for i, point := range revision.LoadPoints {
		prefix := fmt.Sprintf("loadPoints[%d]", i)
		name := strings.TrimSpace(point.Name)
		if name == "" {
			fields = AddFieldError(fields, prefix+".name", "不能为空")
		} else if pointNames[name] {
			fields = AddFieldError(fields, prefix+".name", "吊点名称不能重复")
		}
		pointNames[name] = true
		if point.RatedCapacityKg <= 0 {
			fields = AddFieldError(fields, prefix+".ratedCapacityKg", "必须大于 0")
		}
		if point.PlannedLoadKg < 0 {
			fields = AddFieldError(fields, prefix+".plannedLoadKg", "不能小于 0")
		}
		if point.AngleDeg < 0 || point.AngleDeg >= 90 {
			fields = AddFieldError(fields, prefix+".angleDeg", "应在 0 到 90 度之间")
		}
		if point.SafetyFactor <= 0 {
			fields = AddFieldError(fields, prefix+".safetyFactor", "必须大于 0")
		}
	}
	cueNumbers := map[int]bool{}
	for i, cue := range revision.Cues {
		prefix := fmt.Sprintf("cues[%d]", i)
		if cue.CueNo <= 0 || cueNumbers[cue.CueNo] {
			fields = AddFieldError(fields, prefix+".cueNo", "必须是唯一正整数")
		}
		cueNumbers[cue.CueNo] = true
		if strings.TrimSpace(cue.Label) == "" {
			fields = AddFieldError(fields, prefix+".label", "不能为空")
		}
		if cue.StartOffsetMs < 0 || cue.DurationMs <= 0 {
			fields = AddFieldError(fields, prefix, "起始时间不能为负且持续时间必须大于 0")
		}
		if len(cue.MovingPoints) == 0 {
			fields = AddFieldError(fields, prefix+".movingPoints", "至少需要一个吊点")
		}
		for _, name := range cue.MovingPoints {
			if !pointNames[name] {
				fields = AddFieldError(fields, prefix+".movingPoints", "引用了不存在的吊点 "+name)
			}
		}
	}
	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}
