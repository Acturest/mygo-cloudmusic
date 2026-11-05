package app

import (
	"math"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/lyrics"
)

// The stage's other styles, all after Folia: 心象 scatters a line in
// fragments about the middle, each springing in as it is sung; 回环 wraps
// a line around the front of a great ring that turns a line at a time.

// cadenzaShapes lays out the words of a line for 心象 at font size f:
// classic's springs, moved further apart and turned, so the line reads as
// fragments about the middle.
func cadenzaShapes(c *ui.Context, l *lyrics.Line, f, w, h float32) []wordShape {
	seed := float64(l.At / time.Millisecond)
	shapes := make([]wordShape, len(l.Words))
	widths := make([]float32, len(l.Words))
	for i, word := range l.Words {
		widths[i], _ = c.MeasureText(0, ui.Span{Text: word.Text, Size: f, Weight: 800})
		ws := seed + float64(i)*1.7
		shapes[i] = wordShape{
			x:      (lineRandom(ws, 1) - 0.5) * 40,
			y:      (lineRandom(ws, 2) - 0.5) * 40,
			rotate: (lineRandom(ws, 3) - 0.5) * 18,
			scale:  1.05 + lineRandom(ws, 4)*0.35,
			drift:  (lineRandom(ws, 5) - 0.5) * 36,
			pulse:  true,
			ghost:  true,
		}
	}
	spaceWords(shapes, widths, f*0.14)
	// Where each word sits about the middle: a little further out than
	// classic moves them, and a little up or down.
	for i := range shapes {
		ws := seed + float64(i)*1.7
		shapes[i].x += (lineRandom(ws, 11) - 0.5) * w * 0.09
		shapes[i].y += (lineRandom(ws, 12) - 0.5) * h * 0.18
	}
	if len(shapes) > 0 {
		shapes[len(shapes)-1].last = true
	}
	return shapes
}

// cadenzaLine shows l on 心象 as it is at pos: its words about the middle
// of the stage, springing in as they are sung, swelling a little while
// they are sung, then drifting.
func (a *app) cadenzaLine(c *ui.Context, l *lyrics.Line, pos time.Duration, f, w, h, fade float32, accent ui.Color) {
	shapes := cadenzaShapes(c, l, f, w, h)
	ui.Row(c.Key(l.At)).Absolute().Top(0).Left(0).Right(0).Bottom(0).Padding(0, w*0.10).
		Wrap().Justify(ui.Center).AlignContent(ui.Center).AlignItems(ui.Center).
		Opacity(fade).PassThrough().Children(func() {
		for i, word := range l.Words {
			a.stageWord(c.Key(i), word, shapes[i], pos, f, accent)
		}
	})
}

// ringGlyph is one character on 回环: what it is, when it is sung, how wide
// it is, a space too.
type ringGlyph struct {
	text       string
	space      bool
	start, end time.Duration
	width      float32
}

// ringGlyphs times the characters of l one by one, spreading each word's
// time over its characters, and measures them with 回环's letter spacing.
func ringGlyphs(c *ui.Context, l *lyrics.Line, f float32) []ringGlyph {
	var out []ringGlyph
	for _, word := range l.Words {
		text := word.Text
		n := utf8.RuneCountInString(text)
		if n == 0 {
			continue
		}
		each := (word.End - word.Start) / time.Duration(n)
		at := word.Start
		for _, r := range text {
			width, _ := c.MeasureText(0, ui.Span{Text: string(r), Size: f, Weight: 700})
			out = append(out, ringGlyph{
				text:  string(r),
				space: r == ' ',
				start: at,
				end:   at + each,
				width: width + f*0.18,
			})
			at += each
		}
	}
	return out
}

