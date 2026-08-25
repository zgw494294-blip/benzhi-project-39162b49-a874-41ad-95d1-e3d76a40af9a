package application

import (
	"context"
	"time"

	"benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a/internal/domain"
)

type Repository interface {
	Replay(context.Context, string) (domain.RiggingPlan, bool, error)
	Create(context.Context, domain.RiggingPlan, string) (domain.RiggingPlan, error)
	Update(context.Context, domain.RiggingPlan, int64, string) (domain.RiggingPlan, error)
	Get(context.Context, string) (domain.RiggingPlan, error)
	List(context.Context) ([]domain.RiggingPlan, error)
	FindByAuthorization(context.Context, string) (domain.RiggingPlan, error)
}

type PlanQueryRepository interface {
	ListFiltered(context.Context, domain.PlanListFilter) ([]domain.RiggingPlan, domain.PlanStateCounts, error)
}

type AuthorizationRepository interface {
	GetRevision(context.Context, string, int) (domain.PlanRevision, error)
}

type SafetyChecker interface {
	Evaluate(domain.PlanRevision, string, time.Time) domain.CheckRun
}

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID() string
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }
