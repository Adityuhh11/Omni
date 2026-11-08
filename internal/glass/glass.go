// Package glass gives the windows that draw a card frosted glass: the system blurs what is behind the window, inside the rounded
// shape the page says its card has, and the page paints its theme's plate, part see-through, on top. A web page cannot blur what
// is behind its own window, so this is the system's work: a compositor visual with the blurred backdrop, clipped to the shape, on
// Windows (glass_windows.go), a visual effect view masked to the shape behind the page on a Mac (glass_darwin.go). While Windows'
// energy saver stops the system's blur, the glass is the wallpaper under the window, blurred here (wallpaper.go).
package glass

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// Service is bound once and called by every page that draws a card; it acts on the window of the page that calls it.
type Service struct {
	watched sync.Map // windows whose closing releases their glass, by window id
}

// What the glass behind a card is, as Shape reports it to the page, which paints its plate to match.
const (
	Blur      = "blur"      // the system blurs what is behind the card; the plate is part see-through
	Wallpaper = "wallpaper" // the system draws no blur for now (Windows' energy saver): the page lays Glass.Image under the plate
	Tint      = "tint"      // no blur and no wallpaper to show: the plate is nearly opaque, without a blur
	Off       = "off"       // the person asked the system for no transparency; the plate is opaque
)

// Glass is what Shape reports to the page.
type Glass struct {
	Look string // Blur, Wallpaper, Tint or Off
	// While Look is Wallpaper: the wallpaper of the window's screen, small and blurred (a data: URL), and where that screen lies
	// in CSS pixels from the window's top left, for the page to lay it under its card.
	Image                    string
	Left, Top, Width, Height float64
}

var logged sync.Map // failures already in the log, by message: the page asks again on every resize and every move

// logOnce writes a failure to the log the first time it happens.
func logOnce(err error, outcome string) {
	if _, seen := logged.LoadOrStore(err.Error(), true); !seen {
		log.Printf("%v; %s", err, outcome)
	}
}

// Shape puts the glass behind the calling window's card: x, y, width, height and corner radius in CSS pixels from the window's
// top left. A width of 0 takes it away. dark says whether the page's theme is dark, which picks the Mac's material; Windows'
// blur has no colour of its own. It says what the glass is.
func (s *Service) Shape(ctx context.Context, x, y, width, height, radius float64, dark bool) (Glass, error) {
	win, ok := ctx.Value(application.WindowKey).(application.Window)
	if !ok || win == nil {
		return Glass{}, errors.New("glass: the call came from no window")
	}
	hwnd := uintptr(win.NativeWindow())
	if _, seen := s.watched.LoadOrStore(win.ID(), true); !seen {
		win.OnWindowEvent(events.Common.WindowClosing, func(*application.WindowEvent) {
			s.watched.Delete(win.ID())
			forget(hwnd)
		})
	}
	g, err := shape(hwnd, Rect{X: x, Y: y, W: width, H: height, R: radius}, dark)
	if err != nil {
		logOnce(err, "the cards are opaque")
	}
	return g, err
}

// Rect is a rounded rectangle in CSS pixels.
type Rect struct{ X, Y, W, H, R float64 }
