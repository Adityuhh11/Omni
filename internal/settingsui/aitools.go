package settingsui

// toolsView fills the page's Tools section.
func (p *OmniAI) toolsView(v *OmniAIView) {
	if p.o.Tools != nil {
		v.Tools = p.o.Tools()
	}
	v.ScriptsFolder = p.o.ScriptsDir
}

// SetTool gives the model a group of tools, or takes it away.
func (p *OmniAI) SetTool(group string, on bool) string { return reason(p.o.SetTool(group, on)) }

// SetAlwaysRun lets a group's tools that change something run without asking first, or makes them ask.
func (p *OmniAI) SetAlwaysRun(group string, on bool) string {
	return reason(p.o.SetAlwaysRun(group, on))
}

// OpenScriptsFolder shows the scripts folder.
func (p *OmniAI) OpenScriptsFolder() string { return reason(p.o.OpenFolder(p.o.ScriptsDir)) }
