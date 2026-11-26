package glass

import (
	"errors"
	"fmt"
	"log"
	"math"
	"sync"
	"syscall"
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// Windows' own acrylic covers the whole window rectangle with its own small corners, and a window region does not cut it (tried).
// So the glass is drawn with the system compositor, the way WinUI draws acrylic: a visual behind the page, filled with the
// "host backdrop" (what is behind the window, already blurred by the system) and clipped to the card's rounded rectangle, which
// keeps the bar a capsule, the corners smooth and the page's shadow around the card.
//
// The compositor is a Windows Runtime object. Its methods are called through their tables by slot number, which is the order of
// the methods in Windows.UI.Composition's metadata (counted after IInspectable's six). Every call is made on the windows' thread.

var (
	combase         = windows.NewLazySystemDLL("combase.dll")
	coremessaging   = windows.NewLazySystemDLL("CoreMessaging.dll")
	dwmapi          = windows.NewLazySystemDLL("dwmapi.dll")
	user32          = windows.NewLazySystemDLL("user32.dll")
	kernel32        = windows.NewLazySystemDLL("kernel32.dll")
	procRoActivate  = combase.NewProc("RoActivateInstance")
	procMakeString  = combase.NewProc("WindowsCreateString")
	procFreeString  = combase.NewProc("WindowsDeleteString")
	procQueue       = coremessaging.NewProc("CreateDispatcherQueueController")
	procDwmSetAttr  = dwmapi.NewProc("DwmSetWindowAttribute")
	procDpiOfWindow = user32.NewProc("GetDpiForWindow")
	procPowerStatus = kernel32.NewProc("GetSystemPowerStatus")
)

var (
	iidCompositor        = guid("{B403CA50-7F8C-4E83-985F-CC45060036D8}")
	iidCompositor3       = guid("{C9DD8EF0-6EB1-4E3C-A658-675D9C64D4AB}")
	iidCompositor7       = guid("{D3483FAD-9A12-53BA-BFC8-88B7FF7977C6}")
	iidDesktopInterop    = guid("{29E691FA-4567-4DCA-B319-D0F207EB6807}")
	iidCompositionTarget = guid("{A1BEA8BA-D726-4663-8129-6B5E7927FFA6}")
	iidVisual            = guid("{117E202D-A859-4C89-873B-C2AA566788E3}")
	iidSpriteVisual      = guid("{08E05581-1AD1-4F97-9757-402D76E4233B}")
	iidBrush             = guid("{AB0D7608-30C0-40E9-B568-B60A6BD1FB46}")
	iidRectangleClip     = guid("{B3E7549E-00B4-5B53-8BE8-353F6C433101}")
	iidClip              = guid("{1CCD2A52-CFC7-4ACE-9983-146BB8EB6A3C}")
)

// Slots in the method tables (IUnknown has 3, IInspectable 6).
const (
	slotQueryInterface            = 0
	slotRelease                   = 2
	slotCreateDesktopWindowTarget = 3  // ICompositorDesktopInterop
	slotCreateContainerVisual     = 9  // ICompositor
	slotCreateSpriteVisual        = 22 // ICompositor
	slotCreateHostBackdropBrush   = 6  // ICompositor3
	slotCreateRectangleClip       = 8  // ICompositor7, no arguments
	slotPutRoot                   = 7  // ICompositionTarget
	slotGetChildren               = 6  // IContainerVisual
	slotInsertAtTop               = 9  // IVisualCollection
	slotPutBrush                  = 7  // ISpriteVisual
	slotPutClip                   = 15 // IVisual
	slotPutIsVisible              = 19
	slotPutOffset                 = 21
	slotPutSize                   = 36
	slotPutBottom                 = 7 // IRectangleClip
	slotPutBottomLeftRadius       = 9
	slotPutBottomRightRadius      = 11
	slotPutLeft                   = 13
	slotPutRight                  = 15
	slotPutTop                    = 17
	slotPutTopLeftRadius          = 19
	slotPutTopRightRadius         = 21
)

const dwmwaUseHostBackdropBrush = 17

// object is a COM object: its first word points at its table of methods.
type object struct{ table *[64]uintptr }

// The arguments may be pointers converted to uintptr (a place for a result, a Vector3): uintptrescapes keeps what they point at on
// the heap and alive until the call returns, so a stack that grows meanwhile cannot move it from under the system.
//
//go:uintptrescapes
func (o *object) call(slot int, args ...uintptr) error {
	hr, _, _ := syscall.SyscallN(o.table[slot], append([]uintptr{uintptr(unsafe.Pointer(o))}, args...)...)
	if int32(hr) < 0 {
		return fmt.Errorf("HRESULT 0x%08X", uint32(hr))
	}
	return nil
}

func (o *object) as(iid *windows.GUID) (*object, error) {
	var out *object
	if err := o.call(slotQueryInterface, uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&out))); err != nil {
		return nil, err
	}
	return out, nil
}

