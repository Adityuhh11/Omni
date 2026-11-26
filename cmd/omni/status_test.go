package main

import (
	"testing"

	"omni/internal/indexing"
	"omni/internal/sidecar"
)

func TestTheTrayStatusLineNamesTheMostUrgentThing(t *testing.T) {
	ready := sidecar.Status{State: sidecar.StateReady}
	done := indexing.Status{State: indexing.UpToDate, Files: 2, Chunks: 9}
	tests := []struct {
		name   string
		hotkey string
		engine sidecar.Status
		held   bool
		asleep bool // the engine starts when file search is used
		want   string
	}{
		{"a taken hotkey beats everything", "Alt+Space is taken", ready, true, false, "Alt+Space is taken"},
		{"off", "", sidecar.Status{State: sidecar.StateOff}, false, false, "File search is off. Open Settings to turn it on."},
		{"asleep until a file search", "", sidecar.Status{State: sidecar.StateOff}, false, true, "File search starts when you search for a file."},
		{"held after a first download", "", ready, true, false, "File search is ready. Open Settings to choose what to index."},
		{"not held", "", ready, false, false, "Up to date: 9 chunks in 2 files"},
	}
	for _, tt := range tests {
		if got := statusLine(tt.hotkey, tt.engine, done, tt.held, tt.asleep); got != tt.want {
			t.Errorf("%s: %q, want %q", tt.name, got, tt.want)
		}
	}
}
