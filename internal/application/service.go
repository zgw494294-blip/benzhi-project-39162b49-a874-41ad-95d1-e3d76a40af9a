package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a/internal/domain"
)

type Service struct {
	repository Repository
	checker    SafetyChecker
	clock      Clock
	ids        IDGenerator
}

func NewService(repository Repository, checker SafetyChecker) *Service {
	return &Service{
		repository: repository,
		checker:    checker,
		clock:      systemClock{},
		ids:        RandomIDs{},
	}
}

func NewServiceWithDependencies(repository Repository, checker SafetyChecker, clock Clock, ids IDGenerator) *Service {
	return &Service{repository: repository, checker: checker, clock: clock, ids: ids}
}

func (s *Service) GetPlan(ctx context.Context, id string) (domain.RiggingPlan, error) {
	plan, err := s.repository.Get(ctx, strings.TrimSpace(id))
	if err == nil {
		plan.RevisionDiffs = plan.BuildRevisionDiffs()
	}
	return plan, err
}

func (s *Service) ListPlans(ctx context.Context) ([]domain.RiggingPlan, error) {
	return s.repository.List(ctx)
}

func (s *Service) ListPlansFiltered(ctx context.Context, query PlanListQuery) (PlanListResult, error) {
	filter, err := validatePlanListQuery(query)
	if err != nil {
		return PlanListResult{}, err
	}
	filtered, ok := s.repository.(PlanQueryRepository)
	if !ok {
		return PlanListResult{}, fmt.Errorf("仓储不支持方案筛选")
	}
	plans, counts, err := filtered.ListFiltered(ctx, filter)
	if err != nil {
		return PlanListResult{}, err
	}
	return PlanListResult{Plans: plans, StateCounts: counts}, nil
}

func validatePlanListQuery(query PlanListQuery) (domain.PlanListFilter, error) {
	filter := domain.PlanListFilter{
		State: domain.PlanState(strings.TrimSpace(query.State)), VenueKeyword: strings.TrimSpace(query.Venue),
		PerformanceDateFrom: strings.TrimSpace(query.PerformanceDateFrom), PerformanceDateTo: strings.TrimSpace(query.PerformanceDateTo),
	}
	var fields []domain.FieldError
	if filter.State != "" && !domain.ValidState(filter.State) {
		fields = domain.AddFieldError(fields, "state", "不是有效的方案状态")
	}
	from, fromOK := parseOptionalISODate(filter.PerformanceDateFrom, "performanceDateFrom", &fields)
	to, toOK := parseOptionalISODate(filter.PerformanceDateTo, "performanceDateTo", &fields)
	if fromOK && toOK && from.After(to) {
		fields = domain.AddFieldError(fields, "performanceDateTo", "不能早于 performanceDateFrom")
	}
	if len(fields) > 0 {
		return domain.PlanListFilter{}, &domain.ValidationError{Fields: fields}
	}
	return filter, nil
}

func parseOptionalISODate(value, field string, fields *[]domain.FieldError) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		*fields = domain.AddFieldError(*fields, field, "必须使用 YYYY-MM-DD")
		return time.Time{}, false
	}
	return parsed, true
}

func (s *Service) requestKey(value string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return s.ids.NewID()
}

func (s *Service) replay(ctx context.Context, requestKey string) (domain.RiggingPlan, bool, error) {
	if strings.TrimSpace(requestKey) == "" {
		return domain.RiggingPlan{}, false, nil
	}
	plan, ok, err := s.repository.Replay(ctx, strings.TrimSpace(requestKey))
	if err != nil {
		return domain.RiggingPlan{}, false, fmt.Errorf("读取请求重放结果：%v", err)
	}
	return plan, ok, nil
}
