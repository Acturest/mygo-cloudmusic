package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo"

	"netease-music-mygo/internal/player"
)

// The data kept under the app's old name moves to the directory of its
// name, which Path made empty, but not over data kept there already.
func TestMoveOldData(t *testing.T) {
	root := t.TempDir()
	old, dir := filepath.Join(root, oldName), filepath.Join(root, "MyGo 云音乐")
	os.MkdirAll(old, 0o700)
	os.WriteFile(filepath.Join(old, "settings.json"), []byte("old"), 0o600)
	os.MkdirAll(dir, 0o700)
	moveDir(old, dir)
	if b, _ := os.ReadFile(filepath.Join(dir, "settings.json")); string(b) != "old" {
		t.Fatalf("settings %q after moving", b)
	}

	os.MkdirAll(old, 0o700)
	os.WriteFile(filepath.Join(old, "settings.json"), []byte("older"), 0o600)
	moveDir(old, dir)
	if b, _ := os.ReadFile(filepath.Join(dir, "settings.json")); string(b) != "old" {
		t.Fatalf("settings %q, moved over what was there", b)
	}
}

// The directory chosen is kept in the settings, the songs are kept under
// it from then on, and under the app's cache directory again once the
// choice is cleared.
func TestCacheDir(t *testing.T) {
	base := t.TempDir()
	appDir, dataDir := filepath.Join(base, "app"), filepath.Join(base, "data")
	mygo.App.SetPath(mygo.PathCache, appDir)
	mygo.App.SetPath(mygo.PathUserData, dataDir)
	t.Cleanup(func() {
		mygo.App.SetPath(mygo.PathCache, "")
		mygo.App.SetPath(mygo.PathUserData, "")
	})
	a := newTestApp()
	a.settings = defaultSettings()
	a.settings.path = filepath.Join(dataDir, "settings.json")
	a.pl.Cache = player.NewCache(filepath.Join(appDir, "songs"), 1<<20)
	chosen := filepath.Join(base, "chosen")
	if err := usableCacheDir(chosen); err != nil {
		t.Fatal(err)
	}
	a.settings.CacheDir = chosen
	a.settings.save()
	if got := loadSettings().CacheDir; got != chosen {
		t.Errorf("kept the directory as %q, want %q", got, chosen)
	}
	a.applyCacheDir()
	if got, want := a.pl.Cache.Dir(), filepath.Join(chosen, "songs"); got != want {
		t.Fatalf("keeps songs in %s, want %s", got, want)
	}
	a.settings.CacheDir = ""
	a.applyCacheDir()
	if got, want := a.pl.Cache.Dir(), filepath.Join(appDir, "songs"); got != want {
		t.Fatalf("keeps songs in %s after going back, want %s", got, want)
	}
}
