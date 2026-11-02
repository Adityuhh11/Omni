package browsers

import (
	"os"
	"path/filepath"
)

func sources() []source {
	roaming, local := os.Getenv("APPDATA"), os.Getenv("LOCALAPPDATA")
	return []source{
		{"Firefox", filepath.Join(roaming, "Mozilla", "Firefox", "Profiles"), firefox},
		{"Zen", filepath.Join(roaming, "zen", "Profiles"), firefox},
		{"Chrome", filepath.Join(local, "Google", "Chrome", "User Data"), chromium},
		{"Edge", filepath.Join(local, "Microsoft", "Edge", "User Data"), chromium},
		{"Brave", filepath.Join(local, "BraveSoftware", "Brave-Browser", "User Data"), chromium},
	}
}
