package mcpconnect

import (
	"os"
	"path/filepath"
)

func supportDir(home string) string {
	if d := os.Getenv("APPDATA"); d != "" {
		return d
	}
	return filepath.Join(home, "AppData", "Roaming")
}
