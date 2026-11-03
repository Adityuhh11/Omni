package system

import (
	"fmt"
	"os/exec"
	"syscall"
)

// Commands are the rows the bar offers on Windows. Sleep and Hibernate are offered only when this computer can do them (init).
var Commands = []Command{
	{ID: "sys:lock", Title: "Lock the computer"},
	{ID: "sys:displayoff", Title: "Turn off the display"},
	{ID: "sys:signout", Title: "Sign out", Confirm: true},
	{ID: "sys:restart", Title: "Restart", Confirm: true},
	{ID: "sys:shutdown", Title: "Shut down", Confirm: true},
	{ID: "sys:taskmgr", Title: "Task Manager"},
	{ID: "sys:recycle", Title: "Empty the Recycle Bin"},
	{ID: "sys:winsettings", Title: "Windows Settings"},
}

// program is an action that starts a program with arguments.
type program struct {
	exe  string
	args []string
}

// programs are the actions that are just a command line. shutdown.exe holds the privileges it needs itself.
var programs = map[string]program{
	"sys:lock":        {"rundll32.exe", []string{"user32.dll,LockWorkStation"}},
	"sys:signout":     {"shutdown.exe", []string{"/l"}},
	"sys:restart":     {"shutdown.exe", []string{"/r", "/t", "0"}},
	"sys:shutdown":    {"shutdown.exe", []string{"/s", "/t", "0"}},
	"sys:hibernate":   {"shutdown.exe", []string{"/h"}},
	"sys:taskmgr":     {"taskmgr.exe", nil},
	"sys:winsettings": {"explorer.exe", []string{"ms-settings:"}},
}

func init() {
	for id, p := range programs {
		actions[id] = func() error { return start(p) }
	}
	actions["sys:recycle"] = emptyRecycleBin
	actions["sys:displayoff"] = displayOff

	c, err := powerCapabilities()
	if err != nil {
		return // no Sleep or Hibernate row: better none than one that does the other
	}
	if c.s3 || c.modernStandby {
		actions["sys:sleep"] = func() error { return sleep(c) }
		Commands = append(Commands, Command{ID: "sys:sleep", Title: "Sleep"})
	}
	if c.hibernate {
		Commands = append(Commands, Command{ID: "sys:hibernate", Title: "Hibernate"})
	}
}

func start(p program) error {
	cmd := exec.Command(p.exe, p.args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", p.exe, err)
	}
	return cmd.Process.Release()
}

// emptyRecycleBin asks Windows to empty the bin; with no flags it asks the person to confirm first.
func emptyRecycleBin() error {
	proc := syscall.NewLazyDLL("shell32.dll").NewProc("SHEmptyRecycleBinW")
	r, _, _ := proc.Call(0, 0, 0)
	if r != 0 && r != 0x8000FFFF { // 0 is S_OK; the second is what Windows answers when the bin is already empty
		return fmt.Errorf("empty the Recycle Bin: Windows answered %#x", r)
	}
	return nil
}

// Beep plays the system's notification sound (a finished timer). It makes no sound on a computer whose sounds are off, which is right.
func Beep() {
	const iconAsterisk = 0x40
	_, _, _ = user32.NewProc("MessageBeep").Call(iconAsterisk)
}
