package aitools

import (
	"context"
	"regexp"
	"strings"

	"omni/internal/chat"
)

// scriptOutputChars is the most of a script's output the model is given.
const scriptOutputChars = 4000

var notToolName = regexp.MustCompile(`[^a-z0-9_]+`)

// scriptTools are the scripts with an @omni.tool line, each its own group: switched on, and allowed to run without asking, one
// by one. A script can do anything, so every one counts as changing something.
func (d Deps) scriptTools() []chat.Local {
	var out []chat.Local
	names := map[string]bool{}
	for _, s := range d.Scripts.List() {
		if s.Tool == "" {
			continue
		}
		name := "script_" + strings.Trim(notToolName.ReplaceAllString(strings.ToLower(s.Name), "_"), "_")
		if names[name] { // two files whose names differ only in what a tool name cannot hold: the first keeps it
			continue
		}
		names[name] = true
		params, required := map[string]string{}, []string(nil)
		if s.Argument != "" {
			params["input"], required = s.Argument, []string{"input"}
		}
		out = append(out, chat.Local{
			Group: "script:" + s.Name, GroupTitle: s.Title, Busy: "Running " + s.Title, Writes: true,
			Def:    chat.ObjectTool(name, s.Tool, params, required...),
			Prompt: "Some tools are the person's own scripts: call one only when the question asks for what it does.",
			Run: func(ctx context.Context, args map[string]string) (string, []chat.Source, error) {
				out, err := d.RunScript(ctx, s, args["input"])
				if err != nil {
					return "", nil, err
				}
				if strings.TrimSpace(out) == "" {
					return s.Title + " ran and printed nothing.", nil, nil
				}
				return "Output of " + s.Title + ":\n" + cut(out, scriptOutputChars), nil, nil
			},
		})
	}
	return out
}
