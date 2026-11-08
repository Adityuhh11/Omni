package glass

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
void omniGlassShape(unsigned long window, double x, double y, double w, double h, double r, int dark);
*/
import "C"

import "errors"

// shape puts a visual effect view, masked to the card, behind the page (glass_darwin.m). The window stays as it is around the
// card, so the shadow the page draws still shows; a Mac's coordinates are points, which are CSS pixels.
func shape(window uintptr, r Rect, dark bool) (string, error) {
	if window == 0 {
		return "", errors.New("glass: the window has no handle yet")
	}
	d := 0
	if dark {
		d = 1
	}
	C.omniGlassShape(C.ulong(window), C.double(r.X), C.double(r.Y), C.double(r.W), C.double(r.H), C.double(r.R), C.int(d))
	return Blur, nil // the Mac's own Reduce Transparency reaches the page as prefers-reduced-transparency
}

// forget has nothing to do on a Mac: the effect view belongs to its window and goes with it.
func forget(uintptr) {}
