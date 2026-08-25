package application

import (
	"time"

	"benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a/internal/domain"
)

type CreatePlanCommand struct {
	RequestKey      string               `json:"requestKey"`
	Title           string               `json:"title"`
	Venue           string               `json:"venue"`
	PerformanceDate string               `json:"performanceDate"`
	Owner           string               `json:"owner"`
	ChangeReason    string               `json:"changeReason"`
	SubmittedBy     string               `json:"submittedBy"`
	LoadPoints      []domain.LoadPoint   `json:"loadPoints"`
	Cues            []domain.MovementCue `json:"cues"`
}

type SubmitRevisionCommand struct {
	RequestKey   string               `json:"requestKey"`
	PlanID       string               `json:"-"`
	Version      int64                `json:"version"`
	ChangeReason string               `json:"changeReason"`
	SubmittedBy  string               `json:"submittedBy"`
	LoadPoints   []domain.LoadPoint   `json:"loadPoints"`
	Cues         []domain.MovementCue `json:"cues"`
}

type RunChecksCommand struct {
	RequestKey string `json:"requestKey"`
	PlanID     string `json:"-"`
	Version    int64  `json:"version"`
	Actor      string `json:"actor"`
}

type RecordRehearsalCommand struct {
	RequestKey   string                  `json:"requestKey"`
	PlanID       string                  `json:"-"`
	RevisionID   string                  `json:"revisionId"`
	Version      int64                   `json:"version"`
	StartedAt    time.Time               `json:"startedAt"`
	CompletedAt  time.Time               `json:"completedAt"`
	Observer     string                  `json:"observer"`
	Outcome      domain.RehearsalOutcome `json:"outcome"`
	Observations string                  `json:"observations"`
	EvidenceRefs []string                `json:"evidenceRefs"`
}

type DecideReviewCommand struct {
	RequestKey string                `json:"requestKey"`
	PlanID     string                `json:"-"`
	Version    int64                 `json:"version"`
	Reviewer   string                `json:"reviewer"`
	Decision   domain.ReviewDecision `json:"decision"`
	Comment    string                `json:"comment"`
}

type PlanListQuery struct {
	State               string
	Venue               string
	PerformanceDateFrom string
	PerformanceDateTo   string
}

type PlanListResult struct {
	Plans       []domain.RiggingPlan   `json:"plans"`
	StateCounts domain.PlanStateCounts `json:"stateCounts"`
}

type AuthorizationVerification struct {
	Valid                bool   `json:"valid"`
	AuthorizationCode    string `json:"authorizationCode"`
	PlanID               string `json:"planId,omitempty"`
	Title                string `json:"title,omitempty"`
	Venue                string `json:"venue,omitempty"`
	PerformanceDate      string `json:"performanceDate,omitempty"`
	RevisionNo           int    `json:"revisionNo,omitempty"`
	FrozenDigest         string `json:"frozenDigest,omitempty"`
	FrozenRevisionDigest string `json:"frozenRevisionDigest,omitempty"`
	Reason               string `json:"reason"`
	Message              string `json:"message"`
}
