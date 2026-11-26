package notes

import "errors"

// errUnavailable is what every call answers when the notes folder could not be opened at start: the rest of the program does not
// depend on it.
var errUnavailable = errors.New("notes are not available: the notes folder could not be opened (see the log)")

// Service is the store as the notes window uses it; its exported methods become TypeScript functions.
type Service struct {
	store *Store
	w     Windows
}

// Windows are what the notes' two windows ask of the program; any of them may be nil.
type Windows struct {
	// SetOnTop keeps the note window above other windows, or lets it sink.
	SetOnTop func(bool)
	// OpenNote shows the note window on the note with this id ("new" for an empty one).
	OpenNote func(id string)
	// OpenHistory shows the window that lists every note.
	OpenHistory func()
}

// NewService returns the service. A nil store is allowed: every call then says that notes are not available.
func NewService(store *Store, windows Windows) *Service {
	return &Service{store: store, w: windows}
}

// List returns the notes that contain every word of query, newest change first; an empty query returns all of them.
func (s *Service) List(query string) ([]Note, error) {
	if s.store == nil {
		return nil, errUnavailable
	}
	return s.store.Search(query, 0)
}

// Get returns a note with its text.
func (s *Service) Get(id string) (Note, error) {
	if s.store == nil {
		return Note{}, errUnavailable
	}
	return s.store.Get(id)
}

// Save writes a note, or makes a new one when id is empty, and returns it as saved.
func (s *Service) Save(id, text string) (Note, error) {
	if s.store == nil {
		return Note{}, errUnavailable
	}
	return s.store.Save(id, text)
}

// Delete removes a note.
func (s *Service) Delete(id string) error {
	if s.store == nil {
		return errUnavailable
	}
	return s.store.Delete(id)
}

// SetOnTop keeps the notes window above other windows (on) or lets other windows cover it (off).
func (s *Service) SetOnTop(on bool) {
	if s.w.SetOnTop != nil {
		s.w.SetOnTop(on)
	}
}

// OpenNote shows the note window on a note ("new" for an empty one): what the history window does when a note is picked.
func (s *Service) OpenNote(id string) {
	if s.w.OpenNote != nil {
		s.w.OpenNote(id)
	}
}

// OpenHistory shows the window with every note in it: the note window's History button.
func (s *Service) OpenHistory() {
	if s.w.OpenHistory != nil {
		s.w.OpenHistory()
	}
}
