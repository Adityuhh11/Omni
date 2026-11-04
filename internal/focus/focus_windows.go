package focus

import (
	"fmt"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32        = windows.NewLazySystemDLL("user32.dll")
	showWindow    = user32.NewProc("ShowWindow")
	getWindowText = user32.NewProc("GetWindowTextW")
)

// browsers are the programs whose address bar is read, by file name.
var browsers = map[string]bool{
	"chrome.exe": true, "msedge.exe": true, "brave.exe": true, "vivaldi.exe": true, "opera.exe": true, "arc.exe": true,
	"firefox.exe": true, "zen.exe": true, "librewolf.exe": true, "floorp.exe": true, "waterfox.exe": true,
}

// winProbe reads the foreground window. The address bar is read again only when the window or its title changes (a new tab or a
// new page changes the title), which keeps UI Automation off the browser almost all of the time.
type winProbe struct {
	bar *addressBar

	lastWindow windows.HWND
	lastTitle  string
	lastURL    string
}

func newProbe() (probe, error) {
	bar, err := openAddressBar()
	if err != nil {
		return nil, err
	}
	return &winProbe{bar: bar}, nil
}

func (p *winProbe) front() (front, error) {
	w := windows.GetForegroundWindow()
	if w == 0 {
		return front{}, fmt.Errorf("no window in front")
	}
	exe, err := programOf(w)
	if err != nil {
		return front{}, err
	}
	f := front{app: exe, handle: uintptr(w)}
	if !browsers[exe] {
		return f, nil
	}
	title := titleOf(w)
	if w != p.lastWindow || title != p.lastTitle {
		p.lastWindow, p.lastTitle = w, title
		p.lastURL = p.bar.read(w) // "" when it cannot be read: the program rules still apply
	}
	f.url = p.lastURL
	return f, nil
}

// hide minimizes the window: nothing is closed or lost, and it is one click away once the period ends.
// ponytail: the whole browser window goes, not just the tab; closing the tab would need keystrokes into the browser.
func (p *winProbe) hide(f front) error {
	const swMinimize = 6
	showWindow.Call(f.handle, swMinimize) // its answer is whether the window was visible before, not success
	p.lastWindow = 0                      // whatever comes to the front next is read afresh
	return nil
}

func (p *winProbe) close() { p.bar.close() }

// programOf is the lower-case file name of the program that owns the window.
func programOf(w windows.HWND) (string, error) {
	var pid uint32
	if _, err := windows.GetWindowThreadProcessId(w, &pid); err != nil {
		return "", fmt.Errorf("owner of the window: %w", err)
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return "", fmt.Errorf("open process %d: %w", pid, err) // an elevated program: nothing Omni may look into
	}
	defer windows.CloseHandle(h)
	buf := make([]uint16, windows.MAX_LONG_PATH)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil {
		return "", fmt.Errorf("name of process %d: %w", pid, err)
	}
	return strings.ToLower(filepath.Base(windows.UTF16ToString(buf[:size]))), nil
}

func titleOf(w windows.HWND) string {
	buf := make([]uint16, 512)
	n, _, _ := getWindowText.Call(uintptr(w), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return windows.UTF16ToString(buf[:n])
}
