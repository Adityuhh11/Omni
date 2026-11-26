package settingsui

import "unsafe"

// setGlass does nothing on Windows: only a Mac puts its blur behind the Glass theme's windows.
func setGlass(unsafe.Pointer, bool, float64) {}
