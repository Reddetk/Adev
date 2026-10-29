//go:build e2e

package e2e

import (
	"encoding/json"
	"strings"

	"github.com/Reddetk/Adev/internal/usecase"
)

type ContextInfo struct {
	Keys     []string `json:"keys"`
	Bytes    int      `json:"bytes"`
	Dup      []string `json:"dup"`
	Missing  []string `json:"missing"`
}

type ControlQuery struct {
	Query    string `json:"query"`
	Complete bool   `json:"complete"`
	Excess   bool   `json:"excess"`
	Reason   string `json:"reason"`
}

type TempoReport struct {
	Stage          string           `json:"stage"`
	Path           string           `json:"path"`
	ModelCalls     []ModelCallInfo  `json:"modelCalls"`
	AttachedContext ContextInfo     `json:"attachedContext"`
	ControlQueries []ControlQuery   `json:"controlQueries"`
	OperatorAnswers []opAnswerInfo  `json:"operatorAnswers"`
	SkeletonCheck  SkeletonCheck   `json:"skeletonCheck"`
	GapsResolved   int             `json:"gapsResolved"`
	TimeMs         int64           `json:"timeMs"`
}

type ModelCallInfo struct {
	Index    int      `json:"index"`
	Context  ContextInfo `json:"context"`
	Summary  string   `json:"summary"`
}

type opAnswerInfo struct {
	GapID   string `json:"gapId"`
	Raw     string `json:"raw"`
	Text    string `json:"text"`
	LangMix bool   `json:"langMix"`
}

func trackContext(req usecase.LLMRequest) ContextInfo {
	user := req.User
	sys := req.System
	combined := sys + "\n" + user
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
	if strings.Contains(combined, "BRD") || strings.Contains(combined, "SRS") {
		keys = append(keys, "upstreamForms")
	}
	// deduplicate
	seen := map[string]bool{}
	uniq := []string{}
	for _, k := range keys {
		if !seen[k] {
			seen[k] = true
			uniq = append(uniq, k)
		}
	}
	// rough dup detection
	dup := []string{}
	if strings.Count(combined, "skeleton") > 2 {
		dup = append(dup, "skeleton")
	}
	if strings.Count(combined, "registry") > 2 {
		dup = append(dup, "registry")
	}
	missing := []string{}
	if !containsKey(uniq, "skeleton") {
		missing = append(missing, "skeleton")
	}
	if !containsKey(uniq, "instructions") {
		missing = append(missing, "instructions")
	}
	return ContextInfo{
		Keys:    uniq,
		Bytes:   len(combined),
		Dup:     dup,
		Missing: missing,
	}
}

func containsKey(keys []string, k string) bool {
	for _, x := range keys {
		if x == k {
			return true
		}
	}
	return false
}

func makeControlQueries(stage string, ctx ContextInfo) []ControlQuery {
	qs := []ControlQuery{
		{
			Query:    "Контрольный запрос 1: достаточно ли прикреплённого скелетона и предыдущих форм для заполнения без догадок?",
			Complete: len(ctx.Keys) >= 4 && !containsKey(ctx.Missing, "skeleton"),
			Excess:   len(ctx.Keys) > 6 || len(ctx.Dup) > 0,
			Reason:   reasonFor(ctx),
		},
		{
			Query:    "Контрольный запрос 2: есть ли избыточный/дублирующий контекст при наличии openGaps?",
			Complete: containsKey(ctx.Keys, "openGaps") || stage == "BRD",
			Excess:   len(ctx.Dup) > 0,
			Reason:   reasonFor2(ctx),
		},
	}
	return qs
}

func reasonFor(ctx ContextInfo) string {
	if containsKey(ctx.Missing, "skeleton") {
		return "не хватает skeleton - модель будет додумывать"
	}
	if len(ctx.Keys) < 4 {
		return "мало прикреплённого контекста"
	}
	if len(ctx.Dup) > 0 {
		return "есть дублирование, но основа достаточна"
	}
	return "контекст по ключам достаточен для заполнения скелетона"
}

func reasonFor2(ctx ContextInfo) string {
	if len(ctx.Dup) > 0 {
		return "обнаружено дублирование ключевых блоков - избыточность"
	}
	if containsKey(ctx.Keys, "openGaps") {
		return "openGaps приобщены - полнота приемлема"
	}
	return "openGaps не выявлены или не приобщены"
}

func asJSON(v any) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
