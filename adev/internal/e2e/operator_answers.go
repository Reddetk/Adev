//go:build e2e

package e2e

import (
	"fmt"
	"math/rand"
	"strings"
)

type operatorAnswer struct {
	GapID   string
	Text    string
	Raw     string
	LangMix bool
}

func genOperatorAnswer(path Path, gapID string, hint string) operatorAnswer {
	switch path {
	case PathHappy:
		base := fmt.Sprintf("%s должен быть понятен и покрывать требование %s", hint, gapID)
		raw := chaosWrap(base + " в рамках системы")
		return operatorAnswer{GapID: gapID, Text: strings.ReplaceAll(raw, "   ", " "), Raw: raw, LangMix: false}
	case PathNeg1:
		// incomplete + mixed punctuation + broken
		base := fmt.Sprintf("ну в общем про %s только часть...", hint)
		raw := chaosWrap(base + "  пожалуй   недостаточно   но   как есть?!")
		raw = strings.ReplaceAll(raw, "   ", " ")
		// break it
		if len(raw) > 20 {
			raw = raw[:rand.Intn(15)+10] + "..."
		}
		return operatorAnswer{GapID: gapID, Text: raw, Raw: raw, LangMix: true}
	case PathNeg2:
		// excessive / off-topic with EN mix
		base := fmt.Sprintf("we need to consider %s; also maybe legacy stuff and some other details... not sure", hint)
		raw := chaosWrap(base)
		raw = strings.ReplaceAll(raw, "   ", " ")
		raw += "   and also   whatever   comes to mind!!!"
		return operatorAnswer{GapID: gapID, Text: raw, Raw: raw, LangMix: true}
	case PathEmpty:
		return operatorAnswer{GapID: gapID, Text: "", Raw: "", LangMix: false}
	}
	return operatorAnswer{GapID: gapID, Text: hint, Raw: hint}
}
