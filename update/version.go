package update

import (
	"strconv"
	"strings"
)

// developmentVersion is what the binary reports when it was built without the
// release ldflags: go run ./cmd/simdiag, and the default in common/types.go.
const developmentVersion = "dev"

// IsDevelopmentBuild reports a version no release can be ordered against.
func IsDevelopmentBuild(version string) bool {
	return strings.TrimSpace(version) == "" || version == developmentVersion
}

// Compare orders two SimDiag versions, returning -1, 0 or 1.
//
// String equality is not enough, and getting it wrong points the wrong way. A
// snapshot is stamped "0.3.1-next" (.goreleaser.yml, snapshot.version_template)
// and never equals the "0.3.0" GitHub reports. The equality test this replaces
// therefore called a newer local build outdated and would have installed the
// older release over it.
//
// The rule is the semantic-versioning one, narrowed to what SimDiag produces:
// compare major, minor and patch numerically, and on a tie a version carrying a
// pre-release suffix loses. 0.3.1-next sits between 0.3.0 and 0.3.1.
func Compare(a, b string) int {
	majorA, preA := parseVersion(a)
	majorB, preB := parseVersion(b)

	for i := range majorA {
		if majorA[i] != majorB[i] {
			if majorA[i] < majorB[i] {
				return -1
			}
			return 1
		}
	}

	switch {
	case preA == preB:
		return 0
	case preA:
		// Same numbers, but this one is still on its way there.
		return -1
	default:
		return 1
	}
}

// parseVersion splits a version into its three numbers and whether it carries a
// pre-release suffix. Anything unparseable counts as zero, which keeps a
// malformed version older than a real one rather than making it win.
func parseVersion(version string) (numbers [3]int, prerelease bool) {
	version = strings.TrimSpace(version)
	version = strings.TrimPrefix(version, "v")

	// Both the "-next" of a snapshot and a hand-made "-rc1" land here.
	if cut := strings.IndexAny(version, "-+"); cut >= 0 {
		version = version[:cut]
		prerelease = true
	}

	for i, part := range strings.SplitN(version, ".", 3) {
		if i >= len(numbers) {
			break
		}
		numbers[i], _ = strconv.Atoi(part)
	}

	return numbers, prerelease
}
