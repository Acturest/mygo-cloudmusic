package app

import (
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"

	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/lyrics"
	"netease-music-mygo/internal/player"
)

// 回环, after Folia's claddagh: the line being sung runs round the front of
// a great ellipse seen at 45 degrees, the line before it fading round the
// back, the two after it coming on, and the ring turning a line at a time
// with a spring. Characters shrink, blur and dim as they go round the back,
// and light up one by one as they are sung.
const (
	ringMaxArc      = 4.25 // the most of the ring one line may span, radians
	ringLetterEm    = 0.04 // letter spacing, in em
	ringTrackEm     = 0.18 // extra tracking between characters, in em
	ringBackFollow  = 0.28 // how far a line behind follows the one being sung
	ringOrbitFollow = 0.52 // how far a line behind orbits with the song
	ringLead        = 340 * time.Millisecond
	ringMinorRatio  = 0.09 // the ellipse's minor radius, of the major
	// The ellipse is edge-on, so the back it turns into is a flat strip:
	// characters turning past the end crowd into a heap there and read as
	// stray glyphs cut loose from the line. Folia works the same geometry
	// but fades everything outside the window it is focused on; the same
	// window keeps a line's own wrapped text from making that heap.
	ringTurnFade = math.Pi/2 - 0.55 // where a character starts to dissolve
	ringTurnSpan = 0.28             // how far past it before it is gone
	ringTiltDeg     = 45
	ringPowerMult   = 0.25 // normal animation intensity
	ringPowerMax    = 1.25
	ringFocusScale  = 0.65
	ringWeight      = 700
	// The spring that turns the ring from one line to the next, as Folia's
	// stiffness 55, damping 14, mass 0.9.
	ringTurnStiffness = 55.0 / 0.9
	ringTurnDamping   = 14.0 / 0.9
)

// A ringGlyph is one grapheme of a line: when it is sung, and where on the
// ring it sits.
type ringGlyph struct {
	text  string
	start time.Duration
	end   time.Duration
	angle float32 // its centre's angle on the ring, radians
	scale float32 // below 1 for a line too long to fit the ring
}

// ringMetrics is a line laid out for the ring: its glyphs, and the width of
// each, for placing them.
type ringMetrics struct {
	glyphs []ringGlyph
	width  []float32
	height float32
}

// ringState is what 回环 keeps between frames: how far the ring has turned
// and how fast, the sound it listens to, and the lines measured so far.
type ringState struct {
	started bool
	base    int // the line at the front of the ring
	lastCur int
	lastPos time.Duration // the song time of the last frame, seconds
	holdPos time.Duration
	lastAt  time.Time
	offset  float64 // how far the ring has turned, radians
	offsetV float64
	bass    float64
	bassV   float64
	vocal   float64
	vocalV  float64
	glow    float64 // how much the axis glows, 0 to 1
	cache   map[string]*ringMetrics
	keys    []string
}

// claddaghRing draws the song on 回环 at pos.
func (a *app) claddaghRing(c *ui.Context, lines []lyrics.Line, cur int, pos time.Duration, w, h, f float32, accent ui.Color, live bool) {
	if cur < 0 || cur >= len(lines) {
		return
	}
	s := &a.ring
	if s.cache == nil {
		s.cache = make(map[string]*ringMetrics)
	}
	now := time.Now()
	dt := 1.0 / 60
	// After the song is taken back to its start the ring would spring all
	// the way back round; Folia holds the frame it had instead.
	hold := s.started && cur > 0 && pos <= 500*time.Millisecond && s.lastPos-pos > 500*time.Millisecond
	if s.started {
		if d := now.Sub(s.lastAt).Seconds(); d > 0 && d < 0.1 {
			dt = d
		}
	}
	s.lastAt = now
	playPos := pos
	if hold {
		playPos = s.holdPos
	}
	if !s.started {
		s.started = true
		s.base, s.lastCur = cur, cur
		s.offset = float64(cur) * math.Pi
		s.lastPos, s.holdPos = pos, pos
	} else if !hold {
		if cur != s.lastCur {
			if absInt(cur-s.lastCur) > 1 {
				// A jump, as a seek makes: no spring, the ring is there.
				s.offset, s.offsetV = float64(cur)*math.Pi, 0
			}
			s.lastCur = cur
		}
		s.base = cur
		s.offset, s.offsetV = ringSpring(s.offset, s.offsetV, float64(cur)*math.Pi,
			ringTurnStiffness, ringTurnDamping, dt)
	}
	s.lastPos = pos

	// The sound drives the ring's size and the glow of its axis. NetEase
	// gives 24 bands from 40 Hz to 14 kHz: the first six are Folia's bass,
	// the five and a half octaves up from them its vocal.
	var bass, vocal float64
	if live {
		if levels, ok := a.pl.Levels(); ok {
			bass = math.Pow(ringGroup(levels, 0, 6), 1.8) * 255
			vocal = math.Pow(ringGroup(levels, 13, 19), 1.5) * 255
		}
	}
	if !hold {
		s.bass, s.bassV = ringSpring(s.bass, s.bassV, bass, 150, 25, dt)
		s.vocal, s.vocalV = ringSpring(s.vocal, s.vocalV, vocal, 120, 24, dt)
	}
	power := float32(ringClamp01(s.bass / 255))
	vocalPower := float32(ringClamp01(s.vocal / 255))
	chorus := lines[cur].Chorus
	s.glow = ringApproach(s.glow, ringBool(chorus), 0.05)

	if live {
		c.AnimationFrame()
	}

	cx, cy := w/2, h/2
	rx := min(w*0.44, 560)
	scaleFactor := float32(math.Min(1+float64(power)*ringPowerMult, ringPowerMax))
	curRx := rx * scaleFactor
	rMajor := curRx
	rMinor := curRx * ringMinorRatio
	// Folia sets its glyphs at 72 points and its blur and glow in pixels at
	// that size; the stage sets lyrics smaller, so they keep their ratio.
	px := f / 72

	a.ringAxis(c, cx, cy, accent, power, vocalPower, chorus, float32(s.glow))

	active := a.ringMetrics(c, &lines[s.base], f, rx)
	if active == nil {
		return
	}
	activeEnd := ringRenderEnd(lines, s.base)
	activeWord := ringWordOffset(active, playPos, activeEnd)
	progress := ringLineProgress(active, playPos, activeEnd)
	orbit := float32(math.Pi) * ringOrbitFollow *
		(1 - float32(math.Pow(float64(1-progress), 1.35)))
	currentLen := ringVisualLength(lines[s.base].Text)

	// The four lines on the ring: the one behind, the one being sung, and
	// the two coming round.
	for idx := s.base - 1; idx <= s.base+2; idx++ {
		if idx < 0 || idx >= len(lines) {
			continue
		}
		m := active
		if idx != s.base {
			m = a.ringMetrics(c, &lines[idx], f, rx)
		}
		if m == nil {
			continue
		}
		line := lines[idx]
		lineDiff := float32(math.Abs(s.offset-float64(idx)*math.Pi)) / math.Pi
		act := max(0, 1-lineDiff)

		ownEnd := ringRenderEnd(lines, idx)
		ownWord := ringWordOffset(m, playPos, ownEnd)
		follow := float32(1)
		if idx == s.base {
			follow = clamp01(lineDiff)
		}
		wordOffset := ownWord
		if idx >= s.base {
			wordOffset += (activeWord*ringBackFollow + orbit) * follow
		}

		// A line that is still on the ring after the next one is due is
		// faded and shrunk by how much longer it is than the new one, so
		// that the two do not overlap on the back of the ring.
		lengthFade, lengthScale := float32(1), float32(1)
		if idx > s.base && currentLen > 10 {
			targetLen := ringVisualLength(line.Text)
			fadeStrength := clamp01((currentLen - 10) / 8)
			targetStrength := min(1, max(0.4, clamp01((targetLen-5)/5)))
			combined := fadeStrength * targetStrength
			targetOpacity := 1 - combined
			targetScale := 1 - 0.25*combined
			tp := clamp01((lineDiff - 0.4) / 0.5)
			lengthFade = 1 - (1-targetOpacity)*tp
			lengthScale = 1 - (1-targetScale)*tp
		}

		upcoming := float32(0)
		if idx > s.base {
			entry := clamp01(float32((playPos - (line.At - ringLead)).Seconds() / ringLead.Seconds()))
			upcoming = 0.18 + 0.82*entry
		}

		for i := range m.glyphs {
			g := m.glyphs[i]
			if g.text == "" {
				continue
			}
			theta := float64(idx)*math.Pi + float64(g.angle)
			psi := float32(theta - s.offset - float64(wordOffset))
			deltaDist := psi * curRx
			thetaCurve := deltaDist / rMajor
			localCos := float32(math.Cos(float64(thetaCurve)))
			depth := (localCos + 1) / 2

			// Folia compresses x by 0.35+0.65*D^1.2, which pulls the glyphs near the
			// sides of the ring towards the centre and off the ellipse (with a 0.09
			// minor axis they read as detached characters). Keeping the radius on
			// the ellipse fixes that; the front of the ring is identical either way.
			rawX := ringSin(thetaCurve) * rMajor
			rawY := localCos * rMinor
			if line.Chorus {
				stagger := f * (0.06 + power*0.12)
				if i%2 == 0 {
					rawY += stagger
				} else {
					rawY -= stagger
				}
			}
			rot := float64(-ringTiltDeg) * math.Pi / 180
			cosRot, sinRot := float32(math.Cos(rot)), float32(math.Sin(rot))
			x := rawX*cosRot - rawY*sinRot
			y := rawX*sinRot + rawY*cosRot
			tangentX := ringCos(thetaCurve) * rMajor
			tangentY := -ringSin(thetaCurve) * rMinor
			rtx := tangentX*cosRot - tangentY*sinRot
			rty := tangentX*sinRot + tangentY*cosRot
			tangent := ringNormalizeAngle(float32(math.Atan2(float64(rty), float64(rtx)) * 180 / math.Pi))
			tilt := min(38, max(-38, tangent*(0.4+0.6*depth)))

			maxVisible := curRx * 0.48
			distRatio := min(1, float32(math.Abs(float64(deltaDist)))/maxVisible)
			focus := act * ringPow(1-distRatio, 1.8)

			distanceOpacity := 0.68 + 0.32*ringPow(depth, 1.9)
			opacity := (0.35 + 0.65*ringPow(depth, 1.5)*(0.35+0.65*focus)) * distanceOpacity
			opacity *= clamp01(2 - lineDiff)
			if idx < s.base {
				opacity *= max(0, 1-lineDiff)
			}
			opacity *= lengthFade
			if idx > s.base {
				opacity *= upcoming
			}
			if lineDiff > 0.02 {
				boundary := clamp01((lineDiff - 0.3) / 0.6)
				threshold := 1 - 1.2*boundary
				if localCos > threshold {
					opacity *= clamp01(1 - (localCos-threshold)/0.15)
				}
			}
			if over := float32(math.Abs(float64(psi))) - ringTurnFade; over > 0 {
				opacity *= clamp01(1 - over/ringTurnSpan)
			}
			if opacity < 0.004 {
				continue
			}

			scale := (0.22 + 0.98*ringPow(depth, 1.5)) * (1 + ringFocusScale*focus) * lengthScale * g.scale
			blur := 8 * (1 - depth) * (1 - 0.5*focus) * px

			// The colour comes on at a stroke as the character is sung, with
			// a flash that fades: Folia's hard cut, not a slow fade.
			var color ui.Color
			alpha := float32(0.55)
			flash := float32(0)
			progressChar := float32(0)
			if playPos >= g.end {
				progressChar = 1
			} else if playPos > g.start {
				if d := (g.end - g.start).Seconds(); d > 0 {
					progressChar = float32((playPos - g.start).Seconds() / d)
				} else {
					progressChar = 1
				}
			}
			if progressChar >= 1 {
				color, alpha = accent, 1
			} else if progressChar > 0 {
				pf := min(1, progressChar*15)
				alpha = 0.55 + 0.45*pf
				color = white.Mix(accent, pf).Alpha(alpha)
				if progressChar < 0.4 {
					flash = ringPow(1-progressChar/0.4, 2) * 0.65
				}
			} else {
				color = white.Alpha(alpha)
			}

			var radius float32
			if line.Chorus {
				radius = (36 + power*24) * ringPow(focus, 1.5)
			} else {
				radius = 24 * ringPow(focus, 2)
			}
			radius *= 1 + flash

			ringDrawGlyph(c, ringFace{
				text:    g.text,
				cx:      cx + x,
				cy:      cy + y,
				width:   m.width[i],
				height:  m.height,
				font:    f,
				tilt:    tilt,
				scale:   scale,
				opacity: opacity,
				blur:    blur,
				color:   color,
				alpha:   alpha,
				radius:  radius * px,
				chorus:  line.Chorus,
			}, white, accent, idx*manyIndex+i)
		}
	}
}

// manyIndex keeps the keys of the glyphs of different lines apart.
const manyIndex = 10000

// ringFace is a glyph ready to be drawn on the ring.
type ringFace struct {
	text    string
	cx, cy  float32
	width   float32
	height  float32
	font    float32
	tilt    float32
	scale   float32
	opacity float32
	blur    float32
	color   ui.Color
	alpha   float32
	radius  float32
	chorus  bool
}

// ringDrawGlyph draws one character with the glow Folia gives it: a shadow
// of the glyph, blurred, behind it; three layers of it for a chorus.
func ringDrawGlyph(c *ui.Context, fc ringFace, primary, accent ui.Color, key int) {
	if fc.opacity < 0.004 || fc.text == "" {
		return
	}
	place := func(k int, col ui.Color, blur float32) {
		t := ui.Text(c.Key(k), fc.text).Absolute().Left(fc.cx).Top(fc.cy).
			Scale(fc.scale, fc.scale).Rotate(fc.tilt).
			Translate(-fc.width/2, -fc.height/2).
			FontSize(fc.font).FontWeight(ringWeight).NoWrap().
			LetterSpacing(fc.font * ringLetterEm).TextColor(col).Opacity(fc.opacity)
		if blur >= 0.2 {
			t.Blur(blur)
		}
	}
	if fc.radius > 0.5 {
		shadowFade := clamp01((fc.radius - 0.5) / 2)
		if shadowFade > 0.01 {
			faded := fc.color.Alpha(fc.alpha * shadowFade)
			// A text-shadow of radius r spreads as a blur of about half r.
			if fc.chorus {
				inner := fc.color.Mix(primary, 0.65).Alpha(shadowFade)
				place(key*10+1, inner, fc.radius*0.35/2)
				place(key*10+2, faded, fc.radius/2)
				place(key*10+3, faded, fc.radius*1.6/2)
			} else {
				place(key*10+1, faded, fc.radius/2)
			}
		}
	}
	place(key*10, fc.color, fc.blur)
	_ = accent
}

// ringAxis draws the faint upright axis the ring turns about: a gradient
// line, breathing with the bass, glowing while a chorus is sung.
func (a *app) ringAxis(c *ui.Context, cx, cy float32, accent ui.Color, power, vocalPower float32, chorus bool, glow float32) {
	ratio := min(1, max(0, max(power, vocalPower)-0.02)/0.58)
	mixed := white.Mix(accent, ratio)
	alpha := 0.2 + 0.75*ratio
	bass2 := power * power
	length, thick := float32(300), float32(4)
	gradient := func(col ui.Color) ui.LinearGradient {
		return ui.LinearGradient{
			Angle: 90,
			Stops: []ui.GradientStop{
				{Color: col.Alpha(0), At: 0},
				{Color: col, At: 0.2},
				{Color: col, At: 0.8},
				{Color: col.Alpha(0), At: 1},
			},
		}
	}
	bar := func(key string, col ui.Color, blur float32) {
		b := ui.Box(c.Key(key)).Absolute().Left(cx).Top(cy).
			Width(length).Height(thick).Scale(1+bass2*1.5, 1+bass2*0.5).
			Rotate(ringTiltDeg).Translate(-length/2, -thick/2).
			LinearGradient(gradient(col))
		if blur >= 0.2 {
			b.Blur(blur)
		}
	}
	if chorus && glow > 0.01 {
		// Folia's drop-shadow on the axis, a blurred copy behind it.
		bar("ring-axis-glow", mixed.Alpha(alpha*glow), (4+power*12)*glow/2)
	}
	bar("ring-axis", mixed.Alpha(alpha), 0)
}

