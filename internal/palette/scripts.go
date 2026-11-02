package palette

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"omni/internal/rank"
	"omni/internal/scripts"
)

// ScriptBox is the scripts folder as the bar uses it; *scripts.Folder satisfies it.
type ScriptBox interface {
	Dir() string
	List() []scripts.Script
	Find(name string) (scripts.Script, bool)
}

// scriptSep parts a script's name from the text typed after its keyword in a row's id; no file name holds it.
const scriptSep = "\x1f"

const scriptsFolderID = "scripts:folder"

// scriptItems are the scripts whose keyword starts the text (`port 3000` runs the script with "3000") or whose title the
// text starts or abbreviates, and a row that opens the scripts folder.
func (s *Service) scriptItems(text string) []Item {
	if s.x.Scripts == nil || utf8.RuneCountInString(text) < 2 {
		return nil
	}
	var items []Item
	for _, sc := range s.x.Scripts.List() {
		word, rest, _ := strings.Cut(text, " ")
		if strings.EqualFold(word, sc.Keyword) && (sc.Argument != "" || strings.TrimSpace(rest) == "") {
			items = append(items, scriptItem(sc, strings.TrimSpace(rest)))
		} else if rank.Match(text, sc.Title).Strong {
			items = append(items, scriptItem(sc, ""))
		}
	}
	if rank.Match(text, "Scripts folder").Strong {
		items = append(items, Item{ID: scriptsFolderID, Kind: "script", Title: "Open the scripts folder", Subtitle: "Scripts", Actions: []string{}})
	}
	return items
}

func scriptItem(sc scripts.Script, input string) Item {
	item := Item{ID: "script:" + sc.Name + scriptSep + input, Kind: "script", Title: sc.Title, Subtitle: "Script", Actions: []string{"reveal"}}
	switch {
	case input != "":
		item.Title += ": " + input
	case sc.Argument != "":
		item.Subtitle = "Script · type " + sc.Keyword + " and " + sc.Argument
		item.Fill = sc.Keyword + " "
	}
	return item
}

// doScript runs a script row and does with its output what the script asks: nothing, copy it, or paste it into the program
// the bar came from. The bar stays open while the script runs, so a failure can be shown in it.
func (s *Service) doScript(id, action string) error {
	if s.x.Scripts == nil {
		return errors.New("scripts are not available")
	}
	if id == scriptsFolderID {
		return s.files.Open(s.x.Scripts.Dir(), 0)
	}
	name, input, _ := strings.Cut(strings.TrimPrefix(id, "script:"), scriptSep)
	sc, ok := s.x.Scripts.Find(name)
	if !ok {
		return fmt.Errorf("there is no script %q any more", name)
	}
	switch action {
	case "reveal":
		return s.files.Reveal(sc.Path)
	case "open":
	default:
		return fmt.Errorf("unknown action %q", action)
	}
	if sc.Argument != "" && input == "" {
		return fmt.Errorf("type %s and %s", sc.Keyword, sc.Argument)
	}
	// ponytail: runs while the bar waits (up to scripts.RunTimeout); a script that takes long can start something and return
	out, err := s.x.RunScript(context.Background(), sc, input)
	if err != nil {
		return err
	}
	switch sc.Output {
	case "copy":
		if s.x.Copy == nil {
			return errors.New("copying is not available")
		}
		if err := s.x.Copy(out); err != nil {
			return fmt.Errorf("copy what %s printed: %w", sc.Title, err)
		}
	case "paste":
		if s.x.Paste == nil {
			return errors.New("pasting is not available")
		}
		return s.x.Paste(out) // it closes the bar itself, to give the keyboard back
	}
	s.dismiss()
	return nil
}
