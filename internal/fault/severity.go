package fault

import "strings"

// Severity ranks a fault from an internal code.
type SeverityRank int

const (
	RankLow SeverityRank = iota
	RankMedium
	RankHigh
	RankCritical
)

// SeverityForCode maps a fault code to a rank using its leading digit.
func SeverityForCode(code string) SeverityRank {
	switch {
	case strings.HasPrefix(code, "F1"):
		return RankCritical
	case strings.HasPrefix(code, "F2"):
		return RankHigh
	case strings.HasPrefix(code, "F3"):
		return RankMedium
	default:
		return RankLow
	}
}

// Escalate reports whether a fault of the given rank needs immediate attention.
func Escalate(r SeverityRank) bool { return r >= RankHigh }

// RankLabel returns a human-readable severity label.
func RankLabel(r SeverityRank) string {
	switch r {
	case RankCritical:
		return "critical"
	case RankHigh:
		return "high"
	case RankMedium:
		return "medium"
	default:
		return "low"
	}
}
