package lyrics

import (
	"strings"
	"time"
	"unicode"
)

// yrcSlack is how far a word-timed line may sit from the timed line with
// the same words before the two count as telling different times.
const yrcSlack = 1200 * time.Millisecond

// yrcReach is how far either way a word-timed line's twin is looked for.
const yrcReach = 60 * time.Second

// lineKey is what two copies of a line are compared by: its words, without
// the spaces the word-timed version puts between them.
func lineKey(text string) string {
	var b strings.Builder
	for _, r := range text {
		if !unicode.IsSpace(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// trustYRC reports whether NetEase's word-timed lines can be believed.
//
// They are made by machine, and some of them lose a break in the song: 場末
// - DADARAY's skip the 28 seconds between 2:31.98 and 3:02.95, so from 2:35
// on they are half a minute early and the lyrics walk through a stretch
// that is nothing but humming. The timed lines are made by hand, and the
// translation keeps their times, so where the two tell different times the
// word-timed ones are the ones to doubt.
//
// Lines are paired by their words, and the pair counts as disagreeing when
// their times are more than yrcSlack apart. Fewer than a handful of pairs
// is no evidence either way, and the word-timed lines stand.
func trustYRC(timed, plain []Line) bool {
	if len(timed) == 0 {
		return false
	}
	if len(plain) == 0 {
		return true
	}
	byText := make(map[string][]time.Duration, len(plain))
	for _, l := range plain {
		if key := lineKey(l.Text); key != "" {
			byText[key] = append(byText[key], l.At)
		}
	}
	pairs, off := 0, 0
	for _, l := range timed {
		times, ok := byText[lineKey(l.Text)]
		if !ok {
			continue
		}
		gap := yrcReach
		for _, at := range times {
			d := at - l.At
			if d < 0 {
				d = -d
			}
			if d < gap {
				gap = d
			}
		}
		if gap >= yrcReach {
			continue
		}
		pairs++
		if gap > yrcSlack {
			off++
		}
	}
	if pairs < 4 && len(timed) == len(plain) {
		// Nothing matched by its words, but the two lists run the song line
		// by line: compare them in order.
		pairs, off = 0, 0
		for i := range timed {
			pairs++
			if d := timed[i].At - plain[i].At; d > yrcSlack || d < -yrcSlack {
				off++
			}
		}
	}
	if pairs < 4 {
		return true
	}
	// A line or two off is the tail of a machine alignment; a sixth of them
	// is the alignment having lost its place.
	return off*6 <= pairs
}
