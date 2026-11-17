package player

import (
	"math"
	"math/rand/v2"
	"sync/atomic"

	"github.com/egoist/mygo/audio"
)

// Effects is the audio effect chain's settings: a ten band equalizer and
// the effects after it, as the effects panel keeps them. Enabled off is a
// bypass, sample for sample.
type Effects struct {
	Enabled  bool
	Gains    [10]float64 // dB per band, -12 to 12
	Highpass float64     // Hz, 20 to 2000
	Lowpass  float64     // Hz, 500 to 20000
	Drive    float64     // 0 to 1: saturation
	Crush    float64     // 0 to 1: bit depth
	Wow      float64     // 0 to 1: tape wow and flutter
	Noise    float64     // 0 to 1: vinyl noise
	Width    float64     // 0 to 2: stereo width, 1 neutral
	Space    float64     // 0 to 1: reverb
	Punch    float64     // 0 to 1: compression
}

// NeutralEffects is the effect chain doing nothing: flat equalizer, both
// cutoffs at their ends, no drive or noise, stereo as it came.
func NeutralEffects() Effects {
	return Effects{Highpass: 20, Lowpass: 20000, Width: 1}
}

// effectBands are the equalizer's bands, and their shape's Q, as Folia's.
var effectBands = [10]float64{31, 62, 125, 250, 500, 1000, 2000, 4000, 8000, 16000}

const effectBandQ = 1.4

// effectDecoder applies the processor to what the decoder reads, so that
// the chain sits between the song and the player that plays it.
type effectDecoder struct {
	audio.Decoder
	proc *effectProcessor
}

func (d *effectDecoder) Read(buf []float32) (int, error) {
	n, err := d.Decoder.Read(buf)
	if n > 0 {
		d.proc.process(buf[:n])
	}
	return n, err
}

func (d *effectDecoder) SetPosition(frame int64) error {
	d.proc.reset()
	return d.Decoder.SetPosition(frame)
}

// effectProcessor holds the filters and delays of one song's sound. It is
// used by the goroutine decoding the song alone.
type effectProcessor struct {
	rate     int
	channels int
	params   *atomic.Pointer[Effects]
	last     *Effects

	eq  [10]biquad // [band], per channel below
	eqR [10]biquad
	hp  biquad
	hpR biquad
	lp  biquad
	lpR biquad

	driveShape, driveScale, driveComp float64
	crushSteps                        float64

	wow          [2][]float64
	wowAt        int
	wowPhase     float64
	flutterPhase float64

	compEnv float64
	attack  float64
	release float64

	noiseL, noiseR               biquad
	noiseCrackleL, noiseCrackleR float64

	rev *reverb
}

func newEffectProcessor(rate, channels int, params *atomic.Pointer[Effects]) *effectProcessor {
	return &effectProcessor{
		rate:     rate,
		channels: channels,
		params:   params,
		wow:      [2][]float64{make([]float64, rate/5+2), make([]float64, rate/5+2)},
	}
}

// reset forgets the filters' and delays' sound, as a seek makes.
func (p *effectProcessor) reset() {
	for i := range p.eq {
		p.eq[i].reset()
		p.eqR[i].reset()
	}
	p.hp.reset()
	p.hpR.reset()
	p.lp.reset()
	p.lpR.reset()
	clear(p.wow[0])
	clear(p.wow[1])
	p.wowAt = 0
	p.compEnv = 0
	p.noiseCrackleL, p.noiseCrackleR = 0, 0
	if p.rev != nil {
		p.rev.reset()
	}
}

// process applies the chain to interleaved stereo samples, in place.
func (p *effectProcessor) process(buf []float32) {
	e := p.params.Load()
	if e == nil || !e.Enabled {
		p.last = nil
		return
	}
	if p.last != e {
		p.configure(e)
		p.last = e
	}
	neutralFilters := e.Highpass <= 20.001 && e.Lowpass >= 19999
	for i := 0; i+1 < len(buf); i += 2 {
		l, r := float64(buf[i]), float64(buf[i+1])

		// Ten bands: the first a low shelf, the last a high shelf.
		for b := range p.eq {
			if e.Gains[b] == 0 {
				continue
			}
			l = p.eq[b].process(l)
			r = p.eqR[b].process(r)
		}
		if !neutralFilters {
			l = p.lp.process(p.hp.process(l))
			r = p.lpR.process(p.hpR.process(r))
		}
		if e.Drive > 0 {
			l = math.Tanh(p.driveShape*l) / p.driveScale * p.driveComp
			r = math.Tanh(p.driveShape*r) / p.driveScale * p.driveComp
		}
		if e.Crush > 0 {
			l = math.Round(l*p.crushSteps) / p.crushSteps
			r = math.Round(r*p.crushSteps) / p.crushSteps
		}
		if e.Wow > 0 {
			l, r = p.wowAt2(l, r, e.Wow)
		}
		if e.Width != 1 {
			direct := (1 + e.Width) / 2
			cross := (1 - e.Width) / 2
			l, r = l*direct+r*cross, r*direct+l*cross
		}
		if e.Punch > 0 {
			l, r = p.compress(l, r, e.Punch)
		}
		if e.Space > 0 {
			wl, wr := p.rev.process(l, r)
			l = l*(1-e.Space*0.2) + wl*e.Space*0.75
			r = r*(1-e.Space*0.2) + wr*e.Space*0.75
		}
		if e.Noise > 0 {
			gain := math.Pow(e.Noise, 1.4) * 0.55
			nl := p.noiseCrackleL*0.9 + (rand.Float64()*2-1)*0.12
			p.noiseCrackleL *= 0.88
			if rand.Float64() < 0.0007 {
				p.noiseCrackleL = rand.Float64()*2 - 1
			}
			nr := p.noiseCrackleR*0.9 + (rand.Float64()*2-1)*0.12
			p.noiseCrackleR *= 0.88
			if rand.Float64() < 0.0007 {
				p.noiseCrackleR = rand.Float64()*2 - 1
			}
			l += p.noiseL.process(nl) * gain
			r += p.noiseR.process(nr) * gain
		}
		buf[i], buf[i+1] = float32(l), float32(r)
	}
}

