package utils

import (
	"fmt"
)

// HumanizeDuration formats seconds to a human-friendly string.
func HumanizeDuration(s int) string {
	switch {
	case s < 60:
		return fmt.Sprintf("%ds", s)
	case s < 3600:
		return fmt.Sprintf("%dm", s/60)
	case s < 86400:
		return fmt.Sprintf("%dh", s/3600)
	case s < 604800:
		return fmt.Sprintf("%dd", s/86400)
	default:
		return fmt.Sprintf("%dw", s/604800)
	}
}
