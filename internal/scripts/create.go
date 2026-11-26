package scripts

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// MaxBody is the longest script Add writes: a command for the bar, not a program.
const MaxBody = 256 << 10

// Language is a kind of script this system can run, as the Add dialog offers it.
type Language struct{ Label, Ext string }

// comments is how each kind of script starts a comment line, which is where its @omni lines go.
var comments = map[string]string{
	".ps1": "#", ".py": "#", ".sh": "#", ".zsh": "#", ".rb": "#", ".js": "//", ".bat": "REM", ".cmd": "REM", ".applescript": "--",
}

// New is a script made in the settings window instead of dropped into the folder.
type New struct {
	// Title is the name the person gave it: what the bar lists it as. The file is named after it.
	Title string
	// Ext is the language's extension, one of Languages.
	Ext string
	// Output is one of Outputs; empty means silent.
	Output string
	Body   string
}

// Add writes the script into the folder with its @omni lines on top, so it is a command at once, and returns the file's path. It
// never replaces a file, and two scripts do not share a title: that would be two rows the bar cannot tell apart.
func (f *Folder) Add(n New) (string, error) {
	title := strings.TrimSpace(n.Title)
	if title == "" {
		return "", errors.New("give the script a name")
	}
	if strings.ContainsAny(title, "\r\n") {
		return "", errors.New("the name must be on one line")
	}
	if !slices.ContainsFunc(Languages, func(l Language) bool { return l.Ext == n.Ext }) {
		return "", fmt.Errorf("Omni cannot run %q scripts on this computer", n.Ext)
	}
	if n.Output == "" {
		n.Output = "silent"
	}
	if !slices.Contains(Outputs, n.Output) {
		return "", fmt.Errorf("what to do with the output must be one of %s", strings.Join(Outputs, ", "))
	}
	if strings.TrimSpace(n.Body) == "" {
		return "", errors.New("the script has no code")
	}
	if len(n.Body) > MaxBody {
		return "", fmt.Errorf("the script is longer than %d KB", MaxBody>>10)
	}
	f.refresh()
	f.mu.Lock()
	for _, s := range f.list {
		if strings.EqualFold(s.Title, title) {
			f.mu.Unlock()
			return "", fmt.Errorf("there is already a script called %q", s.Title)
		}
	}
	f.mu.Unlock()

	text := compose(title, n)
	if strings.ToLower(n.Ext) == ".bat" || strings.ToLower(n.Ext) == ".cmd" {
		text = strings.ReplaceAll(text, "\n", "\r\n") // cmd.exe loses its labels in a file with bare line feeds
	}
	stem := slug(title)
	for i := 1; ; i++ {
		name := stem + n.Ext
		if i > 1 {
			name = fmt.Sprintf("%s-%d%s", stem, i, n.Ext)
		}
		path := filepath.Join(f.dir, name)
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("write the script: %w", err)
		}
		_, werr := file.WriteString(text)
		if cerr := file.Close(); werr == nil {
			werr = cerr
		}
		if werr != nil {
			os.Remove(path)
			return "", fmt.Errorf("write the script: %w", werr)
		}
		f.refresh()
		return path, nil
	}
}

// compose puts the @omni lines on top of the code (below a #! line, which must stay first), dropping a title the code brought: the
// name the person just chose wins.
func compose(title string, n New) string {
	lines := strings.Split(strings.ReplaceAll(strings.TrimPrefix(n.Body, "\xef\xbb\xbf"), "\r\n", "\n"), "\n")
	var shebang []string
	if strings.HasPrefix(lines[0], "#!") {
		shebang, lines = lines[:1], lines[1:]
	}
	c := comments[strings.ToLower(n.Ext)]
	head := []string{c + " @omni.title " + title}
	if n.Output != "silent" {
		head = append(head, c+" @omni.output "+n.Output)
	}
	var code []string
	for i, line := range lines {
		if m := header.FindStringSubmatch(line); i < headerLines && m != nil && m[1] == "title" {
			continue
		}
		code = append(code, line)
	}
	return strings.TrimRight(strings.Join(slices.Concat(shebang, head, code), "\n"), "\n") + "\n"
}

// slug is a file name for a title: lower case letters and digits, with dashes between the words.
func slug(title string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(title) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			if dash && b.Len() > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r)
			dash = false
		default:
			dash = true
		}
	}
	if b.Len() == 0 {
		return "script"
	}
	return b.String()
}
