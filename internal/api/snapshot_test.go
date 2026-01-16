package api

import (
	"testing"
)

func TestResolveSnapshotMethod(t *testing.T) {
	tests := []struct {
		name    string
		version Version
		method  string
		want    string
	}{
		{
			name:    "24.10 uses zfs.snapshot",
			version: Version{Major: 24, Minor: 10},
			method:  MethodSnapshotCreate,
			want:    "zfs.snapshot.create",
		},
		{
			name:    "25.04 uses zfs.snapshot",
			version: Version{Major: 25, Minor: 4},
			method:  MethodSnapshotQuery,
			want:    "zfs.snapshot.query",
		},
		{
			name:    "25.10 uses pool.snapshot",
			version: Version{Major: 25, Minor: 10},
			method:  MethodSnapshotCreate,
			want:    "pool.snapshot.create",
		},
		{
			name:    "26.0 uses pool.snapshot",
			version: Version{Major: 26, Minor: 0},
			method:  MethodSnapshotDelete,
			want:    "pool.snapshot.delete",
		},
		{
			name:    "hold method",
			version: Version{Major: 24, Minor: 10},
			method:  MethodSnapshotHold,
			want:    "zfs.snapshot.hold",
		},
		{
			name:    "release method",
			version: Version{Major: 25, Minor: 10},
			method:  MethodSnapshotRelease,
			want:    "pool.snapshot.release",
		},
		{
			name:    "clone method",
			version: Version{Major: 24, Minor: 10},
			method:  MethodSnapshotClone,
			want:    "zfs.snapshot.clone",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveSnapshotMethod(tt.version, tt.method); got != tt.want {
				t.Errorf("ResolveSnapshotMethod() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSnapshotResponse_HasHold(t *testing.T) {
	tests := []struct {
		name     string
		snapshot SnapshotResponse
		want     bool
	}{
		{
			name:     "no holds",
			snapshot: SnapshotResponse{Holds: nil},
			want:     false,
		},
		{
			name:     "empty holds map",
			snapshot: SnapshotResponse{Holds: map[string]any{}},
			want:     false,
		},
		{
			name:     "has holds",
			snapshot: SnapshotResponse{Holds: map[string]any{"terraform": true}},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.snapshot.HasHold(); got != tt.want {
				t.Errorf("HasHold() = %v, want %v", got, tt.want)
			}
		})
	}
}
