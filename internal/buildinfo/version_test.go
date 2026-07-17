package buildinfo

import (
	"runtime/debug"
	"testing"
)

func TestCurrentCanonicalizesLinkerVersion(t *testing.T) {
	original := linkerVersion
	linkerVersion = " v1.2.3 "
	t.Cleanup(func() {
		linkerVersion = original
	})

	if got, want := Current(), "1.2.3"; got != want {
		t.Fatalf("Current() = %q, want %q", got, want)
	}
}

func TestResolveVersion(t *testing.T) {
	tests := []struct {
		name   string
		linker string
		info   *debug.BuildInfo
		ok     bool
		want   string
	}{
		{
			name:   "linker version takes precedence",
			linker: " v1.2.3 ",
			info:   buildInfo("v9.9.9"),
			ok:     true,
			want:   "1.2.3",
		},
		{
			name:   "linker version wins over local VCS build",
			linker: "v1.2.3",
			info: buildInfoWithSettings(
				"v9.9.9-0.20260717190216-abcdef123456",
				debug.BuildSetting{Key: "vcs", Value: "git"},
				debug.BuildSetting{Key: "vcs.revision", Value: "abcdef1234567890"},
			),
			ok:   true,
			want: "1.2.3",
		},
		{
			name:   "only one leading lowercase v is removed",
			linker: "vv1.2.3",
			info:   buildInfo("v9.9.9"),
			ok:     true,
			want:   "v1.2.3",
		},
		{
			name:   "uppercase V is preserved",
			linker: "V1.2.3",
			info:   buildInfo("v9.9.9"),
			ok:     true,
			want:   "V1.2.3",
		},
		{
			name:   "linker devel is accepted",
			linker: "(devel)",
			info:   buildInfo("v9.9.9"),
			ok:     true,
			want:   "(devel)",
		},
		{
			name:   "empty linker falls back to module",
			linker: "",
			info:   buildInfo("v2.3.4"),
			ok:     true,
			want:   "2.3.4",
		},
		{
			name:   "local VCS pseudo version returns dev",
			linker: "dev",
			info: buildInfoWithSettings(
				"v2.3.5-0.20260717190216-abcdef123456",
				debug.BuildSetting{Key: "vcs", Value: "git"},
				debug.BuildSetting{Key: "vcs.revision", Value: "abcdef1234567890"},
			),
			ok:   true,
			want: "dev",
		},
		{
			name:   "module proxy pseudo version is preserved",
			linker: "dev",
			info:   buildInfo("v2.3.5-0.20260717190216-abcdef123456"),
			ok:     true,
			want:   "2.3.5-0.20260717190216-abcdef123456",
		},
		{
			name:   "whitespace linker falls back to module",
			linker: " \t\n ",
			info:   buildInfo(" v2.3.4 "),
			ok:     true,
			want:   "2.3.4",
		},
		{
			name:   "v only linker falls back to module",
			linker: " v ",
			info:   buildInfo("v2.3.4"),
			ok:     true,
			want:   "2.3.4",
		},
		{
			name:   "dev linker falls back to module",
			linker: " vdev ",
			info:   buildInfo("v2.3.4"),
			ok:     true,
			want:   "2.3.4",
		},
		{
			name:   "module removes at most one leading lowercase v",
			linker: "dev",
			info:   buildInfo(" vv2.3.4 "),
			ok:     true,
			want:   "v2.3.4",
		},
		{
			name:   "unavailable build info returns dev",
			linker: "dev",
			info:   buildInfo("v2.3.4"),
			ok:     false,
			want:   "dev",
		},
		{
			name:   "nil build info returns dev",
			linker: "dev",
			info:   nil,
			ok:     true,
			want:   "dev",
		},
		{
			name:   "empty module version returns dev",
			linker: "dev",
			info:   buildInfo(""),
			ok:     true,
			want:   "dev",
		},
		{
			name:   "whitespace module version returns dev",
			linker: "dev",
			info:   buildInfo(" \t\n "),
			ok:     true,
			want:   "dev",
		},
		{
			name:   "v only module version returns dev",
			linker: "dev",
			info:   buildInfo(" v "),
			ok:     true,
			want:   "dev",
		},
		{
			name:   "dev module version returns dev",
			linker: "dev",
			info:   buildInfo(" vdev "),
			ok:     true,
			want:   "dev",
		},
		{
			name:   "devel module version returns dev",
			linker: "dev",
			info:   buildInfo(" (devel) "),
			ok:     true,
			want:   "dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveVersion(tt.linker, tt.info, tt.ok); got != tt.want {
				t.Errorf("resolveVersion(%q, %#v, %t) = %q, want %q", tt.linker, tt.info, tt.ok, got, tt.want)
			}
		})
	}
}

func buildInfo(version string) *debug.BuildInfo {
	return buildInfoWithSettings(version)
}

func buildInfoWithSettings(version string, settings ...debug.BuildSetting) *debug.BuildInfo {
	return &debug.BuildInfo{
		Main:     debug.Module{Version: version},
		Settings: settings,
	}
}
