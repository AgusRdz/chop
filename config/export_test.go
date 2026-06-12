package config

import "sync"

// ResetCacheForTest zeros all sync.Once config caches.
// Call this at the start of any test that manipulates env vars
// (XDG_CONFIG_HOME, HOME, etc.) to prevent stale cached results.
// Note: ConfigDir and DataDir no longer cache — no reset needed for them.
func ResetCacheForTest() {
	globalCfgOnce = sync.Once{}
	globalCfg = Config{}
	globalFiltersOnce = sync.Once{}
	globalFilters = nil
}
