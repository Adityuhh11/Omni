package focus

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// UI Automation, called through its COM vtables: the few calls needed to read the text of a window's first edit field, which in
// every supported browser is the address bar. The slot numbers are the order of the methods in UIAutomationClient.h.
const (
	slotRelease                 = 2
	slotElementFromHandle       = 6  // IUIAutomation
	slotCreatePropertyCondition = 23 // IUIAutomation
	slotFindFirst               = 5  // IUIAutomationElement
	slotGetCurrentPropertyValue = 10 // IUIAutomationElement

	propControlType = 30003
	propValueValue  = 30045
	controlEdit     = 50004
	treeDescendants = 4

	vtI4   = 3
	vtBSTR = 8
)

var (
	clsidCUIAutomation = windows.GUID{Data1: 0xff48dba4, Data2: 0x60ef, Data3: 0x4201, Data4: [8]byte{0xaa, 0x87, 0x54, 0x10, 0x3e, 0xef, 0x59, 0x4e}}
	iidIUIAutomation   = windows.GUID{Data1: 0x30cbe57d, Data2: 0xd9d0, Data3: 0x452a, Data4: [8]byte{0xab, 0x13, 0x7a, 0xc5, 0xac, 0x48, 0x25, 0xee}}

	ole32            = windows.NewLazySystemDLL("ole32.dll")
	oleaut32         = windows.NewLazySystemDLL("oleaut32.dll")
	coCreateInstance = ole32.NewProc("CoCreateInstance")
	variantClear     = oleaut32.NewProc("VariantClear")
)

// variant is a COM VARIANT on 64-bit Windows: the type, three reserved words, then the value.
type variant struct {
	vt  uint16
	_   [3]uint16
	val uintptr
	_   uintptr
}

// comObject is a COM interface pointer: the object starts with the address of its table of methods.
type comObject struct{ vtbl *[32]uintptr }

// call calls method number slot of obj, which is always its own first argument.
func call(obj *comObject, slot int, args ...uintptr) uintptr {
	r, _, _ := syscall.SyscallN(obj.vtbl[slot], append([]uintptr{uintptr(unsafe.Pointer(obj))}, args...)...)
	return r
}

func release(obj *comObject) {
	if obj != nil {
		call(obj, slotRelease)
	}
}

// addressBar holds the automation object and the "is an edit field" condition, made once.
type addressBar struct {
	automation, isEdit *comObject
}

func openAddressBar() (*addressBar, error) {
	if err := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED); err != nil {
		return nil, fmt.Errorf("start COM: %w", err)
	}
	var a *comObject
	const clsctxInprocServer = 1
	if hr, _, _ := coCreateInstance.Call(uintptr(unsafe.Pointer(&clsidCUIAutomation)), 0, clsctxInprocServer,
		uintptr(unsafe.Pointer(&iidIUIAutomation)), uintptr(unsafe.Pointer(&a))); hr != 0 {
		windows.CoUninitialize()
		return nil, fmt.Errorf("create UI Automation: %#x", hr)
	}
	edit := variant{vt: vtI4, val: controlEdit}
	var cond *comObject
	// A VARIANT is 16 bytes, so the x64 calling convention passes it by address.
	if hr := call(a, slotCreatePropertyCondition, propControlType, uintptr(unsafe.Pointer(&edit)), uintptr(unsafe.Pointer(&cond))); hr != 0 {
		release(a)
		windows.CoUninitialize()
		return nil, fmt.Errorf("make the edit-field condition: %#x", hr)
	}
	return &addressBar{automation: a, isEdit: cond}, nil
}

// read is the text of the window's address bar, or "" when there is none or it cannot be read.
func (b *addressBar) read(w windows.HWND) string {
	var window, field *comObject
	if call(b.automation, slotElementFromHandle, uintptr(w), uintptr(unsafe.Pointer(&window))) != 0 || window == nil {
		return ""
	}
	defer release(window)
	if call(window, slotFindFirst, treeDescendants, uintptr(unsafe.Pointer(b.isEdit)), uintptr(unsafe.Pointer(&field))) != 0 || field == nil {
		return ""
	}
	defer release(field)
	var v variant
	if call(field, slotGetCurrentPropertyValue, propValueValue, uintptr(unsafe.Pointer(&v))) != 0 {
		return ""
	}
	defer variantClear.Call(uintptr(unsafe.Pointer(&v)))
	if v.vt != vtBSTR || v.val == 0 {
		return ""
	}
	return windows.UTF16PtrToString(*(**uint16)(unsafe.Pointer(&v.val))) // the value is a BSTR: a pointer to UTF-16
}

func (b *addressBar) close() {
	release(b.isEdit)
	release(b.automation)
	windows.CoUninitialize()
}
