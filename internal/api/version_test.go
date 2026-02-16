package api

import (
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    Version
		wantErr bool
	}{
		{
			name: "TrueNAS SCALE 24.10",
			raw:  "TrueNAS-SCALE-24.10.2.4",
			want: Version{
				Major:  24,
				Minor:  10,
				Patch:  2,
				Build:  4,
				Flavor: FlavorScale,
				Raw:    "TrueNAS-SCALE-24.10.2.4",
			},
		},
		{
			name: "TrueNAS COMMUNITY edition",
			raw:  "TrueNAS-COMMUNITY-24.10.1.0",
			want: Version{
				Major:  24,
				Minor:  10,
				Patch:  1,
				Build:  0,
				Flavor: FlavorCommunity,
				Raw:    "TrueNAS-COMMUNITY-24.10.1.0",
			},
		},
		{
			name: "TrueNAS 25.04 without SCALE",
			raw:  "TrueNAS-25.04.2.4",
			want: Version{
				Major:  25,
				Minor:  4,
				Patch:  2,
				Build:  4,
				Flavor: FlavorUnknown,
				Raw:    "TrueNAS-25.04.2.4",
			},
		},
		{
			name: "TrueNAS SCALE 25.10",
			raw:  "TrueNAS-SCALE-25.10.0.0",
			want: Version{
				Major:  25,
				Minor:  10,
				Patch:  0,
				Build:  0,
				Flavor: FlavorScale,
				Raw:    "TrueNAS-SCALE-25.10.0.0",
			},
		},
		{
			name: "TrueNAS 25.10 three-part version",
			raw:  "TrueNAS-25.10.1",
			want: Version{
				Major:  25,
				Minor:  10,
				Patch:  1,
				Build:  0,
				Flavor: FlavorUnknown,
				Raw:    "TrueNAS-25.10.1",
			},
		},
		{
			name:    "invalid version string",
			raw:     "not-a-version",
			wantErr: true,
		},
		{
			name:    "empty string",
			raw:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseVersion(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.Major != tt.want.Major {
				t.Errorf("Major = %v, want %v", got.Major, tt.want.Major)
			}
			if got.Minor != tt.want.Minor {
				t.Errorf("Minor = %v, want %v", got.Minor, tt.want.Minor)
			}
			if got.Patch != tt.want.Patch {
				t.Errorf("Patch = %v, want %v", got.Patch, tt.want.Patch)
			}
			if got.Build != tt.want.Build {
				t.Errorf("Build = %v, want %v", got.Build, tt.want.Build)
			}
			if got.Flavor != tt.want.Flavor {
				t.Errorf("Flavor = %v, want %v", got.Flavor, tt.want.Flavor)
			}
			if got.Raw != tt.want.Raw {
				t.Errorf("Raw = %v, want %v", got.Raw, tt.want.Raw)
			}
		})
	}
}

func TestVersion_AtLeast(t *testing.T) {
	tests := []struct {
		name    string
		version Version
		major   int
		minor   int
		want    bool
	}{
		{
			name:    "24.10 at least 25.10",
			version: Version{Major: 24, Minor: 10},
			major:   25,
			minor:   10,
			want:    false,
		},
		{
			name:    "25.04 at least 25.10",
			version: Version{Major: 25, Minor: 4},
			major:   25,
			minor:   10,
			want:    false,
		},
		{
			name:    "25.10 at least 25.10",
			version: Version{Major: 25, Minor: 10},
			major:   25,
			minor:   10,
			want:    true,
		},
		{
			name:    "25.11 at least 25.10",
			version: Version{Major: 25, Minor: 11},
			major:   25,
			minor:   10,
			want:    true,
		},
		{
			name:    "26.0 at least 25.10",
			version: Version{Major: 26, Minor: 0},
			major:   25,
			minor:   10,
			want:    true,
		},
		// Additional tests against 25.0 baseline (container support threshold)
		{
			name:    "25.04 at least 25.0",
			version: Version{Major: 25, Minor: 4},
			major:   25,
			minor:   0,
			want:    true,
		},
		{
			name:    "25.0 at least 25.0",
			version: Version{Major: 25, Minor: 0},
			major:   25,
			minor:   0,
			want:    true,
		},
		{
			name:    "24.10 at least 25.0",
			version: Version{Major: 24, Minor: 10},
			major:   25,
			minor:   0,
			want:    false,
		},
		{
			name:    "26.0 at least 25.0",
			version: Version{Major: 26, Minor: 0},
			major:   25,
			minor:   0,
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.version.AtLeast(tt.major, tt.minor); got != tt.want {
				t.Errorf("AtLeast(%d, %d) = %v, want %v", tt.major, tt.minor, got, tt.want)
			}
		})
	}
}

func TestVersion_String(t *testing.T) {
	tests := []struct {
		name    string
		version Version
		want    string
	}{
		{
			name:    "full version",
			version: Version{Major: 24, Minor: 10, Patch: 2, Build: 4},
			want:    "24.10.2.4",
		},
		{
			name:    "zero values",
			version: Version{Major: 25, Minor: 0, Patch: 0, Build: 0},
			want:    "25.0.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.version.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVersion_Compare(t *testing.T) {
	tests := []struct {
		name  string
		a     Version
		b     Version
		want  int
	}{
		{
			name: "equal versions",
			a:    Version{Major: 24, Minor: 10, Patch: 2, Build: 4},
			b:    Version{Major: 24, Minor: 10, Patch: 2, Build: 4},
			want: 0,
		},
		{
			name: "a < b by major",
			a:    Version{Major: 24, Minor: 10},
			b:    Version{Major: 25, Minor: 4},
			want: -1,
		},
		{
			name: "a > b by major",
			a:    Version{Major: 25, Minor: 4},
			b:    Version{Major: 24, Minor: 10},
			want: 1,
		},
		{
			name: "a < b by minor",
			a:    Version{Major: 25, Minor: 4},
			b:    Version{Major: 25, Minor: 10},
			want: -1,
		},
		{
			name: "a > b by minor",
			a:    Version{Major: 25, Minor: 10},
			b:    Version{Major: 25, Minor: 4},
			want: 1,
		},
		{
			name: "a < b by patch",
			a:    Version{Major: 25, Minor: 10, Patch: 0},
			b:    Version{Major: 25, Minor: 10, Patch: 1},
			want: -1,
		},
		{
			name: "a < b by build",
			a:    Version{Major: 25, Minor: 10, Patch: 1, Build: 0},
			b:    Version{Major: 25, Minor: 10, Patch: 1, Build: 1},
			want: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.Compare(tt.b); got != tt.want {
				t.Errorf("Compare() = %v, want %v", got, tt.want)
			}
		})
	}
}

