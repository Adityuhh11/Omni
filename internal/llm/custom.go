package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// CustomEndpoint is the Endpoint of a server somebody else runs (LM Studio, Ollama, a cloud service) from the address the person
// pasted. Services give their address with a version ("https://api.openai.com/v1"); the chat adds its own "/v1/...", so a
// trailing "/v1" and slashes are dropped.
func CustomEndpoint(rawURL, model, key string) Endpoint {
	u := strings.TrimSuffix(strings.TrimRight(strings.TrimSpace(rawURL), "/"), "/v1")
	return Endpoint{URL: u, Key: key, Model: strings.TrimSpace(model), Device: "custom"}
}

// Loopback reports whether an address is on this computer, where what is asked does not leave it.
func Loopback(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := u.Hostname()
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// Switch is the model Omni AI asks: the bundled server, or, when Custom says so, the one the person pointed it at. It decides at
// each call, so a changed setting applies to the next question.
type Switch struct {
	Bundled *Server
	// Custom returns the address and model name of the server to use instead of the bundled model, or false for the bundled one.
	Custom func() (url, model string, ok bool)
	// Key returns that server's API key ("" if it needs none). It is asked only when a question is sent, since reading the system
	// credential store is not free and Status is asked every second by the settings window.
	Key func() string
}

// Ensure returns the endpoint to ask.
func (s Switch) Ensure(ctx context.Context) (Endpoint, error) {
	if url, model, ok := s.Custom(); ok {
		return CustomEndpoint(url, model, s.Key()), nil
	}
	return s.Bundled.Ensure(ctx)
}

// Hold keeps the bundled model loaded; another server is not Omni's to keep.
func (s Switch) Hold() func() {
	if _, _, ok := s.Custom(); ok {
		return func() {}
	}
	return s.Bundled.Hold()
}

// UseProcessor moves the bundled model off the graphics card.
func (s Switch) UseProcessor() { s.Bundled.UseProcessor() }

// Status is the bundled model's, or ready for a server that is not Omni's to start or stop.
func (s Switch) Status() Status {
	if _, _, ok := s.Custom(); ok {
		return Status{State: Ready, Device: "custom"}
	}
	return s.Bundled.Status()
}

// ListModels asks a server which models it offers (GET /v1/models), by the ids it gives them.
func ListModels(ctx context.Context, ep Endpoint) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ep.URL+"/v1/models", nil)
	if err != nil {
		return nil, fmt.Errorf("ask %s: %w", ep.URL, err)
	}
	if ep.Key != "" {
		req.Header.Set("Authorization", "Bearer "+ep.Key)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("reach %s: %w", ep.URL, err)
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return nil, fmt.Errorf("%s did not accept the key (%d)", ep.URL, resp.StatusCode)
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("%s answered %d", ep.URL, resp.StatusCode)
	}
	var out struct{ Data []struct{ ID string } }
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&out); err != nil {
		return nil, fmt.Errorf("%s did not answer like an OpenAI-style server: %w", ep.URL, err)
	}
	ids := make([]string, 0, len(out.Data))
	for _, m := range out.Data {
		ids = append(ids, m.ID)
	}
	sort.Strings(ids)
	return ids, nil
}
