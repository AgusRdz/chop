package filters

import (
	"strings"

	"github.com/AgusRdz/chop/config"
)

// FilterFunc takes raw command output and returns filtered output.
type FilterFunc func(raw string) (string, error)

// userFilters holds user-defined custom filters loaded at startup.
var userFilters map[string]config.CustomFilter

// SetUserFilters registers user-defined custom filters for lookup.
// Call once at startup with the result of config.LoadCustomFiltersWithLocal.
func SetUserFilters(filters map[string]config.CustomFilter) {
	userFilters = filters
}

// Get returns a filter for the given command, or nil for passthrough.
// Checks user-defined custom filters first, then falls back to built-ins.
func Get(command string, args []string) FilterFunc {
	// User-defined filters take priority
	if cf := config.LookupCustomFilter(userFilters, command, args); cf != nil {
		if fn := BuildUserFilter(cf); fn != nil {
			return fn
		}
	}
	return get(command, args)
}

// HasFilter reports whether a registered filter exists for the given command.
func HasFilter(command string, args []string) bool {
	if config.LookupCustomFilter(userFilters, command, args) != nil {
		return true
	}
	return get(command, args) != nil
}

func get(command string, args []string) FilterFunc {
	switch command {
	case "git":
		return getGitFilter(args)
	case "npm":
		return getNpmFilter(args)
	case "npx":
		return getNpxFilter(args)
	case "pnpm":
		return getPnpmFilter(args)
	case "yarn":
		return getYarnFilter(args)
	case "bun":
		return getBunFilter(args)
	case "docker":
		return getDockerFilter(args)
	case "dotnet":
		return getDotnetFilter(args)
	case "kubectl":
		return getKubectlFilter(args)
	case "helm":
		return getHelmFilter(args)
	case "terraform":
		return getTerraformFilter(args)
	case "cargo":
		return getCargoFilter(args)
	case "go":
		return getGoFilter(args)
	case "tsc":
		return filterTsc
	case "eslint":
		return filterEslint
	case "biome":
		return filterEslint
	case "gh":
		return getGhFilter(args)
	case "grep":
		return filterGrep
	case "rg":
		return filterGrep
	case "curl":
		return filterCurl
	case "http":
		return filterHttpie
	case "aws":
		return getAwsFilter(args)
	case "az":
		return getAzFilter(args)
	case "gcloud":
		return getGcloudFilter(args)
	case "mvn":
		return getMavenFilter(args)
	case "gradle", "gradlew":
		return getGradleFilter(args)
	// Angular / Nx
	case "ng":
		return getAngularFilter(args)
	case "nx":
		return getNxFilter(args)
	// Python
	case "pytest":
		return filterPytest
	case "pip", "pip3":
		return getPipFilter(args)
	case "uv":
		return getUvFilter(args)
	case "mypy":
		return filterMypy
	case "ruff":
		return filterRuff
	case "flake8":
		return filterRuff
	case "pylint":
		return filterPylint
	// Ruby
	case "bundle", "bundler":
		return getBundleFilter(args)
	case "rspec":
		return filterRspec
	case "rubocop":
		return filterRubocop
	// PHP
	case "composer":
		return getComposerFilter(args)
	// Build tools
	case "make":
		return filterMake
	case "cmake":
		return filterCmake
	case "gcc", "g++", "cc", "c++", "clang", "clang++":
		return filterCompiler
	// System
	case "ping":
		return filterPing
	case "ps":
		return filterPsCmd
	case "ss", "netstat":
		return filterNetstat
	case "df", "du":
		return filterDf
	case "cat", "tail", "less", "more":
		return filterAutoDetect
	case "ls":
		return filterAutoDetect
	case "find":
		return filterAutoDetect
	case "node", "node16", "node18", "node20", "node22":
		return filterAutoDetect
	case "acli":
		return getAcliFilter(args)
	default:
		return nil
	}
}

