package settingsui

import (
	"context"
	"net/url"
	"strings"
	"time"

	"omni/internal/config"
	"omni/internal/llm"
)

// Using a model somebody else runs (LM Studio, Ollama, a cloud service) instead of the bundled one
// (docs/decisions/0014-bring-your-own-model.md). The address and the model's name are in config.toml; the key is in the system
// credential store, and the page is only told whether there is one.

// CustomView is that choice as the page shows it.
type CustomView struct {
	On         bool
	URL, Model string
	HasKey     bool
	Local      bool // the address is on this computer, so nothing asked leaves it
	Host       string
}

// CustomTest is the answer of a test of the connection: why it failed, or the models the server offers.
type CustomTest struct {
	Error  string
	Models []string
}

func (p *OmniAI) customView(cfg config.Chat) CustomView {
	v := CustomView{On: cfg.Custom, URL: cfg.CustomURL, Model: cfg.CustomModel, Local: llm.Loopback(cfg.CustomURL)}
	if u, err := url.Parse(cfg.CustomURL); err == nil {
		v.Host = u.Host
	}
	if p.o.CustomKey != nil {
		v.HasKey = p.o.CustomKey() != ""
	}
	return v
}

// UseCustom saves the address and the model's name and makes Omni AI ask that model. It turns Omni AI on, since there is nothing
// to download, and frees the memory the bundled model held.
func (p *OmniAI) UseCustom(address, model string) string {
	address, model = strings.TrimSpace(address), strings.TrimSpace(model)
	msg := reason(p.o.Store.Update(func(c *config.Config) {
		c.Chat.Custom, c.Chat.CustomURL, c.Chat.CustomModel, c.Chat.Enabled = true, address, model, true
	}))
	if msg == "" {
		p.o.Model.Unload()
	}
	return msg
}

// UseBundled goes back to the model Omni downloads. Omni AI stays on only if that model is installed.
func (p *OmniAI) UseBundled() string {
	installed := p.o.Setup.State().Installed
	return reason(p.o.Store.Update(func(c *config.Config) {
		c.Chat.Custom = false
		c.Chat.Enabled = c.Chat.Enabled && installed
	}))
}

// SetCustomKey keeps the API key in the system credential store; an empty key removes it.
func (p *OmniAI) SetCustomKey(key string) string {
	if p.o.SetCustomKey == nil {
		return "this computer has no credential store Omni can use"
	}
	if err := p.o.SetCustomKey(strings.TrimSpace(key)); err != nil {
		return err.Error()
	}
	return ""
}

// TestCustom asks the server at address which models it has, with the key typed (or, if none was, the one saved). The model's
// name is not needed to test, so the page can offer the server's own list.
func (p *OmniAI) TestCustom(address, key string) CustomTest {
	address = strings.TrimSpace(address)
	if u, err := url.Parse(address); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return CustomTest{Error: "That is not an http or https address."}
	}
	if key = strings.TrimSpace(key); key == "" && p.o.CustomKey != nil {
		key = p.o.CustomKey()
	}
	list := p.o.ListModels
	if list == nil {
		list = llm.ListModels
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	models, err := list(ctx, llm.CustomEndpoint(address, "", key))
	if err != nil {
		return CustomTest{Error: err.Error()}
	}
	return CustomTest{Models: models}
}
