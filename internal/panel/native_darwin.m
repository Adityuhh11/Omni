#import <Cocoa/Cocoa.h>
#import <objc/runtime.h>
#import <QuartzCore/QuartzCore.h>
#import <WebKit/WebKit.h>
#include "native_darwin.h"

// AppKit may only be used from the main thread; the hotkey and the page call in from others.
static void onMain(void (^block)(void)) {
	if ([NSThread isMainThread]) {
		block();
	} else {
		dispatch_sync(dispatch_get_main_queue(), block);
	}
}

static NSWindow *windowOf(uintptr_t window) {
	return (__bridge NSWindow *)(void *)window;
}

// The height of the primary screen (the one with the menu bar, first in the list): Cocoa's y grows upward from its bottom edge.
static CGFloat primaryHeight(void) {
	NSArray<NSScreen *> *screens = [NSScreen screens];
	return screens.count > 0 ? screens[0].frame.size.height : 0;
}

int omniFrontmostPID(void) {
	__block int pid = 0;
	onMain(^{
		NSRunningApplication *app = [[NSWorkspace sharedWorkspace] frontmostApplication];
		pid = app ? app.processIdentifier : 0;
	});
	return pid;
}

int omniIsRunning(int pid) {
	return [NSRunningApplication runningApplicationWithProcessIdentifier:pid] != nil;
}

// Brings the application forward, but only when it is not the one in front already: activating it raises every one of its windows, which
// would put two Safari windows above an Xcode window each time the bar is closed with Escape.
// The page's web view in a window, and its zoom: the bar's page is zoomed to the screen (omniSetPageZoom), so what it measures in CSS
// pixels is that many times as large in points.
static WKWebView *webViewIn(NSView *view) {
	if ([view isKindOfClass:[WKWebView class]]) {
		return (WKWebView *)view;
	}
	for (NSView *child in view.subviews) {
		WKWebView *found = webViewIn(child);
		if (found != nil) {
			return found;
		}
	}
	return nil;
}

static CGFloat pageZoomOf(NSWindow *win) {
	WKWebView *web = webViewIn(win.contentView);
	return web != nil && web.pageZoom > 0 ? web.pageZoom : 1;
}

// referencePPI is the density the bar is drawn for: a MacBook's Retina screen at its default size (about 128 points per inch).
static const double referencePPI = 128;

// How much smaller the bar is drawn on this screen, so that it is not much larger to the eye than on a MacBook: a screen with fewer points
// per inch (a 24-inch 1080p monitor has about 92) draws every point larger. Never larger than 1, never below 0.8; 1 when the
// screen does not say how large it is.
static double densityZoom(NSScreen *screen) {
	NSNumber *number = screen.deviceDescription[@"NSScreenNumber"];
	CGSize mm = CGDisplayScreenSize(number.unsignedIntValue);
	if (mm.width <= 0 || screen.frame.size.width <= 0) {
		return 1;
	}
	double ppi = screen.frame.size.width / (mm.width / 25.4);
	// Halfway (a square root) to the same size to the eye: fully matched, the bar looks too small next to everything else on a
	// big screen, which is drawn larger too. A 24-inch 1080p monitor gets about 0.85.
	return fmin(1, fmax(0.8, sqrt(ppi / referencePPI)));
}

static void placeGlass(NSWindow *win, double ms);
static void placePanes(NSWindow *win);

// Zooms the bar's page. The page lays out the same in CSS pixels at any zoom, so it measures nothing new and asks for no new
// glass: the glass behind it, here and in internal/glass (told by the notification), is placed again for the new zoom.
void omniSetPageZoom(uintptr_t window, double zoom) {
	onMain(^{
		NSWindow *win = windowOf(window);
		WKWebView *web = webViewIn(win.contentView);
		if (web != nil && fabs(web.pageZoom - zoom) > 0.001) {
			web.pageZoom = zoom;
			placeGlass(win, 0);
			placePanes(win);
			[[NSNotificationCenter defaultCenter] postNotificationName:@"OmniPageZoomChanged" object:win];
		}
	});
}

double omniPageZoom(uintptr_t window) {
	__block double zoom = 1;
	onMain(^{ zoom = pageZoomOf(windowOf(window)); });
	return zoom;
}

void omniActivate(int pid) {
	onMain(^{
		NSRunningApplication *front = [[NSWorkspace sharedWorkspace] frontmostApplication];
		if (front != nil && front.processIdentifier == pid) {
			return;
		}
		[[NSRunningApplication runningApplicationWithProcessIdentifier:pid] activateWithOptions:NSApplicationActivateAllWindows];
	});
}

// orderFrontRegardless shows the panel without making this program the active one.
void omniOrderFront(uintptr_t window) {
	onMain(^{ [windowOf(window) orderFrontRegardless]; });
}

void omniOrderOut(uintptr_t window) {
	onMain(^{ [windowOf(window) orderOut:nil]; });
}

