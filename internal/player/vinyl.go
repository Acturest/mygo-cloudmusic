package player

import (
	"math"
	"math/rand/v2"
)

// VinylMix is the record surface noise's mix, layer by layer, as the
// 配比 panel keeps it: each level is a ratio of the factory balance, and
// the crackle's probability scales how often its grains fall.
type VinylMix struct {
	Hiss        float64
	Rumble      float64
	Crackle     float64
	Hum         float64
	CrackleProb float64
}

// Active reports whether the mix makes any round.
func (m VinylMix) Active() bool {
	return m.Hiss > 0 || m.Rumble > 0 || m.Crackle > 0 || m.Hum > 0
}

// vinylNoise synthesises a record's surface under the music, as the
// vinyl-groove project does: hiss from the dust, the rumble of the
// turntable at 33 1/3 turns, crackle from scratches, and the motor's hum.
type vinylNoise struct {
	rate     int
	hpL, hpR biquad
	lpL, lpR biquad
	airL     biquad
	airR     biquad
	rumbleL  biquad
	rumbleR  biquad
	dustL    biquad
	dustR    biquad
	brownL   float64
	brownR   float64
	lfo      float64
	crackL   float64
	crackR   float64
	hum      float64
}

func newVinylNoise(rate int) *vinylNoise {
	n := &vinylNoise{rate: rate}
	n.hpL.highpass(float64(rate), 620, 0.7)
	n.hpR.highpass(float64(rate), 620, 0.7)
	n.lpL.lowpass(float64(rate), 7200, 0.7)
	n.lpR.lowpass(float64(rate), 7200, 0.7)
	n.airL.bandpass(float64(rate), 3200, 0.7)
	n.airR.bandpass(float64(rate), 3200, 0.7)
	n.rumbleL.lowpass(float64(rate), 78, 0.6)
	n.rumbleR.lowpass(float64(rate), 78, 0.6)
	n.dustL.highpass(float64(rate), 820, 0.7)
	n.dustR.highpass(float64(rate), 820, 0.7)
	return n
}

func (n *vinylNoise) reset() {
	n.hpL.reset()
	n.hpR.reset()
	n.lpL.reset()
	n.lpR.reset()
	n.airL.reset()
	n.airR.reset()
	n.rumbleL.reset()
	n.rumbleR.reset()
	n.dustL.reset()
	n.dustR.reset()
	n.brownL, n.brownR, n.crackL, n.crackR = 0, 0, 0, 0
}

// sample returns one frame of the surface noise at mix.
func (n *vinylNoise) sample(mix VinylMix) (float64, float64) {
	// Hiss: white noise through a band of the surface's own.
	wl := rand.Float64()*2 - 1
	wr := rand.Float64()*2 - 1
	hissL := n.lpL.process(n.hpL.process(wl)) * 0.095 * mix.Hiss
	hissR := n.lpR.process(n.hpR.process(wr)) * 0.095 * mix.Hiss
	// The air over the surface rides with the hiss.
	airL := n.airL.process(rand.Float64()*2-1) * 0.016 * mix.Hiss
	airR := n.airR.process(rand.Float64()*2-1) * 0.016 * mix.Hiss

	// Rumble: brown noise under 78 Hz, swelling once a turn, 1 ± 0.48.
	n.brownL = (n.brownL + 0.02*wl) / 1.02
	n.brownR = (n.brownR + 0.02*wr) / 1.02
	// One swell a turn, the turn 19.8 s long.
	n.lfo += 2 * math.Pi / 19.8 / float64(n.rate)
	swell := 1 + 0.48*math.Sin(n.lfo)
	rumbleL := n.rumbleL.process(n.brownL*3.4) * 0.62 * swell * mix.Rumble
	rumbleR := n.rumbleR.process(n.brownR*3.4) * 0.62 * swell * mix.Rumble

	// Crackle: sparse grains and the odd scratch, each decaying at once.
	probability := 30 / float64(n.rate) * clamp01f(mix.CrackleProb)
	if rand.Float64() < probability {
		amp := math.Pow(rand.Float64(), 2.8)*0.5 + 0.004
		if rand.Float64() < 0.5 {
			amp = -amp
		}
		n.crackL += amp
	}
	if rand.Float64() < probability {
		amp := math.Pow(rand.Float64(), 2.8)*0.5 + 0.004
		if rand.Float64() < 0.5 {
			amp = -amp
		}
		n.crackR += amp
	}
	crackL := n.dustL.process(n.crackL) * 0.46 * mix.Crackle
	crackR := n.dustR.process(n.crackR) * 0.46 * mix.Crackle
	n.crackL *= 0.55
	n.crackR *= 0.55

	// Hum: the motor's two harmonics, well under everything else.
	n.hum += 2 * math.Pi * 50 / float64(n.rate)
	hum := (math.Sin(n.hum)*0.0055 + math.Sin(n.hum*2)*0.0022) * mix.Hum

	l := hissL + airL + rumbleL + crackL + hum
	r := hissR + airR + rumbleR + crackR + hum
	// Make room: the surface sits under the music, not beside it.
	return l * 0.55, r * 0.55
}

func clamp01f(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
