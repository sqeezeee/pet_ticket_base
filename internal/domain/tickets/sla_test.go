package tickets

import (
	"testing"
	"time"
)

func TestCalculateSLAStatus(t *testing.T) {
	createdAt := time.Date(2026, 10, 18, 8, 0, 0, 0, time.UTC)
	base := Ticket{
		CreatedAt: createdAt, ResponseDeadline: createdAt.Add(time.Hour),
		ResolutionDeadline: createdAt.Add(10 * time.Hour),
	}
	tests := []struct {
		name string
		now  time.Time
		want SLAStatus
	}{
		{name: "ok", now: createdAt.Add(30 * time.Minute), want: SLAStatusOK},
		{name: "warning below twenty percent", now: createdAt.Add(49 * time.Minute), want: SLAStatusWarning},
		{name: "violated", now: createdAt.Add(61 * time.Minute), want: SLAStatusViolated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CalculateSLAStatus(base, tt.now).ResponseStatus; got != tt.want {
				t.Fatalf("response status = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCalculateSLAStatusKeepsLateCompletionViolated(t *testing.T) {
	createdAt := time.Date(2026, 10, 18, 8, 0, 0, 0, time.UTC)
	lateResponse := createdAt.Add(2 * time.Hour)
	ticket := Ticket{
		CreatedAt: createdAt, ResponseDeadline: createdAt.Add(time.Hour),
		ResolutionDeadline: createdAt.Add(4 * time.Hour), FirstResponseAt: &lateResponse,
	}
	metrics := CalculateSLAStatus(ticket, createdAt.Add(3*time.Hour))
	if metrics.ResponseStatus != SLAStatusViolated || metrics.OverallStatus != SLAStatusViolated {
		t.Fatalf("unexpected metrics: %+v", metrics)
	}
}
