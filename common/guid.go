package common

import "strings"

// NormalizeGUID normalizes a GUID for comparison: trims braces and lowercases.
// Example: "{EE6F1C30-3F2E-11F0-8001-444553540000}" -> "ee6f1c30-3f2e-11f0-8001-444553540000"
func NormalizeGUID(guid string) string {
	return strings.ToLower(strings.Trim(guid, "{}"))
}

// NormalizeGUIDUpper normalizes a GUID for comparison: trims braces and uppercases.
// Used by Gremlins and OpenKneeboard which store GUIDs in uppercase.
// Example: "{ee6f1c30-3f2e-11f0-8001-444553540000}" -> "EE6F1C30-3F2E-11F0-8001-444553540000"
func NormalizeGUIDUpper(guid string) string {
	return strings.ToUpper(strings.Trim(guid, "{}"))
}

// NormalizeGUIDBraced normalizes a GUID to uppercase with braces (Gremlins format).
// Example: "ee6f1c30-3f2e-11f0-8001-444553540000" -> "{EE6F1C30-3F2E-11F0-8001-444553540000}"
func NormalizeGUIDBraced(guid string) string {
	guid = strings.ToUpper(guid)
	if !strings.HasPrefix(guid, "{") {
		guid = "{" + guid
	}
	if !strings.HasSuffix(guid, "}") {
		guid += "}"
	}
	return guid
}

// NormalizeGUIDShort returns the first 8 characters of a normalized GUID (device ID prefix).
// Example: "{EE6F1C30-3F2E-11F0-8001-444553540000}" -> "ee6f1c30"
func NormalizeGUIDShort(guid string) string {
	normalized := NormalizeGUID(guid)
	if len(normalized) >= 8 {
		return normalized[:8]
	}
	return normalized
}

// MatchGUIDPartial reports whether two GUIDs name the same physical controller.
//
// - Same segment count (both 5 or both 4): exact match on all segments
// - Mixed segment count (5 vs 4): partial match on first 3 segments (handles IL-2 vs DCS)
//
// This prevents false positives between different devices with the same first 3 segments,
// while still allowing cross-simulator matching (DCS 5-segment vs IL-2 4-segment).
// Example: "EE6F1C30-3F2E-11F0-8001-444553540000" matches "EE6F1C30-3F2E-11F0-0000545345440180"
//
// It already answers the exact case, braces and letter case included, so callers
// do not need to try an equality test first. Three of the four enrichers did,
// each spelling the fallback slightly differently.
func MatchGUIDPartial(guid1, guid2 string) bool {
	clean1 := strings.ToUpper(strings.Trim(guid1, "{}"))
	clean2 := strings.ToUpper(strings.Trim(guid2, "{}"))

	parts1 := strings.Split(clean1, "-")
	parts2 := strings.Split(clean2, "-")

	if len(parts1) < 3 || len(parts2) < 3 {
		return false
	}

	// Same format: require exact match on all segments
	if len(parts1) == len(parts2) {
		for i := range parts1 {
			if parts1[i] != parts2[i] {
				return false
			}
		}
		return true
	}

	// Different format (cross-simulator): match first 3 segments only
	return parts1[0] == parts2[0] && parts1[1] == parts2[1] && parts1[2] == parts2[2]
}