// claddaghRing shows l on 回环 as it is at pos: its characters along the
// front of a great ellipse, nearer ones larger and sharper, the ring
// turning a line's distance as a new line is due, the new line coming
// round from the right as the old one slides away to the left.
func claddaghRing(c *ui.Context, lines []lyrics.Line, cur int, pos time.Duration, w, h, f float32, accent ui.Color, live bool) {
	if cur < 0 || cur >= len(lines) {
		return
	}
	size := min(72, max(40, w*0.055)) * 0.9
	rx := min(w*0.42, 520)
	ry := rx * 0.22
	// The front of the ring, where the line sits, is the ring's lowest
	// point: keep that just under the middle of the stage.
	cx, cy := w/2, h*0.42
	// The ring turns by a line over about a second as a line is due: the
	// new line comes round from behind where the last one was.
	since := float32((pos - lines[cur].At).Seconds())
	turn := springStep(since / 0.9)
	if live && turn < 0.995 {
		c.AnimationFrame()
	}
	ui.Box(c.Key("ring")).Absolute().Top(0).Left(0).Right(0).Bottom(0).PassThrough().
		Draw(func(p *ui.Painter, r ui.Rect) {
			// A faint upright axis through the middle, fading at its ends:
			// the ring turns about it.
			const n = 30
			axis := min(420, h*0.5)
			for i := range n {
				t := float32(i) / n
				alpha := 0.2 * float32(math.Sin(float64(math.Pi)*(0.1+0.8*float64(t))))
				p.Line(cx, cy-axis/2+axis*t, cx, cy-axis/2+axis*(t+1.0/n), 1, accent.Alpha(alpha))
			}
		}).Children(func() {
		// The old line goes, then the new one; the far side of the ring is
		// left out, being behind the near one.
		for idx := cur - 1; idx <= cur; idx++ {
			if idx < 0 {
				continue
			}
			glyphs := ringGlyphs(c, &lines[idx], size)
			if len(glyphs) == 0 {
				continue
			}
			// A character takes its own width of the arc, and at least a
			// thirteenth of a radian, so a short line still curves.
			advances := make([]float32, len(glyphs))
			total := float32(0)
			for i, g := range glyphs {
				advances[i] = max(g.width*1.25/rx, 0.13)
				total += advances[i]
			}
			angle := -total / 2
			for i, g := range glyphs {
				nominal := angle + advances[i]/2
				angle += advances[i]
				psi := nominal - float32(idx-cur)*math.Pi + math.Pi*(1-turn)
				if !g.space {
					drawRingGlyph(c, g, psi, pos, size, rx, ry, cx, cy, accent)
				}
			}
		}
	})
}

// drawRingGlyph draws one character of 回环 at angle psi, lit as it is
// sung: nearer the front of the ring it is larger, sharper and brighter,
// and a light runs along the ring with the song.
func drawRingGlyph(c *ui.Context, g ringGlyph, psi float32, pos time.Duration, size, rx, ry, cx, cy float32, accent ui.Color) {
	cos, sin := float32(math.Cos(float64(psi))), float32(math.Sin(float64(psi)))
	depth := (cos + 1) / 2
	if depth < 0.18 {
		return
	}
	x := sin * rx
	y := cos * ry
	scale := 0.38 + 0.82*pow(depth, 1.5)
	// Each character swells as it is sung, then settles back to its size.
	if pos > g.start {
		t := clamp01(float32((pos - g.start).Seconds()) / max(0.12, float32((g.end-g.start).Seconds())))
		scale *= 1 + 0.26*float32(math.Sin(math.Pi*float64(t)))
	}
	lit := clamp01(float32((pos - g.start).Seconds()) / 0.2)
	opacity := clamp01((0.35 + 0.65*pow(depth, 1.5)) * (0.55 + 0.45*lit))
	blur := 6 * (1 - depth)
	tangent := float32(math.Atan2(float64(-sin*ry), float64(cos*rx)) * 180 / math.Pi)
	tangent = max(-26, min(26, tangent)) * (0.4 + 0.6*depth)
	glow := float32(0)
	if pos >= g.start {
		glow = 1 - clamp01(float32((pos-(g.end+350*time.Millisecond)).Seconds())/0.35)
	}
	color := white.Mix(accent, 0.85*glow)
	box := ui.Box(c).Absolute().Left(cx+x).Top(cy+y).
		Translate(-g.width/2, -size*0.65).Rotate(tangent).Scale(scale, scale).Opacity(opacity)
	box.Children(func() {
		if glow > 0.05 {
			ui.Text(c.Key("glow"), g.text).Absolute().Top(0).Left(0).FontSize(size).FontWeight(700).NoWrap().
				TextColor(accent.Alpha(0.55 * glow * opacity)).Blur(size * 0.25).PassThrough()
		}
		text := ui.Text(c.Key("body"), g.text).FontSize(size).FontWeight(700).NoWrap().TextColor(color)
		if blur > 0.3 {
			text.Blur(blur)
		}
	})
}

func pow(v, e float32) float32 { return float32(math.Pow(float64(v), float64(e))) }
