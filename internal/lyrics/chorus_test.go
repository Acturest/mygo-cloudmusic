package lyrics

import "testing"

// The chorus is the text that comes round most often, as Folia finds it.
func TestMarkChorus(t *testing.T) {
	lrc := "[00:10.00]夜空下最亮的星\n" +
		"[00:14.00]能否听清\n" +
		"[00:18.00]夜空下最亮的星\n" +
		"[00:22.00]......\n" +
		"[00:26.00]夜空下最亮的星\n" +
		"[00:30.00]能否听清"
	lines := Build(lrc, "", "")
	if len(lines) != 6 {
		t.Fatalf("%d lines, want 6", len(lines))
	}
	for _, i := range []int{0, 2, 4} {
		if !lines[i].Chorus {
			t.Errorf("line %d %q is not marked as chorus", i, lines[i].Text)
		}
	}
	for _, i := range []int{1, 3, 5} {
		if lines[i].Chorus {
			t.Errorf("line %d %q is marked as chorus", i, lines[i].Text)
		}
	}
}

// With no text sung twice there is no chorus.
func TestMarkChorusNone(t *testing.T) {
	lines := Build("[00:10.00]一\n[00:12.00]二二\n[00:14.00]三三三", "", "")
	for i, l := range lines {
		if l.Chorus {
			t.Errorf("line %d %q is marked as chorus", i, l.Text)
		}
	}
}
