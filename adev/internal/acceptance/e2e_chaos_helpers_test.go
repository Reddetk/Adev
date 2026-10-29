//go:build tdd

package acceptance

import (
	"encoding/json"
	"math/rand"
	"strings"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type e2ePath string

const (
	pathHappy e2ePath = "happyPath"
	pathNeg1  e2ePath = "negPath-1"
	pathNeg2  e2ePath = "negPath-2"
	pathEmpty e2ePath = "empty"
)

var chaosRUPrefix = []string{
	"нужно  указать   вот это...   ",
	"короче,     не совсем ясно, но   ",
	"вот   кратко:   ",
	"как бы   ",
	"мне кажется   надо   добавить   ",
	"извините за сумбур   ",
	"по сути   ",
	"в общем   ",
}

var chaosENPrefix = []string{
	"well,   ",
	"basically   ",
	"kinda   ",
	"sort of   ",
	"just   saying   ",
	"hm...   ",
	"actually   ",
	"roughly   ",
}

var chaosPunct = []string{"...", "?!", "?!?", ".", ",", ";", ":", "!"}

func chaosWrapText(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	prefix := chaosRUPrefix[rand.Intn(len(chaosRUPrefix))]
	if rand.Intn(2) == 0 {
		prefix = chaosENPrefix[rand.Intn(len(chaosENPrefix))]
	}
	s = prefix + s
	s = strings.ReplaceAll(s, " ", "   ")
	if rand.Intn(3) == 0 {
		s += " " + chaosPunct[rand.Intn(len(chaosPunct))]
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
		} else {
			if len(s) > 10 {
				s = strings.ToUpper(s[:10]) + s[10:]
			}
		}
	}
	return strings.TrimSpace(s)
}

type chaosOpAns struct {
	Raw     string
	Text    string
	LangMix bool
}

func genChaosOpAnswer(p e2ePath, gapHint string) chaosOpAns {
	switch p {
	case pathHappy:
		base := gapHint + " должно быть конкретно и покрывать требование в рамках системы"
		raw := chaosWrapText(base)
		return chaosOpAns{Raw: raw, Text: strings.ReplaceAll(raw, "   ", " "), LangMix: false}
	case pathNeg1:
		base := "ну в общем про " + gapHint + " только часть..."
		raw := chaosWrapText(base + "   пожалуй   недостаточно   но   как есть?!")
		raw = strings.ReplaceAll(raw, "   ", " ")
		if len(raw) > 18 {
			raw = raw[:rand.Intn(10)+8] + "..."
		}
		return chaosOpAns{Raw: raw, Text: raw, LangMix: true}
	case pathNeg2:
		base := "we need to consider " + gapHint + "; also maybe legacy stuff and some other details... not sure"
		raw := chaosWrapText(base)
		raw = strings.ReplaceAll(raw, "   ", " ")
		raw += "   and also   whatever   comes to mind!!!"
		return chaosOpAns{Raw: raw, Text: raw, LangMix: true}
	case pathEmpty:
		return chaosOpAns{Raw: "", Text: "", LangMix: false}
	}
	base := gapHint
	raw := chaosWrapText(base)
	return chaosOpAns{Raw: raw, Text: strings.ReplaceAll(raw, "   ", " "), LangMix: false}
}

func mustJSONRaw(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