// The usable area (menu bar and Dock left out) of the screen that holds the mouse.
void omniScreenUnderMouse(double *x, double *y, double *w, double *h, double *zoom) {
	onMain(^{
		NSPoint mouse = [NSEvent mouseLocation];
		NSScreen *found = nil;
		for (NSScreen *screen in [NSScreen screens]) {
			if (NSPointInRect(mouse, screen.frame)) {
				found = screen;
				break;
			}
		}
		if (found == nil) {
			found = [NSScreen mainScreen] ?: [NSScreen screens].firstObject;
		}
		NSRect area = found.visibleFrame;
		*x = area.origin.x;
		*y = primaryHeight() - (area.origin.y + area.size.height);
		*w = area.size.width;
		*h = area.size.height;
		*zoom = densityZoom(found);
	});
}

void omniWindowFrame(uintptr_t window, double *x, double *y, double *w, double *h) {
	onMain(^{
		NSRect frame = windowOf(window).frame;
		*x = frame.origin.x;
		*y = primaryHeight() - (frame.origin.y + frame.size.height);
		*w = frame.size.width;
		*h = frame.size.height;
	});
}

void omniSetFrame(uintptr_t window, double x, double y, double w, double h) {
	onMain(^{
		[windowOf(window) setFrame:NSMakeRect(x, primaryHeight() - y - h, w, h) display:NO];
	});
}

static void placeGlass(NSWindow *win, double ms);
static void placePanes(NSWindow *win);

// Changes the height and keeps the top edge where it is.
void omniSetHeight(uintptr_t window, double h) {
	onMain(^{
		NSWindow *win = windowOf(window);
		NSRect frame = win.frame;
		frame.origin.y += frame.size.height - h;
		frame.size.height = h;
		[win setFrame:frame display:YES];
		placeGlass(win, -1);
		placePanes(win);
	});
}


static const void *glassKey = &glassKey;

// A blur view ignores its layer's corners on some systems; a mask image that stretches between its corners is the way AppKit rounds one.
static NSImage *roundedMask(CGFloat radius) {
	CGFloat side = radius * 2 + 1;
	NSImage *mask = [NSImage imageWithSize:NSMakeSize(side, side) flipped:NO drawingHandler:^BOOL(NSRect rect) {
		[[NSColor blackColor] setFill];
		[[NSBezierPath bezierPathWithRoundedRect:rect xRadius:radius yRadius:radius] fill];
		return YES;
	}];
	mask.capInsets = NSEdgeInsetsMake(radius, radius, radius, radius);
	mask.resizingMode = NSImageResizingModeStretch;
	return mask;
}
// Liquid Glass (NSGlassEffectView, macOS 26 and later) where the system has it: it bends and lights what is behind the card as well as
// blurring it, and takes the card's corners itself. Older systems get the plain blur, rounded by a mask image.
static NSView *newGlass(NSRect frame) {
	if (@available(macOS 26.0, *)) {
		NSGlassEffectView *glass = [[NSGlassEffectView alloc] initWithFrame:frame];
		glass.style = NSGlassEffectViewStyleRegular; // frosted: the clear style barely blurs, and busy windows behind show through sharp
		glass.tintColor = [NSColor colorWithWhite:0.06 alpha:0.22]; // light: more of the desktop's colour comes through the bar
		if (@available(macOS 27.0, *)) {
			glass.effectIsInteractive = YES;
		}
		return glass;
	}
	NSVisualEffectView *blur = [[NSVisualEffectView alloc] initWithFrame:frame];
	blur.material = NSVisualEffectMaterialHUDWindow;
	blur.blendingMode = NSVisualEffectBlendingModeBehindWindow;
	blur.state = NSVisualEffectStateActive;
	return blur;
}

static void roundGlass(NSView *glass, CGFloat radius) {
	if (@available(macOS 26.0, *)) {
		if ([glass isKindOfClass:[NSGlassEffectView class]]) {
			((NSGlassEffectView *)glass).cornerRadius = radius;
			return;
		}
	}
	((NSVisualEffectView *)glass).maskImage = roundedMask(radius);
}

static const void *glassShapeKey = &glassShapeKey; // @[x, top, w, h, radius, ms]: the card's place in CSS pixels, from the window's top

// Moves the blur to the card's place as last asked for, over ms milliseconds (-1: the time last asked for). Cocoa measures from the
// bottom, so the place depends on the window's height; the page asks for the window's new height and the blur's new place in two calls
// that can land in either order, and so the blur is placed again after every change of height.
static void placeGlass(NSWindow *win, double ms) {
	NSView *glass = objc_getAssociatedObject(win, glassKey);
	NSArray<NSNumber *> *shape = objc_getAssociatedObject(win, glassShapeKey);
	if (glass == nil || shape == nil) {
		return;
	}
	// The shape is kept in the page's CSS pixels; the zoom now (omniSetPageZoom) turns it into points.
	NSView *host = win.contentView.superview;
	CGFloat zoom = pageZoomOf(win);
	double top = shape[1].doubleValue * zoom, h = shape[3].doubleValue * zoom;
	NSRect target = NSMakeRect(shape[0].doubleValue * zoom, host.bounds.size.height - top - h, shape[2].doubleValue * zoom, h);
	roundGlass(glass, shape[4].doubleValue * zoom);
	if (ms < 0) {
		ms = shape[5].doubleValue;
	}
	if (ms <= 0) {
		glass.frame = target;
		return;
	}
	[NSAnimationContext runAnimationGroup:^(NSAnimationContext *context) {
		context.duration = ms / 1000.0;
		context.timingFunction = [CAMediaTimingFunction functionWithControlPoints:0.16 :1 :0.3 :1];
		glass.animator.frame = target;
	}];
}

