package app

import (
	"time"

	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/lyrics"
)

// 心象, after Folia's cadenza: a line scattered in fragments about the
// middle, each springing in as it is sung. 回环 lives in ui_stage_ring.go.

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
			scale:  1.0 + lineRandom(ws, 4)*0.28,
			drift:  (lineRandom(ws, 5) - 0.5) * 36,
			pulse:  true,
		}
	}
	spaceWords(shapes, widths, f*0.14)
	// Where each word sits about the middle: a little further out than
	// classic moves them, and a little up or down, so that the line reads
	// as fragments about the stage. Where each comes in from is worked out
	// as it is sung, from where it sits.
	for i := range shapes {
		ws := seed + float64(i)*1.7
		shapes[i].x += (lineRandom(ws, 11) - 0.5) * w * 0.07
		shapes[i].y += (lineRandom(ws, 12) - 0.5) * h * 0.12
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

