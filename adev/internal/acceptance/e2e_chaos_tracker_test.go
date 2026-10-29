//go:build tdd

package acceptance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ctxInfo struct {
	Keys    []string `json:"keys"`
	Bytes   int      `json:"bytes"`
	Dup     []string `json:"dup"`
	Missing []string `json:"missing"`
}

type ctrlQuery struct {
	Query    string `json:"query"`
	Complete bool   `json:"complete"`
	Excess   bool   `json:"excess"`
	Reason   string `json:"reason"`
}

type skCheck struct {
	OK      bool     `json:"ok"`
	Missing []string `json:"missing"`
	Extra   []string `json:"extra"`
}

type opAnsInfo struct {
	GapHint string `json:"gapHint"`
	Raw     string `json:"raw"`
	Text    string `json:"text"`
	LangMix bool   `json:"langMix"`
}

type modelCallInfo struct {
	Index   int     `json:"index"`
	Context ctxInfo `json:"context"`
	Summary string  `json:"summary"`
}

type tempoRpt struct {
	Stage           string         `json:"stage"`
	Path            string         `json:"path"`
	ModelCalls      []modelCallInfo `json:"modelCalls"`
	AttachedContext ctxInfo        `json:"attachedContext"`
	ControlQueries  []ctrlQuery    `json:"controlQueries"`
	OperatorAnswers []opAnsInfo    `json:"operatorAnswers"`
	SkeletonCheck   skCheck        `json:"skeletonCheck"`
	GapsResolved    int            `json:"gapsResolved"`
	TimeMs          int64          `json:"timeMs"`
}

func trackCtxFromUser(user string) ctxInfo {
	combined := user
	keys := []string{}
	if strings.Contains(combined, "\"skeleton\"") || strings.Contains(combined, "skeleton") {
		keys = append(keys, "skeleton")
	}
	if strings.Contains(combined, "registry") || strings.Contains(combined, "object") {
		keys = append(keys, "registry")
	}
	if strings.Contains(combined, "previous") || strings.Contains(combined, "form") {
		keys = append(keys, "previousForms")
	}
	if strings.Contains(combined, "openGaps") {
		keys = append(keys, "openGaps")
	}
	if strings.Contains(combined, "instruction") {
		keys = append(keys, "instructions")
	}
	if strings.Contains(combined, "history") {
		keys = append(keys, "history")
	}
	if strings.Contains(combined, "BRD") || strings.Contains(combined, "SRS") || strings.Contains(combined, "HLD") {
		keys = append(keys, "upstreamForms")
	}
	seen := map[string]bool{}
	uniq := []string{}
	for _, k := range keys {
		if !seen[k] {
			seen[k] = true
			uniq = append(uniq, k)
		}
	}
	dup := []string{}
	if strings.Count(combined, "skeleton") > 2 {
		dup = append(dup, "skeleton")
	}
	if strings.Count(combined, "registry") > 2 {
		dup = append(dup, "registry")
	}
	missing := []string{}
	has := func(k string) bool {
		for _, x := range uniq {
			if x == k {
				return true
			}
		}
		return false
	}
	if !has("skeleton") {
		missing = append(missing, "skeleton")
	}
	if !has("instructions") {
		missing = append(missing, "instructions")
	}
	return ctxInfo{Keys: uniq, Bytes: len(combined), Dup: dup, Missing: missing}
}

func makeCtrlQueries(stage string, c ctxInfo) []ctrlQuery {
	has := func(k string) bool {
		for _, x := range c.Keys {
			if x == k {
				return true
			}
		}
		return false
	}
	complete1 := len(c.Keys) >= 4 && !func() bool {
		for _, m := range c.Missing {
			if m == "skeleton" {
				return true
			}
		}
		return false
	}()
	reason1 := "контекст по ключам достаточен для заполнения скелетона"
	if func() bool {
		for _, m := range c.Missing {
			if m == "skeleton" {
				return true
			}
		}
		return false
	}() {
		reason1 = "не хватает skeleton - модель будет додумывать"
	} else if len(c.Keys) < 4 {
		reason1 = "мало прикреплённого контекста"
	} else if len(c.Dup) > 0 {
		reason1 = "есть дублирование, но основа достаточна"
	}
	reason2 := "openGaps не выявлены или не приобщены"
	if len(c.Dup) > 0 {
		reason2 = "обнаружено дублирование ключевых блоков - избыточность"
	} else if has("openGaps") {
		reason2 = "openGaps приобщены - полнота приемлема"
	}
	return []ctrlQuery{
		{Query: "Контрольный запрос 1: достаточно ли прикреплённого скелетона и предыдущих форм для заполнения без догадок?", Complete: complete1, Excess: len(c.Keys) > 6 || len(c.Dup) > 0, Reason: reason1},
		{Query: "Контрольный запрос 2: есть ли избыточный/дублирующий контекст при наличии openGaps?", Complete: has("openGaps") || stage == "00", Excess: len(c.Dup) > 0, Reason: reason2},
	}
}

func writeTempo(r tempoRpt) {
	dir := filepath.Join("..", "logs")
	_ = os.MkdirAll(dir, 0755)
	name := filepath.Join(dir, "e2e-tempo-"+r.Stage+"-"+r.Path+"-"+time.Now().Format("20060102-150405")+".json")
	b, _ := json.MarshalIndent(r, "", "  ")
	_ = os.WriteFile(name, b, 0644)
	println("=== TEMPO-REPORT ===")
	println("stage:", r.Stage, "path:", r.Path, "ctx.keys:", strings.Join(r.AttachedContext.Keys, ","))
	println("ctx.bytes:", r.AttachedContext.Bytes, "missing:", strings.Join(r.AttachedContext.Missing, ","), "dup:", strings.Join(r.AttachedContext.Dup, ","))
	for i, cq := range r.ControlQueries {
		println("  cq["+itoa2(i)+"] complete:", cq.Complete, "excess:", cq.Excess, "reason:", cq.Reason)
	}
	println("skeleton.ok:", r.SkeletonCheck.OK, "missing:", strings.Join(r.SkeletonCheck.Missing, ","))
	println("opAnswers:", len(r.OperatorAnswers), "gapsResolved:", r.GapsResolved)
	println("=== END TEMPO-REPORT ===")
}

func itoa2(i int) string {
	if i == 0 {
		return "0"
	}
	if i == 1 {
		return "1"
	}
	return "2"
}
