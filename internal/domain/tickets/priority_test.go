package tickets

import "testing"

func TestPriority(t *testing.T) {
	tests := []struct {
		priority  Priority
		name      string
		escalated Priority
	}{
		{PriorityLow, "low", PriorityMedium},
		{PriorityMedium, "medium", PriorityHigh},
		{PriorityHigh, "high", PriorityCritical},
		{PriorityCritical, "critical", PriorityCritical},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.priority.IsValid() {
				t.Fatalf("priority %d must be valid", tt.priority)
			}
			if got := tt.priority.String(); got != tt.name {
				t.Fatalf("String() = %q, want %q", got, tt.name)
			}
			if got := tt.priority.Escalate(); got != tt.escalated {
				t.Fatalf("Escalate() = %v, want %v", got, tt.escalated)
			}
			parsed, err := ParsePriority(tt.name)
			if err != nil || parsed != tt.priority {
				t.Fatalf("ParsePriority(%q) = %v, %v", tt.name, parsed, err)
			}
		})
	}
}

func TestPriorityRejectsInvalidValues(t *testing.T) {
	for _, priority := range []Priority{0, 5, -1} {
		if priority.IsValid() {
			t.Errorf("priority %d must be invalid", priority)
		}
	}
	if _, err := ParsePriority("urgent"); err == nil {
		t.Fatal("expected invalid priority parse error")
	}
}

func TestTicketValidateRejectsInvalidPriority(t *testing.T) {
	ticket := Ticket{UserID: 1, TopicID: 1, Status: StatusNew}
	if err := ticket.Validate(); err == nil {
		t.Fatal("expected invalid priority error")
	}
}
