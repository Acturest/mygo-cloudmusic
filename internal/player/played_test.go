package player

import (
	"testing"
	"time"

	"netease-music-mygo/internal/netease"
)

// heardBy makes p tell the test of what it reports, in order.
func heardBy(p *Player) *[]time.Duration {
	heard := &[]time.Duration{}
	p.Played = func(t netease.Track, played time.Duration) { *heard = append(*heard, played) }
	return heard
}

// A track reported as it changes counts only once half of it played, and
// then only once.
func TestReportPlayedNeedsHalf(t *testing.T) {
	p := New(func(func()) {})
	heard := heardBy(p)
	p.Cue([]netease.Track{{ID: 7, Name: "宝船", Duration: 200000}})

	p.heard = 99 * time.Second
	p.reportPlayed(false)
	if len(*heard) != 0 {
		t.Fatalf("reported after 99s of 200: %v", *heard)
	}
	p.heard = 100 * time.Second
	p.reportPlayed(false)
	if len(*heard) != 1 || (*heard)[0] != 100*time.Second {
		t.Fatalf("reported %v, want one of 100s", *heard)
	}
	p.reportPlayed(false)
	if len(*heard) != 1 {
		t.Fatalf("reported twice: %v", *heard)
	}
}

// A track that reached its end is reported whole, however little of it the
// position caught, as a seek to the end of a short track would have it.
func TestReportPlayedFinished(t *testing.T) {
	p := New(func(func()) {})
	heard := heardBy(p)
	p.Cue([]netease.Track{{ID: 7, Name: "宝船", Duration: 200000}})

	p.heard = 20 * time.Second
	p.reportPlayed(true)
	if len(*heard) != 1 || (*heard)[0] != 200*time.Second {
		t.Fatalf("reported %v, want one of 200s", *heard)
	}
}

// Changing a track reports the one that goes, through the stop that clears
// it, and only if it played enough.
func TestReportPlayedOnStop(t *testing.T) {
	p := New(func(func()) {})
	heard := heardBy(p)
	p.Cue([]netease.Track{{ID: 7, Name: "宝船", Duration: 200000}})

	p.heard = 10 * time.Second
	p.stop()
	if len(*heard) != 0 {
		t.Fatalf("reported after 10s of 200: %v", *heard)
	}
	p.heard = 150 * time.Second
	p.stop()
	if len(*heard) != 1 || (*heard)[0] != 150*time.Second {
		t.Fatalf("reported %v, want one of 150s", *heard)
	}
}
