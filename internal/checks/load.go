package checks

import (
	"fmt"
	"math"

	"benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a/internal/domain"
)

func (e *Engine) checkLoads(points []domain.LoadPoint) []domain.CheckFinding {
	findings := make([]domain.CheckFinding, 0)
	for _, point := range points {
		angleRadians := point.AngleDeg * math.Pi / 180
		cosine := math.Cos(angleRadians)
		effectiveLoad := point.PlannedLoadKg
		if cosine > 0 {
			effectiveLoad = point.PlannedLoadKg / cosine
		}
		if effectiveLoad > point.RatedCapacityKg {
			findings = append(findings, domain.CheckFinding{
				Code: "LOAD-001", Severity: domain.SeverityBlocker, Subject: point.Name,
				Description: fmt.Sprintf("角度修正后合力 %.1f kg 超过额定能力 %.1f kg（角度 %.1f°）", effectiveLoad, point.RatedCapacityKg, point.AngleDeg),
			})
		}
		availableFactor := math.Inf(1)
		if effectiveLoad > 0 {
			availableFactor = point.RatedCapacityKg / effectiveLoad
		}
		requiredFactor := math.Max(e.config.MinimumSafetyFactor, point.SafetyFactor)
		if availableFactor < requiredFactor {
			findings = append(findings, domain.CheckFinding{
				Code: "LOAD-002", Severity: domain.SeverityBlocker, Subject: point.Name,
				Description: fmt.Sprintf("可用安全系数 %.2f 低于要求 %.2f（计划声明 %.2f）", availableFactor, requiredFactor, point.SafetyFactor),
			})
		}
		if point.PlannedLoadKg > 0 && point.PlannedLoadKg < point.RatedCapacityKg*0.05 {
			findings = append(findings, domain.CheckFinding{
				Code: "LOAD-101", Severity: domain.SeverityWarning, Subject: point.Name,
				Description: fmt.Sprintf("计划载荷 %.1f kg 低于额定能力的 5%%，请确认录入单位", point.PlannedLoadKg),
			})
		}
	}
	return findings
}
