package settingsui

import "github.com/wailsapp/wails/v3/pkg/application"

// platformWindow adds what only a Mac needs to a window's options: the windows the page draws itself (the cards and the timer) are on a clear
// window, which a Mac only gives when asked (a transparent backdrop, and a borderless window rather than a titled one with its title bar hidden),
// or there is an opaque plate behind the page's rounded edge. Their glass is internal/glass, cut to the page's card; a window with a title bar
// (the setup) is frosted whole by the system.
func platformWindow(options *application.WebviewWindowOptions, spec windowSpec) {
	if spec.card || spec.overlay {
		options.Mac = application.MacWindow{Backdrop: application.MacBackdropTransparent, CornerType: application.MacWindowCornerTypeSquare}
		return
	}
	options.Mac = application.MacWindow{Backdrop: application.MacBackdropTranslucent}
	options.BackgroundType = application.BackgroundTypeTranslucent
}
