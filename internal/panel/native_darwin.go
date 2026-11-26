package panel

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework QuartzCore -framework WebKit
#include "native_darwin.h"
*/
import "C"

import (
	"math"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// The window calls the panel needs from the system, for a Mac (native_darwin.m; the Windows ones are win32_windows.go). A "window"
// here is the NSWindow's address as a number, and the window that "had the keyboard" is the application that was frontmost, by its
// process id: the panel never activates this program, so that application stays the active one while the panel is up.

// newWindowTakesTheKeyboard is false on a Mac: the panel never activates this program, so there is nothing to give back.
const newWindowTakesTheKeyboard = false

// platformWindow makes the panel window a non-activating NSPanel: it takes typing without making this program the active one (so the app
// underneath keeps its place, and a full-screen app's Space is not left), floats above ordinary windows, appears on whichever Space is
// current and over full-screen apps, and leaves Mission Control and ⌘` alone. The page draws its own shadow.
func platformWindow(options *application.WebviewWindowOptions) {
	options.Mac = application.MacWindow{
		WindowClass:      application.MacWindowClassPanel,
		PanelPreferences: application.MacPanelPreferences{FloatingPanel: true, NonActivating: true},
		WindowLevel:      application.MacWindowLevelFloating,
		CollectionBehavior: application.MacWindowCollectionBehaviorCanJoinAllSpaces |
			application.MacWindowCollectionBehaviorFullScreenAuxiliary |
			application.MacWindowCollectionBehaviorIgnoresCycle,
		DisableShadow: true,
		// The page draws the whole bar, its rounded edge and its shadow, on a clear window. A frameless Mac window is still a titled
		// one with a hidden title bar, which keeps the system's corners and an opaque plate behind the page; "square" gives a
		// borderless window with nothing of its own.
		Backdrop:   application.MacBackdropTransparent,
		CornerType: application.MacWindowCornerTypeSquare,
	}
}

// foregroundWindow is the application that has the keyboard, or 0 when that is this program (its own windows are not somewhere to paste).
func foregroundWindow() uintptr {
	pid := int(C.omniFrontmostPID())
	if pid == os.Getpid() {
		return 0
	}
	return uintptr(pid)
}

// setForeground brings the application back to the front. The panel never took that from it, so this only matters when something else did.
func setForeground(app uintptr) { C.omniActivate(C.int(app)) }

// isWindow says whether the application is still running.
func isWindow(app uintptr) bool { return app != 0 && C.omniIsRunning(C.int(app)) != 0 }

// cloak on a Mac is plain ordering: out when on, in when off. (ponytail: if the first frame after showing turns out stale or blank, keep
// the panel ordered in at alpha 0 instead.)
func cloak(window uintptr, on bool) {
	if on {
		C.omniOrderOut(C.uintptr_t(window))
	} else {
		C.omniOrderFront(C.uintptr_t(window))
	}
}

// moveTopmost puts the window at r (points).
func moveTopmost(window uintptr, r Rect) {
	C.omniSetFrame(C.uintptr_t(window), C.double(r.X), C.double(r.Y), C.double(r.W), C.double(r.H))
}

// cursorMonitor returns the usable area of the screen that holds the mouse, in points, and the "dpi" the bar is drawn at there:
// 96 (a point per device-independent pixel) on a screen as dense as a MacBook's, less on a screen that draws each point larger
// (a 24-inch 1080p monitor), so the bar keeps its size to the eye. setPageScale zooms the page to match.
func cursorMonitor() (Rect, uint32) {
	var x, y, w, h, zoom C.double
	C.omniScreenUnderMouse(&x, &y, &w, &h, &zoom)
	return rect(x, y, w, h), uint32(math.Round(96 * float64(zoom)))
}

// setPageScale zooms the page to dpi (see cursorMonitor), so its CSS pixels fill the window place() sized.
func setPageScale(window uintptr, dpi uint32) {
	C.omniSetPageZoom(C.uintptr_t(window), C.double(float64(dpi)/96))
}

// windowRect is where the window is, in points.
func windowRect(window uintptr) Rect {
	var x, y, w, h C.double
	C.omniWindowFrame(C.uintptr_t(window), &x, &y, &w, &h)
	return rect(x, y, w, h)
}

// windowDPI is the dpi the page is drawn at now (96 times its zoom), for turning its CSS pixels into points.
func windowDPI(window uintptr) uint32 {
	return uint32(math.Round(96 * float64(C.omniPageZoom(C.uintptr_t(window)))))
}

// resizeHeight changes the height in place, keeping the top edge.
func resizeHeight(window uintptr, heightPx int) {
	C.omniSetHeight(C.uintptr_t(window), C.double(heightPx))
}

func rect(x, y, w, h C.double) Rect {
	return Rect{X: int(math.Round(float64(x))), Y: int(math.Round(float64(y))), W: int(math.Round(float64(w))), H: int(math.Round(float64(h)))}
}

// setGlass puts the system's blur under the card (or takes it away): the card is the rounded rectangle at (x, top) of size w by h in the
// window, and a change of shape takes ms milliseconds.
func setGlass(window uintptr, on bool, x, top, w, h, radius, ms int) {
	var flag C.int
	if on {
		flag = 1
	}
	C.omniGlass(C.uintptr_t(window), flag, C.double(x), C.double(top), C.double(w), C.double(h), C.double(radius), C.double(ms))
}

// setGlassPanes puts a round pane of the card's glass behind each quick-open tile: their left edges at xs, their top at top, size
// across; none takes them away.
func setGlassPanes(window uintptr, xs []int, top, size int) {
	lefts := make([]C.double, len(xs)+1) // +1: never empty, so there is always an element to point at
	for i, x := range xs {
		lefts[i] = C.double(x)
	}
	C.omniGlassPanes(C.uintptr_t(window), &lefts[0], C.int(len(xs)), C.double(top), C.double(size))
}
