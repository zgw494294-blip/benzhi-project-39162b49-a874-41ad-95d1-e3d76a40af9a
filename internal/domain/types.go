package domain

import "time"

type PlanState string

const (
	StateDraft               PlanState = "DRAFT"
	StateCheckBlocked        PlanState = "CHECK_BLOCKED"
	StateRehearsalReady      PlanState = "REHEARSAL_READY"
	StateRemediationRequired PlanState = "REMEDIATION_REQUIRED"
	StateReviewPending       PlanState = "REVIEW_PENDING"
	StateAuthorized          PlanState = "AUTHORIZED"
)

type RiggingPlan struct {
	ID                string            `json:"id"`
	Title             string            `json:"title"`
	Venue             string            `json:"venue"`
	PerformanceDate   string            `json:"performanceDate"`
	Owner             string            `json:"owner"`
	State             PlanState         `json:"state"`
	CurrentRevision   int               `json:"currentRevision"`
	Version           int64             `json:"version"`
	CreatedAt         time.Time         `json:"createdAt"`
	UpdatedAt         time.Time         `json:"updatedAt"`
	Revisions         []PlanRevision    `json:"revisions"`
	CheckRuns         []CheckRun        `json:"checkRuns"`
	Rehearsals        []RehearsalRecord `json:"rehearsals"`
	Reviews           []SafetyReview    `json:"reviews"`
	Timeline          []AuditEvent      `json:"timeline"`
	AuthorizationCode string            `json:"authorizationCode,omitempty"`
	FrozenDigest      string            `json:"frozenDigest,omitempty"`
	RevisionDiffs     []RevisionDiff    `json:"revisionDiffs,omitempty"`
}

type PlanListFilter struct {
	State               PlanState
	VenueKeyword        string
	PerformanceDateFrom string
	PerformanceDateTo   string
}

type PlanStateCounts map[PlanState]int

type RevisionDiff struct {
	FromRevisionID string             `json:"fromRevisionId"`
	ToRevisionID   string             `json:"toRevisionId"`
	FromRevisionNo int                `json:"fromRevisionNo"`
	ToRevisionNo   int                `json:"toRevisionNo"`
	Entries        []RevisionChange   `json:"entries"`
	Closure        RemediationClosure `json:"closure"`
}

type RevisionChange struct {
	Kind       string `json:"kind"`
	Subject    string `json:"subject"`
	Identifier string `json:"identifier"`
	Field      string `json:"field,omitempty"`
	OldValue   any    `json:"oldValue,omitempty"`
	NewValue   any    `json:"newValue,omitempty"`
}

type RemediationClosure struct {
	OldRevisionID          string           `json:"oldRevisionId"`
	BlockingFindings       []CheckFinding   `json:"blockingFindings"`
	OldRehearsalOutcome    RehearsalOutcome `json:"oldRehearsalOutcome,omitempty"`
	OldObservations        string           `json:"oldObservations,omitempty"`
	OldEvidenceRefs        []string         `json:"oldEvidenceRefs,omitempty"`
	CurrentRechecked       bool             `json:"currentRechecked"`
	CurrentRehearsalPassed bool             `json:"currentRehearsalPassed"`
}

type PlanRevision struct {
	ID           string        `json:"id"`
	PlanID       string        `json:"planId"`
	RevisionNo   int           `json:"revisionNo"`
	ChangeReason string        `json:"changeReason"`
	LoadPoints   []LoadPoint   `json:"loadPoints"`
	Cues         []MovementCue `json:"cues"`
	SubmittedBy  string        `json:"submittedBy"`
	SubmittedAt  time.Time     `json:"submittedAt"`
	SupersedesID string        `json:"supersedesId,omitempty"`
}

type LoadPoint struct {
	ID              string  `json:"id"`
	RevisionID      string  `json:"revisionId"`
	Name            string  `json:"name"`
	RatedCapacityKg float64 `json:"ratedCapacityKg"`
	PlannedLoadKg   float64 `json:"plannedLoadKg"`
	AngleDeg        float64 `json:"angleDeg"`
	SafetyFactor    float64 `json:"safetyFactor"`
	Position        string  `json:"position"`
}

type MovementCue struct {
	ID            string   `json:"id"`
	RevisionID    string   `json:"revisionId"`
	CueNo         int      `json:"cueNo"`
	Label         string   `json:"label"`
	StartOffsetMs int64    `json:"startOffsetMs"`
	DurationMs    int64    `json:"durationMs"`
	MovingPoints  []string `json:"movingPoints"`
	ClearanceCm   float64  `json:"clearanceCm"`
	Operator      string   `json:"operator"`
}

type CheckSeverity string

const (
	SeverityBlocker CheckSeverity = "BLOCKER"
	SeverityWarning CheckSeverity = "WARNING"
)

type CheckFinding struct {
	Code        string        `json:"code"`
	Severity    CheckSeverity `json:"severity"`
	Subject     string        `json:"subject"`
	Description string        `json:"description"`
}

type CheckRun struct {
	ID         string         `json:"id"`
	RevisionID string         `json:"revisionId"`
	CheckedAt  time.Time      `json:"checkedAt"`
	Findings   []CheckFinding `json:"findings"`
	Passed     bool           `json:"passed"`
}

type RehearsalOutcome string

const (
	RehearsalPassed  RehearsalOutcome = "PASSED"
	RehearsalBlocked RehearsalOutcome = "BLOCKED"
)

type RehearsalRecord struct {
	ID           string           `json:"id"`
	PlanID       string           `json:"planId"`
	RevisionID   string           `json:"revisionId"`
	StartedAt    time.Time        `json:"startedAt"`
	CompletedAt  time.Time        `json:"completedAt"`
	Observer     string           `json:"observer"`
	Outcome      RehearsalOutcome `json:"outcome"`
	Observations string           `json:"observations"`
	EvidenceRefs []string         `json:"evidenceRefs"`
}

type ReviewDecision string

const (
	ReviewApproved ReviewDecision = "APPROVED"
	ReviewRejected ReviewDecision = "REJECTED"
)

type SafetyReview struct {
	ID                string         `json:"id"`
	PlanID            string         `json:"planId"`
	RevisionID        string         `json:"revisionId"`
	Reviewer          string         `json:"reviewer"`
	Decision          ReviewDecision `json:"decision"`
	Comment           string         `json:"comment"`
	DecidedAt         time.Time      `json:"decidedAt"`
	FrozenDigest      string         `json:"frozenDigest,omitempty"`
	AuthorizationCode string         `json:"authorizationCode,omitempty"`
}

type AuditEvent struct {
	ID         string    `json:"id"`
	PlanID     string    `json:"planId"`
	RevisionID string    `json:"revisionId,omitempty"`
	Type       string    `json:"type"`
	Actor      string    `json:"actor"`
	Summary    string    `json:"summary"`
	State      PlanState `json:"state"`
	OccurredAt time.Time `json:"occurredAt"`
}