// ringMetrics lays l out on a ring of radius rx, at font size f.
func (a *app) ringMetrics(c *ui.Context, l *lyrics.Line, f, rx float32) *ringMetrics {
	if l == nil {
		return nil
	}
	key := fmt.Sprintf("%d|%s|%.2f|%.2f", l.At, l.Text, f, rx)
	if m, ok := a.ring.cache[key]; ok {
		return m
	}
	m := ringMeasure(c, l, f, rx)
	if m == nil {
		return nil
	}
	if len(a.ring.cache) >= 240 && len(a.ring.keys) > 0 {
		old := a.ring.keys[0]
		a.ring.keys = a.ring.keys[1:]
		delete(a.ring.cache, old)
	}
	a.ring.cache[key] = m
	a.ring.keys = append(a.ring.keys, key)
	return m
}

// ringMeasure builds the glyphs of l and where on the ring each sits: the
// widest a line may run is ringMaxArc, and a longer one is set smaller.
func ringMeasure(c *ui.Context, l *lyrics.Line, f, rx float32) *ringMetrics {
	glyphs := ringTimeline(l)
	if len(glyphs) == 0 {
		return nil
	}
	n := len(glyphs)
	offsets := make([]float32, n+1)
	var prefix strings.Builder
	var fallback float32
	for k := 1; k <= n; k++ {
		prefix.WriteString(glyphs[k-1].text)
		fallback += ringFallbackWidth(glyphs[k-1].text, f)
		w, _ := c.MeasureText(0, ui.Span{
			Text:          prefix.String(),
			Size:          f,
			Weight:        ringWeight,
			LetterSpacing: f * ringLetterEm,
		})
		if !(w > 0) {
			w = fallback
		}
		off := w + float32(k-1)*f*ringTrackEm
		if off < offsets[k-1] {
			off = offsets[k-1]
		}
		offsets[k] = off
	}
	_, height := c.MeasureText(0, ui.Span{Text: "Hg", Size: f, Weight: ringWeight})
	if !(height > 0) {
		height = f
	}
	safeRadius := max(rx, f*2, 1)
	span := offsets[n] / safeRadius
	shrink := float32(1)
	if span > ringMaxArc {
		shrink = ringMaxArc / span
	}
	m := &ringMetrics{
		glyphs: glyphs,
		width:  make([]float32, n),
		height: height,
	}
	for i := range glyphs {
		centre := (offsets[i] + offsets[i+1]) / 2
		glyphs[i].angle = (centre/safeRadius - span/2) * shrink
		glyphs[i].scale = shrink
		m.width[i] = offsets[i+1] - offsets[i]
	}
	return m
}

