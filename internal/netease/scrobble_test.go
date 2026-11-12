package netease

import "testing"

// The log speaks as the macOS client, whatever this app is: that is the only
// picture the record credits.
func TestScrobbleHeaderIsTheMacClient(t *testing.T) {
	h := scrobbleHeader()
	for key, want := range map[string]string{
		"os":          "osx",
		"osver":       "15.5",
		"appver":      "3.1.10.5100",
		"versioncode": "140",
		"resolution":  "1920x1080",
		"channel":     "netease",
	} {
		if got := h[key]; got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
	if h["deviceId"] == "" || h["buildver"] == "" || h["requestId"] == "" {
		t.Fatalf("header lacks its device or time: %v", h)
	}
}
