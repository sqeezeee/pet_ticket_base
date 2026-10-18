package tickets

import (
	"context"
	"time"

	domain "pet-ticket/internal/domain/tickets"
)

// SLACalculator coordinates SLA rules from storage with domain calculations.
type SLACalculator struct {
	repo SLARuleRepository
	now  func() time.Time
}

type SLARuleRepository interface {
	GetSLARule(context.Context, int64, domain.Priority) (domain.SLARule, error)
}

func NewSLACalculator(repo SLARuleRepository) *SLACalculator {
	return &SLACalculator{repo: repo, now: time.Now}
}

// CalculateDeadlines sets both deadlines using the applicable SLA rule.
func (c *SLACalculator) CalculateDeadlines(ctx context.Context, ticket *domain.Ticket) error {
	rule, err := c.repo.GetSLARule(ctx, ticket.TopicID, ticket.Priority)
	if err != nil {
		return err
	}
	createdAt := ticket.CreatedAt
	if createdAt.IsZero() {
		createdAt = c.now()
		ticket.CreatedAt = createdAt
	}
	ticket.ResponseDeadline, ticket.ResolutionDeadline = rule.CalculateDeadlines(createdAt)
	return nil
}

// CheckFirstResponse records the first support response and never overwrites it.
func (c *SLACalculator) CheckFirstResponse(ticket *domain.Ticket, isSupport bool) bool {
	if !isSupport || ticket.IsFirstResponse() {
		return false
	}
	now := c.now()
	ticket.FirstResponseAt = &now
	return true
}

// CheckResolved records the first transition into resolved/closed.
func (c *SLACalculator) CheckResolved(ticket *domain.Ticket) bool {
	if !ticket.IsResolved() || ticket.ResolvedAt != nil {
		return false
	}
	now := c.now()
	ticket.ResolvedAt = &now
	return true
}