// ringTimeline gives each grapheme of l when it is sung, shared through its
// word where the word has no times of its own, and spreads the gaps.
func ringTimeline(l *lyrics.Line) []ringGlyph {
	graphemes := ringGraphemes(l.Text)
	if len(graphemes) == 0 {
		return nil
	}
	start, end := l.At.Seconds(), l.End.Seconds()
	spans := make([]ringSpan, len(graphemes))
	resolved := make([]bool, len(graphemes))
	if len(l.Words) == 0 {
		dur := (end - start) / float64(len(graphemes))
		for i := range spans {
			spans[i] = ringSpan{start + dur*float64(i), start + dur*float64(i+1)}
			resolved[i] = true
		}
	} else {
		cursor := 0
		lastResolved := start
		for _, word := range l.Words {
			wordGraphemes := ringGraphemes(word.Text)
			if len(wordGraphemes) == 0 {
				cursor = max(cursor, ringFindSequence(graphemes, wordGraphemes, cursor))
				continue
			}
			at := ringFindSequence(graphemes, wordGraphemes, cursor)
			ws, we := word.Start.Seconds(), word.End.Seconds()
			for k := cursor; k < at; k++ {
				spans[k] = ringSpan{ws, ws}
				resolved[k] = true
			}
			dur := (we - ws) / float64(len(wordGraphemes))
			for k := range wordGraphemes {
				if at+k >= len(spans) {
					break
				}
				spans[at+k] = ringSpan{ws + dur*float64(k), ws + dur*float64(k+1)}
				resolved[at+k] = true
			}
			lastResolved = math.Max(lastResolved, we)
			cursor = max(cursor, at+len(wordGraphemes))
		}
		for i := range spans {
			if !resolved[i] {
				spans[i] = ringSpan{lastResolved, lastResolved}
			}
		}
	}
	spans = ringAdjustTimeline(spans, start, end)
	out := make([]ringGlyph, len(graphemes))
	for i := range graphemes {
		out[i] = ringGlyph{
			text:  graphemes[i],
			start: ringDuration(spans[i].start),
			end:   ringDuration(spans[i].end),
		}
	}
	return out
}

// ringSpan is when something is sung, in seconds.
type ringSpan struct{ start, end float64 }

// ringAdjustTimeline gives zero-length stretches, the gaps between words,
// time of their own, taking it from the words either side when there is
// too little, as Folia does, so that the ring does not sit still.
func ringAdjustTimeline(spans []ringSpan, lineStart, lineEnd float64) []ringSpan {
	for i := 0; i < len(spans); {
		if spans[i].start != spans[i].end {
			i++
			continue
		}
		j := i
		for j < len(spans) && spans[j].start == spans[j].end {
			j++
		}
		gapStart, gapEnd := lineStart, lineEnd
		if i > 0 {
			gapStart = spans[i-1].end
		}
		if j < len(spans) {
			gapEnd = spans[j].start
		}
		minNeeded := float64(j-i) * 0.06
		duration := gapEnd - gapStart
		if duration < minNeeded {
			deficit := minNeeded - duration
			switch {
			case i > 0 && j < len(spans):
				prevDur := spans[i-1].end - spans[i-1].start
				prevSteal := math.Min(deficit/2, math.Max(0, prevDur-0.04))
				nextDur := spans[j].end - spans[j].start
				nextSteal := math.Min(deficit-prevSteal, math.Max(0, nextDur-0.04))
				gapStart -= prevSteal
				gapEnd += nextSteal
				spans[i-1].end = gapStart
				spans[j].start = gapEnd
			case i > 0:
				prevDur := spans[i-1].end - spans[i-1].start
				steal := math.Min(deficit, math.Max(0, prevDur-0.04))
				gapStart -= steal
				spans[i-1].end = gapStart
			case j < len(spans):
				nextDur := spans[j].end - spans[j].start
				steal := math.Min(deficit, math.Max(0, nextDur-0.04))
				gapEnd += steal
				spans[j].start = gapEnd
			}
			duration = gapEnd - gapStart
		}
		unit := 0.0
		if n := j - i; n > 0 && duration > 0 {
			unit = duration / float64(n)
		}
		for k := i; k < j; k++ {
			spans[k].start = gapStart + unit*float64(k-i)
			spans[k].end = gapStart + unit*float64(k-i+1)
		}
		i = j
	}
	return spans
}

