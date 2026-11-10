package lyrics

import (
	"regexp"
	"strings"
)

var chorusTime = regexp.MustCompile(`\[(\d{2,3}):(\d{2})[.:](\d{2,3})\]`)

// markChorus marks the lines of the chorus, as Folia finds it: the text that
// comes round most often in the lyrics, counted with the times taken off.
func markChorus(lines []Line, lrc string) {
	counts := make(map[string]int)
	note := func(text string) {
		t := strings.TrimSpace(chorusTime.ReplaceAllString(text, ""))
		if t == "" || t == "......" || len([]rune(t)) < 2 {
			return
		}
		counts[t]++
	}
	if strings.TrimSpace(lrc) != "" {
		for _, raw := range strings.Split(lrc, "\n") {
			note(raw)
		}
	} else {
		for _, l := range lines {
			note(l.Text)
		}
	}
	maxCount := 0
	for _, n := range counts {
		if n > maxCount {
			maxCount = n
		}
	}
	if maxCount <= 1 {
		return
	}
	for i := range lines {
		if counts[strings.TrimSpace(lines[i].Text)] == maxCount {
			lines[i].Chorus = true
		}
	}
}
