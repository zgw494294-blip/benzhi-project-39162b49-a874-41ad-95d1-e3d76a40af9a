package application

import (
	"context"
	"time"

	"benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a/internal/domain"
)

func (s *Service) RecordRehearsal(ctx context.Context, command RecordRehearsalCommand) (domain.RiggingPlan, error) {
	if replayed, ok, err := s.replay(ctx, command.RequestKey); err != nil || ok {
		return replayed, err
	}
	plan, err := s.repository.Get(ctx, command.PlanID)
	if err != nil {
		return domain.RiggingPlan{}, err
	}
	if command.Version != plan.Version {
		return domain.RiggingPlan{}, domain.ErrConflict
	}
	current, err := plan.Current()
	if err != nil {
		return domain.RiggingPlan{}, err
	}
	now := s.clock.Now().UTC()
	startedAt := command.StartedAt.UTC()
	completedAt := command.CompletedAt.UTC()
	if startedAt.IsZero() {
		startedAt = now.Add(-30 * time.Minute)
	}
	if completedAt.IsZero() {
		completedAt = now
	}
	revisionID := command.RevisionID
	if revisionID == "" {
		revisionID = current.ID
	}
	record := domain.RehearsalRecord{
		ID: s.ids.NewID(), PlanID: plan.ID, RevisionID: revisionID,
		StartedAt: startedAt, CompletedAt: completedAt,
		Observer: command.Observer, Outcome: command.Outcome,
		Observations: command.Observations, EvidenceRefs: command.EvidenceRefs,
	}
	if err := plan.RecordRehearsal(record, now); err != nil {
		return domain.RiggingPlan{}, err
	}
	plan.Version++
	return s.repository.Update(commitContext(ctx), plan, command.Version, s.requestKey(command.RequestKey))
}
