package system

import (
	"fmt"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	powrprof = windows.NewLazySystemDLL("powrprof.dll")
	user32   = windows.NewLazySystemDLL("user32.dll")
)

// capabilities is what this computer can do with its power, from GetPwrCapabilities.
type capabilities struct {
	s3, hibernate, modernStandby bool
}

// Offsets into SYSTEM_POWER_CAPABILITIES: a run of one-byte BOOLEANs, in the order winnt.h declares them.
const (
	offS3           = 5
	offS4           = 6
	offHiberFile    = 8
	offAoAc         = 20 // "always on, always connected": Modern Standby, where S3 is switched off
	capabilitiesLen = 76
)

func parseCapabilities(b []byte) capabilities {
	return capabilities{s3: b[offS3] != 0, hibernate: b[offS4] != 0 && b[offHiberFile] != 0, modernStandby: b[offAoAc] != 0}
}

func powerCapabilities() (capabilities, error) {
	buf := make([]byte, capabilitiesLen)
	if r, _, err := powrprof.NewProc("GetPwrCapabilities").Call(uintptr(unsafe.Pointer(&buf[0]))); r == 0 {
		return capabilities{}, fmt.Errorf("read the power capabilities: %w", err)
	}
	return parseCapabilities(buf), nil
}

// sleep puts the computer to sleep the way it sleeps. On a Modern Standby computer SetSuspendState is refused (ERROR_NOT_SUPPORTED);
// there the Start menu turns the screen off, and the system goes into standby from there, so Omni does the same.
func sleep(c capabilities) error {
	if c.modernStandby {
		return displayOff()
	}
	if err := enableShutdownPrivilege(); err != nil {
		return err
	}
	// Called directly, not through rundll32: rundll32 hands its own window handle in as the first argument, which reads as "hibernate".
	if r, _, err := powrprof.NewProc("SetSuspendState").Call(0, 0, 0); r == 0 {
		return fmt.Errorf("sleep: %w", err)
	}
	return nil
}

// displayOff turns the screens off. It waits a moment first: the Enter that chose the row is still being let go, and a key event
// would wake the screens again at once.
func displayOff() error {
	const hwndBroadcast, wmSyscommand, scMonitorpower, off = 0xFFFF, 0x0112, 0xF170, 2
	time.Sleep(500 * time.Millisecond)
	if r, _, err := user32.NewProc("PostMessageW").Call(hwndBroadcast, wmSyscommand, scMonitorpower, off); r == 0 {
		return fmt.Errorf("turn the display off: %w", err)
	}
	return nil
}

// enableShutdownPrivilege switches on the right to suspend, which every account holds but a process starts with switched off.
func enableShutdownPrivilege() error {
	var token windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_ADJUST_PRIVILEGES|windows.TOKEN_QUERY, &token); err != nil {
		return fmt.Errorf("open the process token: %w", err)
	}
	defer token.Close()
	var luid windows.LUID
	name, err := windows.UTF16PtrFromString("SeShutdownPrivilege")
	if err != nil {
		return fmt.Errorf("look up the shutdown privilege: %w", err)
	}
	if err := windows.LookupPrivilegeValue(nil, name, &luid); err != nil {
		return fmt.Errorf("look up the shutdown privilege: %w", err)
	}
	privileges := windows.Tokenprivileges{PrivilegeCount: 1, Privileges: [1]windows.LUIDAndAttributes{{Luid: luid, Attributes: windows.SE_PRIVILEGE_ENABLED}}}
	if err := windows.AdjustTokenPrivileges(token, false, &privileges, 0, nil, nil); err != nil {
		return fmt.Errorf("enable the shutdown privilege: %w", err)
	}
	return nil
}