static const void *panesKey = &panesKey;         // NSMutableArray of the glass views behind the quick-open tiles
static const void *panesShapeKey = &panesShapeKey; // @[@[x, ...], top, size]: the tiles' places in CSS pixels, from the window's top

// Puts the quick-open tiles' glass where the page last said they are: a round pane of the same glass as the card behind each.
static void placePanes(NSWindow *win) {
	NSMutableArray<NSView *> *panes = objc_getAssociatedObject(win, panesKey);
	NSArray *shape = objc_getAssociatedObject(win, panesShapeKey);
	if (panes == nil || shape == nil) {
		return;
	}
	NSView *host = win.contentView.superview;
	CGFloat zoom = pageZoomOf(win);
	NSArray<NSNumber *> *xs = shape[0];
	double top = [shape[1] doubleValue] * zoom, size = [shape[2] doubleValue] * zoom;
	for (NSUInteger i = 0; i < panes.count && i < xs.count; i++) {
		panes[i].frame = NSMakeRect(xs[i].doubleValue * zoom, host.bounds.size.height - top - size, size, size);
		roundGlass(panes[i], size / 2);
	}
}

// Puts a blur of whatever is behind the window under the page, cut to the rounded rectangle at (x, top) of size w by h (points, from
// the window's top left). The page draws the card on a clear window, so the system's blur has to be a view of its own below the web
// view, shaped like the card. The top edge stays put when the window is resized (the margin below is what changes); a change of
// shape runs over ms milliseconds on the page's own easing, so the blur follows the card as it grows or shrinks.
void omniGlass(uintptr_t window, int on, double cx, double ctop, double cw, double ch, double cradius, double motion) {
	onMain(^{
		double ms = motion;
		NSWindow *win = windowOf(window);
		NSView *host = win.contentView.superview;
		NSView *glass = objc_getAssociatedObject(win, glassKey);
		if (!on) {
			[glass removeFromSuperview];
			objc_setAssociatedObject(win, glassKey, nil, OBJC_ASSOCIATION_RETAIN_NONATOMIC);
			return;
		}
		objc_setAssociatedObject(win, glassShapeKey, @[ @(cx), @(ctop), @(cw), @(ch), @(cradius), @(ms) ], OBJC_ASSOCIATION_RETAIN_NONATOMIC);
		if (glass == nil) {
			glass = newGlass(NSZeroRect);
			glass.autoresizingMask = NSViewMinYMargin;
			[host addSubview:glass positioned:NSWindowBelow relativeTo:win.contentView];
			objc_setAssociatedObject(win, glassKey, glass, OBJC_ASSOCIATION_RETAIN_NONATOMIC);
			ms = 0;
		}
		placeGlass(win, ms);
	});
}

// Puts a round pane of the card's glass behind each quick-open tile: n tiles of size by size CSS pixels, their left edges at xs and
// their top edges at top (from the window's top left). n = 0 takes them all away.
void omniGlassPanes(uintptr_t window, const double *xs, int n, double top, double size) {
	NSMutableArray<NSNumber *> *lefts = [NSMutableArray arrayWithCapacity:n];
	for (int i = 0; i < n; i++) {
		[lefts addObject:@(xs[i])];
	}
	onMain(^{
		NSWindow *win = windowOf(window);
		NSView *host = win.contentView.superview;
		NSMutableArray<NSView *> *panes = objc_getAssociatedObject(win, panesKey);
		if (panes == nil) {
			panes = [NSMutableArray array];
			objc_setAssociatedObject(win, panesKey, panes, OBJC_ASSOCIATION_RETAIN_NONATOMIC);
		}
		while ((int)panes.count > n) {
			[panes.lastObject removeFromSuperview];
			[panes removeLastObject];
		}
		while ((int)panes.count < n) {
			NSView *pane = newGlass(NSZeroRect);
			pane.autoresizingMask = NSViewMinYMargin;
			[host addSubview:pane positioned:NSWindowBelow relativeTo:win.contentView];
			[panes addObject:pane];
		}
		objc_setAssociatedObject(win, panesShapeKey, @[ lefts, @(top), @(size) ], OBJC_ASSOCIATION_RETAIN_NONATOMIC);
		placePanes(win);
	});
}
