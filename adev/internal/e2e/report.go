//go:build e2e

package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

func writeTempoReport(r TempoReport) error {
	dir := "logs"
	_ = os.MkdirAll(dir, 0755)
	name := filepath.Join(dir, "e2e-tempo-"+r.Stage+"-"+r.Path+"-"+time.Now().Format("20060102-150405")+".json")
	b, _ := json.MarshalIndent(r, "", "  ")
	return os.WriteFile(name, b, 0644)
}

func logReport(r TempoReport) {
	println("=== TEMPO-REPORT ===")
	println("stage:", r.Stage, "path:", r.Path)
	println("attachedContext.keys:", join(r.AttachedContext.Keys))
	println("attachedContext.bytes:", r.AttachedContext.Bytes)
	println("attachedContext.missing:", join(r.AttachedContext.Missing))
	println("attachedContext.dup:", join(r.AttachedContext.Dup))
	for i, cq := range r.ControlQueries {
		println("  controlQuery["+itoa(i)+"] complete:", cq.Complete, "excess:", cq.Excess, "reason:", cq.Reason)
	}
	println("skeletonCheck.ok:", r.SkeletonCheck.OK, "missing:", join(r.SkeletonCheck.Missing))
	println("operatorAnswers:", len(r.OperatorAnswers))
	println("gapsResolved:", r.GapsResolved)
	println("=== END TEMPO-REPORT ===")
}

func join(s []string) string {
	if len(s) == 0 {
		return "[]"
	}
	res := ""
	for i, v := range s {
		if i > 0 {
			res += ","
		}
		res += v
	}
	return res
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	if i == 1 {
		return "1"
	}
	return string(rune('0' + i))
}
