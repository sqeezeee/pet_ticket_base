package tickets

import "time"

// SLARule defines the response and resolution targets for a topic/priority pair.
type SLARule struct {
	ID             int64
	TopicID        int64
	Priority       Priority
	ResponseTime   time.Duration
	ResolutionTime time.Duration
}

// CalculateDeadlines converts the relative targets into absolute deadlines.
func (r SLARule) CalculateDeadlines(createdAt time.Time) (time.Time, time.Time) {
	return createdAt.Add(r.ResponseTime), createdAt.Add(r.ResolutionTime)
}

// SLAStatus is the current state of a single SLA target.
type SLAStatus string

const (
	SLAStatusOK       SLAStatus = "ok"
	SLAStatusWarning  SLAStatus = "warning"
	SLAStatusViolated SLAStatus = "violated"
)

// SLAMetrics contains the response, resolution and aggregate SLA states.
type SLAMetrics struct {
	ResponseStatus   SLAStatus
	ResolutionStatus SLAStatus
	OverallStatus    SLAStatus
}

// CalculateSLAStatus calculates SLA state at a supplied point in time. Supplying
// now makes the domain logic deterministic and straightforward to test.
func CalculateSLAStatus(ticket Ticket, now time.Time) SLAMetrics {
	response := calculateMetricStatus(ticket.CreatedAt, ticket.ResponseDeadline, ticket.FirstResponseAt, now)
	resolution := calculateMetricStatus(ticket.CreatedAt, ticket.ResolutionDeadline, ticket.ResolvedAt, now)
	overall := response
	if slaSeverity(resolution) > slaSeverity(overall) {
		overall = resolution
	}

	return SLAMetrics{ResponseStatus: response, ResolutionStatus: resolution, OverallStatus: overall}
}

func calculateMetricStatus(createdAt, deadline time.Time, completedAt *time.Time, now time.Time) SLAStatus {
	if deadline.IsZero() {
		return SLAStatusOK
	}
	if completedAt != nil {
		if completedAt.After(deadline) {
			return SLAStatusViolated
		}
		return SLAStatusOK
	}
	if now.After(deadline) {
		return SLAStatusViolated
	}

	total := deadline.Sub(createdAt)
	if total > 0 && deadline.Sub(now) < total/5 {
		return SLAStatusWarning
	}
	return SLAStatusOK
}

func slaSeverity(status SLAStatus) int {
	switch status {
	case SLAStatusViolated:
		return 2
	case SLAStatusWarning:
		return 1
	default:
		return 0
	}
}
