package checks

import (
	"fmt"
	"sort"
	"strings"

	"benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a/internal/domain"
)

func (e *Engine) checkCues(cues []domain.MovementCue) []domain.CheckFinding {
	ordered := append([]domain.MovementCue(nil), cues...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].StartOffsetMs == ordered[j].StartOffsetMs {
			return ordered[i].CueNo < ordered[j].CueNo
		}
		return ordered[i].StartOffsetMs < ordered[j].StartOffsetMs
	})
	findings := make([]domain.CheckFinding, 0)
	for _, cue := range ordered {
		if cue.ClearanceCm < e.config.MinimumClearanceCm {
			findings = append(findings, domain.CheckFinding{
				Code: "MOVE-001", Severity: domain.SeverityBlocker,
				Subject:     fmt.Sprintf("Q%d %s", cue.CueNo, cue.Label),
				Description: fmt.Sprintf("最小净空 %.1f cm 低于要求 %.1f cm", cue.ClearanceCm, e.config.MinimumClearanceCm),
			})
		}
		if strings.TrimSpace(cue.Operator) == "" {
			findings = append(findings, domain.CheckFinding{
				Code: "MOVE-101", Severity: domain.SeverityWarning,
				Subject:     fmt.Sprintf("Q%d %s", cue.CueNo, cue.Label),
				Description: "动作尚未指定操作员",
			})
		}
	}
	for i := 0; i < len(ordered); i++ {
		for j := i + 1; j < len(ordered); j++ {
			left, right := ordered[i], ordered[j]
			if right.StartOffsetMs >= left.StartOffsetMs+left.DurationMs {
				break
			}
			shared := sharedPoints(left.MovingPoints, right.MovingPoints)
			if len(shared) == 0 {
				continue
			}
			findings = append(findings, domain.CheckFinding{
				Code: "MOVE-002", Severity: domain.SeverityBlocker,
				Subject:     fmt.Sprintf("Q%d / Q%d", left.CueNo, right.CueNo),
				Description: fmt.Sprintf("动作时间重叠且共享吊点：%s", strings.Join(shared, "、")),
			})
		}
	}
	return findings
}

func sharedPoints(left, right []string) []string {
	set := make(map[string]bool, len(left))
	for _, point := range left {
		set[point] = true
	}
	var shared []string
	seen := map[string]bool{}
	for _, point := range right {
		if set[point] && !seen[point] {
			shared = append(shared, point)
			seen[point] = true
		}
	}
	sort.Strings(shared)
	return shared
}
