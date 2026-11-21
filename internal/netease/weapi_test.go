package netease

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

// weapiDecrypt undoes weapiAES for the tests.
func weapiDecrypt(t *testing.T, key, encoded string) string {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("params are not base64: %v", err)
	}
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		t.Fatalf("key: %v", err)
	}
	out := make([]byte, len(raw))
	cipher.NewCBCDecrypter(block, []byte(weapiIV)).CryptBlocks(out, raw)
	pad := int(out[len(out)-1])
	if pad < 1 || pad > aes.BlockSize {
		t.Fatalf("bad padding %d", pad)
	}
	return string(out[:len(out)-pad])
}

func TestWeapiEncryptRoundTrips(t *testing.T) {
	key := strings.Repeat("a", 16)
	data := map[string]any{"alg": "itembased", "trackId": 7, "like": true, "time": 3}
	params, encSecKey, err := weapiEncrypt(data, key)
	if err != nil {
		t.Fatal(err)
	}
	// The first layer is the preset key, the second the request's own.
	inner := weapiDecrypt(t, key, params)
	text := weapiDecrypt(t, weapiPresetKey, inner)
	var got map[string]any
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("payload is not JSON: %v", err)
	}
	if got["alg"] != "itembased" || got["trackId"] != float64(7) || got["like"] != true {
		t.Errorf("payload = %v", got)
	}
	if len(encSecKey) != 256 || strings.Trim(encSecKey, "0123456789abcdef") != "" {
		t.Errorf("encSecKey = %q, want 256 hex digits", encSecKey)
	}
}

func TestSongURLBitrate(t *testing.T) {
	for level, want := range map[string]int{
		"standard": 128000,
		"higher":   192000,
		"exhigh":   320000,
		"lossless": 999000,
		"hires":    999000,
		"":         320000,
	} {
		if got := bitrate(level); got != want {
			t.Errorf("bitrate(%q) = %d, want %d", level, got, want)
		}
	}
}
