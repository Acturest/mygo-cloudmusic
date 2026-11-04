package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"

	"netease-music-mygo/internal/player"
)

// songCacheDir is where the songs played are kept: under the directory the
// listener chose, or under the app's own cache directory.
func (a *app) songCacheDir() string {
	if dir := a.settings.CacheDir; dir != "" {
		return filepath.Join(dir, "songs")
	}
	dir, err := mygo.App.Path(mygo.PathCache)
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "songs")
}

// newSongCache returns the cache of songs played in dir, or nil without
// one.
func newSongCache(dir string, mb int) *player.Cache {
	if dir == "" {
		return nil
	}
	return player.NewCache(dir, int64(mb)<<20)
}

// cacheDirLabel says where the songs are kept, for the settings.
func (a *app) cacheDirLabel() string {
	if dir := a.songCacheDir(); dir != "" {
		return dir
	}
	return "无法得知缓存目录"
}

// chooseCacheDir asks where to keep the songs, then whether the songs in
// the directory left are cleared, and keeps them in the directory chosen.
// It waits on the dialogs, so it runs on a goroutine; old is where the
// songs are kept now.
func (a *app) chooseCacheDir(old string, parent *mygo.Window) {
	opts := mygo.OpenDialogOptions{
		Parent:            parent,
		Title:             "选择缓存目录",
		ButtonLabel:       "选择",
		Directory:         true,
		CreateDirectories: true,
	}
	if old != "" {
		opts.DefaultPath = filepath.Dir(old)
	}
	dirs, err := mygo.Dialog.Open(opts)
	if err != nil || len(dirs) == 0 {
		return
	}
	dir := dirs[0]
	// The songs are kept there already.
	if old != "" && filepath.Clean(dir) == filepath.Dir(filepath.Clean(old)) {
		return
	}
	detail := "旧缓存不会移动，也不会自动删除。请确认已清理旧缓存，再更换目录。"
	if old != "" {
		detail = "旧缓存（" + old + "）不会移动，也不会自动删除。请确认已清理旧缓存，再更换目录。"
	}
	res, err := mygo.Dialog.Message(mygo.MessageOptions{
		Parent:        parent,
		Type:          mygo.MessageWarning,
		Title:         "更换缓存目录",
		Message:       "歌曲缓存将保存到 " + filepath.Join(dir, "songs"),
		Detail:        detail,
		Buttons:       []string{"已清理，更换", "取消"},
		DefaultButton: 1,
		CancelButton:  1,
	})
	if err != nil || res.Button != 0 {
		return
	}
	if err := usableCacheDir(dir); err != nil {
		mygo.Dialog.Error("无法使用这个目录", err.Error())
		return
	}
	a.post(func() { a.setCacheDir(dir) })
}

// usableCacheDir makes sure the songs can be written to dir.
func usableCacheDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".probe-*")
	if err != nil {
		return err
	}
	f.Close()
	return os.Remove(f.Name())
}

// setCacheDir keeps the songs played in dir from now on, and under the
// app's cache directory when empty. The songs in the directory left stay
// there, for the listener to delete. It runs on the main thread.
func (a *app) setCacheDir(dir string) {
	a.settings.CacheDir = dir
	a.applyCacheDir()
	a.changed()
}

// applyCacheDir points the cache of the songs played at the directory the
// settings name, and measures it again.
func (a *app) applyCacheDir() {
	if cache := a.pl.Cache; cache != nil {
		cache.SetDir(a.songCacheDir())
	} else {
		a.pl.Cache = newSongCache(a.songCacheDir(), a.settings.CacheMB)
	}
	a.cacheUsed = -1
	a.measureCache()
}

// applyCacheLimit gives the cache the limit the settings say.
func (a *app) applyCacheLimit() {
	if cache := a.pl.Cache; cache != nil {
		limit := int64(a.settings.CacheMB) << 20
		go func() {
			cache.SetLimit(limit)
			a.measureCache()
		}()
	}
}

// measureCache finds how much the cache takes, on a goroutine, for the
// settings to show.
func (a *app) measureCache() {
	cache := a.pl.Cache
	if cache == nil {
		return
	}
	go func() {
		n := cache.Size()
		a.post(func() {
			a.cacheUsed = n
			if a.settingsWin != nil {
				a.settingsWin.Invalidate()
			}
		})
	}()
}

func (a *app) clearCache() {
	if cache := a.pl.Cache; cache != nil {
		a.cacheUsed = 0
		go func() {
			cache.Clear()
			a.measureCache()
		}()
	}
}

// cacheUsage says how much the cache takes, once measured.
func (a *app) cacheUsage() string {
	switch n := a.cacheUsed; {
	case n < 0:
		return "正在计算已用空间…"
	case n >= 1<<30:
		return fmt.Sprintf("已用 %.1f GB", float64(n)/(1<<30))
	default:
		return fmt.Sprintf("已用 %d MB", n>>20)
	}
}
