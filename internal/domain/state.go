package domain

import "fmt"

var allowedTransitions = map[PlanState]map[PlanState]bool{
	StateDraft: {
		StateCheckBlocked:   true,
		StateRehearsalReady: true,
	},
	StateCheckBlocked: {
		StateDraft: true,
	},
	StateRehearsalReady: {
		StateReviewPending:       true,
		StateRemediationRequired: true,
	},
	StateRemediationRequired: {
		StateDraft: true,
	},
	StateReviewPending: {
		StateAuthorized:          true,
		StateRemediationRequired: true,
	},
	StateAuthorized: {},
}

func (p *RiggingPlan) Transition(next PlanState) error {
	if p.State == next {
		return nil
	}
	if !allowedTransitions[p.State][next] {
		return fmt.Errorf("%w：%s 不能变更为 %s", ErrInvalidState, p.State, next)
	}
	p.State = next
	return nil
}

func (p RiggingPlan) Current() (PlanRevision, error) {
	for i := len(p.Revisions) - 1; i >= 0; i-- {
		if p.Revisions[i].RevisionNo == p.CurrentRevision {
			return p.Revisions[i], nil
		}
	}
	return PlanRevision{}, fmt.Errorf("%w：当前修订不存在", ErrNotFound)
}

func (p RiggingPlan) LatestCheck() (CheckRun, bool) {
	revision, err := p.Current()
	if err != nil {
		return CheckRun{}, false
	}
	for i := len(p.CheckRuns) - 1; i >= 0; i-- {
		if p.CheckRuns[i].RevisionID == revision.ID {
			return p.CheckRuns[i], true
		}
	}
	return CheckRun{}, false
}

func (p RiggingPlan) LatestRehearsal() (RehearsalRecord, bool) {
	revision, err := p.Current()
	if err != nil {
		return RehearsalRecord{}, false
	}
	for i := len(p.Rehearsals) - 1; i >= 0; i-- {
		if p.Rehearsals[i].RevisionID == revision.ID {
			return p.Rehearsals[i], true
		}
	}
	return RehearsalRecord{}, false
}

func ValidState(value PlanState) bool {
	_, ok := allowedTransitions[value]
	return ok
}
