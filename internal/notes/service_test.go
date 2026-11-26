package notes

import "testing"

func TestAServiceWithoutAFolderSaysSoInsteadOfCrashing(t *testing.T) {
	s := NewService(nil, Windows{})
	if _, err := s.List(""); err == nil {
		t.Error("List")
	}
	if _, err := s.Get("20261026-231500"); err == nil {
		t.Error("Get")
	}
	if _, err := s.Save("", "x"); err == nil {
		t.Error("Save")
	}
	if err := s.Delete("20261026-231500"); err == nil {
		t.Error("Delete")
	}
	s.SetOnTop(true) // nothing to move, nothing to fail
	s.OpenNote("new")
	s.OpenHistory()
}

func TestAServiceCarriesTheStoreAndTheWindowsSwitches(t *testing.T) {
	var onTop []bool
	var opened []string
	history := 0
	s := NewService(newStore(t), Windows{
		SetOnTop:    func(on bool) { onTop = append(onTop, on) },
		OpenNote:    func(id string) { opened = append(opened, id) },
		OpenHistory: func() { history++ },
	})
	n, err := s.Save("", "hello")
	if err != nil || n.ID == "" {
		t.Fatalf("Save = %+v, %v", n, err)
	}
	if list, err := s.List("hell"); err != nil || len(list) != 1 {
		t.Fatalf("List = %v, %v", list, err)
	}
	s.SetOnTop(false)
	if len(onTop) != 1 || onTop[0] {
		t.Errorf("on top calls %v", onTop)
	}
	s.OpenNote(n.ID)
	s.OpenHistory()
	if len(opened) != 1 || opened[0] != n.ID || history != 1 {
		t.Errorf("opened %v, history %d times", opened, history)
	}
}
