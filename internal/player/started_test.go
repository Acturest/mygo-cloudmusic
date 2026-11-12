package player

import (
	"testing"

	"netease-music-mygo/internal/netease"
)

// A track tells Started it started once, however often playback is resumed.
func TestBeginPlayTellsStartedOnce(t *testing.T) {
	p := New(func(func()) {})
	var heard []int64
	p.Started = func(tr netease.Track) { heard = append(heard, tr.ID) }
	p.Cue([]netease.Track{{ID: 3, Name: "宝船", Duration: 200000}})

	p.beginPlay()
	p.beginPlay()
	if len(heard) != 1 || heard[0] != 3 {
		t.Fatalf("started %v, want once", heard)
	}

	// A track started again, as after playing it over, starts anew.
	p.began = false
	p.beginPlay()
	if len(heard) != 2 {
		t.Fatalf("started %v, want twice", heard)
	}
}

// Nothing started yet, so nothing to tell.
func TestBeginPlayNeedsATrack(t *testing.T) {
	p := New(func(func()) {})
	p.Started = func(netease.Track) { t.Fatal("heard a track that never started") }
	p.beginPlay()
}
