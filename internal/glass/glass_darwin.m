#import <Cocoa/Cocoa.h>
#import <objc/runtime.h>

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

void omniGlassShape(unsigned long window, double x, double y, double w, double h, double r, int dark) {
	// AppKit only on the main thread; the page calls in from another, and nothing waits for the answer.
	dispatch_async(dispatch_get_main_queue(), ^{
		NSWindow *win = (__bridge NSWindow *)(void *)window;
		NSView *content = win.contentView;
		NSVisualEffectView *fx = objc_getAssociatedObject(win, &glassKey);
		if (fx == nil) {
			fx = [[NSVisualEffectView alloc] initWithFrame:NSZeroRect];
			fx.blendingMode = NSVisualEffectBlendingModeBehindWindow;
			fx.state = NSVisualEffectStateActive; // frosted even while another program is in front
			fx.material = NSVisualEffectMaterialPopover;
			fx.autoresizingMask = NSViewMinYMargin; // stays at the top when the window grows under it, as the page does
			[content addSubview:fx positioned:NSWindowBelow relativeTo:nil];
			objc_setAssociatedObject(win, &glassKey, fx, OBJC_ASSOCIATION_RETAIN_NONATOMIC);
		}
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
	});
}
