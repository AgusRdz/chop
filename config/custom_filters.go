package config

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// CustomFilter defines a user-configured output filter for a specific command.
type CustomFilter struct {
	Keep []string `yaml:"keep"` // regex patterns — only lines matching at least one are kept
	Drop []string `yaml:"drop"` // regex patterns — lines matching any are removed
	Head int      `yaml:"head"` // keep only first N lines (applied after keep/drop)
	Tail int      `yaml:"tail"` // keep only last N lines (applied after keep/drop)
	Exec string   `yaml:"exec"` // external script: pipe output through this command
}

// CustomFiltersConfig holds the full filters.yml content.
type CustomFiltersConfig struct {
	Filters map[string]CustomFilter `yaml:"filters"`
}

// FiltersConfigPath returns the path to the custom filters file.
func FiltersConfigPath() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "chop", "filters.yml")
}

// LoadCustomFilters reads the custom filters config file.
// Returns an empty map if the file doesn't exist or can't be parsed.
func LoadCustomFilters() map[string]CustomFilter {
	return LoadCustomFiltersFrom(FiltersConfigPath())
}

// LoadCustomFiltersFrom reads custom filters from a specific path.
func LoadCustomFiltersFrom(path string) map[string]CustomFilter {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return ParseCustomFilters(data)
}

// ParseCustomFilters parses YAML bytes into a custom filters map.
func ParseCustomFilters(data []byte) map[string]CustomFilter {
	var cfg CustomFiltersConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil
	}
	if cfg.Filters == nil {
		return nil
	}
	return cfg.Filters
}

// LookupCustomFilter finds a custom filter for the given command and args.
// It checks "command subcommand" first, then falls back to "command".
func LookupCustomFilter(filters map[string]CustomFilter, command string, args []string) *CustomFilter {
	if len(filters) == 0 {
		return nil
	}

	// Try "command subcommand" match first
	if len(args) > 0 {
		fullCmd := command + " " + args[0]
		if f, ok := filters[fullCmd]; ok {
			return &f
		}
		// Case-insensitive fallback
		for key, f := range filters {
			if strings.EqualFold(key, fullCmd) {
				return &f
			}
		}
	}

	// Try base command match
	if f, ok := filters[command]; ok {
		return &f
	}
	for key, f := range filters {
		if strings.EqualFold(key, command) {
			return &f
		}
	}

	return nil
}

// LoadCustomFiltersWithLocal loads global custom filters, then overlays
// a local .chop-filters.yml from the given directory (if it exists).
// Local filters are merged on top of global ones (local wins on conflict).
func LoadCustomFiltersWithLocal(cwd string) map[string]CustomFilter {
	global := LoadCustomFilters()

	if cwd == "" {
		return global
	}

	localPath := filepath.Join(cwd, ".chop-filters.yml")
	local := LoadCustomFiltersFrom(localPath)

	if len(local) == 0 {
		return global
	}

	// Merge: local overrides global
	if global == nil {
		return local
	}
	for k, v := range local {
		global[k] = v
	}
	return global
}