// create calls a method whose only argument is where it puts the object it makes.
func (o *object) create(slot int) (*object, error) {
	var out *object
	if err := o.call(slot, uintptr(unsafe.Pointer(&out))); err != nil {
		return nil, err
	}
	return out, nil
}

func (o *object) release() {
	if o != nil {
		_ = o.call(slotRelease) // Release returns a count, not an HRESULT: nothing to check
	}
}

// float is a float32 argument as the system's calling convention takes it; the first four arguments also travel in the float
// registers, and later ones are read from their slot on the stack.
func float(v float64) uintptr { return uintptr(math.Float32bits(float32(v))) }

// vector2 is a Vector2 passed by value: 8 bytes, which travel in one register.
func vector2(x, y float64) uintptr { return float(x) | float(y)<<32 }

func guid(s string) *windows.GUID {
	g, err := windows.GUIDFromString(s)
	if err != nil {
		panic(err)
	}
	return &g
}

// pane is one window's glass: a container visual on the window with a piece of blurred backdrop per shape (the card, and the
// quick-open tiles beside the bar). Pieces are kept once made and hidden while there are fewer shapes.
type pane struct {
	root     *object // IVisual of the container
	children *object // IVisualCollection
	pieces   []piece
	target   *object // kept alive: releasing it takes the visuals off the window
}

// piece is one rounded rectangle of glass.
type piece struct {
	visual *object // IVisual
	clip   *object // IRectangleClip
}

func (p *pane) release() {
	for _, pc := range p.pieces {
		pc.clip.release()
		pc.visual.release()
	}
	p.children.release()
	p.root.release()
	p.target.release()
}

var (
	compositor *object // ICompositor, made on the windows' thread on first use
	panes      = map[uintptr]*pane{}
	mu         sync.Mutex // the map; the compositor is only touched inside InvokeSync
	lastLook   string     // what the glass was the last time a card asked; guarded by mu
)

func shape(hwnd uintptr, shapes []Rect, _ bool) (Glass, error) {
	if hwnd == 0 {
		return Glass{}, errors.New("glass: the window has no handle yet")
	}
	lk := look()
	mu.Lock()
	prev := lastLook
	lastLook = lk
	mu.Unlock()
	if prev != lk && len(shapes) > 0 {
		// A card that shows no blur is otherwise a mystery: this says whether Windows asked for it.
		log.Printf("glass: %q (was %q); Windows' transparency effects and energy saver decide it", lk, prev)
	}
	var err error
	application.InvokeSync(func() {
		// A panic here would leave InvokeSync waiting for ever (Wails recovers it without finishing the call).
		defer func() {
			if v := recover(); v != nil {
				err = fmt.Errorf("glass: %v", v)
			}
		}()
		err = shapeNow(hwnd, shapes, lk)
	})
	return Glass{Look: lk}, err
}

// forget releases a window's glass when the window closes: Windows gives a later window the same handle, which must get a pane
// of its own.
func forget(hwnd uintptr) {
	application.InvokeSync(func() {
		mu.Lock()
		p := panes[hwnd]
		delete(panes, hwnd)
		mu.Unlock()
		if p != nil {
			p.release()
		}
	})
}

// look says what Windows lets the glass be. The host backdrop is blurred only while Windows draws transparency, and WebView2
// does not tell the page when it stops (prefers-reduced-transparency stays "no-preference"), so the window asks.
func look() string {
	if k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, registry.QUERY_VALUE); err == nil {
		v, _, err := k.GetIntegerValue("EnableTransparency")
		k.Close()
		if err == nil && v == 0 {
			return Off // the person turned transparency off: they get none
		}
	}
	var status struct {
		acLine, batteryFlag, batteryPercent, systemStatus uint8
		lifeTime, fullLifeTime                            uint32
	}
	if ok, _, _ := procPowerStatus.Call(uintptr(unsafe.Pointer(&status))); ok != 0 && status.systemStatus&1 != 0 {
		return Tint // energy saver stops the blur, not the person's wish for glass
	}
	return Blur
}

