package application

import (
	"context"
	"fmt"
	"strings"

	"benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a/internal/domain"
)

func (s *Service) SubmitRevision(ctx context.Context, command SubmitRevisionCommand) (domain.RiggingPlan, error) {
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
	now := s.clock.Now().UTC()
	current, err := plan.Current()
	if err != nil {
		return domain.RiggingPlan{}, err
	}
	revisionID := s.ids.NewID()
	assignRevisionIDs(command.LoadPoints, command.Cues, revisionID, s.ids)
	revision := domain.PlanRevision{
		ID: revisionID, PlanID: plan.ID, RevisionNo: plan.CurrentRevision + 1,
		ChangeReason: strings.TrimSpace(command.ChangeReason),
		LoadPoints:   command.LoadPoints, Cues: command.Cues,
		SubmittedBy: strings.TrimSpace(command.SubmittedBy), SubmittedAt: now,
		SupersedesID: current.ID,
	}
	if err := plan.AddRevision(revision, now); err != nil {
		return domain.RiggingPlan{}, err
	}
	plan.Version++
	updated, err := s.repository.Update(ctx, plan, command.Version, s.requestKey(command.RequestKey))
	if err != nil {
		return domain.RiggingPlan{}, fmt.Errorf("保存方案修订：%v", err)
	}
	return updated, nil
}
