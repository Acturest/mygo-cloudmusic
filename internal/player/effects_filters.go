package player

import "math"

// biquad is one RBJ filter's coefficients and its state, for one channel.
type biquad struct {
	b0, b1, b2, a1, a2 float64
	x1, x2, y1, y2     float64
}

func (f *biquad) reset() { f.x1, f.x2, f.y1, f.y2 = 0, 0, 0, 0 }

func (f *biquad) process(x float64) float64 {
	y := f.b0*x + f.b1*f.x1 + f.b2*f.x2 - f.a1*f.y1 - f.a2*f.y2
	f.x2, f.x1 = f.x1, x
	f.y2, f.y1 = f.y1, y
	return y
}

func (f *biquad) set(b0, b1, b2, a0, a1, a2 float64) {
	f.b0, f.b1, f.b2 = b0/a0, b1/a0, b2/a0
	f.a1, f.a2 = a1/a0, a2/a0
}

func (f *biquad) peaking(rate, freq, q, gainDB float64) {
	w := 2 * math.Pi * freq / float64(rate)
	alpha := math.Sin(w) / (2 * q)
	a := math.Pow(10, gainDB/40)
	cos := math.Cos(w)
	f.set(1+alpha*a, -2*cos, 1-alpha*a, 1+alpha/a, -2*cos, 1-alpha/a)
}

func (f *biquad) lowShelf(rate, freq, q, gainDB float64) {
	w := 2 * math.Pi * freq / float64(rate)
	cos, sin := math.Cos(w), math.Sin(w)
	alpha := sin / 2 * math.Sqrt(2)
	a := math.Pow(10, gainDB/40)
	sqrtA := math.Sqrt(a)
	f.set(a*((a+1)-(a-1)*cos+2*sqrtA*alpha),
		2*a*((a-1)-(a+1)*cos),
		a*((a+1)-(a-1)*cos-2*sqrtA*alpha),
		(a+1)+(a-1)*cos+2*sqrtA*alpha,
		-2*((a-1)+(a+1)*cos),
		(a+1)+(a-1)*cos-2*sqrtA*alpha)
}

func (f *biquad) highShelf(rate, freq, q, gainDB float64) {
	w := 2 * math.Pi * freq / float64(rate)
	cos, sin := math.Cos(w), math.Sin(w)
	alpha := sin / 2 * math.Sqrt(2)
	a := math.Pow(10, gainDB/40)
	sqrtA := math.Sqrt(a)
	f.set(a*((a+1)+(a-1)*cos+2*sqrtA*alpha),
		-2*a*((a-1)+(a+1)*cos),
		a*((a+1)+(a-1)*cos-2*sqrtA*alpha),
		(a+1)-(a-1)*cos+2*sqrtA*alpha,
		2*((a-1)-(a+1)*cos),
		(a+1)-(a-1)*cos-2*sqrtA*alpha)
}

func (f *biquad) highpass(rate, freq, q float64) {
	w := 2 * math.Pi * freq / float64(rate)
	cos, sin := math.Cos(w), math.Sin(w)
	alpha := sin / (2 * q)
	f.set((1+cos)/2, -(1 + cos), (1+cos)/2, 1+alpha, -2*cos, 1-alpha)
}

func (f *biquad) lowpass(rate, freq, q float64) {
	w := 2 * math.Pi * freq / float64(rate)
	cos, sin := math.Cos(w), math.Sin(w)
	alpha := sin / (2 * q)
	f.set((1-cos)/2, 1-cos, (1-cos)/2, 1+alpha, -2*cos, 1-alpha)
}

func (f *biquad) bandpass(rate, freq, q float64) {
	w := 2 * math.Pi * freq / float64(rate)
	cos, sin := math.Cos(w), math.Sin(w)
	alpha := sin / (2 * q)
	f.set(alpha, 0, -alpha, 1+alpha, -2*cos, 1-alpha)
}

// reverb is a small feedback delay network in place of Folia's convolution
// reverb: four damped combs and two allpass filters for each channel, with
// the same long, dark tail the impulse response gives.
type reverb struct {
	combs [4][2][]float64
	at    [4][2]int
	damp  [4][2]float64
	fb    [4][2]float64
	ap    [2][2][]float64
	apAt  [2][2]int
}

var reverbCombMillis = [4]float64{29.7, 37.1, 41.1, 43.7}

const (
	reverbAllpassMillis  = 5.0
	reverbAllpass2Millis = 1.7
	reverbDamping        = 0.25
	reverbDecaySeconds   = 2.2
	reverbAllpassGain    = 0.5
)

func newReverb(rate int) *reverb {
	r := &reverb{}
	for c := 0; c < 2; c++ {
		spread := 1.0
		if c == 1 {
			spread = 1.021
		}
		for i := range r.combs {
			ms := reverbCombMillis[i] * spread
			n := int(float64(rate) * ms / 1000)
			r.combs[i][c] = make([]float64, n)
			r.fb[i][c] = math.Pow(10, -3*ms/1000/reverbDecaySeconds)
		}
		r.ap[0][c] = make([]float64, int(float64(rate)*reverbAllpassMillis/1000))
		r.ap[1][c] = make([]float64, int(float64(rate)*reverbAllpass2Millis/1000))
	}
	return r
}

func (r *reverb) reset() {
	for i := range r.combs {
		for c := 0; c < 2; c++ {
			clear(r.combs[i][c])
			r.at[i][c], r.damp[i][c] = 0, 0
		}
	}
	for i := range r.ap {
		for c := 0; c < 2; c++ {
			clear(r.ap[i][c])
			r.apAt[i][c] = 0
		}
	}
}

func (r *reverb) process(l, rIn float64) (float64, float64) {
	return r.channel(0, l), r.channel(1, rIn)
}

func (r *reverb) channel(c int, x float64) float64 {
	var sum float64
	for i := range r.combs {
		buf := r.combs[i][c]
		at := r.at[i][c]
		y := buf[at]
		r.damp[i][c] = y*(1-reverbDamping) + r.damp[i][c]*reverbDamping
		buf[at] = x + r.damp[i][c]*r.fb[i][c]
		r.at[i][c] = (at + 1) % len(buf)
		sum += y
	}
	sum *= 0.25
	for i := range r.ap {
		buf := r.ap[i][c]
		at := r.apAt[i][c]
		delayed := buf[at]
		buf[at] = sum + delayed*reverbAllpassGain
		r.apAt[i][c] = (at + 1) % len(buf)
		sum = delayed - sum*reverbAllpassGain
	}
	return sum
}