func shapeNow(hwnd uintptr, shapes []Rect, lk string) error {
	mu.Lock()
	p := panes[hwnd]
	mu.Unlock()
	if lk != Blur {
		// Released, not hidden: a backdrop that lived through Windows turning its blur off draws black once the blur is back, so
		// the next Blur builds a fresh pane. The bar's window lives for the whole run and would otherwise stay black.
		if p != nil {
			mu.Lock()
			delete(panes, hwnd)
			mu.Unlock()
			p.release()
		}
		return nil
	}
	if p == nil {
		if len(shapes) == 0 {
			return nil
		}
		var err error
		if p, err = newPane(hwnd); err != nil {
			return fmt.Errorf("glass: %w", err)
		}
		mu.Lock()
		panes[hwnd] = p
		mu.Unlock()
	}
	for len(p.pieces) < len(shapes) {
		pc, err := newPiece(p.children)
		if err != nil {
			return fmt.Errorf("glass: %w", err)
		}
		p.pieces = append(p.pieces, pc)
	}
	dpi, _, _ := procDpiOfWindow.Call(hwnd)
	if dpi == 0 {
		dpi = 96
	}
	scale := float64(dpi) / 96
	for i, pc := range p.pieces {
		shown := i < len(shapes)
		if err := pc.visual.call(slotPutIsVisible, boolArg(shown)); err != nil {
			return fmt.Errorf("glass: show the pane: %w", err)
		}
		if shown {
			if err := pc.place(shapes[i], scale); err != nil {
				return err
			}
		}
	}
	return nil
}

// place moves and cuts the piece to r, a rounded rectangle in CSS pixels, on a window at scale device pixels per CSS pixel.
func (pc piece) place(r Rect, scale float64) error {
	x, y, w, h, radius := r.X*scale, r.Y*scale, r.W*scale, r.H*scale, r.R*scale
	offset := [3]float32{float32(x), float32(y), 0} // a Vector3 is 12 bytes, passed by address
	if err := pc.visual.call(slotPutOffset, uintptr(unsafe.Pointer(&offset))); err != nil {
		return fmt.Errorf("glass: place the pane: %w", err)
	}
	steps := []struct {
		o    *object
		slot int
		arg  uintptr
	}{
		{pc.visual, slotPutSize, vector2(w, h)},
		{pc.clip, slotPutLeft, float(0)},
		{pc.clip, slotPutTop, float(0)},
		{pc.clip, slotPutRight, float(w)},
		{pc.clip, slotPutBottom, float(h)},
		{pc.clip, slotPutTopLeftRadius, vector2(radius, radius)},
		{pc.clip, slotPutTopRightRadius, vector2(radius, radius)},
		{pc.clip, slotPutBottomRightRadius, vector2(radius, radius)},
		{pc.clip, slotPutBottomLeftRadius, vector2(radius, radius)},
	}
	for _, step := range steps {
		if err := step.o.call(step.slot, step.arg); err != nil {
			return fmt.Errorf("glass: shape the pane: %w", err)
		}
	}
	return nil
}

func boolArg(b bool) uintptr {
	if b {
		return 1
	}
	return 0
}

// newPane puts an empty container visual behind the window's page, for the pieces of glass to go in.
func newPane(hwnd uintptr) (_ *pane, err error) {
	built := &pane{}
	defer func() {
		if err != nil {
			built.release() // what was made before the step that failed
		}
	}()
	if compositor == nil {
		c, err := newCompositor()
		if err != nil {
			return nil, err
		}
		compositor = c
	}
	on := int32(1)
	if hr, _, _ := procDwmSetAttr.Call(hwnd, dwmwaUseHostBackdropBrush, uintptr(unsafe.Pointer(&on)), 4); int32(hr) < 0 {
		return nil, fmt.Errorf("let the window use the backdrop: HRESULT 0x%08X", uint32(hr))
	}
	interop, err := compositor.as(iidDesktopInterop)
	if err != nil {
		return nil, fmt.Errorf("desktop interop: %w", err)
	}
	defer interop.release()
	// Not topmost: the visuals go behind the window's content, the page.
	if err := interop.call(slotCreateDesktopWindowTarget, hwnd, 0, uintptr(unsafe.Pointer(&built.target))); err != nil {
		return nil, fmt.Errorf("window target: %w", err)
	}
	container, err := compositor.create(slotCreateContainerVisual) // IContainerVisual
	if err != nil {
		return nil, fmt.Errorf("container visual: %w", err)
	}
	defer container.release()
	if built.children, err = container.create(slotGetChildren); err != nil {
		return nil, fmt.Errorf("the container's children: %w", err)
	}
	if built.root, err = container.as(iidVisual); err != nil {
		return nil, err
	}
	asTarget, err := built.target.as(iidCompositionTarget)
	if err != nil {
		return nil, err
	}
	defer asTarget.release()
	if err := asTarget.call(slotPutRoot, uintptr(unsafe.Pointer(built.root))); err != nil {
		return nil, fmt.Errorf("put the glass on the window: %w", err)
	}
	return built, nil
}

