package system

import (
	"os/exec"
	"strings"
	"testing"
)

func TestTheCommandLinesAreWhatWindowsExpects(t *testing.T) {
	want := map[string]string{
		"sys:lock":      "rundll32.exe user32.dll,LockWorkStation",
		"sys:shutdown":  "shutdown.exe /s /t 0",
		"sys:restart":   "shutdown.exe /r /t 0",
		"sys:signout":   "shutdown.exe /l",
		"sys:hibernate": "shutdown.exe /h",
	}
	for id, line := range want {
		p := programs[id]
		if got := strings.Join(append([]string{p.exe}, p.args...), " "); got != line {
			t.Errorf("%s = %q, want %q", id, got, line)
		}
	}
}

func TestTheCapabilitiesAreReadAtTheirOffsets(t *testing.T) {
	tests := []struct {
		name string
		set  []int
		want capabilities
	}{
		{"nothing", nil, capabilities{}},
		{"classic sleep", []int{offS3}, capabilities{s3: true}},
		{"hibernation needs its file", []int{offS4}, capabilities{}},
		{"hibernation", []int{offS4, offHiberFile}, capabilities{hibernate: true}},
		{"modern standby", []int{offAoAc, offS4, offHiberFile}, capabilities{hibernate: true, modernStandby: true}},
	}
	for _, tt := range tests {
		b := make([]byte, capabilitiesLen)
		for _, i := range tt.set {
			b[i] = 1
		}
		if got := parseCapabilities(b); got != tt.want {
			t.Errorf("%s: %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

// The real computer's answer agrees with powercfg about hibernation.
func TestThisComputersCapabilitiesMatchPowercfg(t *testing.T) {
	c, err := powerCapabilities()
	if err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("powercfg", "/a").Output()
	if err != nil {
		t.Skipf("powercfg: %v", err)
	}
	if !strings.Contains(string(out), "sleep states") {
		t.Skip("powercfg answers in another language")
	}
	available, _, _ := strings.Cut(string(out), "not available")
	if hib := strings.Contains(available, "Hibernate"); hib != c.hibernate {
		t.Errorf("powercfg says hibernate %v, GetPwrCapabilities %v", hib, c.hibernate)
	}
	if ms := strings.Contains(available, "S0 Low Power Idle"); ms != c.modernStandby {
		t.Errorf("powercfg says modern standby %v, GetPwrCapabilities %v", ms, c.modernStandby)
	}
}
