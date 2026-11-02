package mcpconnect

import (
	"os"
	"path/filepath"
)

func list() []Target {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return targets(home, supportDir(home))
}

// targets is the table of apps. support is the folder apps keep their settings in (%APPDATA%, ~/Library/Application Support).
func targets(home, support string) []Target {
	app := func(id, name, dir, file, key string) Target {
		return Target{ID: id, Name: name, Found: dir, File: filepath.Join(dir, file), Key: key}
	}
	return []Target{
		app("claude-desktop", "Claude Desktop", filepath.Join(support, "Claude"), "claude_desktop_config.json", "mcpServers"),
		app("cursor", "Cursor", filepath.Join(home, ".cursor"), "mcp.json", "mcpServers"),
		app("vscode", "VS Code", filepath.Join(support, "Code", "User"), "mcp.json", "servers"),
		app("windsurf", "Windsurf", filepath.Join(home, ".codeium", "windsurf"), "mcp_config.json", "mcpServers"),
	}
}
