package domain

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// BuildRevisionDiffs 按吊点名称和动作编号比较每个修订及其被替代修订。
func (p RiggingPlan) BuildRevisionDiffs() []RevisionDiff {
	byID := make(map[string]PlanRevision, len(p.Revisions))
	for _, revision := range p.Revisions {
		byID[revision.ID] = revision
	}
	result := make([]RevisionDiff, 0)
	for _, revision := range p.Revisions {
		if revision.SupersedesID == "" {
			continue
		}
		previous, ok := byID[revision.SupersedesID]
		if !ok {
			continue
		}
		diff := RevisionDiff{
			FromRevisionID: previous.ID, ToRevisionID: revision.ID,
			FromRevisionNo: previous.RevisionNo, ToRevisionNo: revision.RevisionNo,
			Entries: buildRevisionChanges(previous, revision),
			Closure: p.buildRemediationClosure(previous, revision),
		}
		result = append(result, diff)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].ToRevisionNo != result[j].ToRevisionNo {
			return result[i].ToRevisionNo < result[j].ToRevisionNo
		}
		return result[i].ToRevisionID < result[j].ToRevisionID
	})
	return result
}

func buildRevisionChanges(previous, current PlanRevision) []RevisionChange {
	changes := make([]RevisionChange, 0)
	oldPoints := make(map[string]LoadPoint, len(previous.LoadPoints))
	newPoints := make(map[string]LoadPoint, len(current.LoadPoints))
	for _, point := range previous.LoadPoints {
		oldPoints[strings.TrimSpace(point.Name)] = point
	}
	for _, point := range current.LoadPoints {
		newPoints[strings.TrimSpace(point.Name)] = point
	}
	pointNames := unionSortedKeys(oldPoints, newPoints)
	for _, name := range pointNames {
		oldPoint, hadOld := oldPoints[name]
		newPoint, hadNew := newPoints[name]
		switch {
		case !hadOld:
			changes = append(changes, RevisionChange{Kind: "ADDED", Subject: "LOAD_POINT", Identifier: name, NewValue: newPoint})
		case !hadNew:
			changes = append(changes, RevisionChange{Kind: "DELETED", Subject: "LOAD_POINT", Identifier: name, OldValue: oldPoint})
		default:
			changes = appendPointChanges(changes, name, oldPoint, newPoint)
		}
	}

	oldCues := make(map[int]MovementCue, len(previous.Cues))
	newCues := make(map[int]MovementCue, len(current.Cues))
	for _, cue := range previous.Cues {
		oldCues[cue.CueNo] = cue
	}
	for _, cue := range current.Cues {
		newCues[cue.CueNo] = cue
	}
	cueNumbers := make([]int, 0, len(oldCues)+len(newCues))
	seen := map[int]bool{}
	for number := range oldCues {
		seen[number] = true
		cueNumbers = append(cueNumbers, number)
	}
	for number := range newCues {
		if !seen[number] {
			cueNumbers = append(cueNumbers, number)
		}
	}
	sort.Ints(cueNumbers)
	for _, number := range cueNumbers {
		oldCue, hadOld := oldCues[number]
		newCue, hadNew := newCues[number]
		identifier := fmt.Sprintf("%d", number)
		switch {
		case !hadOld:
			changes = append(changes, RevisionChange{Kind: "ADDED", Subject: "CUE", Identifier: identifier, NewValue: newCue})
		case !hadNew:
			changes = append(changes, RevisionChange{Kind: "DELETED", Subject: "CUE", Identifier: identifier, OldValue: oldCue})
		default:
			changes = appendCueChanges(changes, identifier, oldCue, newCue)
		}
	}
	sort.SliceStable(changes, func(i, j int) bool {
		if changes[i].Subject != changes[j].Subject {
			return changes[i].Subject < changes[j].Subject
		}
		if changes[i].Identifier != changes[j].Identifier {
			if changes[i].Subject == "CUE" {
				left, _ := strconv.Atoi(changes[i].Identifier)
				right, _ := strconv.Atoi(changes[j].Identifier)
				return left < right
			}
			return changes[i].Identifier < changes[j].Identifier
		}
		return changes[i].Field < changes[j].Field
	})
	return changes
}

func unionSortedKeys[T any](left, right map[string]T) []string {
	seen := make(map[string]bool, len(left)+len(right))
	keys := make([]string, 0, len(left)+len(right))
	for key := range left {
		seen[key] = true
		keys = append(keys, key)
	}
	for key := range right {
		if !seen[key] {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func appendPointChanges(changes []RevisionChange, identifier string, old, new LoadPoint) []RevisionChange {
	values := []struct {
		field string
		old   any
		new   any
	}{
		{"ratedCapacityKg", old.RatedCapacityKg, new.RatedCapacityKg},
		{"plannedLoadKg", old.PlannedLoadKg, new.PlannedLoadKg},
		{"angleDeg", old.AngleDeg, new.AngleDeg},
		{"safetyFactor", old.SafetyFactor, new.SafetyFactor},
		{"position", old.Position, new.Position},
	}
	return appendChangedValues(changes, "LOAD_POINT", identifier, values)
}

func appendCueChanges(changes []RevisionChange, identifier string, old, new MovementCue) []RevisionChange {
	values := []struct {
		field string
		old   any
		new   any
	}{
		{"label", old.Label, new.Label},
		{"startOffsetMs", old.StartOffsetMs, new.StartOffsetMs},
		{"durationMs", old.DurationMs, new.DurationMs},
		{"movingPoints", old.MovingPoints, new.MovingPoints},
		{"clearanceCm", old.ClearanceCm, new.ClearanceCm},
		{"operator", old.Operator, new.Operator},
	}
	return appendChangedValues(changes, "CUE", identifier, values)
}

func appendChangedValues(changes []RevisionChange, subject, identifier string, values []struct {
	field string
	old   any
	new   any
}) []RevisionChange {
	for _, value := range values {
		if reflect.DeepEqual(value.old, value.new) {
			continue
		}
		changes = append(changes, RevisionChange{Kind: "CHANGED", Subject: subject, Identifier: identifier, Field: value.field, OldValue: value.old, NewValue: value.new})
	}
	return changes
}

func (p RiggingPlan) buildRemediationClosure(previous, current PlanRevision) RemediationClosure {
	closure := RemediationClosure{OldRevisionID: previous.ID, BlockingFindings: make([]CheckFinding, 0), OldEvidenceRefs: make([]string, 0)}
	for _, check := range p.CheckRuns {
		if check.RevisionID != previous.ID {
			continue
		}
		for _, finding := range check.Findings {
			if finding.Severity == SeverityBlocker {
				closure.BlockingFindings = append(closure.BlockingFindings, finding)
			}
		}
	}
	for _, rehearsal := range p.Rehearsals {
		if rehearsal.RevisionID == previous.ID {
			closure.OldRehearsalOutcome = rehearsal.Outcome
			closure.OldObservations = rehearsal.Observations
			closure.OldEvidenceRefs = append([]string(nil), rehearsal.EvidenceRefs...)
		}
	}
	for _, check := range p.CheckRuns {
		if check.RevisionID == current.ID {
			closure.CurrentRechecked = true
			break
		}
	}
	for _, rehearsal := range p.Rehearsals {
		if rehearsal.RevisionID == current.ID && rehearsal.Outcome == RehearsalPassed {
			closure.CurrentRehearsalPassed = true
			break
		}
	}
	return closure
}