func getDockerFilter(args []string) FilterFunc {
	if len(args) == 0 {
		return nil
	}
	switch args[0] {
	case "ps":
		return filterDockerPs
	case "build":
		return filterDockerBuild
	case "images":
		return filterDockerImages
	case "logs":
		return filterDockerLogs
	case "rmi":
		return filterDockerRmi
	case "inspect":
		return filterDockerInspect
	case "stats":
		return filterDockerStats
	case "top":
		return filterDockerTop
	case "diff":
		return filterDockerDiff
	case "history":
		return filterDockerHistory
	case "network":
		if len(args) > 1 && args[1] == "ls" {
			return filterDockerNetworkLs
		}
		return nil
	case "volume":
		if len(args) > 1 && args[1] == "ls" {
			return filterDockerVolumeLs
		}
		return nil
	case "system":
		if len(args) > 1 && args[1] == "df" {
			return filterDockerSystemDf
		}
		return nil
	case "compose":
		return getDockerComposeFilter(args[1:])
	default:
		return nil
	}
}

func getDockerComposeFilter(args []string) FilterFunc {
	if len(args) == 0 {
		// No subcommand — return filterAutoDetect so HasFilter recognizes this
		// command family (docker compose ps/build/logs all have filters).
		return filterAutoDetect
	}
	switch args[0] {
	case "ps":
		return filterDockerPs
	case "build":
		return filterDockerBuild
	case "logs":
		return filterDockerLogs
	case "images":
		return filterDockerImages
	default:
		return nil
	}
}

// skipGitGlobalFlags advances past known git global flags to find the subcommand.
// Handles cases like: git -C /path status, git --no-pager log, git -c key=val diff
func skipGitGlobalFlags(args []string) []string {
	i := 0
	for i < len(args) {
		arg := args[i]
		// Flags that consume the next argument as a value
		if arg == "-C" || arg == "--git-dir" || arg == "--work-tree" || arg == "-c" || arg == "--exec-path" {
			i += 2
			continue
		}
		// Flags with embedded value (--git-dir=path, --work-tree=path)
		if strings.HasPrefix(arg, "--git-dir=") || strings.HasPrefix(arg, "--work-tree=") || strings.HasPrefix(arg, "-c=") {
			i++
			continue
		}
		// Boolean flags
		if arg == "--no-pager" || arg == "--bare" || arg == "--no-replace-objects" || arg == "-p" || arg == "--paginate" {
			i++
			continue
		}
		break
	}
	return args[i:]
}

func getGitFilter(args []string) FilterFunc {
	if len(args) == 0 {
		return nil
	}
	args = skipGitGlobalFlags(args)
	if len(args) == 0 {
		return nil
	}
	switch args[0] {
	case "status":
		return filterGitStatus
	case "log":
		return filterGitLog
	case "diff":
		return filterGitDiff
	case "show":
		return filterGitDiff
	case "branch":
		return filterGitBranch
	case "push":
		return filterGitPush
	case "remote", "tag", "checkout", "reset":
		return filterAutoDetect
	case "stash":
		if len(args) > 1 && args[1] == "list" {
			return filterGitLog
		}
		return nil
	default:
		return nil
	}
}

func getNpmFilter(args []string) FilterFunc {
	if len(args) == 0 {
		return nil
	}
	switch args[0] {
	case "install", "i":
		return filterNpmInstall
	case "update", "up", "upgrade":
		return filterNpmInstall
	case "list", "ls":
		return filterNpmList
	case "view", "info", "show":
		return filterNpmView
	case "test", "t":
		return filterNpmTestCmd
	case "run":
		if len(args) > 1 {
			switch args[1] {
			case "test", "build", "lint":
				return filterNpmTestCmd
			}
		}
		return nil
	default:
		return nil
	}
}

func getDotnetFilter(args []string) FilterFunc {
	if len(args) == 0 {
		return nil
	}
	switch args[0] {
	case "build", "clean", "pack", "publish":
		return filterDotnetBuild
	case "test":
		return filterDotnetTestCmd
	default:
		return nil
	}
}

func getKubectlFilter(args []string) FilterFunc {
	if len(args) == 0 {
		return nil
	}
	switch args[0] {
	case "get":
		return filterKubectlGet
	case "describe":
		return filterKubectlDescribe
	case "logs", "log":
		return filterKubectlLogs
	case "top":
		return filterKubectlTop
	case "apply":
		return filterKubectlApply
	case "delete":
		return filterKubectlDelete
	default:
		return nil
	}
}

func getHelmFilter(args []string) FilterFunc {
	if len(args) == 0 {
		return nil
	}
	switch args[0] {
	case "install", "upgrade":
		return filterHelmInstall
	case "list", "ls":
		return filterHelmList
	case "status":
		return filterHelmInstall // same format
	default:
		return nil
	}
}