// ringFractionalIndex is how many characters of the line have been sung at
// t: whole numbers at each character, linear between them, and allowed to
// run a little past the last so that it comes round the front.
func ringFractionalIndex(glyphs []ringGlyph, t time.Duration, renderEnd time.Duration) float32 {
	n := len(glyphs)
	if n == 0 {
		return 0
	}
	last := n - 1
	if t <= glyphs[0].start {
		return 0
	}
	if t >= glyphs[last].start {
		if renderEnd > glyphs[last].start {
			p := float64(t-glyphs[last].start) / float64(renderEnd-glyphs[last].start)
			return float32(last) + float32(ringClamp01(p))*2
		}
		itemDur := (glyphs[last].end - glyphs[last].start).Seconds()
		gapDur := 0.0
		if last > 0 {
			gapDur = (glyphs[last].start - glyphs[last-1].start).Seconds()
		}
		step := 0.5
		if itemDur > 0 {
			step = itemDur
		} else if gapDur > 0 {
			step = gapDur
		}
		p := (t - glyphs[last].start).Seconds() / step
		return float32(last) + float32(math.Min(p, 1.8))
	}
	for i := 0; i < n-1; i++ {
		tStart, tEnd := glyphs[i].start, glyphs[i+1].start
		if t >= tStart && t < tEnd {
			if tEnd == tStart {
				return float32(i)
			}
			return float32(i) + float32((t-tStart).Seconds()/(tEnd-tStart).Seconds())
		}
	}
	return float32(last)
}

// ringWordOffset is the angle the line has turned by at t: it follows the
// character being sung, and runs past the last one rather than stopping.
func ringWordOffset(m *ringMetrics, t time.Duration, renderEnd time.Duration) float32 {
	n := len(m.glyphs)
	if n == 0 {
		return 0
	}
	frac := ringFractionalIndex(m.glyphs, t, renderEnd)
	last := n - 1
	if frac <= float32(last) {
		i := int(math.Floor(float64(frac)))
		p := frac - float32(i)
		if i >= n {
			i = last
		}
		a := m.glyphs[i].angle
		b := m.glyphs[min(i+1, last)].angle
		return a + (b-a)*p
	}
	lastAngle := m.glyphs[last].angle
	prevAngle := m.glyphs[max(0, last-1)].angle
	return lastAngle + (lastAngle-prevAngle)*(frac-float32(last))
}

// ringLineProgress is how far through the line t is, from 0 to 1.
func ringLineProgress(m *ringMetrics, t time.Duration, renderEnd time.Duration) float32 {
	n := len(m.glyphs)
	if n == 0 {
		return 0
	}
	if n == 1 {
		end := renderEnd
		if end == 0 {
			end = m.glyphs[0].end
		}
		dur := math.Max(0.001, (end - m.glyphs[0].start).Seconds())
		return float32(ringClamp01((t - m.glyphs[0].start).Seconds() / dur))
	}
	frac := ringFractionalIndex(m.glyphs, t, renderEnd)
	return float32(ringClamp01(float64(frac) / float64(max(1, n-1))))
}

// ringRenderEnd is when a line's time on the ring ends: when the next line
// starts to come on, or, for the last line, ten seconds after it ends.
func ringRenderEnd(lines []lyrics.Line, i int) time.Duration {
	end := lines[i].End
	if i+1 < len(lines) {
		end = max(end, lines[i+1].At-ringLead)
	} else {
		end = max(end, lines[i].End+10*time.Second)
	}
	return end
}

