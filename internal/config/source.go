package config

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"slices"
	"strings"
)

// ExcludedUnder returns the folders of exclude that lie inside root, as the '/'-separated paths
// below root that the engine takes. A folder that is root itself, or outside it, is left out.
func ExcludedUnder(root string, exclude []string) []string {
	var below []string
	for _, folder := range exclude {
		rel, err := filepath.Rel(root, folder)
		if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		below = append(below, filepath.ToSlash(rel))
	}
	return below
}

// SourceID names a root in the memory engine. It is derived from the path (case-insensitive,
// as Windows paths are) so the same folder always maps to the same source, and it fits the
// engine's source id rule (letters, digits, dot, dash, underscore; at most 64 characters).
func SourceID(root string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(filepath.Clean(root))))
	return "root-" + hex.EncodeToString(sum[:])[:10]
}

// Inside reports whether folder lies inside root (not root itself).
func Inside(folder, root string) bool { return len(ExcludedUnder(root, []string{folder})) > 0 }

// UseRoots makes roots the searched folders. A folder inside another one is dropped, as the outer folder already indexes
// it (kept, its files would be indexed twice), and a switch or exclusion left without its folder goes with it.
func (c *Config) UseRoots(roots []string) {
	c.Roots = slices.DeleteFunc(slices.Clone(roots), func(folder string) bool {
		return slices.ContainsFunc(roots, func(root string) bool { return Inside(folder, root) })
	})
	searched := func(folder string) bool {
		return slices.ContainsFunc(c.Roots, func(root string) bool { return SourceID(root) == SourceID(folder) })
	}
	c.Off = slices.DeleteFunc(slices.Clone(c.Off), func(folder string) bool { return !searched(folder) })
	c.Exclude = slices.DeleteFunc(slices.Clone(c.Exclude), func(folder string) bool {
		return !slices.ContainsFunc(c.Roots, func(root string) bool { return Inside(folder, root) })
	})
}
