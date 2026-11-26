package settingsui

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
#include "glass_darwin.h"
*/
import "C"

import "unsafe"

// setGlass puts the system's blur behind a card window's page, with the card's corners, or takes it away (glass_darwin.m).
func setGlass(window unsafe.Pointer, on bool, radius float64) {
	if window == nil {
		return
	}
	flag := 0
	if on {
		flag = 1
	}
	C.omniWindowGlass(C.uintptr_t(uintptr(window)), C.int(flag), C.double(radius))
}
