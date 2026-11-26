#ifndef OMNI_SETTINGSUI_GLASS_H
#define OMNI_SETTINGSUI_GLASS_H

#include <stdint.h>

// The window is the NSWindow's address as a number, never a Go pointer; the AppKit work runs on the main thread.
void omniWindowGlass(uintptr_t window, int on, double radius);

#endif
