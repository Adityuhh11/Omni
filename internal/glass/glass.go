// Package glass gives the windows that draw a card frosted glass: the system blurs what is behind the window, inside the rounded
// shape the page says its card has, and the page paints its theme's plate, part see-through, on top. A web page cannot blur what
// is behind its own window, so this is the system's work: acrylic clipped to the shape on Windows (glass_windows.go), a visual
// effect view masked to the shape behind the page on a Mac (glass_darwin.go).
package glass

import (
	"context"
	"errors"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Service is bound once and called by every page that draws a card; it acts on the window of the page that calls it.
type Service struct{}

// Shape puts the glass behind the calling window's card: x, y, width, height and corner radius in CSS pixels from the window's
// top left. A width of 0 takes it away. dark says whether the page's theme is dark, which picks the Mac's material; Windows'
// blur has no colour of its own.
func (*Service) Shape(ctx context.Context, x, y, width, height, radius float64, dark bool) error {
	win, ok := ctx.Value(application.WindowKey).(application.Window)
	if !ok || win == nil {
		return errors.New("glass: the call came from no window")
	}
	return shape(uintptr(win.NativeWindow()), Rect{X: x, Y: y, W: width, H: height, R: radius}, dark)
}

// Rect is a rounded rectangle in CSS pixels.
type Rect struct{ X, Y, W, H, R float64 }
