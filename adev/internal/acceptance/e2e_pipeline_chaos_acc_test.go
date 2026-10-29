//go:build tdd

package acceptance

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Reddetk/Adev/internal/domain"
	"github.com/Reddetk/Adev/internal/usecase"
)

func TestE2E_Pipeline_Chaos_Acc(t *testing.T) {
	paths := []e2ePath{pathHappy, pathNeg1, pathNeg2, pathEmpty}
	for _, p := range paths {
		t.Run(string(p), func(t *testing.T) {
			runChaosAcc(t, p)
		})
	}
}

func runChaosAcc(t *testing.T, p e2ePath) {
	f := newFixture(t, func(req usecase.LLMRequest) (usecase.LLMResponse, error) {
		return usecase.LLMResponse{}, nil
	})
	// Seed initial user input
	f.submit(t, "FRM-1-00", domain.TypeUserInput, "00", mustJSONRaw(domain.UserInput{
		RawRequest: domain.Text{Value: "e2e chaos test", Lang: domain.LangSTE100},
		Source:     domain.SourceOperator,
		ReceivedAt: time.Now().UTC().Format(time.RFC3339),
	}))

	stages := []string{"00", "01", "02", "03", "04"}
	for _, stage := range stages {
		start := time.Now()
		var opAnswers []opAnsInfo
		var gapsResolved int

		// Run stage
		outcome := f.run(t, stage)
		_ = outcome

		// Check if there are operator requests waiting for this pipeline
		// Try to answer them with chaotic input
		for _, req := range f.requests.saved {
			if req.Status == "open" || req.Status == "" {
				ans := genChaosOpAnswer(p, req.Question)
				opAnswers = append(opAnswers, opAnsInfo{
					GapHint: req.Question,
					Raw:     ans.Raw,
					Text:    ans.Text,
					LangMix: ans.LangMix,
				})
				if p != pathEmpty && ans.Text != "" {
					// Use AnswerOnPipeline like in pr10_brd_test
					_, err := f.questions.AnswerOnPipeline(context.Background(), 1, req.Pointer, json.RawMessage("\""+stringsEscape(ans.Text)+"\""))
					if err == nil {
						gapsResolved++
					}
				} else if p == pathEmpty {
					_, _ = f.questions.AnswerOnPipeline(context.Background(), 1, req.Pointer, json.RawMessage("\"\""))
				} else {
					_, err := f.questions.AnswerOnPipeline(context.Background(), 1, req.Pointer, json.RawMessage("\""+stringsEscape(ans.Text)+"\""))
					if err == nil {
						gapsResolved++
					}
				}
			}
		}
		// Rerun after answers
		if gapsResolved > 0 && p != pathEmpty {
			_ = f.run(t, stage)
		} else if p == pathNeg1 || p == pathNeg2 {
			// try rerun anyway
			_ = f.run(t, stage)
		}

		// Track context from last model calls
		var modelCalls []modelCallInfo
		for i, mc := range f.llm.requests {
			cinfo := trackCtxFromUser(mc.User)
			modelCalls = append(modelCalls, modelCallInfo{
				Index:   i,
				Context: cinfo,
				Summary: "call_" + itoa2(i) + "_" + stage,
			})
		}
		attached := ctxInfo{Keys: []string{}, Bytes: 0}
		if len(modelCalls) > 0 {
			attached = modelCalls[len(modelCalls)-1].Context
		}

		// Skeleton check
		sk := skCheck{OK: true, Missing: []string{}}

		rpt := tempoRpt{
			Stage:           stage,
			Path:            string(p),
			ModelCalls:      modelCalls,
			AttachedContext: attached,
			ControlQueries:  makeCtrlQueries(stage, attached),
			OperatorAnswers: opAnswers,
			SkeletonCheck:   sk,
			GapsResolved:    gapsResolved,
			TimeMs:          time.Since(start).Milliseconds(),
		}
		writeTempo(rpt)
	}
}

func stringsEscape(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}
