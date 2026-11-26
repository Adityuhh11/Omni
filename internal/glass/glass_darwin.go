package glass

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework WebKit
void omniGlassShape(unsigned long window, double x, double y, double w, double h, double r, int dark);
*/
import "C"

import "errors"

// shape puts a visual effect view, masked to the card, behind the page (glass_darwin.m). The window stays as it is around the
// card, so the shadow the page draws still shows; a Mac's coordinates are points, which are CSS pixels. Only the card has glass
// here: shapes beside it come from Glass 2, whose glass on a Mac is the launcher's Liquid Glass (internal/panel).
func shape(window uintptr, shapes []Rect, dark bool) (Glass, error) {
	if window == 0 {
		return Glass{}, errors.New("glass: the window has no handle yet")
	}
	d := 0
	if dark {
		d = 1
	}
	var r Rect // none: the glass is taken away
	if len(shapes) > 0 {
		r = shapes[0]
	}
	C.omniGlassShape(C.ulong(window), C.double(r.X), C.double(r.Y), C.double(r.W), C.double(r.H), C.double(r.R), C.int(d))
	return Glass{Look: Blur}, nil // the Mac's own Reduce Transparency reaches the page as prefers-reduced-transparency
}

// forget has nothing to do on a Mac: the effect view belongs to its window and goes with it.
func forget(uintptr) {}
