package settingsui

import (
	"encoding/json"

	"omni/internal/config"
)

// toolsView fills the page's Tools section.
func (p *OmniAI) toolsView(v *OmniAIView, mcp bool) {
	if p.o.Tools != nil {
		v.Tools = p.o.Tools()
	}
	v.MCP, v.ScriptsFolder = mcp, p.o.ScriptsDir
	snippet, _ := json.MarshalIndent(map[string]any{"mcpServers": map[string]any{"omni": map[string]any{"command": p.o.Self, "args": []string{"mcp"}}}}, "", "  ")
	v.MCPConfig = string(snippet)
}

// SetTool gives the model a group of tools, or takes it away.
func (p *OmniAI) SetTool(group string, on bool) string { return reason(p.o.SetTool(group, on)) }

// SetAlwaysRun lets a group's tools that change something run without asking first, or makes them ask.
func (p *OmniAI) SetAlwaysRun(group string, on bool) string {
	return reason(p.o.SetAlwaysRun(group, on))
}

// SetMCP lets other AI apps on this computer use the tools that only read, or stops them.
func (p *OmniAI) SetMCP(on bool) string {
	return reason(p.o.Store.Update(func(c *config.Config) { c.Chat.MCP = on }))
}

// OpenScriptsFolder shows the scripts folder.
func (p *OmniAI) OpenScriptsFolder() string { return reason(p.o.OpenFolder(p.o.ScriptsDir)) }
