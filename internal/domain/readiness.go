package domain

import "fmt"

type ReviewReadiness struct {
	Eligible               bool `json:"eligible"`
	CurrentCheckPassed     bool `json:"currentCheckPassed"`
	CurrentRehearsalPassed bool `json:"currentRehearsalPassed"`
	BlockerCount           int  `json:"blockerCount"`
}

func (p RiggingPlan) ReviewReadiness() ReviewReadiness {
	result := ReviewReadiness{}
	check, found := p.LatestCheck()
	if found {
		result.CurrentCheckPassed = check.Passed
		result.BlockerCount = countBlockers(check.Findings)
	}
	rehearsal, found := p.LatestRehearsal()
	if found {
		result.CurrentRehearsalPassed = rehearsal.Outcome == RehearsalPassed
	}
	result.Eligible = result.CurrentCheckPassed && result.BlockerCount == 0 && result.CurrentRehearsalPassed
	return result
}

func (p RiggingPlan) RequireReviewReadiness() error {
	readiness := p.ReviewReadiness()
	if readiness.Eligible {
		return nil
	}
	if !readiness.CurrentCheckPassed || readiness.BlockerCount > 0 {
		return fmt.Errorf("%w：当前修订仍有阻断校核项", ErrInvalidState)
	}
	return fmt.Errorf("%w：当前修订尚无通过的技术联排", ErrInvalidState)
}
