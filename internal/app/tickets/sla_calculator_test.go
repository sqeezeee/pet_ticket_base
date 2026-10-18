package tickets

import (
	"context"
	"testing"
	"time"

	domain "pet-ticket/internal/domain/tickets"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type slaRuleRepositoryMock struct{ mock.Mock }

func (m *slaRuleRepositoryMock) GetSLARule(
	ctx context.Context,
	topicID int64,
	priority domain.Priority,
) (domain.SLARule, error) {
	args := m.Called(ctx, topicID, priority)
	return args.Get(0).(domain.SLARule), args.Error(1)
}

func TestSLACalculatorCalculateDeadlines(t *testing.T) {
	createdAt := time.Date(2026, 10, 18, 9, 0, 0, 0, time.UTC)
	repo := new(slaRuleRepositoryMock)
	repo.On("GetSLARule", mock.Anything, int64(7), domain.PriorityHigh).Return(domain.SLARule{
		TopicID: 7, Priority: domain.PriorityHigh,
		ResponseTime: 30 * time.Minute, ResolutionTime: 4 * time.Hour,
	}, nil).Once()

	ticket := domain.Ticket{TopicID: 7, Priority: domain.PriorityHigh, CreatedAt: createdAt}
	err := NewSLACalculator(repo).CalculateDeadlines(context.Background(), &ticket)

	require.NoError(t, err)
	require.Equal(t, createdAt.Add(30*time.Minute), ticket.ResponseDeadline)
	require.Equal(t, createdAt.Add(4*time.Hour), ticket.ResolutionDeadline)
	repo.AssertExpectations(t)
}

func TestSLACalculatorRecordsOnlyFirstSupportResponse(t *testing.T) {
	fixedNow := time.Date(2026, 10, 18, 10, 0, 0, 0, time.UTC)
	calculator := NewSLACalculator(new(slaRuleRepositoryMock))
	calculator.now = func() time.Time { return fixedNow }
	ticket := domain.Ticket{}

	require.False(t, calculator.CheckFirstResponse(&ticket, false))
	require.Nil(t, ticket.FirstResponseAt)
	require.True(t, calculator.CheckFirstResponse(&ticket, true))
	require.Equal(t, fixedNow, *ticket.FirstResponseAt)
	require.False(t, calculator.CheckFirstResponse(&ticket, true))
}

func TestSLACalculatorRecordsResolvedAt(t *testing.T) {
	fixedNow := time.Date(2026, 10, 18, 11, 0, 0, 0, time.UTC)
	calculator := NewSLACalculator(new(slaRuleRepositoryMock))
	calculator.now = func() time.Time { return fixedNow }
	ticket := domain.Ticket{Status: domain.StatusInProgress}

	require.False(t, calculator.CheckResolved(&ticket))
	ticket.Status = domain.StatusResolved
	require.True(t, calculator.CheckResolved(&ticket))
	require.Equal(t, fixedNow, *ticket.ResolvedAt)
	require.False(t, calculator.CheckResolved(&ticket))
}
