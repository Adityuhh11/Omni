package glass

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

// The wallpaper of each screen, how it is laid and the colour around it come from Windows' IDesktopWallpaper, a classic COM
// object called through its table like the compositor (slots counted after IUnknown's three).

var (
	ole32               = windows.NewLazySystemDLL("ole32.dll")
	procCoCreate        = ole32.NewProc("CoCreateInstance")
	procMonitorOfWindow = user32.NewProc("MonitorFromWindow")
	procMonitorInfo     = user32.NewProc("GetMonitorInfoW")
	procClientToScreen  = user32.NewProc("ClientToScreen")
)

var (
	clsidDesktopWallpaper = guid("{C2CF3110-460E-4FC1-B9D0-8A1C0C9CC4BD}")
	iidDesktopWallpaper   = guid("{B92B56A9-8B55-4E14-9A89-0199BBB6F93B}")
)

const (
	slotGetWallpaper        = 4
	slotGetMonitorPathAt    = 5
	slotGetMonitorPathCount = 6
	slotGetMonitorRect      = 7
	slotGetBackgroundColor  = 9
	slotGetPosition         = 11
	clsctxAll               = 0x17
	monitorDefaultToNearest = 2
	wallpaperPositionSpan   = 5
)

// Windows' DESKTOP_WALLPAPER_POSITION, in its order: center, tile, stretch, fit, fill (span is fill over every screen).
var fits = []fit{fitCenter, fitTile, fitStretch, fitContain, fitFill, fitFill}

type rect struct{ left, top, right, bottom int32 }

func (r rect) image() image.Rectangle {
	return image.Rect(int(r.left), int(r.top), int(r.right), int(r.bottom))
}

// desktopOf reads how the wallpaper lies on the screen hwnd is on. It runs on the windows' thread, which has COM.
func desktopOf(hwnd uintptr) (desktop, error) {
	d := desktop{fallback: filepath.Join(os.Getenv("APPDATA"), `Microsoft\Windows\Themes\TranscodedWallpaper`)}
	info := struct {
		size          uint32
		monitor, work rect
		flags         uint32
	}{size: 40}
	monitor, _, _ := procMonitorOfWindow.Call(hwnd, monitorDefaultToNearest)
	if ok, _, _ := procMonitorInfo.Call(monitor, uintptr(unsafe.Pointer(&info))); ok == 0 {
		return d, fmt.Errorf("glass: the window's screen: %w", windows.GetLastError())
	}
	d.screen = info.monitor.image()
	var origin struct{ x, y int32 }
	procClientToScreen.Call(hwnd, uintptr(unsafe.Pointer(&origin)))
	d.origin = image.Pt(int(origin.x), int(origin.y))
	d.scale = 1
	if dpi, _, _ := procDpiOfWindow.Call(hwnd); dpi != 0 {
		d.scale = float64(dpi) / 96
	}

	var wp *object
	if hr, _, _ := procCoCreate.Call(uintptr(unsafe.Pointer(clsidDesktopWallpaper)), 0, clsctxAll,
		uintptr(unsafe.Pointer(iidDesktopWallpaper)), uintptr(unsafe.Pointer(&wp))); int32(hr) < 0 {
		return d, fmt.Errorf("glass: the desktop wallpaper object: HRESULT 0x%08X", uint32(hr))
	}
	defer wp.release()
	var position int32
	if err := wp.call(slotGetPosition, uintptr(unsafe.Pointer(&position))); err != nil {
		return d, fmt.Errorf("glass: how the wallpaper is laid: %w", err)
	}
	if position >= 0 && int(position) < len(fits) {
		d.fit = fits[position]
	}
	var bgr uint32 // a COLORREF: 0x00BBGGRR
	if err := wp.call(slotGetBackgroundColor, uintptr(unsafe.Pointer(&bgr))); err != nil {
		return d, fmt.Errorf("glass: the desktop's colour: %w", err)
	}
	d.color = color.RGBA{uint8(bgr), uint8(bgr >> 8), uint8(bgr >> 16), 255}

	var count uint32
	if err := wp.call(slotGetMonitorPathCount, uintptr(unsafe.Pointer(&count))); err != nil {
		return d, fmt.Errorf("glass: the screens: %w", err)
	}
	var id *uint16 // the wallpaper's id for the window's screen
	defer func() { windows.CoTaskMemFree(unsafe.Pointer(id)) }()
	for i := range count {
		var path *uint16
		if wp.call(slotGetMonitorPathAt, uintptr(i), uintptr(unsafe.Pointer(&path))) != nil {
			continue
		}
		var r rect
		if wp.call(slotGetMonitorRect, uintptr(unsafe.Pointer(path)), uintptr(unsafe.Pointer(&r))) != nil {
			windows.CoTaskMemFree(unsafe.Pointer(path)) // a screen that is switched off
			continue
		}
		d.area = d.area.Union(r.image())
		if id == nil && r.image() == d.screen {
			id = path
		} else {
			windows.CoTaskMemFree(unsafe.Pointer(path))
		}
	}
	if position != wallpaperPositionSpan {
		d.area = d.screen
	}
	var file *uint16
	if err := wp.call(slotGetWallpaper, uintptr(unsafe.Pointer(id)), uintptr(unsafe.Pointer(&file))); err != nil {
		return d, fmt.Errorf("glass: the wallpaper's file: %w", err)
	}
	defer windows.CoTaskMemFree(unsafe.Pointer(file))
	d.path = windows.UTF16PtrToString(file) // empty for a plain colour
	return d, nil
}
