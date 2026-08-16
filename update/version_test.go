package update

import "testing"

// The case that matters is the snapshot one. make build stamps a local build
// as "0.3.1-next", which is not equal to the "0.3.0" GitHub reports. The string
// comparison this replaced therefore called a newer build outdated and would
// have installed the older release over it, which the About tab now offers to do
// with one click.
func TestCompare(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want int
	}{
		{"0.3.0", "0.3.0", 0},
		{"v0.4.0", "0.4.0", 0},
		{"0.4.0", "v0.4.0", 0},

		{"0.3.0", "0.4.0", -1},
		{"0.4.0", "0.3.0", 1},
		{"0.9.0", "0.10.0", -1},
		{"1.0.0", "0.99.99", 1},
		{"0.3.1", "0.3.2", -1},

		// A snapshot sits between the release it follows and the one it becomes.
		{"0.3.1-next", "0.3.0", 1},
		{"0.3.1-next", "0.3.1", -1},
		{"0.3.1", "0.3.1-next", 1},
		{"0.3.1-next", "0.3.1-next", 0},
		{"0.4.0-rc1", "0.4.0", -1},

		// Shorter forms and build metadata still order sensibly.
		{"1.0", "1.0.0", 0},
		{"0.3.0+build7", "0.3.0", -1},
	} {
		if got := Compare(tc.a, tc.b); got != tc.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestIsDevelopmentBuild(t *testing.T) {
	for _, version := range []string{"dev", ""} {
		if !IsDevelopmentBuild(version) {
			t.Errorf("IsDevelopmentBuild(%q) = false, want true", version)
		}
	}
	for _, version := range []string{"0.3.0", "0.3.1-next", "v1.0.0"} {
		if IsDevelopmentBuild(version) {
			t.Errorf("IsDevelopmentBuild(%q) = true, want false", version)
		}
	}
}
