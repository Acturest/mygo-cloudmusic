package player

import (
	"math"
	"sync/atomic"
	"testing"
)

// effectsProcessor makes a processor set to e, and the holder to change it.
func effectsProcessor(e Effects) (*effectProcessor, *atomic.Pointer[Effects]) {
	var p atomic.Pointer[Effects]
	p.Store(&e)
	return newEffectProcessor(48000, 2, &p), &p
}

// sineFrames fills buf with a sine at freq, amplitude a.
func sineFrames(buf []float32, rate int, freq, a float64) {
	for i := 0; i+1 < len(buf); i += 2 {
		s := float32(a * math.Sin(2*math.Pi*freq*float64(i/2)/float64(rate)))
		buf[i], buf[i+1] = s, s
	}
}

func rms(buf []float32) float64 {
	var sum float64
	for _, v := range buf {
		sum += float64(v) * float64(v)
	}
	return math.Sqrt(sum / float64(len(buf)))
}

// The chain off leaves the sound as it was, sample for sample, and a chain
// on with neutral settings does too.
func TestEffectsBypass(t *testing.T) {
	in := make([]float32, 2048)
	sineFrames(in, 48000, 440, 0.3)
	for _, e := range []Effects{NeutralEffects(), func() Effects { n := NeutralEffects(); n.Enabled = true; return n }()} {
		buf := append([]float32(nil), in...)
		proc, _ := effectsProcessor(e)
		proc.process(buf)
		for i := range buf {
			if buf[i] != in[i] {
				t.Fatalf("enabled %v: sample %d changed from %v to %v", e.Enabled, i, in[i], buf[i])
			}
		}
	}
}

// A boosted band lifts a sine at its centre.
func TestEffectsEqualizerBoost(t *testing.T) {
	e := NeutralEffects()
	e.Enabled = true
	e.Gains[5] = 12 // 1 kHz
	proc, _ := effectsProcessor(e)
	buf := make([]float32, 48000*2)
	sineFrames(buf, 48000, 1000, 0.1)
	proc.process(buf)
	got := rms(buf[24000:])
	if got < 0.2 {
		t.Errorf("1 kHz at +12 dB comes out at %.3f rms, want nearer 0.4", got)
	}
}

// Width 0 folds both channels into the middle.
func TestEffectsWidthMono(t *testing.T) {
	e := NeutralEffects()
	e.Enabled = true
	e.Width = 0
	proc, _ := effectsProcessor(e)
	buf := make([]float32, 256)
	for i := 0; i+1 < len(buf); i += 2 {
		buf[i], buf[i+1] = 0.5, -0.25
	}
	proc.process(buf)
	for i := 0; i+1 < len(buf); i += 2 {
		if d := math.Abs(float64(buf[i] - buf[i+1])); d > 1e-6 {
			t.Fatalf("frame %d: %v and %v", i/2, buf[i], buf[i+1])
		}
	}
}

// Crush at its top quantizes to a low bit depth.
func TestEffectsCrushQuantizes(t *testing.T) {
	e := NeutralEffects()
	e.Enabled = true
	e.Crush = 1
	steps := math.Pow(2, 16-math.Sqrt(1)*12-1)
	proc, _ := effectsProcessor(e)
	buf := make([]float32, 512)
	sineFrames(buf, 48000, 220, 0.7)
	proc.process(buf)
	for i, v := range buf {
		q := math.Round(float64(v)*steps) / steps
		if math.Abs(float64(v)-q) > 1e-5 {
			t.Fatalf("sample %d = %v, not on the %v step grid", i, v, steps)
		}
	}
}

// Saturation adds harmonics, not loudness.
func TestEffectsDriveLoudness(t *testing.T) {
	e := NeutralEffects()
	e.Enabled = true
	e.Drive = 1
	proc, _ := effectsProcessor(e)
	buf := make([]float32, 48000*2)
	sineFrames(buf, 48000, 220, 0.5)
	before := rms(buf)
	proc.process(buf)
	after := rms(buf[len(buf)/2:])
	if after < before*0.8 || after > before*1.3 {
		t.Errorf("drive changes rms from %.3f to %.3f", before, after)
	}
}

// The reverb rings after the sound and stays finite.
func TestEffectsReverbTails(t *testing.T) {
	e := NeutralEffects()
	e.Enabled = true
	e.Space = 1
	proc, _ := effectsProcessor(e)
	buf := make([]float32, 48000*2)
	buf[0], buf[1] = 0.8, 0.8
	proc.process(buf)
	var tail float64
	for i := 4800; i < len(buf); i++ {
		v := float64(buf[i])
		if math.IsNaN(v) || math.IsInf(v, 0) {
			t.Fatalf("sample %d is %v", i, v)
		}
		tail += math.Abs(v)
	}
	if tail < 1e-6 {
		t.Error("no reverb tail")
	}
}

// The noise floor is audible even under silence.
func TestEffectsNoiseFloor(t *testing.T) {
	e := NeutralEffects()
	e.Enabled = true
	e.Noise = 1
	proc, _ := effectsProcessor(e)
	buf := make([]float32, 48000*2)
	proc.process(buf)
	if rms(buf) < 0.005 {
		t.Errorf("noise rms %.4f", rms(buf))
	}
}
