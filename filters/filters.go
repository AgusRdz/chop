package filters

// FilterFunc takes raw command output and returns filtered output.
type FilterFunc func(raw string) (string, error)

// Get returns a filter for the given command, or nil for passthrough.
func Get(command string, args []string) FilterFunc {
	switch command {
	case "git":
		return getGitFilter(args)
	case "npm":
		return getNpmFilter(args)
	case "npx":
		return getNpxFilter(args)
	case "docker":
		return getDockerFilter(args)
	case "dotnet":
		return getDotnetFilter(args)
	case "kubectl":
		return getKubectlFilter(args)
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
	default:
		return nil
	}
}

func getGitFilter(args []string) FilterFunc {
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
	case "branch":
		return filterGitBranch
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
	case "list", "ls":
		return filterNpmList
	case "test", "t":
		return filterNpmTestCmd
	default:
		return nil
	}
}

func getDotnetFilter(args []string) FilterFunc {
	if len(args) == 0 {
		return nil
	}
	switch args[0] {
	case "build":
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
