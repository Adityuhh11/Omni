package atlas

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"omni/internal/sidecar"
)

type fakeEngine struct {
	mu     sync.Mutex
	state  string
	builds int
	point  sidecar.MapPoint
}

func (f *fakeEngine) MapStatus(context.Context) (sidecar.MapStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return sidecar.MapStatus{State: f.state}, nil
}

func (f *fakeEngine) BuildMap(context.Context) (sidecar.MapStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.builds++
	f.state = "building"
	return sidecar.MapStatus{State: f.state}, nil
}

func (f *fakeEngine) Map(context.Context) (sidecar.MapPoints, error) { return sidecar.MapPoints{}, nil }

func (f *fakeEngine) MapPoint(context.Context, int) (sidecar.MapPoint, error) { return f.point, nil }

func (f *fakeEngine) Locate(context.Context, string) (sidecar.Located, error) {
	return sidecar.Located{}, nil
}

func settle(t *testing.T, s *Service) View {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		busy := s.busy
		s.mu.Unlock()
		if !busy {
			return s.State()
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("the build never finished")
	return View{}
}

func TestBuildInstallsTheExtraOnlyWhenTheEngineLacksIt(t *testing.T) {
	for _, tc := range []struct {
		name, state string
		installErr  error
		installs    int
		builds      int
		want        string
	}{
		{"installed", "none", nil, 0, 1, "building"},
		{"missing", "missing", nil, 1, 1, "building"},
		{"install fails", "missing", errors.New("no network"), 1, 0, "failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			engine := &fakeEngine{state: tc.state}
			installs := 0
			s := New(Options{Engine: engine, Running: func() bool { return true },
				Install: func(context.Context) error { installs++; return tc.installErr }})
			s.Build()
			v := settle(t, s)
			if installs != tc.installs || engine.builds != tc.builds || v.State != tc.want {
				t.Errorf("installs %d builds %d state %q (%s); want %d %d %q", installs, engine.builds, v.State, v.Message,
					tc.installs, tc.builds, tc.want)
			}
		})
	}
}

func TestTheMapIsOffWhileTheEngineIsNot(t *testing.T) {
	s := New(Options{Engine: &fakeEngine{state: "ready"}, Running: func() bool { return false }})
	if v := s.State(); v.State != "off" {
		t.Errorf("state %q", v.State)
	}
}

func TestOpenGoesToTheFileAndTheFirstLineOfItsChunk(t *testing.T) {
	root := filepath.Join(t.TempDir(), "docs")
	for _, tc := range []struct {
		locator  string
		wantLine int
	}{
		{`{"lines":[12,40]}`, 12},
		{`{"page":3}`, 0},
		{``, 0},
	} {
		engine := &fakeEngine{point: sidecar.MapPoint{Root: root, URI: "a/b.md", Locator: tc.locator}}
		var path string
		var line int
		s := New(Options{Engine: engine, Running: func() bool { return true },
			Open: func(p string, l int) error { path, line = p, l; return nil }})
		if err := s.Open(0); err != nil {
			t.Fatal(err)
		}
		if want := filepath.Join(root, "a", "b.md"); path != want || line != tc.wantLine {
			t.Errorf("%s: opened %s at %d, want %s at %d", tc.locator, path, line, want, tc.wantLine)
		}
	}
	gone := New(Options{Engine: &fakeEngine{point: sidecar.MapPoint{URI: "x"}}, Running: func() bool { return true }})
	if err := gone.Open(0); err == nil {
		t.Error("a point of a folder no longer searched opened")
	}
}
