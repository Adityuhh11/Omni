package glass

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

// TestRealWallpaper reads this machine's wallpaper for the screen the desktop window is on and draws it; it runs only with
// OMNI_REAL_GLASS=1. With OMNI_GLASS_DUMP=<folder> it saves the picture there, to look at.
func TestRealWallpaper(t *testing.T) {
	if os.Getenv("OMNI_REAL_GLASS") != "1" {
		t.Skip("set OMNI_REAL_GLASS=1 to read the real wallpaper")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED); err != nil {
		t.Fatal(err)
	}
	defer windows.CoUninitialize()
	desk, _, _ := user32.NewProc("GetDesktopWindow").Call()
	start := time.Now()
	d, err := desktopOf(desk)
	if err != nil {
		t.Fatal(err)
	}
	read := time.Since(start)
	start = time.Now()
	g, err := wallpaperGlass(d)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("wallpaper %q fit %d colour %v screen %v area %v; read in %v, drawn in %v, %d bytes",
		d.path, d.fit, d.color, d.screen, d.area, read, time.Since(start), len(g.Image))
	start = time.Now()
	if _, err := wallpaperGlass(d); err != nil {
		t.Fatal(err)
	}
	t.Logf("again from the cache in %v", time.Since(start))
	if dump := os.Getenv("OMNI_GLASS_DUMP"); dump != "" {
		data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(g.Image, "data:image/png;base64,"))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dump, "wallpaper.png"), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
