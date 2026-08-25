package application

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a/internal/domain"
)

func (s *Service) DecideReview(ctx context.Context, command DecideReviewCommand) (domain.RiggingPlan, error) {
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
	review := domain.SafetyReview{
		ID: s.ids.NewID(), PlanID: plan.ID, RevisionID: current.ID,
		Reviewer: strings.TrimSpace(command.Reviewer), Decision: command.Decision,
		Comment: strings.TrimSpace(command.Comment), DecidedAt: now,
	}
	if review.Decision == domain.ReviewApproved {
		digest, digestErr := domain.FrozenRevisionDigest(plan, current)
		if digestErr != nil {
			return domain.RiggingPlan{}, digestErr
		}
		review.FrozenDigest = digest
		review.AuthorizationCode = domain.BuildAuthorizationCode(digest, review.ID)
	}
	if err := plan.RecordReview(review, now); err != nil {
		return domain.RiggingPlan{}, err
	}
	plan.Version++
	return s.repository.Update(commitContext(ctx), plan, command.Version, s.requestKey(command.RequestKey))
}

func (s *Service) VerifyAuthorization(ctx context.Context, code string) (AuthorizationVerification, error) {
	trimmed := strings.TrimSpace(code)
	if !authorizationCodePattern.MatchString(trimmed) {
		return AuthorizationVerification{Valid: false, AuthorizationCode: trimmed, Reason: "INVALID_FORMAT", Message: "授权码格式无效"}, nil
	}
	plan, err := s.repository.FindByAuthorization(ctx, trimmed)
	if err != nil {
		if err == domain.ErrNotFound {
			return AuthorizationVerification{Valid: false, AuthorizationCode: trimmed, Reason: "NOT_FOUND", Message: "未找到对应的演出启用单"}, nil
		}
		return AuthorizationVerification{}, err
	}
	if plan.State == domain.StateAuthorized {
		filtered, ok := s.repository.(AuthorizationRepository)
		if !ok {
			return AuthorizationVerification{}, fmt.Errorf("仓储不支持读取冻结修订")
		}
		persistedRevision, revisionErr := filtered.GetRevision(ctx, plan.ID, plan.CurrentRevision)
		if revisionErr != nil {
			if revisionErr == domain.ErrNotFound {
				return authorizationResult(plan, trimmed, domain.AuthorizationVerification{Reason: "REVISION_MISSING", Message: "冻结修订不存在"}), nil
			}
			return AuthorizationVerification{}, revisionErr
		}
		plan.Revisions = []domain.PlanRevision{persistedRevision}
	}
	verification := domain.VerifyAuthorizationDetailed(plan, trimmed)
	return authorizationResult(plan, trimmed, verification), nil
}

func authorizationResult(plan domain.RiggingPlan, code string, verification domain.AuthorizationVerification) AuthorizationVerification {
	result := AuthorizationVerification{
		Valid: verification.Valid, AuthorizationCode: code, PlanID: plan.ID, Title: plan.Title,
		Venue: plan.Venue, PerformanceDate: plan.PerformanceDate,
		RevisionNo: verification.RevisionNo, FrozenDigest: plan.FrozenDigest,
		FrozenRevisionDigest: verification.ComputedDigest, Reason: verification.Reason,
		Message: verification.Message,
	}
	return result
}

var authorizationCodePattern = regexp.MustCompile(`(?i)^RIG-[A-Z0-9]{5}(?:-[A-Z0-9]{5}){3}$`)
