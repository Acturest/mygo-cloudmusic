package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// A list set to zebra tints every other row, but never over the row under the
// pointer nor the song playing; a compact list is shorter and has no covers.
func TestListStyle(t *testing.T) {
	a := newTestApp()
	a.pal = paletteFor(&ui.Theme{})

	a.settings.ListStyle = listPlain
	if h, _, _, cover := a.listRow(); h != 54 || !cover {
		t.Fatalf("plain row is %g high, cover %v; want 54 and a cover", h, cover)
	}
	if bg := a.rowBackground(1, false, false); bg != ui.Transparent {
		t.Fatalf("plain list tinted row 1: %v", bg)
	}

	a.settings.ListStyle = listZebra
	if bg := a.rowBackground(0, false, false); bg != ui.Transparent {
		t.Fatalf("zebra tinted the first row")
	}
	if bg := a.rowBackground(1, false, false); bg != a.pal.zebra {
		t.Fatalf("zebra left row 1 untinted: %v", bg)
	}

	a.settings.ListStyle = listCompact
	if h, pad, gap, cover := a.listRow(); h != 42 || pad != 8 || gap != 10 || cover {
		t.Fatalf("compact row is %g high, pad %g, gap %g, cover %v", h, pad, gap, cover)
	}
	if bg := a.rowBackground(1, false, false); bg != a.pal.zebra {
		t.Fatalf("compact list is not zebra: %v", bg)
	}

	// The pointer and the song playing win over the tint, whatever the style.
	for _, style := range []string{listPlain, listZebra, listCompact} {
		a.settings.ListStyle = style
		if bg := a.rowBackground(1, false, true); bg != a.pal.hover {
			t.Fatalf("%q: the row under the pointer is %v, want the hover tint", style, bg)
		}
		if bg := a.rowBackground(1, true, false); bg != a.pal.accent.Alpha(0.09) {
			t.Fatalf("%q: the row playing is %v, want the accent tint", style, bg)
		}
	}
}
