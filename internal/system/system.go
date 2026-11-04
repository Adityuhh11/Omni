// Package system runs the actions on the computer the bar offers beside launching programs: lock, sleep, hibernate, the display, signing
// out, restarting and shutting down, the process monitor, the bin and the system settings. A command that ends the session is marked
// Confirm, and the bar asks for a second Enter before it runs.
//
// What each system can do, and what it calls it, is in system_windows.go and system_darwin.go.
package system

import "fmt"

// Command is an action the bar offers as a row: its id ("sys:lock", which the bar's history keeps), what the row says, and whether it
// closes the person's programs (shut down, restart, sign out), so that one mistyped Enter cannot do it.
type Command struct {
	ID, Title string
	Confirm   bool
}

// Run does the action with this id.
func Run(id string) error {
	if run, ok := actions[id]; ok {
		return run()
	}
	return fmt.Errorf("unknown system action %q", id)
}

// actions are the things Run can do, filled in by each system's file.
var actions = map[string]func() error{}
