//go:build e2e

package e2e

import (
	"math/rand"
	"strings"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var chaosRU = []string{
	"нужно  указать   вот это...   ",
	"короче,     не совсем ясно, но   ",
	"вот   кратко:   ",
	"как бы   ",
	"мне кажется   надо   добавить   ",
	"извините за сумбур   ",
	"по сути   ",
	"в общем   ",
}

var chaosEN = []string{
	"well,   ",
	"basically   ",
	"kinda   ",
	"sort of   ",
	"just   saying   ",
	"hm...   ",
	"actually   ",
	"roughly   ",
}

var puncts = []string{"...", "?!", "?!?", ".", ",", ";", ":", "!"}

func chaosWrap(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	prefix := chaosRU[rand.Intn(len(chaosRU))]
	if rand.Intn(2) == 0 {
		prefix = chaosEN[rand.Intn(len(chaosEN))]
	}
	s = prefix + s
	// add some spaces
	s = strings.ReplaceAll(s, " ", "   ")
	if rand.Intn(3) == 0 {
		s += " " + puncts[rand.Intn(len(puncts))]
	}
	if rand.Intn(4) == 0 {
		// add line breaks
		parts := strings.Split(s, "   ")
		if len(parts) > 2 {
			s = strings.Join(parts[:len(parts)/2], "\n\n") + "\n" + strings.Join(parts[len(parts)/2:], "   ")
		}
	}
	if rand.Intn(3) == 0 {
		// mix case
		if rand.Intn(2) == 0 {
			s = strings.ToLower(s)
		} else {
			s = strings.ToUpper(s[:min(10, len(s))]) + s[min(10, len(s)):]
		}
	}
	return strings.TrimSpace(s)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
