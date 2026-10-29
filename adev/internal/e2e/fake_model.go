//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"github.com/Reddetk/Adev/internal/usecase"
)

type scriptedLLM struct {
	mu      sync.Mutex
	path    Path
	callIdx map[string]int
	track   func(string, usecase.LLMRequest) ModelCallInfo
}

func newScriptedLLM(p Path, tracker func(string, usecase.LLMRequest) ModelCallInfo) *scriptedLLM {
	return &scriptedLLM{
		path:    p,
		callIdx: map[string]int{},
		track:   tracker,
	}
}

func (s *scriptedLLM) Complete(ctx context.Context, req usecase.LLMRequest) (usecase.LLMResponse, error) {
	s.mu.Lock()
	stage := extractStage(req.User)
	s.callIdx[stage]++
	idx := s.callIdx[stage]
	if s.track != nil {
		_ = s.track(stage, req)
	}
	s.mu.Unlock()

	switch s.path {
	case PathHappy:
		return happyResponse(stage)
	case PathNeg1:
		return neg1Response(stage, idx)
	case PathNeg2:
		return neg2Response(stage, idx)
	case PathEmpty:
		return emptyResponse()
	}
	return happyResponse(stage)
}

func extractStage(s string) string {
	for _, st := range []string{"BRD", "SRS", "HLD", "LLD", "TASK_PLAN", "FR", "ADR"} {
		if strings.Contains(s, st) {
			return st
		}
	}
	return "UNKNOWN"
}

func happyResponse(stage string) (usecase.LLMResponse, error) {
	var payload json.RawMessage
	switch stage {
	case "BRD":
		payload = json.RawMessage(`{"goals":[{"id":"GOAL-1","outcome":{"value":"Снизить время оформления","lang":"ste100"},"metricId":"M-1"}],"metrics":[{"id":"M-1","name":"time_min","unit":"min","baseline":60,"target":20}],"scope":{"in":[{"value":"Оформление","lang":"ste100"}]},"successMetrics":[{"metricId":"M-1","threshold":"20","window":"30d"}]}`)
	case "SRS":
		payload = json.RawMessage(`{"fr":[{"id":"FR-1","statement":{"id":"FR-1","text":"Система должна оформлять заявку","modality":"shall","priority":"must"},"actor":"operator","trigger":"request","response":"created","stateChange":"created","priority":"must","acceptance":[{"given":"operator","when":"request","then":"created"}]}],"nfr":[{"id":"NFR-1","statement":{"id":"NFR-1","text":"Ответ < 200мс","modality":"shall"},"metric":"latency","unit":"ms","threshold":"200","measurementPoint":"api","window":"p95"}]}`)
	case "HLD":
		payload = json.RawMessage(`{"boundedContexts":[{"id":"bc-1","name":"Orders","purpose":{"value":"Обработка заказов","lang":"ste100"},"containerIds":["obj-container-1"]}],"containers":[{"objectId":"obj-container-1","purpose":{"value":"API сервис","lang":"ste100"},"boundedContextId":"bc-1","tech":"go"}],"aggregates":[{"id":"agg-1","containerId":"obj-container-1","root":"Order","entityRefs":["Order"],"eventMaps":[],"moduleRef":"obj-module-1"}],"contextRelations":[],"degradationStrategy":[]}`)
	case "LLD":
		payload = json.RawMessage(`{"containers":[{"containerId":"obj-container-1","layers":[{"name":"usecase","packages":[{"path":"internal/usecase","modules":[{"objectId":"obj-module-1","path":"internal/usecase/order"}]}]}]}],"contracts":[{"id":"contract-1","moduleRef":{"formId":"FRM-1-05","formType":"LLD","formVersion":1},"kind":"http","inputType":"Order","outputType":"Order","errors":[{"code":"E-1","condition":"not found"}]}],"paths":[{"id":"path-1","contractRef":"contract-1","kind":"happy","given":"exists","when":"get","then":"ok"},{"id":"path-2","contractRef":"contract-1","kind":"negative","expectedErrorCode":"E-1","given":"no","when":"get","then":"fail"}]}`)
	default:
		payload = json.RawMessage(`{"tasks":[{"id":"T-1","title":{"value":"Реализовать","lang":"ste100"},"taskType":"feature","owner":"dev","acceptance":[{"id":"AC-1","text":"Готово","modality":"shall","priority":"must"}],"acceptanceRefs":["FR-1"]}],"roots":["T-1"]}`)
	}
	return usecase.LLMResponse{Payload: payload, Usage: usecase.LLMUsage{LatencyMs: 15}}, nil
}

func neg1Response(stage string, idx int) (usecase.LLMResponse, error) {
	if idx == 1 {
		// first call - incomplete skeleton
		var payload json.RawMessage
		switch stage {
		case "BRD":
			payload = json.RawMessage(`{"goals":[{"id":"GOAL-1","outcome":{"value":"Снизить","lang":"ste100"},"metricId":"M-1"}],"metrics":[],"scope":{"in":[]},"successMetrics":[]}`)
		case "SRS":
			payload = json.RawMessage(`{"fr":[],"nfr":[]}`)
		case "HLD":
			payload = json.RawMessage(`{"boundedContexts":[],"containers":[],"aggregates":[],"contextRelations":[],"degradationStrategy":[]}`)
		case "LLD":
			payload = json.RawMessage(`{"containers":[],"contracts":[],"paths":[]}`)
		default:
			payload = json.RawMessage(`{"tasks":[],"roots":[]}`)
		}
		return usecase.LLMResponse{Payload: payload, Usage: usecase.LLMUsage{LatencyMs: 10}}, nil
	}
	// second call after operator answers - still partial
	return happyResponse(stage)
}

func neg2Response(stage string, idx int) (usecase.LLMResponse, error) {
	if idx == 1 {
		// excessive/garbage in places but maybe valid structure? try to be weird
		var payload json.RawMessage
		switch stage {
		case "BRD":
			payload = json.RawMessage(`{"goals":[{"id":"GOAL-1","outcome":{"value":"Снизить время оформления заявок очень много лишнего текста для теста избыточности","lang":"ste100"},"metricId":"M-1"},{"id":"GOAL-2","outcome":{"value":"Ещё одна цель не из скоупа но пусть будет","lang":"ste100"},"metricId":"M-2"}],"metrics":[{"id":"M-1","name":"time_min","unit":"min","baseline":60,"target":20},{"id":"M-2","name":"foo","unit":"x","baseline":10,"target":5}],"scope":{"in":[{"value":"Оформление","lang":"ste100"},{"value":"Лишнее","lang":"ste100"}]},"successMetrics":[{"metricId":"M-1","threshold":"20","window":"30d"}]}`)
		default:
			return happyResponse(stage)
		}
		return usecase.LLMResponse{Payload: payload, Usage: usecase.LLMUsage{LatencyMs: 12}}, nil
	}
	return happyResponse(stage)
}

func emptyResponse() (usecase.LLMResponse, error) {
	return usecase.LLMResponse{Payload: json.RawMessage(`{}`), Usage: usecase.LLMUsage{LatencyMs: 5}}, nil
}
