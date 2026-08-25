package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a/internal/domain"
)

func (s *Service) CreatePlan(ctx context.Context, command CreatePlanCommand) (domain.RiggingPlan, error) {
	if replayed, ok, err := s.replay(ctx, command.RequestKey); err != nil || ok {
		return replayed, err
	}
	if err := validateCreatePlan(command); err != nil {
		return domain.RiggingPlan{}, err
	}
	now := s.clock.Now().UTC()
	planID := s.ids.NewID()
	revisionID := s.ids.NewID()
	assignRevisionIDs(command.LoadPoints, command.Cues, revisionID, s.ids)
	plan := domain.RiggingPlan{
		ID: planID, Title: strings.TrimSpace(command.Title), Venue: strings.TrimSpace(command.Venue),
		PerformanceDate: command.PerformanceDate, Owner: strings.TrimSpace(command.Owner),
		State: domain.StateDraft, Version: 1, CreatedAt: now, UpdatedAt: now,
	}
	revision := domain.PlanRevision{
		ID: revisionID, PlanID: planID, RevisionNo: 1,
		ChangeReason: strings.TrimSpace(command.ChangeReason),
		LoadPoints:   command.LoadPoints, Cues: command.Cues,
		SubmittedBy: strings.TrimSpace(command.SubmittedBy), SubmittedAt: now,
	}
	if err := plan.AddRevision(revision, now); err != nil {
		return domain.RiggingPlan{}, err
	}
	plan.Version = 1
	plan.AppendEvent(domain.AuditEvent{
		ID: s.ids.NewID(), PlanID: plan.ID, RevisionID: revision.ID,
		Type: "PLAN_CREATED", Actor: plan.Owner,
		Summary: fmt.Sprintf("建立吊挂方案：%s / %s", plan.Venue, plan.PerformanceDate),
		State:   plan.State, OccurredAt: now,
	})
	return s.repository.Create(ctx, plan, s.requestKey(command.RequestKey))
}

func validateCreatePlan(command CreatePlanCommand) error {
	var fields []domain.FieldError
	if strings.TrimSpace(command.Title) == "" {
		fields = domain.AddFieldError(fields, "title", "不能为空")
	}
	if strings.TrimSpace(command.Venue) == "" {
		fields = domain.AddFieldError(fields, "venue", "不能为空")
	}
	if strings.TrimSpace(command.Owner) == "" {
		fields = domain.AddFieldError(fields, "owner", "不能为空")
	}
	if strings.TrimSpace(command.SubmittedBy) == "" {
		fields = domain.AddFieldError(fields, "submittedBy", "不能为空")
	}
	date, err := time.Parse("2006-01-02", command.PerformanceDate)
	if err != nil || date.IsZero() {
		fields = domain.AddFieldError(fields, "performanceDate", "必须使用 YYYY-MM-DD")
	}
	if len(fields) > 0 {
		return &domain.ValidationError{Fields: fields}
	}
	return nil
}

func assignRevisionIDs(points []domain.LoadPoint, cues []domain.MovementCue, revisionID string, ids IDGenerator) {
	for i := range points {
		if strings.TrimSpace(points[i].ID) == "" {
			points[i].ID = ids.NewID()
		}
		points[i].RevisionID = revisionID
	}
	for i := range cues {
		if strings.TrimSpace(cues[i].ID) == "" {
			cues[i].ID = ids.NewID()
		}
		cues[i].RevisionID = revisionID
	}
}
