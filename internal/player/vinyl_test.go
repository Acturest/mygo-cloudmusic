package player

import (
	"math"
	"sync/atomic"
	"testing"
)

// The record's surface noise is mixed under the music at the ratio the
// 配比 panel gives, and silence stays silence with every layer off.
func TestVinylNoiseMix(t *testing.T) {
	var params atomic.Pointer[Effects]
	neutral := NeutralEffects()
	params.Store(&neutral)
	var mix atomic.Pointer[VinylMix]
	m := VinylMix{Hiss: 1, Rumble: 1, Crackle: 1, Hum: 1, CrackleProb: 1}
	mix.Store(&m)
	proc := newEffectProcessor(48000, 2, &params, &mix)

	buf := make([]float32, 48000*2)
	proc.process(buf)
	level := rms(buf)
	if level < 0.0005 {
		t.Errorf("surface noise at factory mix has rms %.5f", level)
	}
	for i, v := range buf {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			t.Fatalf("sample %d is %v", i, v)
		}
	}

	mix.Store(&VinylMix{})
	silent := make([]float32, 4096)
	proc.process(silent)
	for i, v := range silent {
		if v != 0 {
			t.Fatalf("an empty mix changed sample %d to %v", i, v)
		}
	}
}
