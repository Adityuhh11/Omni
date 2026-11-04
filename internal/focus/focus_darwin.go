package focus

import (
	"fmt"
	"os/exec"
	"strings"
)

// chromium are the browsers that answer AppleScript's "URL of active tab of front window", by the name System Events gives them.
// Firefox and its relatives have no AppleScript address: only the program rules apply to them.
var chromium = map[string]bool{"Google Chrome": true, "Microsoft Edge": true, "Brave Browser": true, "Arc": true, "Vivaldi": true, "Opera": true}

// macProbe asks System Events which app is in front and, for a browser, asks the browser for its address. macOS confirms each
// of those once ("Omni wants to control ..."); declined, that app is read by name only.
// ponytail: one osascript per look (two for a browser), once a second and only during a focus period.
type macProbe struct{}

func newProbe() (probe, error) { return macProbe{}, nil }

func (macProbe) front() (front, error) {
	app, err := osascript(`tell application "System Events" to get name of first application process whose frontmost is true`)
	if err != nil {
		return front{}, err
	}
	f := front{app: app}
	switch {
	case chromium[app]:
		f.url, _ = osascript(`tell application "` + app + `" to get URL of active tab of front window`)
	case app == "Safari":
		f.url, _ = osascript(`tell application "Safari" to get URL of front document`)
	}
	return f, nil
}

// hide hides the app, as ⌘H does: its windows come back when it is chosen again.
func (macProbe) hide(f front) error {
	_, err := osascript(`tell application "System Events" to set visible of application process "` + f.app + `" to false`)
	return err
}

func (macProbe) close() {}

func osascript(script string) (string, error) {
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return "", fmt.Errorf("osascript: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
