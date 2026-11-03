package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// Local is a tool that runs on this computer: a look into Omni's own data or a script of the person's. main builds them (internal/aitools); the chat offers the ones whose group is switched on.
type Local struct {
	// Group is what switches the tool on in [chat] tools ("files", "notes", "script:backup.ps1"); one group may hold several tools.
	Group string
	// GroupTitle names the group in the + menu and in Settings.
	GroupTitle string
	Def        Tool
	// Apps is the same tool as other AI apps see it over MCP: the same name and
	// arguments, described in full for a model that can use it. Unset, Def is used.
	Apps Tool
	// Writes marks a tool that changes something (a new note, any script): the person is asked before each run unless its
	// group is in [chat] always_run.
	Writes bool
	// Prompt is a sentence for the system prompt that tells the model when the group's tools are worth calling.
	Prompt string
	// Busy is what the window shows while it runs ("Searching your files").
	Busy string
	// Run does the work. The text goes to the model; sources are shown under the answer.
	Run func(ctx context.Context, args map[string]string) (text string, sources []Source, err error)
}

// ForApps is the tool as another AI app is shown it.
func (l Local) ForApps() Tool {
	if l.Apps.Function.Name != "" {
		return l.Apps
	}
	return l.Def
}

// ToolGroup is a group of local tools as the + menu and Settings show it.
type ToolGroup struct {
	Group  string `json:"group"`
	Title  string `json:"title"`
	On     bool   `json:"on"`
	Writes bool   `json:"writes"` // it has a tool that changes something, so it can be set to run without asking
	Always bool   `json:"always"` // it runs without asking
}

// Confirm is the question the window shows before a tool that changes something runs.
type Confirm struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Tool  string `json:"tool"`
	Input string `json:"input"`
}

// toolset is what one answer may use.
type toolset struct {
	web    bool
	locals []Local
	always []string // groups that run without asking
}

func (t toolset) any() bool { return t.web || len(t.locals) > 0 }

func (t toolset) list() []Tool {
	var out []Tool
	if t.web {
		out = append(out, webTools()...)
	}
	for _, l := range t.locals {
		out = append(out, l.Def)
	}
	return out
}

// schemaTokens estimates what the tools' descriptions take of the context (they are sent with every request that offers
// them, and the server counts them against the context like the messages).
func (t toolset) schemaTokens() int {
	if !t.any() {
		return 0
	}
	b, _ := json.Marshal(t.list())
	return len(b) / charsPerToken
}

func (t toolset) find(name string) (Local, bool) {
	i := slices.IndexFunc(t.locals, func(l Local) bool { return l.Def.Function.Name == name })
	if i < 0 {
		return Local{}, false
	}
	return t.locals[i], true
}

// prompts are the groups' sentences for the system prompt, each once.
func (t toolset) prompts() []string {
	var out []string
	for _, l := range t.locals {
		if l.Prompt != "" && !slices.Contains(out, l.Prompt) {
			out = append(out, l.Prompt)
		}
	}
	return out
}

// groups lists the tool groups there are, in the order the tools come, with their switches as cfg has them.
func groups(locals []Local, on, always []string) []ToolGroup {
	var out []ToolGroup
	for _, l := range locals {
		i := slices.IndexFunc(out, func(g ToolGroup) bool { return g.Group == l.Group })
		if i < 0 {
			out = append(out, ToolGroup{Group: l.Group, Title: l.GroupTitle, On: slices.Contains(on, l.Group), Always: slices.Contains(always, l.Group)})
			i = len(out) - 1
		}
		out[i].Writes = out[i].Writes || l.Writes
	}
	return out
}

// ObjectTool describes a tool whose arguments are all strings; required lists the ones the model must give.
func ObjectTool(name, description string, params map[string]string, required ...string) Tool {
	props := map[string]any{}
	for p, help := range params {
		props[p] = map[string]any{"type": "string", "description": help}
	}
	if required == nil {
		required = []string{}
	}
	return Tool{Type: "function", Function: ToolFunction{Name: name, Description: description, Parameters: map[string]any{
		"type": "object", "properties": props, "required": required,
	}}}
}

// runLocal runs one local tool, asking the person first when it changes something. Arguments that are not strings (a small
// model writes numbers bare) are passed as their JSON text.
func (l *lookup) runLocal(ctx context.Context, tool Local, raw string) string {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil && strings.TrimSpace(raw) != "" {
		return "Error: the arguments were not valid JSON. Call " + tool.Def.Function.Name + " again with a JSON object."
	}
	args := StringArgs(parsed)
	if tool.Writes && !slices.Contains(l.tools.always, tool.Group) {
		if l.ask == nil || !l.ask(ctx, Confirm{Title: tool.GroupTitle, Tool: tool.Def.Function.Name, Input: describeArgs(args)}) {
			return "The person did not allow " + tool.Def.Function.Name + " to run. Do not call it again for this question; say it was not run."
		}
	}
	busy := tool.Busy
	if busy == "" {
		busy = "Running " + tool.GroupTitle
	}
	if q := args["query"]; q != "" {
		busy += " for “" + q + "”"
	}
	l.status(Status{Phase: "tool", Text: busy + "…"})
	text, sources, err := tool.Run(ctx, args)
	if err != nil {
		l.logf("chat: tool %s: %v", tool.Def.Function.Name, err)
		return fmt.Sprintf("%s failed: %v. Tell the person it did not work. Do not answer as if it had.", tool.Def.Function.Name, err)
	}
	for _, s := range sources {
		l.used = addSource(l.used, s)
	}
	return text
}

// describeArgs is the arguments as the confirmation shows them: one per line, the only one bare.
func describeArgs(args map[string]string) string {
	if len(args) == 1 {
		for _, v := range args {
			return v
		}
	}
	keys := make([]string, 0, len(args))
	for k := range args {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s: %s\n", k, args[k])
	}
	return strings.TrimSpace(b.String())
}

// StringArgs turns a tool call's arguments into text: strings trimmed, anything else (a number a model wrote bare) as its JSON.
func StringArgs(in map[string]any) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		if s, ok := v.(string); ok {
			out[k] = strings.TrimSpace(s)
		} else {
			b, _ := json.Marshal(v)
			out[k] = string(b)
		}
	}
	return out
}
