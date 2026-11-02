package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCustomEndpointDropsTheVersionAndSlashesTheChatAddsItself(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"https://api.openai.com/v1", "https://api.openai.com"},
		{" http://localhost:1234/v1/ ", "http://localhost:1234"},
		{"http://localhost:11434", "http://localhost:11434"},
		{"https://openrouter.ai/api/v1", "https://openrouter.ai/api"},
	} {
		if got := CustomEndpoint(c.in, " m ", "k"); got.URL != c.want || got.Model != "m" || got.Key != "k" {
			t.Errorf("CustomEndpoint(%q) = %+v, want %s", c.in, got, c.want)
		}
	}
}

func TestLoopbackIsOnlyThisComputer(t *testing.T) {
	for in, want := range map[string]bool{
		"http://localhost:1234/v1": true, "http://127.0.0.1:11434": true, "http://[::1]:8080": true,
		"https://api.openai.com/v1": false, "http://192.168.1.20:1234": false, "http://localhost.evil.test": false, "not a url": false,
	} {
		if got := Loopback(in); got != want {
			t.Errorf("Loopback(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestSwitchUsesTheCustomModelOnlyWhenAskedAndReadsTheKeyOnlyToAsk(t *testing.T) {
	custom, keyReads := false, 0
	s := Switch{
		Custom: func() (string, string, bool) { return "http://x/v1", "m", custom },
		Key:    func() string { keyReads++; return "k" },
	}
	custom = true
	if st := s.Status(); st.State != Ready || st.Device != "custom" {
		t.Fatalf("status = %+v", st)
	}
	s.Hold()() // does not touch the (nil) bundled server
	if keyReads != 0 {
		t.Fatalf("the key was read %d times without a question", keyReads)
	}
	ep, err := s.Ensure(context.Background())
	if err != nil || ep.URL != "http://x" || ep.Model != "m" || ep.Key != "k" || keyReads != 1 {
		t.Fatalf("Ensure = %+v, %v (key reads %d)", ep, err, keyReads)
	}
}

func TestListModelsReadsTheIdsAndSaysWhyItFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path != "/v1/models":
			w.WriteHeader(http.StatusNotFound)
		case r.Header.Get("Authorization") != "Bearer good":
			w.WriteHeader(http.StatusUnauthorized)
		default:
			fmt.Fprint(w, `{"data":[{"id":"zeta"},{"id":"alpha"}]}`)
		}
	}))
	defer srv.Close()
	ids, err := ListModels(context.Background(), CustomEndpoint(srv.URL+"/v1", "", "good"))
	if err != nil || len(ids) != 2 || ids[0] != "alpha" {
		t.Fatalf("ids = %v, %v", ids, err)
	}
	if _, err := ListModels(context.Background(), CustomEndpoint(srv.URL, "", "bad")); err == nil || !strings.Contains(err.Error(), "did not accept the key") {
		t.Fatalf("err = %v", err)
	}
	if _, err := ListModels(context.Background(), CustomEndpoint("http://127.0.0.1:1", "", "")); err == nil || !strings.Contains(err.Error(), "reach") {
		t.Fatalf("err = %v", err)
	}
}
