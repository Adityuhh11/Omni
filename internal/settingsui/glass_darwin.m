#import <Cocoa/Cocoa.h>
#import <objc/runtime.h>
#include "glass_darwin.h"

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
		glass.style = NSGlassEffectViewStyleRegular; // frosted: the clear style barely blurs, and pages of text need a calm background
		glass.tintColor = [NSColor colorWithWhite:0.06 alpha:0.5];
// Only built with an SDK that has the property: @available checks at run time, not when compiling against an older SDK.
#if defined(__MAC_27_0) && __MAC_OS_X_VERSION_MAX_ALLOWED >= __MAC_27_0
		if (@available(macOS 27.0, *)) {
			glass.effectIsInteractive = YES;
		}
#endif
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

// Puts a blur of whatever is behind the window under the page, as large as the window and with the card's corners: the card windows
// are clear and their page is one rounded card that fills them, so the blur is a view of its own below the web view that grows with it.
void omniWindowGlass(uintptr_t window, int on, double radius) {
	void (^block)(void) = ^{
		NSWindow *win = (__bridge NSWindow *)(void *)window;
		NSView *host = win.contentView.superview;
		NSView *glass = objc_getAssociatedObject(win, glassKey);
		if (!on) {
			[glass removeFromSuperview];
			objc_setAssociatedObject(win, glassKey, nil, OBJC_ASSOCIATION_RETAIN_NONATOMIC);
			return;
		}
		if (glass == nil) {
			glass = newGlass(host.bounds);
			glass.autoresizingMask = NSViewWidthSizable | NSViewHeightSizable;
			[host addSubview:glass positioned:NSWindowBelow relativeTo:win.contentView];
			objc_setAssociatedObject(win, glassKey, glass, OBJC_ASSOCIATION_RETAIN_NONATOMIC);
		}
		roundGlass(glass, radius);
	};
	if ([NSThread isMainThread]) {
		block();
	} else {
		dispatch_sync(dispatch_get_main_queue(), block);
	}
}
