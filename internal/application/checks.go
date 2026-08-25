package application

import (
	"context"
	"strings"

	"benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a/internal/domain"
)

func (s *Service) RunChecks(ctx context.Context, command RunChecksCommand) (domain.RiggingPlan, error) {
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
	run := s.checker.Evaluate(current, s.ids.NewID(), now)
	actor := strings.TrimSpace(command.Actor)
	if actor == "" {
		actor = "自动安全校核"
	}
	if err := plan.RecordCheck(run, actor, now); err != nil {
		return domain.RiggingPlan{}, err
	}
	plan.Version++
	return s.repository.Update(commitContext(ctx), plan, command.Version, s.requestKey(command.RequestKey))
}
