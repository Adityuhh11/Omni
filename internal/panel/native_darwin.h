#ifndef OMNI_PANEL_NATIVE_H
#define OMNI_PANEL_NATIVE_H

#include <stdint.h>

// The window is passed as a number (the NSWindow's address), never as a Go pointer. Every function runs its AppKit work on the main thread.
// Coordinates are points with the origin at the top left of the primary screen, as the Go side thinks of them.

int omniFrontmostPID(void);
int omniIsRunning(int pid);
void omniActivate(int pid);
void omniOrderFront(uintptr_t window);
void omniOrderOut(uintptr_t window);
void omniScreenUnderMouse(double *x, double *y, double *w, double *h, double *zoom);
void omniSetPageZoom(uintptr_t window, double zoom);
double omniPageZoom(uintptr_t window);
void omniWindowFrame(uintptr_t window, double *x, double *y, double *w, double *h);
void omniSetFrame(uintptr_t window, double x, double y, double w, double h);
void omniSetHeight(uintptr_t window, double h);
void omniGlass(uintptr_t window, int on, double x, double top, double w, double h, double radius, double ms);
void omniGlassPanes(uintptr_t window, const double *xs, int n, double top, double size);

#endif
