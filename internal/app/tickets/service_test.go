package tickets

import (
	"context"
	"errors"
	"io"
	"testing"

	domain "pet-ticket/internal/domain/tickets"

	"github.com/rs/zerolog"
)

// mockRepository — мок репозитория для тестов
//
//nolint:dupl // Interface and mock have similar structure by design
type mockRepository struct {
	createFunc      func(ctx context.Context, ticket domain.Ticket) (domain.Ticket, error)
	getByIDFunc     func(ctx context.Context, id int64) (domain.Ticket, error)
	updateFunc      func(ctx context.Context, ticket domain.Ticket) (domain.Ticket, error)
	deleteFunc      func(ctx context.Context, id int64) error
	listFunc        func(ctx context.Context, filter ListFilter) ([]domain.Ticket, error)
	addHistoryFunc  func(ctx context.Context, history domain.History) error
	getHistoryFunc  func(ctx context.Context, ticketID int64, limit, offset int) ([]domain.History, error)
	getStatusesFunc func(ctx context.Context) ([]StatusInfo, error)
	getTopicsFunc   func(ctx context.Context) ([]domain.Topic, error)
}

func (m *mockRepository) Create(ctx context.Context, ticket domain.Ticket) (domain.Ticket, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, ticket)
	}
	return domain.Ticket{}, errors.New("not implemented")
}

func (m *mockRepository) GetByID(ctx context.Context, id int64) (domain.Ticket, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return domain.Ticket{}, errors.New("not implemented")
}

func (m *mockRepository) Update(ctx context.Context, ticket domain.Ticket) (domain.Ticket, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, ticket)
	}
	return domain.Ticket{}, errors.New("not implemented")
}

func (m *mockRepository) Delete(ctx context.Context, id int64) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return errors.New("not implemented")
}

func (m *mockRepository) List(ctx context.Context, filter ListFilter) ([]domain.Ticket, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, filter)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRepository) AddHistory(ctx context.Context, history domain.History) error {
	if m.addHistoryFunc != nil {
		return m.addHistoryFunc(ctx, history)
	}
	return errors.New("not implemented")
}

