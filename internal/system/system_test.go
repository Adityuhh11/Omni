package system

import "testing"

func TestEveryOfferedActionIsKnownAndAnUnknownOneIsRefused(t *testing.T) {
	if len(Commands) == 0 {
		t.Fatal("this system offers no actions")
	}
	for _, c := range Commands {
		if _, ok := actions[c.ID]; !ok {
			t.Errorf("%s has no action", c.ID)
		}
	}
	if err := Run("sys:nothing"); err == nil {
		t.Error("an unknown action ran")
	}
}

func TestEndingTheSessionAsksFirst(t *testing.T) {
	ends := map[string]bool{"sys:shutdown": true, "sys:restart": true, "sys:signout": true}
	for _, c := range Commands {
		if c.Confirm != ends[c.ID] {
			t.Errorf("%s: confirm = %v", c.ID, c.Confirm)
		}
		delete(ends, c.ID)
	}
	if len(ends) > 0 {
		t.Errorf("not offered: %v", ends)
	}
}
