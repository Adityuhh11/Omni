package settingsui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"omni/internal/scripts"
)

// toolsView fills the page's Tools section.
func (p *OmniAI) toolsView(v *OmniAIView) {
	if p.o.Tools != nil {
		v.Tools = p.o.Tools()
	}
	v.ScriptsFolder = p.o.ScriptsDir
	v.ScriptLanguages = scripts.Languages
}

// SetTool gives the model a group of tools, or takes it away.
func (p *OmniAI) SetTool(group string, on bool) string { return reason(p.o.SetTool(group, on)) }

// SetAlwaysRun lets a group's tools that change something run without asking first, or makes them ask.
func (p *OmniAI) SetAlwaysRun(group string, on bool) string {
	return reason(p.o.SetAlwaysRun(group, on))
}

// OpenScriptsFolder shows the scripts folder.
func (p *OmniAI) OpenScriptsFolder() string { return reason(p.o.OpenFolder(p.o.ScriptsDir)) }

// AddScript writes a script the person typed or loaded into the scripts folder, under the name they gave it.
func (p *OmniAI) AddScript(title, ext, output, body string) string {
	if p.o.AddScript == nil {
		return "the scripts folder could not be opened"
	}
	_, err := p.o.AddScript(scripts.New{Title: title, Ext: ext, Output: output, Body: body})
	return reason(err)
}

// ScriptFile is a file chosen for the Add dialog: its text, and the name and language it suggests.
type ScriptFile struct {
	Name, Ext, Text string
	Error           string
}

// LoadScript asks for a file and returns what the Add dialog fills its box with; nothing when the person cancels.
func (p *OmniAI) LoadScript() ScriptFile {
	if p.o.PickScript == nil {
		return ScriptFile{Error: "choosing a file is not available"}
	}
	path, err := p.o.PickScript()
	if err != nil {
		return ScriptFile{Error: err.Error()}
	}
	if path == "" {
		return ScriptFile{}
	}
	if info, err := os.Stat(path); err != nil || info.Size() > scripts.MaxBody {
		return ScriptFile{Error: fmt.Sprintf("%s is not a script Omni can add: it must be a text file under %d KB", filepath.Base(path), scripts.MaxBody>>10)}
	}
	text, err := os.ReadFile(path)
	if err != nil {
		return ScriptFile{Error: fmt.Sprintf("read the file: %v", err)}
	}
	file := ScriptFile{Name: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)), Text: string(text)}
	for _, l := range scripts.Languages {
		if strings.EqualFold(filepath.Ext(path), l.Ext) {
			file.Ext = l.Ext
		}
	}
	return file
}
