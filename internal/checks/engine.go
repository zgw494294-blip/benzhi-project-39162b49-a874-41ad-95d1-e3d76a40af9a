package checks

import (
	"sort"
	"time"

	"benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a/internal/domain"
)

func (e *Engine) Evaluate(revision domain.PlanRevision, runID string, now time.Time) domain.CheckRun {
	findings := make([]domain.CheckFinding, 0)
	findings = append(findings, e.checkLoads(revision.LoadPoints)...)
	findings = append(findings, e.checkCues(revision.Cues)...)
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Code == findings[j].Code {
			return findings[i].Subject < findings[j].Subject
		}
		return findings[i].Code < findings[j].Code
	})
	passed := true
	for _, finding := range findings {
		if finding.Severity == domain.SeverityBlocker {
			passed = false
			break
		}
	}
	return domain.CheckRun{
		ID: runID, RevisionID: revision.ID, CheckedAt: now,
		Findings: findings, Passed: passed,
	}
}
