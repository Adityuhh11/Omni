package mcpconnect

import "path/filepath"

func supportDir(home string) string { return filepath.Join(home, "Library", "Application Support") }
