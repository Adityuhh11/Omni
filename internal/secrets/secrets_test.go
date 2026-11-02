package secrets

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

// TestRoundTrip uses the real store of the computer it runs on, under a name of its own that it deletes.
func TestRoundTrip(t *testing.T) {
	name := fmt.Sprintf("test-%d", time.Now().UnixNano())
	t.Cleanup(func() { Delete(name) })
	if _, err := Get(name); !errors.Is(err, ErrNotFound) {
		t.Fatalf("before: %v", err)
	}
	for _, v := range []string{"sk-first-1234", `sk-with "quotes" and \ and ünïcode`} {
		if err := Set(name, v); err != nil {
			t.Skipf("the credential store is not usable here: %v", err) // a CI runner without a signed-in keychain
		}
		if got, err := Get(name); err != nil || got != v {
			t.Fatalf("Get after Set(%q) = %q, %v", v, got, err)
		}
	}
	if err := Set(name, ""); err != nil { // empty deletes
		t.Fatal(err)
	}
	if _, err := Get(name); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete: %v", err)
	}
	if err := Delete(name); err != nil {
		t.Fatalf("deleting what is not there: %v", err)
	}
}
