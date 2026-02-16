package api

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Flavor represents the TrueNAS edition.
type Flavor string

const (
	FlavorScale     Flavor = "SCALE"
	FlavorCommunity Flavor = "COMMUNITY"
	FlavorUnknown   Flavor = ""
)

// Version represents a parsed TrueNAS version.
type Version struct {
	Major  int
	Minor  int
	Patch  int
	Build  int
	Flavor Flavor
	Raw    string
}

// versionRegex extracts version numbers from strings like "TrueNAS-SCALE-24.10.2.4" or "TrueNAS-25.10.1"
var versionRegex = regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)(?:\.(\d+))?`)

// ParseVersion parses a TrueNAS version string.
// Examples: "TrueNAS-SCALE-24.10.2.4", "TrueNAS-25.04.2.4", "TrueNAS-25.10.1"
func ParseVersion(raw string) (Version, error) {
	v := Version{Raw: raw}

	// Detect flavor
	switch {
	case strings.Contains(raw, "SCALE"):
		v.Flavor = FlavorScale
	case strings.Contains(raw, "COMMUNITY"):
		v.Flavor = FlavorCommunity
	default:
		v.Flavor = FlavorUnknown
	}

	// Extract version numbers
	match := versionRegex.FindStringSubmatch(raw)
	if match == nil {
		return v, fmt.Errorf("unable to parse version from %q", raw)
	}

	var err error
	v.Major, err = strconv.Atoi(match[1])
	if err != nil {
		return v, fmt.Errorf("invalid major version: %w", err)
	}
	v.Minor, err = strconv.Atoi(match[2])
	if err != nil {
		return v, fmt.Errorf("invalid minor version: %w", err)
	}
	v.Patch, err = strconv.Atoi(match[3])
	if err != nil {
		return v, fmt.Errorf("invalid patch version: %w", err)
	}
	if match[4] != "" {
		v.Build, err = strconv.Atoi(match[4])
		if err != nil {
			return v, fmt.Errorf("invalid build version: %w", err)
		}
	}

	return v, nil
}

// Compare returns -1 if v < other, 0 if equal, 1 if v > other.
func (v Version) Compare(other Version) int {
	if v.Major != other.Major {
		return cmp(v.Major, other.Major)
	}
	if v.Minor != other.Minor {
		return cmp(v.Minor, other.Minor)
	}
	if v.Patch != other.Patch {
		return cmp(v.Patch, other.Patch)
	}
	return cmp(v.Build, other.Build)
}

// AtLeast returns true if v >= major.minor.
func (v Version) AtLeast(major, minor int) bool {
	if v.Major != major {
		return v.Major > major
	}
	return v.Minor >= minor
}

// String returns the version as "major.minor.patch.build".
func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d.%d", v.Major, v.Minor, v.Patch, v.Build)
}

func cmp(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
