// Package atlas is the map of the index for its window: every chunk as a point in 3D, laid out by the engine. It installs the
// engine's optional map extra the first time the map is built, and relays the map's points, a point's chunk and where a query
// lands.
package atlas

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"omni/internal/sidecar"
)

// Engine is the part of the engine client the map needs.
type Engine interface {
	MapStatus(ctx context.Context) (sidecar.MapStatus, error)
	BuildMap(ctx context.Context) (sidecar.MapStatus, error)
	Map(ctx context.Context) (sidecar.MapPoints, error)
	MapPoint(ctx context.Context, index int) (sidecar.MapPoint, error)
	Locate(ctx context.Context, query string) (sidecar.Located, error)
}

// Options are what the service is built from.
type Options struct {
	Engine  Engine
	Running func() bool                       // the engine answers (it may still be loading its model)
	Install func(ctx context.Context) error   // adds the engine's map extra
	Open    func(path string, line int) error // opens a file, at a line when it has one
	Show    func()                            // brings the map's window up
}

// View is how far the map is, for its window.
type View struct {
	State   string // off, none, installing, building, ready, failed, missing
	Message string
	Count   int
	Stale   bool // the index has changed since the map was built
}

const (
	callTimeout = 10 * time.Second
	// installTimeout covers downloading UMAP and the libraries it needs (about 150 MB).
	installTimeout = 15 * time.Minute
)

// Service is bound to the map's page.
type Service struct {
	o Options

	mu         sync.Mutex
	busy       bool   // a Build is under way
	installing bool   // and it is installing the extra
	problem    string // why the last Build failed
}

// New returns the service.
func New(o Options) *Service { return &Service{o: o} }

// Show opens the map's window (the settings page has the button).
func (s *Service) Show() { s.o.Show() }

// State says how far the map is.
func (s *Service) State() View {
	if !s.o.Running() {
		return View{State: "off", Message: "File search is not running. Turn it on in Settings to see the map."}
	}
	s.mu.Lock()
	installing, problem := s.installing, s.problem
	s.mu.Unlock()
	if installing {
		return View{State: "installing", Message: "Installing the map's layout library (about 150 MB)."}
	}
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()
	st, err := s.o.Engine.MapStatus(ctx)
	if err != nil {
		return View{State: "failed", Message: "The engine did not answer: " + err.Error()}
	}
	if problem != "" && st.State != "building" && st.State != "ready" {
		return View{State: "failed", Message: problem}
	}
	return View{State: st.State, Message: st.Message, Count: st.Count, Stale: st.Stale}
}

// Build lays the index out again, first installing the map's extra if the engine lacks it. It returns at once; State follows it.
func (s *Service) Build() {
	s.mu.Lock()
	if s.busy {
		s.mu.Unlock()
		return
	}
	s.busy, s.problem = true, ""
	s.mu.Unlock()
	go s.build()
}

func (s *Service) build() {
	finish := func(problem string) {
		s.mu.Lock()
		s.busy, s.installing, s.problem = false, false, problem
		s.mu.Unlock()
	}
	ctx, cancel := context.WithTimeout(context.Background(), installTimeout)
	defer cancel()
	st, err := s.o.Engine.MapStatus(ctx)
	if err == nil && st.State == "missing" {
		s.mu.Lock()
		s.installing = true
		s.mu.Unlock()
		err = s.o.Install(ctx)
	}
	if err == nil {
		_, err = s.o.Engine.BuildMap(ctx)
	}
	if err != nil {
		finish("The map could not be built: " + err.Error())
		return
	}
	finish("")
}

// Points returns the whole map.
func (s *Service) Points() (sidecar.MapPoints, error) {
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()
	return s.o.Engine.Map(ctx)
}

// Point returns the chunk behind point index.
func (s *Service) Point(index int) (sidecar.MapPoint, error) {
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()
	return s.o.Engine.MapPoint(ctx, index)
}

// Locate finds where query lands on the map.
func (s *Service) Locate(query string) (sidecar.Located, error) {
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()
	return s.o.Engine.Locate(ctx, query)
}

// Open opens the file of point index, at its first line when the chunk is a range of lines.
func (s *Service) Open(index int) error {
	p, err := s.Point(index)
	if err != nil {
		return err
	}
	if p.Root == "" {
		return errors.New("that folder is no longer searched")
	}
	if err := s.o.Open(filepath.Join(p.Root, filepath.FromSlash(p.URI)), firstLine(p.Locator)); err != nil {
		return fmt.Errorf("open %s: %w", p.URI, err)
	}
	return nil
}

// firstLine reads a text chunk's locator ({"lines":[12,40]}); anything else has no line.
func firstLine(locator string) int {
	var at struct{ Lines []int }
	if json.Unmarshal([]byte(locator), &at) != nil || len(at.Lines) == 0 {
		return 0
	}
	return at.Lines[0]
}
