package settingsui

import "github.com/wailsapp/wails/v3/pkg/application"

// platformWindow adds what only Windows needs to a window's options: a dark title bar and Windows' own Mica glass for a window that has one (the
// setup), and no frame decorations for the ones the page draws itself (the cards and the timer, which also has no taskbar button; their glass is
// internal/glass, cut to the page's card).
func platformWindow(options *application.WebviewWindowOptions, spec windowSpec) {
	switch {
	case spec.overlay:
		options.Windows = application.WindowsWindow{HiddenOnTaskbar: true, DisableFramelessWindowDecorations: true}
	case spec.card:
		options.Windows = application.WindowsWindow{DisableFramelessWindowDecorations: true}
	default:
		options.Windows = application.WindowsWindow{Theme: application.Dark, BackdropType: application.Mica}
		options.BackgroundType = application.BackgroundTypeTranslucent
	}
}
