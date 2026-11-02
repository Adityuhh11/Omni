package browsers

import (
	"os"
	"path/filepath"
)

func sources() []source {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	support := filepath.Join(home, "Library", "Application Support")
	return []source{
		{"Firefox", filepath.Join(support, "Firefox", "Profiles"), firefox},
		{"Zen", filepath.Join(support, "zen", "Profiles"), firefox},
		{"Chrome", filepath.Join(support, "Google", "Chrome"), chromium},
		{"Edge", filepath.Join(support, "Microsoft Edge"), chromium},
		{"Brave", filepath.Join(support, "BraveSoftware", "Brave-Browser"), chromium},
	}
}
