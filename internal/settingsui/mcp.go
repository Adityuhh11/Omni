package settingsui

import (
	"encoding/json"
	"slices"

	"omni/internal/chat"
	"omni/internal/config"
	"omni/internal/mcpconnect"
	"omni/internal/settings"
)

// The AI apps page: whether other AI apps on this computer may reach Omni over MCP (internal/mcp), and which groups of tools
// they get. Its switches are apart from Omni AI's (docs/decisions/0012-mcp-own-switches.md).

// MCPOptions are the parts of the program the page needs; main supplies them.
type MCPOptions struct {
	Store *settings.Store
	// Tools are all the local tools there are, as aitools builds them; the page shows those that only read.
	Tools func() []chat.Local
	// Self is this program, which an AI app starts as `<Self> mcp`.
	Self string
	// Apps, Connect and Disconnect are mcpconnect's; the page's tests give it folders of their own.
	Apps       func(self string) []mcpconnect.App
	Connect    func(id, self string) error
	Disconnect func(id string) error
}

// MCP is bound to the AI apps page of the settings window. Methods that change something return the reason they were refused,
// or "" when it worked.
type MCP struct{ o MCPOptions }

// NewMCP returns the page's service.
func NewMCP(o MCPOptions) *MCP { return &MCP{o: o} }

// MCPView is what the page shows.
type MCPView struct {
	// Enabled is the server's switch; Config is what to add to the other app's MCP settings.
	Enabled bool
	Config  string
	// Self is the path an AI app starts, for the one command line the page shows.
	Self string
	// Apps are the AI apps found on this computer that Omni can be added to with a button.
	Apps   []mcpconnect.App
	Groups []MCPGroup
}

// MCPGroup is a group of tools with the switch that offers it to other apps; only its tools that read are listed.
type MCPGroup struct {
	Group string
	Title string
	On    bool
	Tools []MCPTool
}

// MCPTool is one tool as the other app is shown it.
type MCPTool struct {
	Name        string
	Description string
}

// View returns the page; the window asks again every second or two, since scripts and tools come and go.
func (p *MCP) View() MCPView {
	cfg := p.o.Store.Get().Chat
	snippet, _ := json.MarshalIndent(map[string]any{"mcpServers": map[string]any{"omni": map[string]any{"command": p.o.Self, "args": []string{"mcp"}}}}, "", "  ")
	v := MCPView{Enabled: cfg.MCP, Config: string(snippet), Self: p.o.Self, Apps: p.o.Apps(p.o.Self)}
	for _, l := range p.o.Tools() {
		if l.Writes {
			continue
		}
		i := slices.IndexFunc(v.Groups, func(g MCPGroup) bool { return g.Group == l.Group })
		if i < 0 {
			v.Groups = append(v.Groups, MCPGroup{Group: l.Group, Title: l.GroupTitle, On: slices.Contains(cfg.MCPTools, l.Group)})
			i = len(v.Groups) - 1
		}
		def := l.ForApps().Function
		v.Groups[i].Tools = append(v.Groups[i].Tools, MCPTool{Name: def.Name, Description: def.Description})
	}
	return v
}

// SetEnabled lets other AI apps on this computer reach Omni, or stops them.
func (p *MCP) SetEnabled(on bool) string {
	return reason(p.o.Store.Update(func(c *config.Config) { c.Chat.MCP = on }))
}

// Connect adds Omni to the settings of the AI app with this id, on the person's press of its button.
func (p *MCP) Connect(id string) string { return reason(p.o.Connect(id, p.o.Self)) }

// Disconnect removes Omni from that app's settings.
func (p *MCP) Disconnect(id string) string { return reason(p.o.Disconnect(id)) }

// SetTool gives other AI apps a group of tools, or takes it away.
func (p *MCP) SetTool(group string, on bool) string {
	return reason(p.o.Store.Update(func(c *config.Config) {
		c.Chat.MCPTools = slices.DeleteFunc(slices.Clone(c.Chat.MCPTools), func(g string) bool { return g == group })
		if on {
			c.Chat.MCPTools = append(c.Chat.MCPTools, group)
		}
	}))
}
