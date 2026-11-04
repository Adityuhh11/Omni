package focus

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// TestRealAddressBar opens Edge on a page of its own, off the screen, and reads its address bar. It starts a real browser, so it
// runs only with OMNI_REAL_BROWSER=1.
func TestRealAddressBar(t *testing.T) {
	if os.Getenv("OMNI_REAL_BROWSER") != "1" {
		t.Skip("set OMNI_REAL_BROWSER=1 to open Edge")
	}
	edge := filepath.Join(os.Getenv("ProgramFiles(x86)"), `Microsoft\Edge\Application\msedge.exe`)
	dir := t.TempDir()
	page := filepath.Join(dir, "omni-focus-test.html")
	if err := os.WriteFile(page, []byte("<title>OmniFocusTest</title>hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(edge, "--user-data-dir="+filepath.Join(dir, "profile"), "--no-first-run", "--window-position=-3000,0", "--new-window", page)
	if err := cmd.Start(); err != nil {
		t.Skipf("no Edge: %v", err)
	}
	defer func() { _ = exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run() }()

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	bar, err := openAddressBar()
	if err != nil {
		t.Fatal(err)
	}
	defer bar.close()

	var got string
	for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline) && !strings.Contains(got, "omni-focus-test"); time.Sleep(500 * time.Millisecond) {
		if w := windowTitled("OmniFocusTest"); w != 0 {
			start := time.Now()
			got = bar.read(w)
			t.Logf("read %q in %v", got, time.Since(start))
		}
	}
	if !strings.Contains(got, "omni-focus-test") {
		t.Fatalf("address bar = %q", got)
	}
}

// windowTitled is a top-level window whose title starts with prefix (Edge adds " - Profile 1 - Microsoft Edge").
func windowTitled(prefix string) windows.HWND {
	var found windows.HWND
	cb := windows.NewCallback(func(w windows.HWND, _ uintptr) uintptr {
		if strings.HasPrefix(titleOf(w), prefix) {
			found = w
			return 0
		}
		return 1
	})
	_, _, _ = user32.NewProc("EnumWindows").Call(cb, uintptr(unsafe.Pointer(nil)))
	return found
}
