package apps

import (
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

// makeWindow creates a visible top-level window of a class registered here, and returns its handle.
func makeWindow(t *testing.T, class string) windows.HWND {
	t.Helper()
	k32 := windows.NewLazySystemDLL("kernel32.dll")
	module, _, _ := k32.NewProc("GetModuleHandleW").Call(0)
	type wndClassEx struct {
		size                                         uint32
		style                                        uint32
		proc                                         uintptr
		clsExtra, wndExtra                           int32
		instance, icon, cursor, background, menuName uintptr
		className                                    *uint16
		iconSm                                       uintptr
	}
	name, _ := windows.UTF16PtrFromString(class)
	wc := wndClassEx{proc: user32.NewProc("DefWindowProcW").Addr(), instance: module, className: name}
	wc.size = uint32(unsafe.Sizeof(wc))
	if atom, _, err := user32.NewProc("RegisterClassExW").Call(uintptr(unsafe.Pointer(&wc))); atom == 0 {
		t.Fatalf("register the window class: %v", err)
	}
	const wsOverlappedVisible = 0x00CF0000 | 0x10000000
	hwnd, _, err := user32.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(name)), wsOverlappedVisible,
		0, 0, 200, 100, 0, 0, module, 0)
	if hwnd == 0 {
		t.Fatalf("create the window: %v", err)
	}
	t.Cleanup(func() { user32.NewProc("DestroyWindow").Call(hwnd) })
	return windows.HWND(hwnd)
}

func TestAStoreAppsHiddenCoreWindowIsNotTheWindowToBringForward(t *testing.T) {
	if hwnd := makeWindow(t, "OmniTestPlainWindow"); !isMainWindow(hwnd) {
		t.Fatal("an ordinary visible window was not taken for a main window")
	}
	// A Store app keeps this window for as long as it runs, open or not; bringing it forward shows nothing, and counting it
	// stopped Enter from opening Settings that was minimized.
	if hwnd := makeWindow(t, coreWindowClass); isMainWindow(hwnd) {
		t.Fatal("the core window of a Store app counted as its main window")
	}
}
