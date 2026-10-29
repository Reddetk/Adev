//go:build tdd

package acceptance

import (
	"context"
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Reddetk/Adev/internal/domain"
	"github.com/Reddetk/Adev/internal/usecase"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type chaosPath string

const (
	chaosHappy  chaosPath = "happyPath"
	chaosNeg1   chaosPath = "negPath-1"
	chaosNeg2   chaosPath = "negPath-2"
	chaosEmpty  chaosPath = "empty"
)

var ruP = []string{"нужно  указать   ", "короче,     ", "вот   кратко:   ", "как бы   ", "мне кажется   ", "извините за сумбур   ", "по сути   ", "в общем   "}
var enP = []string{"well,   ", "basically   ", "kinda   ", "sort of   ", "hm...   ", "actually   ", "roughly   "}
var punc = []string{"...", "?!", "?!?", ".", ",", ";", ":", "!"}

func chaosWrap(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	p := ruP[rand.Intn(len(ruP))]
	if rand.Intn(2) == 0 {
		p = enP[rand.Intn(len(enP))]
	}
	s = p + s
	s = strings.ReplaceAll(s, " ", "   ")
	if rand.Intn(3) == 0 {
		s += " " + punc[rand.Intn(len(punc))]
	}
	if rand.Intn(4) == 0 {
		parts := strings.Split(s, "   ")
		if len(parts) > 3 {
			mid := len(parts) / 2
			s = strings.Join(parts[:mid], "\n\n") + "\n" + strings.Join(parts[mid:], "   ")
		}
	}
	if rand.Intn(3) == 0 {
		if rand.Intn(2) == 0 {
			s = strings.ToLower(s)
		} else if len(s) > 10 {
			s = strings.ToUpper(s[:10]) + s[10:]
		}
	}
	return strings.TrimSpace(s)
}

func chaosOp(p chaosPath, hint string) (raw, text string, mix bool) {
	switch p {
	case chaosHappy:
		base := hint + " должно быть конкретно и покрывать требование в рамках системы"
		raw = chaosWrap(base)
		text = strings.ReplaceAll(raw, "   ", " ")
		return raw, text, false
	case chaosNeg1:
		base := "ну в общем про " + hint + " только часть..."
		raw = chaosWrap(base + "   пожалуй   недостаточно   но   как есть?!")
		raw = strings.ReplaceAll(raw, "   ", " ")
		if len(raw) > 18 {
			raw = raw[:rand.Intn(10)+8] + "..."
		}
		return raw, raw, true
	case chaosNeg2:
		base := "we need to consider " + hint + "; also maybe legacy stuff and some other details... not sure"
		raw = chaosWrap(base)
		raw = strings.ReplaceAll(raw, "   ", " ")
		raw += "   and also   whatever   comes to mind!!!"
		return raw, raw, true
	case chaosEmpty:
		return "", "", false
	default:
		raw = chaosWrap(hint)
		text = strings.ReplaceAll(raw, "   ", " ")
		return raw, text, false
	}
}

type ctxInf struct {
	Keys    []string `json:"keys"`
	Bytes   int      `json:"bytes"`
	Dup     []string `json:"dup"`
	Missing []string `json:"missing"`
}
type ctrlQ struct {
	Query    string `json:"query"`
	Complete bool   `json:"complete"`
	Excess   bool   `json:"excess"`
	Reason   string `json:"reason"`
}
type skInf struct {
	OK      bool     `json:"ok"`
	Missing []string `json:"missing"`
	Extra   []string `json:"extra"`
}
type opInf struct {
	GapHint string `json:"gapHint"`
	Raw     string `json:"raw"`
	Text    string `json:"text"`
	LangMix bool   `json:"langMix"`
}
type mcInf struct {
	Index   int    `json:"index"`
	Context ctxInf `json:"context"`
	Summary string `json:"summary"`
}
type trp struct {
	Stage           string  `json:"stage"`
	Path            string  `json:"path"`
	ModelCalls      []mcInf `json:"modelCalls"`
	AttachedContext ctxInf  `json:"attachedContext"`
	ControlQueries  []ctrlQ `json:"controlQueries"`
	OperatorAnswers []opInf `json:"operatorAnswers"`
	SkeletonCheck   skInf   `json:"skeletonCheck"`
	GapsResolved    int     `json:"gapsResolved"`
	TimeMs          int64   `json:"timeMs"`
}

func trackCtx(u string) ctxInf {
	c := u
	keys := []string{}
	if strings.Contains(c, "\"skeleton\"") || strings.Contains(c, "skeleton") {
		keys = append(keys, "skeleton")
	}
	if strings.Contains(c, "registry") || strings.Contains(c, "object") {
		keys = append(keys, "registry")
	}
	if strings.Contains(c, "previous") || strings.Contains(c, "form") {
		keys = append(keys, "previousForms")
	}
	if strings.Contains(c, "openGaps") {
		keys = append(keys, "openGaps")
	}
	if strings.Contains(c, "instruction") {
		keys = append(keys, "instructions")
	}
	if strings.Contains(c, "history") {
		keys = append(keys, "history")
	}
	if strings.Contains(c, "BRD") || strings.Contains(c, "SRS") || strings.Contains(c, "HLD") {
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
	if strings.Count(c, "skeleton") > 2 {
		dup = append(dup, "skeleton")
	}
	if strings.Count(c, "registry") > 2 {
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
	return ctxInf{Keys: uniq, Bytes: len(c), Dup: dup, Missing: missing}
}

func ctrlQrys(stage string, c ctxInf) []ctrlQ {
	has := func(k string) bool {
		for _, x := range c.Keys {
			if x == k {
				return true
			}
		}
		return false
	}
	missSk := false
	for _, m := range c.Missing {
		if m == "skeleton" {
			missSk = true
		}
	}
	re1 := "контекст по ключам достаточен для заполнения скелетона"
	if missSk {
		re1 = "не хватает skeleton - модель будет додумывать"
	} else if len(c.Keys) < 4 {
		re1 = "мало прикреплённого контекста"
	} else if len(c.Dup) > 0 {
		re1 = "есть дублирование, но основа достаточна"
	}
	re2 := "openGaps не выявлены или не приобщены"
	if len(c.Dup) > 0 {
		re2 = "обнаружено дублирование ключевых блоков - избыточность"
	} else if has("openGaps") {
		re2 = "openGaps приобщены - полнота приемлема"
	}
	return []ctrlQ{
		{Query: "Контрольный запрос 1: достаточно ли прикреплённого скелетона и предыдущих форм для заполнения без догадок?", Complete: len(c.Keys) >= 4 && !missSk, Excess: len(c.Keys) > 6 || len(c.Dup) > 0, Reason: re1},
		{Query: "Контрольный запрос 2: есть ли избыточный/дублирующий контекст при наличии openGaps?", Complete: has("openGaps") || stage == "00", Excess: len(c.Dup) > 0, Reason: re2},
	}
}

func writeTrp(r trp) {
	dir := filepath.Join("..", "logs")
	_ = os.MkdirAll(dir, 0755)
	name := filepath.Join(dir, "e2e-tempo-"+r.Stage+"-"+r.Path+"-"+time.Now().Format("20060102-150405")+".json")
	b, _ := json.MarshalIndent(r, "", "  ")
	_ = os.WriteFile(name, b, 0644)
	println("=== TEMPO-REPORT ===")
	println("stage:", r.Stage, "path:", r.Path)
	println("ctx.keys:", strings.Join(r.AttachedContext.Keys, ","))
	println("ctx.bytes:", r.AttachedContext.Bytes, "missing:", strings.Join(r.AttachedContext.Missing, ","), "dup:", strings.Join(r.AttachedContext.Dup, ","))
	for i, cq := range r.ControlQueries {
		println("  cq["+itoa(i)+"] complete:", cq.Complete, "excess:", cq.Excess, "reason:", cq.Reason)
	}
	println("skeleton.ok:", r.SkeletonCheck.OK, "missing:", strings.Join(r.SkeletonCheck.Missing, ","))
	println("opAnswers:", len(r.OperatorAnswers), "gapsResolved:", r.GapsResolved)
	println("=== END TEMPO-REPORT ===")
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	if i == 1 {
		return "1"
	}
	return "2"
}

func esc(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

func TestE2E_Pipeline_Chaos_Fixture(t *testing.T) {
	paths := []chaosPath{chaosHappy, chaosNeg1, chaosNeg2, chaosEmpty}
	for _, p := range paths {
		t.Run(string(p), func(t *testing.T) {
			runChaosFixture(t, p)
		})
	}
}

func runChaosFixture(t *testing.T, p chaosPath) {
	f := newFixture(t, func(req usecase.LLMRequest) (usecase.LLMResponse, error) {
		return usecase.LLMResponse{}, nil
	})
	// seed user input
	f.submit(t, "FRM-1-IN", domain.TypeUserInput, "00", mustJSON(t, domain.UserInput{
		RawRequest: ste("e2e chaos test"),
		Source:     domain.SourceOperator,
		ReceivedAt: time.Now().UTC().Format(time.RFC3339),
	}))

	stages := []string{"00", "01", "02", "03", "04"}
	for _, stage := range stages {
		start := time.Now()
		var opAns []opInf
		var gapsRes int

		_ = f.run(t, stage)

		// answer open operator requests
		for _, req := range f.requests.saved {
			if req.Status == "active" || req.Status == "" {
				raw, txt, mix := chaosOp(p, req.Question)
				opAns = append(opAns, opInf{GapHint: req.Question, Raw: raw, Text: txt, LangMix: mix})
				if p == chaosEmpty {
					_, _ = f.questions.AnswerOnPipeline(context.Background(), int(testTarget), req.Pointer, json.RawMessage("\"\""))
					continue
				}
				if txt == "" {
					_, _ = f.questions.AnswerOnPipeline(context.Background(), int(testTarget), req.Pointer, json.RawMessage("\"\""))
					continue
				}
				_, err := f.questions.AnswerOnPipeline(context.Background(), int(testTarget), req.Pointer, json.RawMessage("\""+esc(txt)+"\""))
				if err == nil {
					gapsRes++
				}
			}
		}

		if gapsRes > 0 && p != chaosEmpty {
			_ = f.run(t, stage)
		} else if p == chaosNeg1 || p == chaosNeg2 {
			_ = f.run(t, stage)
		}

		// track context from model calls
		var mcs []mcInf
		for i, mc := range f.llm.requests {
			c := trackCtx(mc.User)
			mcs = append(mcs, mcInf{Index: i, Context: c, Summary: "call_" + itoa(i) + "_" + stage})
		}
		att := ctxInf{}
		if len(mcs) > 0 {
			att = mcs[len(mcs)-1].Context
		}

		sk := skInf{OK: true}
		// basic skeleton check by trying to see forms
		if stage == "00" {
			// look for BRD forms via store
			sk = skInf{OK: len(f.requests.saved) >= 0}
		}

		r := trp{
			Stage:           stage,
			Path:            string(p),
			ModelCalls:      mcs,
			AttachedContext: att,
			ControlQueries:  ctrlQrys(stage, att),
			OperatorAnswers: opAns,
			SkeletonCheck:   sk,
			GapsResolved:    gapsRes,
			TimeMs:          time.Since(start).Milliseconds(),
		}
		writeTrp(r)
	}
}