// ringGraphemes splits text into the characters a reader sees: a base and
// its accents, a joined emoji, a flag, one glyph.
func ringGraphemes(text string) []string {
	var out []string
	var cur strings.Builder
	var prev rune
	prevRI := false
	inRI := 0
	flush := func() {
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}
	for _, r := range text {
		joined := false
		switch {
		case cur.Len() == 0:
		case unicode.Is(unicode.Mn, r), unicode.Is(unicode.Mc, r), unicode.Is(unicode.Me, r):
			joined = true
		case r == 0x200D, prev == 0x200D: // zero width joiner
			joined = true
		case r >= 0xFE00 && r <= 0xFE0F, r >= 0xE0100 && r <= 0xE01EF: // variation selectors
			joined = true
		case r >= 0x1F3FB && r <= 0x1F3FF: // skin tones
			joined = true
		case r == 0x20E3: // keycap
			joined = true
		case r >= 0xE0020 && r <= 0xE007F: // tags
			joined = true
		case r >= 0x1F1E6 && r <= 0x1F1FF && prevRI && inRI%2 == 1:
			joined = true
		}
		isRI := r >= 0x1F1E6 && r <= 0x1F1FF
		if joined {
			cur.WriteRune(r)
			if isRI {
				inRI++
			}
		} else {
			flush()
			cur.WriteRune(r)
			inRI = 0
			if isRI {
				inRI = 1
			}
		}
		prev = r
		prevRI = isRI
	}
	flush()
	return out
}

// ringFindSequence finds the graphemes of a word in a line, from at.
func ringFindSequence(haystack, needle []string, at int) int {
	if len(needle) == 0 {
		return at
	}
	for i := at; i+len(needle) <= len(haystack); i++ {
		ok := true
		for k := range needle {
			if haystack[i+k] != needle[k] {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return at
}

// ringFallbackWidth is how wide a grapheme is taken to be when it cannot be
// measured.
func ringFallbackWidth(g string, f float32) float32 {
	if g == "" {
		return 0
	}
	if strings.TrimSpace(g) == "" {
		return f * 0.36
	}
	if ringCJK(g) {
		return f
	}
	return f * 0.62
}

// ringVisualLength scores a line by how much room it takes: a CJK character
// counts 1, any other 0.5.
func ringVisualLength(text string) float32 {
	var n float32
	for _, r := range text {
		if ringCJK(string(r)) {
			n++
		} else {
			n += 0.5
		}
	}
	return n
}

func ringCJK(g string) bool {
	for _, r := range g {
		if r >= 0x4E00 && r <= 0x9FA5 || r >= 0x3040 && r <= 0x30FF || r >= 0xAC00 && r <= 0xD7AF {
			return true
		}
	}
	return false
}

// ringGroup is the mean of a range of the spectrum.
func ringGroup(levels [player.Bands]float32, from, to int) float64 {
	sum := 0.0
	for i := from; i < to && i < len(levels); i++ {
		sum += float64(levels[i])
	}
	if to <= from {
		return 0
	}
	return sum / float64(to-from)
}

// ringSpring advances a spring towards target, as Folia's useSpring does,
// in small fixed steps so that the result does not depend on the frame rate.
func ringSpring(x, v, target, stiffness, damping, dt float64) (float64, float64) {
	if dt <= 0 {
		return x, v
	}
	steps := int(math.Ceil(dt / 0.004))
	if steps < 1 {
		steps = 1
	}
	if steps > 64 {
		steps = 64
	}
	h := dt / float64(steps)
	for range steps {
		acc := -stiffness*(x-target) - damping*v
		v += acc * h
		x += v * h
	}
	return x, v
}

// ringApproach moves v towards target by at most step.
func ringApproach(v, target, step float64) float64 {
	if v < target {
		return math.Min(target, v+step)
	}
	return math.Max(target, v-step)
}

func ringPow(v, e float32) float32 { return float32(math.Pow(float64(v), float64(e))) }
func ringSin(v float32) float32    { return float32(math.Sin(float64(v))) }
func ringCos(v float32) float32    { return float32(math.Cos(float64(v))) }

func ringClamp01(v float64) float64 { return math.Max(0, math.Min(1, v)) }

func ringBool(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func ringDuration(seconds float64) time.Duration {
	return time.Duration(seconds * float64(time.Second))
}

// ringNormalizeAngle keeps a tangent readable: past a right angle the
// glyph would hang upside down, so it turns back instead.
func ringNormalizeAngle(deg float32) float32 {
	for deg > 90 {
		deg -= 180
	}
	for deg < -90 {
		deg += 180
	}
	return deg
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
