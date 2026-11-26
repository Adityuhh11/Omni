#import <Cocoa/Cocoa.h>
#import <objc/runtime.h>
#import <WebKit/WebKit.h>

static char glassKey; // the window's effect view, kept on the window itself

// A mask the size of the corners that stretches in the middle: the effect view is cut to the card's rounded shape at any size.
static NSImage *roundedMask(CGFloat r) {
	CGFloat side = 2 * r + 1;
	NSImage *mask = [NSImage imageWithSize:NSMakeSize(side, side) flipped:NO drawingHandler:^BOOL(NSRect rect) {
		[[NSColor blackColor] set];
		[[NSBezierPath bezierPathWithRoundedRect:rect xRadius:r yRadius:r] fill];
		return YES;
	}];
	mask.capInsets = NSEdgeInsetsMake(r, r, r, r);
	mask.resizingMode = NSImageResizingModeStretch;
	return mask;
}

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

static char shapeKey; // the last shape the page asked for, in CSS pixels: @[x, y, w, h, r, dark]

// Puts the effect view where the page's card is: the shape is in CSS pixels, and the page may be zoomed to the screen
// (internal/panel's omniSetPageZoom), each CSS pixel then being that many points.
static void placeFx(NSWindow *win) {
	NSView *content = win.contentView;
	NSVisualEffectView *fx = objc_getAssociatedObject(win, &glassKey);
	NSArray<NSNumber *> *shape = objc_getAssociatedObject(win, &shapeKey);
	if (fx == nil || shape == nil) {
		return;
	}
	CGFloat zoom = pageZoomOf(win);
	CGFloat x = shape[0].doubleValue * zoom, y = shape[1].doubleValue * zoom, w = shape[2].doubleValue * zoom,
		h = shape[3].doubleValue * zoom, r = shape[4].doubleValue * zoom;
	BOOL dark = shape[5].boolValue;
	fx.hidden = w <= 0 || h <= 0;
	if (fx.hidden) {
		return;
	}
	// The page's theme, not the system's appearance, decides whether the glass is dark or light.
	fx.appearance = [NSAppearance appearanceNamed:dark ? NSAppearanceNameDarkAqua : NSAppearanceNameAqua];
	// The page measures from the top; a content view that is not flipped counts from the bottom.
	CGFloat top = content.isFlipped ? y : content.bounds.size.height - y - h;
	fx.frame = NSMakeRect(x, top, w, h);
	NSNumber *radius = objc_getAssociatedObject(fx, &glassKey);
	if (radius == nil || radius.doubleValue != r) {
		fx.maskImage = roundedMask(r);
		objc_setAssociatedObject(fx, &glassKey, @(r), OBJC_ASSOCIATION_RETAIN_NONATOMIC);
	}
}

void omniGlassShape(unsigned long window, double x, double y, double w, double h, double r, int dark) {
	// AppKit only on the main thread; the page calls in from another, and nothing waits for the answer.
	dispatch_async(dispatch_get_main_queue(), ^{
		NSWindow *win = (__bridge NSWindow *)(void *)window;
		NSVisualEffectView *fx = objc_getAssociatedObject(win, &glassKey);
		if (fx == nil) {
			fx = [[NSVisualEffectView alloc] initWithFrame:NSZeroRect];
			fx.blendingMode = NSVisualEffectBlendingModeBehindWindow;
			fx.state = NSVisualEffectStateActive; // frosted even while another program is in front
			fx.material = NSVisualEffectMaterialPopover;
			fx.autoresizingMask = NSViewMinYMargin; // stays at the top when the window grows under it, as the page does
			[win.contentView addSubview:fx positioned:NSWindowBelow relativeTo:nil];
			objc_setAssociatedObject(win, &glassKey, fx, OBJC_ASSOCIATION_RETAIN_NONATOMIC);
			// The page lays out the same at any zoom and does not ask again when it changes: place it again then.
			__weak NSWindow *weakWin = win;
			[[NSNotificationCenter defaultCenter] addObserverForName:@"OmniPageZoomChanged" object:win queue:[NSOperationQueue mainQueue]
			                                              usingBlock:^(NSNotification *note) {
				NSWindow *w = weakWin;
				if (w != nil) {
					placeFx(w);
				}
			}];
		}
		objc_setAssociatedObject(win, &shapeKey, @[ @(x), @(y), @(w), @(h), @(r), @(dark != 0) ], OBJC_ASSOCIATION_RETAIN_NONATOMIC);
		placeFx(win);
	});
}