func (m *mockRepository) GetHistory(ctx context.Context, ticketID int64, limit, offset int) ([]domain.History, error) {
	if m.getHistoryFunc != nil {
		return m.getHistoryFunc(ctx, ticketID, limit, offset)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRepository) GetAllStatuses(ctx context.Context) ([]StatusInfo, error) {
	if m.getStatusesFunc != nil {
		return m.getStatusesFunc(ctx)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRepository) GetAllTopics(ctx context.Context) ([]domain.Topic, error) {
	if m.getTopicsFunc != nil {
		return m.getTopicsFunc(ctx)
	}
	return nil, errors.New("not implemented")
}

// mockDB — мок для работы с транзакциями
type mockDB struct {
	beginTxFunc func(ctx context.Context) (TxCommitter, error)
}

func (m *mockDB) BeginTx(ctx context.Context) (TxCommitter, error) {
	if m.beginTxFunc != nil {
		return m.beginTxFunc(ctx)
	}
	return &mockTx{}, nil
}

// mockTx — мок транзакции
type mockTx struct {
	commitFunc   func() error
	rollbackFunc func() error
}

func (m *mockTx) Commit() error {
	if m.commitFunc != nil {
		return m.commitFunc()
	}
	return nil
}

func (m *mockTx) Rollback() error {
	if m.rollbackFunc != nil {
		return m.rollbackFunc()
	}
	return nil
}

// TestGetTicket_Success — успешное получение тикета
func TestGetTicket_Success(t *testing.T) {
	// Arrange
	expectedTicket := domain.Ticket{
		ID:       1,
		UserID:   100,
		TopicID:  1,
		Status:   domain.StatusNew,
		Priority: domain.PriorityMedium,
		Comment:  "Test ticket",
	}

	repo := &mockRepository{
		getByIDFunc: func(ctx context.Context, id int64) (domain.Ticket, error) {
			if id == 1 {
				return expectedTicket, nil
			}
			return domain.Ticket{}, ErrNotFound
		},
	}

	db := &mockDB{}
	logger := testLogger()
	svc := NewService(repo, db, logger)

	// Act
	ticket, err := svc.GetTicket(context.Background(), 1)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if ticket.ID != expectedTicket.ID {
		t.Errorf("expected ticket ID %d, got %d", expectedTicket.ID, ticket.ID)
	}
	if ticket.UserID != expectedTicket.UserID {
		t.Errorf("expected user ID %d, got %d", expectedTicket.UserID, ticket.UserID)
	}
}

// TestGetTicket_NotFound — тикет не найден
func TestGetTicket_NotFound(t *testing.T) {
	// Arrange
	repo := &mockRepository{
		getByIDFunc: func(ctx context.Context, id int64) (domain.Ticket, error) {
			return domain.Ticket{}, ErrNotFound
		},
	}

	db := &mockDB{}
	logger := testLogger()
	svc := NewService(repo, db, logger)

	// Act
	_, err := svc.GetTicket(context.Background(), 999)

	// Assert
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

// TestCreateTicket_Success — успешное создание тикета
func TestCreateTicket_Success(t *testing.T) {
	// Arrange
	var capturedTicket domain.Ticket
	var capturedHistory domain.History
	committed := false

	repo := &mockRepository{
		createFunc: func(ctx context.Context, ticket domain.Ticket) (domain.Ticket, error) {
			capturedTicket = ticket
			ticket.ID = 42
			return ticket, nil
		},
		addHistoryFunc: func(ctx context.Context, history domain.History) error {
			capturedHistory = history
			return nil
		},
	}

	db := &mockDB{
		beginTxFunc: func(ctx context.Context) (TxCommitter, error) {
			return &mockTx{
				commitFunc: func() error {
					committed = true
					return nil
				},
			}, nil
		},
	}

	logger := testLogger()
	svc := NewService(repo, db, logger)

	input := CreateTicketInput{
		UserID:  100,
		TopicID: 1,
		Comment: "Test ticket",
	}

	// Act
	ticket, err := svc.CreateTicket(context.Background(), input)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if ticket.ID != 42 {
		t.Errorf("expected ticket ID 42, got %d", ticket.ID)
	}
	if capturedTicket.UserID != 100 {
		t.Errorf("expected captured user ID 100, got %d", capturedTicket.UserID)
	}
	if capturedTicket.Priority != domain.PriorityMedium {
		t.Errorf("expected default priority medium, got %s", capturedTicket.Priority)
	}
	if capturedHistory.Action != domain.ActionCreated {
		t.Errorf("expected history action 'created', got %s", capturedHistory.Action)
	}
	if !committed {
		t.Error("expected transaction to be committed")
	}
}

// TestCreateTicket_InvalidInput — валидация входных данных
func TestCreateTicket_InvalidInput(t *testing.T) {
	// Arrange
	repo := &mockRepository{}
	db := &mockDB{}
	logger := testLogger()
	svc := NewService(repo, db, logger)

	tests := []struct {
		name  string
		input CreateTicketInput
	}{
		{
			name: "invalid user_id",
			input: CreateTicketInput{
				UserID:  0, // невалидный
				TopicID: 1,
				Comment: "Test",
			},
		},
		{
			name: "invalid topic_id",
			input: CreateTicketInput{
				UserID:  1,
				TopicID: 0, // невалидный
				Comment: "Test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			_, err := svc.CreateTicket(context.Background(), tt.input)

			// Assert
			if !errors.Is(err, ErrInvalidInput) {
				t.Errorf("expected ErrInvalidInput, got: %v", err)
			}
		})
	}
}

func TestCreateTicket_UsesExplicitPriority(t *testing.T) {
	priority := domain.PriorityHigh
	var captured domain.Ticket
	repo := &mockRepository{
		createFunc: func(_ context.Context, ticket domain.Ticket) (domain.Ticket, error) {
			captured = ticket
			ticket.ID = 1
			return ticket, nil
		},
		addHistoryFunc: func(context.Context, domain.History) error { return nil },
	}

	got, err := NewService(repo, &mockDB{}, testLogger()).CreateTicket(context.Background(), CreateTicketInput{
		UserID: 1, TopicID: 1, Priority: &priority, Comment: "High priority ticket",
	})
	if err != nil {
		t.Fatal(err)
	}
	if captured.Priority != domain.PriorityHigh || got.Priority != domain.PriorityHigh {
		t.Fatalf("priority = %s, want high", got.Priority)
	}
}

// TestUpdateTicket_StatusChange — обновление статуса с историей
func TestUpdateTicket_StatusChange(t *testing.T) {
	// Arrange
	existingTicket := domain.Ticket{
		ID:       1,
		UserID:   100,
		TopicID:  1,
		Status:   domain.StatusNew,
		Priority: domain.PriorityMedium,
		Comment:  "Original comment",
	}

	var capturedHistory domain.History
	committed := false

	repo := &mockRepository{
		getByIDFunc: func(ctx context.Context, id int64) (domain.Ticket, error) {
			return existingTicket, nil
		},
		updateFunc: func(ctx context.Context, ticket domain.Ticket) (domain.Ticket, error) {
			return ticket, nil
		},
		addHistoryFunc: func(ctx context.Context, history domain.History) error {
			capturedHistory = history
			return nil
		},
	}

	db := &mockDB{
		beginTxFunc: func(ctx context.Context) (TxCommitter, error) {
			return &mockTx{
				commitFunc: func() error {
					committed = true
					return nil
				},
			}, nil
		},
	}

	logger := testLogger()
	svc := NewService(repo, db, logger)

	newStatus := domain.StatusInProgress
	input := UpdateTicketInput{
		ID:     1,
		Status: &newStatus,
	}

	// Act
	ticket, err := svc.UpdateTicket(context.Background(), input)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if ticket.Status != domain.StatusInProgress {
		t.Errorf("expected status in_progress, got %s", ticket.Status.String())
	}
	if capturedHistory.Action != domain.ActionStatusChanged {
		t.Errorf("expected history action 'status_changed', got %s", capturedHistory.Action)
	}
	if capturedHistory.OldValue != "new" {
		t.Errorf("expected old value 'new', got %s", capturedHistory.OldValue)
	}
	if capturedHistory.NewValue != "in_progress" {
		t.Errorf("expected new value 'in_progress', got %s", capturedHistory.NewValue)
	}
	if !committed {
		t.Error("expected transaction to be committed")
	}
}

// TestUpdateTicket_TransactionRollback — откат при ошибке истории
func TestUpdateTicket_TransactionRollback(t *testing.T) {
	// Arrange
	existingTicket := domain.Ticket{
		ID:       1,
		UserID:   100,
		TopicID:  1,
		Status:   domain.StatusNew,
		Priority: domain.PriorityMedium,
		Comment:  "Original",
	}

	rolledBack := false

	repo := &mockRepository{
		getByIDFunc: func(ctx context.Context, id int64) (domain.Ticket, error) {
			return existingTicket, nil
		},
		updateFunc: func(ctx context.Context, ticket domain.Ticket) (domain.Ticket, error) {
			return ticket, nil
		},
		addHistoryFunc: func(ctx context.Context, history domain.History) error {
			return errors.New("database error")
		},
	}

	db := &mockDB{
		beginTxFunc: func(ctx context.Context) (TxCommitter, error) {
			return &mockTx{
				rollbackFunc: func() error {
					rolledBack = true
					return nil
				},
			}, nil
		},
	}

	logger := testLogger()
	svc := NewService(repo, db, logger)

	newStatus := domain.StatusInProgress
	input := UpdateTicketInput{
		ID:     1,
		Status: &newStatus,
	}

	// Act
	_, err := svc.UpdateTicket(context.Background(), input)

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !rolledBack {
		t.Error("expected transaction to be rolled back")
	}
}

func TestEscalateTicket_MediumToHigh(t *testing.T) {
	existing := domain.Ticket{
		ID: 1, UserID: 100, TopicID: 1, Status: domain.StatusNew,
		Priority: domain.PriorityMedium, Comment: "Test ticket",
	}
	var updated domain.Ticket
	var history domain.History
	committed := false
	repo := &mockRepository{
		getByIDFunc: func(ctx context.Context, id int64) (domain.Ticket, error) {
			if TxFromContext(ctx) == nil {
				t.Fatal("expected transaction in repository context")
			}
			return existing, nil
		},
		updateFunc: func(ctx context.Context, ticket domain.Ticket) (domain.Ticket, error) {
			updated = ticket
			return ticket, nil
		},
		addHistoryFunc: func(ctx context.Context, value domain.History) error {
			history = value
			return nil
		},
	}
	db := &mockDB{beginTxFunc: func(context.Context) (TxCommitter, error) {
		return &mockTx{commitFunc: func() error { committed = true; return nil }}, nil
	}}

	got, err := NewService(repo, db, testLogger()).EscalateTicket(context.Background(), existing.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Priority != domain.PriorityHigh || updated.Priority != domain.PriorityHigh {
		t.Fatalf("priority = %s, want high", got.Priority)
	}
	if history.Action != domain.ActionPriorityEscalated || history.OldValue != "medium" || history.NewValue != "high" {
		t.Fatalf("unexpected history: %+v", history)
	}
	if !committed {
		t.Fatal("expected transaction commit")
	}
}

func TestUpdatePriority_RecordsHistory(t *testing.T) {
	existing := domain.Ticket{
		ID: 1, UserID: 100, TopicID: 1, Status: domain.StatusNew,
		Priority: domain.PriorityLow, Comment: "Test ticket",
	}
	var history domain.History
	repo := &mockRepository{
		getByIDFunc: func(context.Context, int64) (domain.Ticket, error) { return existing, nil },
		updateFunc:  func(_ context.Context, ticket domain.Ticket) (domain.Ticket, error) { return ticket, nil },
		addHistoryFunc: func(_ context.Context, value domain.History) error {
			history = value
			return nil
		},
	}

	got, err := NewService(repo, &mockDB{}, testLogger()).UpdatePriority(
		context.Background(), existing.ID, domain.PriorityCritical,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Priority != domain.PriorityCritical {
		t.Fatalf("priority = %s, want critical", got.Priority)
	}
	if history.Action != domain.ActionPriorityChanged || history.OldValue != "low" || history.NewValue != "critical" {
		t.Fatalf("unexpected history: %+v", history)
	}
}

func TestEscalateTicket_CriticalIsNoOp(t *testing.T) {
	existing := domain.Ticket{
		ID: 1, UserID: 100, TopicID: 1, Status: domain.StatusNew,
		Priority: domain.PriorityCritical, Comment: "Test ticket",
	}
	repo := &mockRepository{
		getByIDFunc: func(context.Context, int64) (domain.Ticket, error) { return existing, nil },
		updateFunc: func(context.Context, domain.Ticket) (domain.Ticket, error) {
			t.Fatal("critical ticket must not be updated")
			return domain.Ticket{}, nil
		},
		addHistoryFunc: func(context.Context, domain.History) error {
			t.Fatal("critical no-op must not create history")
			return nil
		},
	}

	got, err := NewService(repo, &mockDB{}, testLogger()).EscalateTicket(context.Background(), existing.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Priority != domain.PriorityCritical {
		t.Fatalf("priority = %s, want critical", got.Priority)
	}
}

func TestEscalateTicket_RejectsTerminalStatuses(t *testing.T) {
	for _, status := range []domain.Status{domain.StatusClosed, domain.StatusCancelled} {
		t.Run(status.String(), func(t *testing.T) {
			repo := &mockRepository{getByIDFunc: func(context.Context, int64) (domain.Ticket, error) {
				return domain.Ticket{ID: 1, UserID: 1, TopicID: 1, Status: status, Priority: domain.PriorityMedium}, nil
			}}
			_, err := NewService(repo, &mockDB{}, testLogger()).EscalateTicket(context.Background(), 1)
			if !errors.Is(err, ErrConflict) {
				t.Fatalf("error = %v, want ErrConflict", err)
			}
		})
	}
}

func TestEscalateTicket_HistoryFailureRollsBack(t *testing.T) {
	rolledBack := false
	repo := &mockRepository{
		getByIDFunc: func(context.Context, int64) (domain.Ticket, error) {
			return domain.Ticket{ID: 1, UserID: 1, TopicID: 1, Status: domain.StatusNew, Priority: domain.PriorityMedium}, nil
		},
		updateFunc:     func(_ context.Context, ticket domain.Ticket) (domain.Ticket, error) { return ticket, nil },
		addHistoryFunc: func(context.Context, domain.History) error { return errors.New("history failed") },
	}
	db := &mockDB{beginTxFunc: func(context.Context) (TxCommitter, error) {
		return &mockTx{rollbackFunc: func() error { rolledBack = true; return nil }}, nil
	}}

	if _, err := NewService(repo, db, testLogger()).EscalateTicket(context.Background(), 1); err == nil {
		t.Fatal("expected history error")
	}
	if !rolledBack {
		t.Fatal("expected transaction rollback")
	}
}

// testLogger возвращает no-op logger для тестов
func testLogger() zerolog.Logger {
	return zerolog.New(io.Discard)
}
