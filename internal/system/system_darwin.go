package system

import (
	"fmt"
	"os"
	"os/exec"
)

// Commands are the rows the bar offers on a Mac. There is no Hibernate: a Mac decides that itself (pmset hibernatemode).
var Commands = []Command{
	{ID: "sys:lock", Title: "Lock the screen"},
	{ID: "sys:sleep", Title: "Sleep"},
	{ID: "sys:displayoff", Title: "Turn off the display"},
	{ID: "sys:signout", Title: "Log out", Confirm: true},
	{ID: "sys:restart", Title: "Restart", Confirm: true},
	{ID: "sys:shutdown", Title: "Shut down", Confirm: true},
	{ID: "sys:activitymonitor", Title: "Activity Monitor"},
	{ID: "sys:recycle", Title: "Empty the Trash"},
	{ID: "sys:systemsettings", Title: "System Settings"},
}

// systemEvents asks System Events to end the session. The bar has already asked for a second Enter, so no system dialog follows: System
// Events restarts and shuts down without one, and «event aevtrlgo» is the log out that skips it. Programs may still ask to save their work.
// The first time, macOS asks whether Omni may control System Events.
func systemEvents(verb string) func() error {
	return func() error { return run("osascript", "-e", `tell application "System Events" to `+verb) }
}

// cgSession is the helper that locks the session without asking for any permission. Apple has moved it before, so it is used when it is
// there; otherwise the display is put to sleep, which locks the Mac when it is set to ask for the password at once (the macOS default).
// (ponytail: verify on the macOS versions in use; if neither locks reliably the row should go.)
const cgSession = "/System/Library/CoreServices/Menu Extras/User.menu/Contents/Resources/CGSession"

func lock() error {
	if _, err := os.Stat(cgSession); err == nil {
		return run(cgSession, "-suspend")
	}
	return run("pmset", "displaysleepnow")
}

func init() {
	actions["sys:lock"] = lock
	actions["sys:sleep"] = func() error { return run("pmset", "sleepnow") }
	actions["sys:displayoff"] = func() error { return run("pmset", "displaysleepnow") }
	actions["sys:signout"] = systemEvents("«event aevtrlgo»")
	actions["sys:restart"] = systemEvents("restart")
	actions["sys:shutdown"] = systemEvents("shut down")
	actions["sys:activitymonitor"] = func() error { return run("open", "-a", "Activity Monitor") }
	actions["sys:systemsettings"] = func() error { return run("open", "x-apple.systempreferences:") }
	// Finder asks the person to confirm, as the Recycle Bin does on Windows. The first time, macOS asks whether Omni may control Finder.
	actions["sys:recycle"] = func() error {
		return run("osascript", "-e", `tell application "Finder" to empty the trash`)
	}
}

// Beep plays the system's notification sound (a finished timer). It is silent on a Mac whose sound is muted, which is right.
func Beep() {
	_ = run("afplay", "/System/Library/Sounds/Glass.aiff")
}

// run starts a program and lets it go. It is waited for in the background: on a Mac a child that is never waited for stays in the process table
// as a zombie until the launcher ends.
func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", name, err)
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
