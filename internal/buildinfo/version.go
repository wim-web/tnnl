package buildinfo

import (
	"runtime/debug"
	"strings"
)

var linkerVersion = "dev"

func Current() string {
	info, ok := debug.ReadBuildInfo()

	return resolveVersion(linkerVersion, info, ok)
}

func resolveVersion(linker string, info *debug.BuildInfo, ok bool) string {
	if version := canonicalizeVersion(linker); version != "" && version != "dev" {
		return version
	}

	if !ok || info == nil {
		return "dev"
	}
	if isLocalCheckoutBuild(info) {
		return "dev"
	}

	version := canonicalizeVersion(info.Main.Version)
	if version == "" || version == "dev" || version == "(devel)" {
		return "dev"
	}

	return version
}

// Go 1.24 and later stamp a VCS-derived version into Main.Version for local
// builds. A go install module@version build has no vcs.revision setting, so its
// module version remains a valid fallback.
func isLocalCheckoutBuild(info *debug.BuildInfo) bool {
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			return true
		}
	}

	return false
}

func canonicalizeVersion(version string) string {
	return strings.TrimPrefix(strings.TrimSpace(version), "v")
}