func getNpxFilter(args []string) FilterFunc {
	if len(args) == 0 {
		return nil
	}
	switch args[0] {
	case "jest", "vitest", "mocha":
		return filterNpmTestCmd
	case "nx":
		return getNxFilter(args[1:])
	case "playwright":
		if len(args) == 1 || (len(args) > 1 && args[1] == "test") {
			return filterPlaywright
		}
		return nil
	case "tsc":
		return filterTsc
	case "ng":
		return getAngularFilter(args[1:])
	default:
		return nil
	}
}

func getCargoFilter(args []string) FilterFunc {
	if len(args) == 0 {
		return nil
	}
	switch args[0] {
	case "test":
		return filterCargoTestCmd
	case "build", "check":
		return filterCargoBuild
	case "clippy":
		return filterCargoClippy
	default:
		return nil
	}
}

func getGoFilter(args []string) FilterFunc {
	if len(args) == 0 {
		return nil
	}
	switch args[0] {
	case "test":
		return filterGoTestCmd
	case "build", "vet":
		return filterGoBuild
	default:
		return nil
	}
}

func getGhFilter(args []string) FilterFunc {
	if len(args) == 0 {
		return nil
	}
	switch args[0] {
	case "pr":
		return getGhPrFilter(args[1:])
	case "issue":
		return getGhIssueFilter(args[1:])
	case "run":
		return getGhRunFilter(args[1:])
	default:
		return nil
	}
}

func getTerraformFilter(args []string) FilterFunc {
	if len(args) == 0 {
		return nil
	}
	switch args[0] {
	case "plan":
		return filterTerraformPlan
	case "apply":
		return filterTerraformApply
	case "init":
		return filterTerraformInit
	default:
		return nil
	}
}

func getPnpmFilter(args []string) FilterFunc {
	if len(args) == 0 {
		return nil
	}
	switch args[0] {
	case "install", "i", "add":
		return filterPnpmInstall
	case "list", "ls":
		return filterNpmList
	case "test", "t":
		return filterNpmTestCmd
	default:
		return nil
	}
}

func getYarnFilter(args []string) FilterFunc {
	if len(args) == 0 {
		return nil
	}
	switch args[0] {
	case "install", "add":
		return filterYarnInstall
	case "list":
		return filterNpmList
	case "test":
		return filterNpmTestCmd
	default:
		return nil
	}
}

func getBunFilter(args []string) FilterFunc {
	if len(args) == 0 {
		return nil
	}
	switch args[0] {
	case "install", "i", "add":
		return filterBunInstall
	case "test", "t":
		return filterNpmTestCmd
	default:
		return nil
	}
}

func getAngularFilter(args []string) FilterFunc {
	if len(args) == 0 {
		return nil
	}
	switch args[0] {
	case "build", "b":
		return filterNgBuild
	case "test", "t":
		return filterNgTest
	case "serve", "s":
		return filterNgServe
	case "lint":
		return filterEslint
	default:
		return nil
	}
}

func getNxFilter(args []string) FilterFunc {
	if len(args) == 0 {
		return filterAutoDetect
	}
	switch args[0] {
	case "build", "run":
		return filterNxBuild
	case "test":
		return filterNxTest
	case "lint":
		return filterEslint
	default:
		return nil
	}
}

func getPipFilter(args []string) FilterFunc {
	if len(args) == 0 {
		return nil
	}
	switch args[0] {
	case "install":
		return filterPipInstall
	case "list":
		return filterPipList
	default:
		return nil
	}
}

func getUvFilter(args []string) FilterFunc {
	if len(args) == 0 {
		return nil
	}
	switch args[0] {
	case "pip":
		if len(args) > 1 {
			switch args[1] {
			case "install":
				return filterUvInstall
			case "list":
				return filterPipList
			}
		}
		return nil
	case "install", "add":
		return filterUvInstall
	default:
		return nil
	}
}

func getBundleFilter(args []string) FilterFunc {
	if len(args) == 0 {
		return filterBundleInstall
	}
	switch args[0] {
	case "install":
		return filterBundleInstall
	default:
		return nil
	}
}

func getComposerFilter(args []string) FilterFunc {
	if len(args) == 0 {
		return nil
	}
	switch args[0] {
	case "install", "update", "require":
		return filterComposerInstall
	default:
		return nil
	}
}
