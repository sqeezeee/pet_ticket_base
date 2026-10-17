package tickets

import (
	"fmt"
	"strings"
)

// Priority represents a ticket priority level.
type Priority int

const (
	PriorityLow      Priority = 1
	PriorityMedium   Priority = 2
	PriorityHigh     Priority = 3
	PriorityCritical Priority = 4
)

// String returns the textual representation of a priority.
func (p Priority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityMedium:
		return "medium"
	case PriorityHigh:
		return "high"
	case PriorityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// IsValid reports whether the priority is supported by the domain.
func (p Priority) IsValid() bool {
	return p >= PriorityLow && p <= PriorityCritical
}

// Escalate raises the priority by one level, capped at Critical.
func (p Priority) Escalate() Priority {
	if !p.IsValid() {
		return p
	}
	if p == PriorityCritical {
		return PriorityCritical
	}
	return p + 1
}

// ParsePriority parses a textual priority name.
func ParsePriority(value string) (Priority, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case PriorityLow.String():
		return PriorityLow, nil
	case PriorityMedium.String():
		return PriorityMedium, nil
	case PriorityHigh.String():
		return PriorityHigh, nil
	case PriorityCritical.String():
		return PriorityCritical, nil
	default:
		return 0, fmt.Errorf("invalid priority %q", value)
	}
}
