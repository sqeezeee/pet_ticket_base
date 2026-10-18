package tickets

import (
	"errors"
	"time"
)

// Ticket представляет доменную сущность тикета
type Ticket struct {
	ID                 int64
	UserID             int64
	TopicID            int64
	Status             Status
	Priority           Priority
	Amount             *float64
	Comment            string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	ResponseDeadline   time.Time
	ResolutionDeadline time.Time
	FirstResponseAt    *time.Time
	ResolvedAt         *time.Time
}

// GetSLAStatus returns the ticket's current SLA metrics.
func (t Ticket) GetSLAStatus() SLAMetrics { return CalculateSLAStatus(t, time.Now()) }

// IsFirstResponse reports whether support has already answered the ticket.
func (t Ticket) IsFirstResponse() bool { return t.FirstResponseAt != nil }

// IsResolved reports whether the ticket has reached an SLA-completing status.
func (t Ticket) IsResolved() bool {
	return t.Status == StatusResolved || t.Status == StatusClosed
}

// Validate проверяет валидность тикета
func (t Ticket) Validate() error {
	if t.UserID <= 0 {
		return errors.New("invalid user_id")
	}
	if t.TopicID <= 0 {
		return errors.New("invalid topic_id")
	}
	if !t.Status.IsValid() {
		return errors.New("invalid status")
	}
	if !t.Priority.IsValid() {
		return errors.New("invalid priority")
	}
	if t.Amount != nil && *t.Amount < 0 {
		return errors.New("amount cannot be negative")
	}
	return nil
}

// IsNew проверяет, является ли тикет новым (не сохранённым)
func (t Ticket) IsNew() bool {
	return t.ID == 0
}