// configure sets the filters and delays for a change of settings.
func (p *effectProcessor) configure(e *Effects) {
	for b := range p.eq {
		q := effectBandQ
		switch b {
		case 0:
			p.eq[b].lowShelf(float64(p.rate), effectBands[b], q, e.Gains[b])
			p.eqR[b].lowShelf(float64(p.rate), effectBands[b], q, e.Gains[b])
		case len(p.eq) - 1:
			p.eq[b].highShelf(float64(p.rate), effectBands[b], q, e.Gains[b])
			p.eqR[b].highShelf(float64(p.rate), effectBands[b], q, e.Gains[b])
		default:
			p.eq[b].peaking(float64(p.rate), effectBands[b], q, e.Gains[b])
			p.eqR[b].peaking(float64(p.rate), effectBands[b], q, e.Gains[b])
		}
	}
	p.hp.highpass(float64(p.rate), e.Highpass, 0.7)
	p.hpR.highpass(float64(p.rate), e.Highpass, 0.7)
	p.lp.lowpass(float64(p.rate), math.Min(e.Lowpass, float64(p.rate)*0.475), 0.7)
	p.lpR.lowpass(float64(p.rate), math.Min(e.Lowpass, float64(p.rate)*0.475), 0.7)

	p.driveShape = 1 + e.Drive*4
	p.driveScale = math.Tanh(p.driveShape)
	p.driveComp = 1
	if e.Drive > 0 {
		p.driveComp = saturationCompensation(p.driveShape, p.driveScale)
	}
	bits := 16 - math.Sqrt(e.Crush)*12
	p.crushSteps = math.Max(2, math.Pow(2, bits-1))

	p.attack = math.Exp(-1 / (0.008 * float64(p.rate)))
	p.release = math.Exp(-1 / (0.18 * float64(p.rate)))

	p.noiseL.bandpass(float64(p.rate), 2600, 0.5)
	p.noiseR.bandpass(float64(p.rate), 2600, 0.5)
	if p.rev == nil {
		p.rev = newReverb(p.rate)
	}
}

// saturationCompensation measures how much a transfer shape lifts a
// reference sine, so that the curve does not make the chain louder.
func saturationCompensation(shape, scale float64) float64 {
	const cycle = 1024
	var energy float64
	for i := range cycle {
		in := 0.5 * math.Sin(2*math.Pi*float64(i)/cycle)
		shaped := math.Tanh(shape*in) / scale
		energy += shaped * shaped
	}
	rms := math.Sqrt(energy / cycle)
	reference := 0.5 / math.Sqrt2
	if rms == 0 {
		return 1
	}
	return reference / rms
}

// wowAt2 reads the delay line at the wow and flutter's drift, and writes
// the sound being heard into it.
func (p *effectProcessor) wowAt2(l, r, amount float64) (float64, float64) {
	p.wowPhase += 2 * math.Pi * 0.7 / float64(p.rate)
	p.flutterPhase += 2 * math.Pi * 6.8 / float64(p.rate)
	delay := 0.012 + amount*0.0045*math.Sin(p.wowPhase) + amount*0.0006*math.Sin(p.flutterPhase)
	frames := delay * float64(p.rate)
	n := float64(len(p.wow[0]))
	at := float64(p.wowAt) - frames
	if at < 0 {
		at += n
	}
	i0 := int(at) % len(p.wow[0])
	i1 := (i0 + 1) % len(p.wow[0])
	frac := at - math.Floor(at)
	p.wow[0][p.wowAt] = l
	p.wow[1][p.wowAt] = r
	p.wowAt = (p.wowAt + 1) % len(p.wow[0])
	return p.wow[0][i0]*(1-frac) + p.wow[0][i1]*frac,
		p.wow[1][i0]*(1-frac) + p.wow[1][i1]*frac
}

// compress applies a soft knee compressor to both channels, as one gain,
// then the makeup the settings give back.
func (p *effectProcessor) compress(l, r, punch float64) (float64, float64) {
	threshold := -6 - punch*26
	ratio := 1 + punch*7
	knee := 30 - punch*18
	peak := math.Max(math.Abs(l), math.Abs(r))
	db := 20 * math.Log10(peak+1e-9)
	var out float64
	switch {
	case db <= threshold-knee/2:
		out = db
	case db >= threshold+knee/2:
		out = threshold + (db-threshold)/ratio
	default:
		t := db - (threshold - knee/2)
		out = db + (1/ratio-1)*t*t/(2*knee)
	}
	target := 1.0
	if db > -90 {
		target = math.Pow(10, (out-db)/20)
	}
	coef := p.attack
	if target > p.compEnv {
		coef = p.release
	}
	p.compEnv = target + (p.compEnv-target)*coef
	makeup := math.Pow(10, punch*3/20)
	return l * p.compEnv * makeup, r * p.compEnv * makeup
}