// newPiece adds a visual filled with the blurred backdrop, clipped to a rounded rectangle, on top of the pane's others.
func newPiece(children *object) (_ piece, err error) {
	var built piece
	defer func() {
		if err != nil {
			built.clip.release()
			built.visual.release()
		}
	}()
	sprite, err := compositor.create(slotCreateSpriteVisual)
	if err != nil {
		return piece{}, fmt.Errorf("sprite visual: %w", err)
	}
	defer sprite.release()
	c3, err := compositor.as(iidCompositor3)
	if err != nil {
		return piece{}, fmt.Errorf("compositor 3 (Windows 10 1703 or later): %w", err)
	}
	defer c3.release()
	backdrop, err := c3.create(slotCreateHostBackdropBrush)
	if err != nil {
		return piece{}, fmt.Errorf("backdrop brush: %w", err)
	}
	defer backdrop.release()
	brush, err := backdrop.as(iidBrush)
	if err != nil {
		return piece{}, err
	}
	defer brush.release()
	spriteVisual, err := sprite.as(iidSpriteVisual)
	if err != nil {
		return piece{}, err
	}
	defer spriteVisual.release()
	if err := spriteVisual.call(slotPutBrush, uintptr(unsafe.Pointer(brush))); err != nil {
		return piece{}, fmt.Errorf("fill the visual: %w", err)
	}
	if built.visual, err = sprite.as(iidVisual); err != nil {
		return piece{}, err
	}
	c7, err := compositor.as(iidCompositor7)
	if err != nil {
		return piece{}, fmt.Errorf("compositor 7 (Windows 10 2004 or later): %w", err)
	}
	defer c7.release()
	rect, err := c7.create(slotCreateRectangleClip)
	if err != nil {
		return piece{}, fmt.Errorf("clip: %w", err)
	}
	defer rect.release()
	if built.clip, err = rect.as(iidRectangleClip); err != nil {
		return piece{}, err
	}
	asClip, err := rect.as(iidClip)
	if err != nil {
		return piece{}, err
	}
	defer asClip.release()
	if err := built.visual.call(slotPutClip, uintptr(unsafe.Pointer(asClip))); err != nil {
		return piece{}, fmt.Errorf("clip the visual: %w", err)
	}
	if err := children.call(slotInsertAtTop, uintptr(unsafe.Pointer(built.visual))); err != nil {
		return piece{}, fmt.Errorf("add the visual to the glass: %w", err)
	}
	return built, nil
}

// newCompositor makes the compositor on this thread, with the dispatcher queue it needs to send its changes to the screen.
func newCompositor() (*object, error) {
	options := struct{ size, threadType, apartment uint32 }{12, 2, 2} // this thread, a single-threaded apartment
	var controller uintptr
	// A struct of 12 bytes is passed by address. A thread that already has a queue keeps it (the call fails, which is fine).
	_, _, _ = procQueue.Call(uintptr(unsafe.Pointer(&options)), uintptr(unsafe.Pointer(&controller)))
	name, err := windows.UTF16FromString("Windows.UI.Composition.Compositor")
	if err != nil {
		return nil, err
	}
	var class uintptr
	if hr, _, _ := procMakeString.Call(uintptr(unsafe.Pointer(&name[0])), uintptr(len(name)-1), uintptr(unsafe.Pointer(&class))); int32(hr) < 0 {
		return nil, fmt.Errorf("class name: HRESULT 0x%08X", uint32(hr))
	}
	defer procFreeString.Call(class)
	var inspectable *object
	if hr, _, _ := procRoActivate.Call(class, uintptr(unsafe.Pointer(&inspectable))); int32(hr) < 0 {
		return nil, fmt.Errorf("make the compositor: HRESULT 0x%08X", uint32(hr))
	}
	defer inspectable.release()
	return inspectable.as(iidCompositor)
}
