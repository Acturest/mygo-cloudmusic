package app

import (
	"math"
	"testing"
	"time"

	"netease-music-mygo/internal/lyrics"
)

// A word's gap takes time from the words either side when it has too little
// of its own, as Folia spreads it, so that the ring keeps turning.
func TestRingAdjustTimeline(t *testing.T) {
	line := &lyrics.Line{At: 10 * time.Second, End: 12 * time.Second, Text: "ab cd"}
	line.Words = []lyrics.Word{
		{Text: "ab", Start: 10 * time.Second, End: 11 * time.Second},
		{Text: "cd", Start: 11 * time.Second, End: 12 * time.Second},
	}
	glyphs := ringTimeline(line)
	if len(glyphs) != 5 {
		t.Fatalf("%d glyphs, want 5", len(glyphs))
	}
	gap := glyphs[2]
	if got := gap.start.Seconds(); math.Abs(got-10.97) > 0.001 {
		t.Errorf("the space starts at %v, want 10.97", got)
	}
	if got := gap.end.Seconds(); math.Abs(got-11.03) > 0.001 {
		t.Errorf("the space ends at %v, want 11.03", got)
	}
	for i := 1; i < len(glyphs); i++ {
		if glyphs[i].start < glyphs[i-1].start {
			t.Fatalf("glyph %d starts before the one before it", i)
		}
	}
}

// A line whose characters carry their own times is followed character by
// character, and runs past the last while the line is still sung.
func TestRingFractionalIndex(t *testing.T) {
	glyphs := []ringGlyph{
		{start: 0, end: time.Second},
		{start: time.Second, end: 2 * time.Second},
		{start: 2 * time.Second, end: 3 * time.Second},
	}
	if got := ringFractionalIndex(glyphs, -time.Second, 0); got != 0 {
		t.Errorf("before the line: %v, want 0", got)
	}
	if got := ringFractionalIndex(glyphs, 1500*time.Millisecond, 0); math.Abs(float64(got)-1.5) > 0.001 {
		t.Errorf("halfway through: %v, want 1.5", got)
	}
	if got := ringFractionalIndex(glyphs, 2500*time.Millisecond, 3*time.Second); math.Abs(float64(got)-3) > 0.001 {
		t.Errorf("past the last, with a line after: %v, want 3", got)
	}
	if got := ringFractionalIndex(glyphs, 3*time.Second, 0); math.Abs(float64(got)-3) > 0.001 {
		t.Errorf("past the last, no line after: %v, want 3", got)
	}
	if got := ringFractionalIndex(glyphs, 4*time.Second, 0); math.Abs(float64(got)-3.8) > 0.001 {
		t.Errorf("well past the last: %v, want 3.8", got)
	}
}

// Characters are cut into graphemes as Folia cuts them: a family emoji is
// one character, not six.
func TestRingGraphemes(t *testing.T) {
	if got := ringGraphemes("👨‍👩‍👧‍👦a"); len(got) != 2 {
		t.Errorf("%d graphemes, want 2: %q", len(got), got)
	}
	if got := ringGraphemes("回环"); len(got) != 2 || got[0] != "回" {
		t.Errorf("%q, want 回 and 环", got)
	}
}

// A CJK character counts as two of the half-width characters Folia counts,
// so that long lines make way for it.
func TestRingVisualLength(t *testing.T) {
	if got := ringVisualLength("回环"); got != 2 {
		t.Errorf("回环 is %v long, want 2", got)
	}
	if got := ringVisualLength("ab"); got != 1 {
		t.Errorf("ab is %v long, want 1", got)
	}
}
