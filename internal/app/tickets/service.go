package tickets

import (
	"context"
	"fmt"
	"time"

	"pet-ticket/internal/domain/tickets"

	"github.com/rs/zerolog"
)

// CreateTicketInput представляет входные данные для создания тикета
type CreateTicketInput struct {
	UserID   int64
	TopicID  int64
	Priority tickets.Priority
	Amount   *float64
	Comment  string
}

// UpdateTicketInput представляет входные данные для обновления тикета
type UpdateTicketInput struct {
	ID      int64
	Status  *tickets.Status
	Comment *string
}

// AddCommentInput contains a comment author and whether that author is support.
type AddCommentInput struct {
	TicketID  int64
	UserID    int64
	Comment   string
	IsSupport bool
}

// ListTicketsInput представляет входные данные для получения списка тикетов
type ListTicketsInput struct {
	UserID   *int64
	TopicID  *int64
	Status   *tickets.Status
	Priority *tickets.Priority
	Limit    int
	Offset   int
	SortBy   string
	SortDesc bool
}

// Service определяет бизнес-логику работы с тикетами
type Service interface {
	CreateTicket(ctx context.Context, input CreateTicketInput) (tickets.Ticket, error)
	GetTicket(ctx context.Context, id int64) (tickets.Ticket, error)
	UpdateTicket(ctx context.Context, input UpdateTicketInput) (tickets.Ticket, error)
	UpdatePriority(ctx context.Context, ticketID int64, priority tickets.Priority) (tickets.Ticket, error)
	EscalateTicket(ctx context.Context, ticketID int64) (tickets.Ticket, error)
	DeleteTicket(ctx context.Context, id int64) error
	ListTickets(ctx context.Context, input ListTicketsInput) ([]tickets.Ticket, error)
	GetTicketHistory(ctx context.Context, ticketID int64, limit, offset int) ([]tickets.History, error)
	GetAllStatuses(ctx context.Context) ([]StatusInfo, error)
	GetAllTopics(ctx context.Context) ([]tickets.Topic, error)
	AddComment(ctx context.Context, input AddCommentInput) (tickets.Ticket, error)
	GetSLAViolations(ctx context.Context) ([]tickets.Ticket, error)
}

// service реализует интерфейс Service
type service struct {
	repo   Repository
	db     TxBeginner
	logger zerolog.Logger
	sla    *SLACalculator
}

// TxBeginner интерфейс для создания транзакций
type TxBeginner interface {
	BeginTx(context.Context) (TxCommitter, error)
}

// TxCommitter интерфейс для работы с транзакциями
type TxCommitter interface {
	Commit() error
	Rollback() error
}

// NewService создаёт новый экземпляр сервиса
func NewService(repo Repository, db TxBeginner, logger zerolog.Logger) Service {
	return &service{
		repo:   repo,
		db:     db,
		logger: logger,
		sla:    NewSLACalculator(repo),
	}
}

// CreateTicket создаёт новый тикет
func (s *service) CreateTicket(ctx context.Context, input CreateTicketInput) (tickets.Ticket, error) {
	priority := input.Priority
	if priority == 0 {
		priority = tickets.PriorityMedium
	} else if !priority.IsValid() {
		return tickets.Ticket{}, ErrInvalidPriority
	}

	// Создаём доменную сущность
	ticket := tickets.Ticket{
		UserID:   input.UserID,
		TopicID:  input.TopicID,
		Status:   tickets.StatusNew,
		Priority: priority,
		Amount:   input.Amount,
		Comment:  input.Comment,
	}

	// Валидация доменной модели
	if err := ticket.Validate(); err != nil {
		s.logger.Warn().Err(err).Msg("invalid ticket data")
		return tickets.Ticket{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	if err := s.sla.CalculateDeadlines(ctx, &ticket); err != nil {
		return tickets.Ticket{}, fmt.Errorf("failed to calculate SLA deadlines: %w", err)
	}

	// Начинаем транзакцию
	tx, err := s.db.BeginTx(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to begin transaction")
		return tickets.Ticket{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Контекст с транзакцией
	txCtx := ContextWithTx(ctx, tx)

	// Сохранение в репозиторий
	created, err := s.repo.Create(txCtx, ticket)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to create ticket")
		return tickets.Ticket{}, fmt.Errorf("failed to create ticket: %w", err)
	}

	// Добавление записи в историю
	history := tickets.History{
		TicketID: created.ID,
		UserID:   input.UserID,
		Action:   tickets.ActionCreated,
		NewValue: fmt.Sprintf("status=%s", created.Status.String()),
	}
	if err = s.repo.AddHistory(txCtx, history); err != nil {
		s.logger.Error().Err(err).Int64("ticket_id", created.ID).Msg("failed to add history")
		return tickets.Ticket{}, fmt.Errorf("failed to add history: %w", err)
	}

	// Коммит транзакции
	if err = tx.Commit(); err != nil {
		s.logger.Error().Err(err).Msg("failed to commit transaction")
		return tickets.Ticket{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Info().Int64("ticket_id", created.ID).Msg("ticket created")
	return created, nil
}

// UpdatePriority changes a ticket priority and records the change in history.
func (s *service) UpdatePriority(
	ctx context.Context,
	ticketID int64,
	priority tickets.Priority,
) (tickets.Ticket, error) {
	if !priority.IsValid() {
		return tickets.Ticket{}, ErrInvalidPriority
	}

	return s.changePriority(ctx, ticketID, priority, tickets.ActionPriorityChanged, false)
}

// EscalateTicket raises a ticket priority by one level.
func (s *service) EscalateTicket(ctx context.Context, ticketID int64) (tickets.Ticket, error) {
	return s.changePriority(ctx, ticketID, 0, tickets.ActionPriorityEscalated, true)
}

func (s *service) changePriority(
	ctx context.Context,
	ticketID int64,
	priority tickets.Priority,
	action tickets.HistoryAction,
	escalate bool,
) (tickets.Ticket, error) {
	tx, err := s.db.BeginTx(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to begin priority transaction")
		return tickets.Ticket{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := ContextWithTx(ctx, tx)
	existing, err := s.repo.GetByID(txCtx, ticketID)
	if err != nil {
		return tickets.Ticket{}, fmt.Errorf("failed to get ticket: %w", err)
	}

	if escalate {
		if existing.Status == tickets.StatusClosed || existing.Status == tickets.StatusCancelled {
			return tickets.Ticket{}, ErrConflict
		}
		priority = existing.Priority.Escalate()
	}

	if !priority.IsValid() {
		return tickets.Ticket{}, ErrInvalidPriority
	}
	if priority == existing.Priority {
		return existing, nil
	}

	oldPriority := existing.Priority
	existing.Priority = priority
	if err := s.sla.CalculateDeadlines(txCtx, &existing); err != nil {
		return tickets.Ticket{}, fmt.Errorf("failed to recalculate SLA deadlines: %w", err)
	}
	if err := existing.Validate(); err != nil {
		return tickets.Ticket{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	updated, err := s.repo.Update(txCtx, existing)
	if err != nil {
		return tickets.Ticket{}, fmt.Errorf("failed to update ticket priority: %w", err)
	}

	history := tickets.History{
		TicketID: updated.ID,
		UserID:   updated.UserID,
		Action:   action,
		OldValue: oldPriority.String(),
		NewValue: priority.String(),
	}
	if err = s.repo.AddHistory(txCtx, history); err != nil {
		return tickets.Ticket{}, fmt.Errorf("failed to add priority history: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return tickets.Ticket{}, fmt.Errorf("failed to commit priority transaction: %w", err)
	}

	s.logger.Info().
		Int64("ticket_id", updated.ID).
		Str("old_priority", oldPriority.String()).
		Str("new_priority", priority.String()).
		Msg("ticket priority changed")

	return updated, nil
}

// GetTicket возвращает тикет по ID
func (s *service) GetTicket(ctx context.Context, id int64) (tickets.Ticket, error) {
	ticket, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error().Err(err).Int64("id", id).Msg("failed to get ticket")
		return tickets.Ticket{}, fmt.Errorf("failed to get ticket: %w", err)
	}

	return ticket, nil
}

// UpdateTicket обновляет существующий тикет
func (s *service) UpdateTicket(ctx context.Context, input UpdateTicketInput) (tickets.Ticket, error) {
	// Начинаем транзакцию
	tx, err := s.db.BeginTx(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to begin transaction")
		return tickets.Ticket{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Контекст с транзакцией
	txCtx := ContextWithTx(ctx, tx)

	// Получаем существующий тикет
	existing, err := s.repo.GetByID(txCtx, input.ID)
	if err != nil {
		s.logger.Error().Err(err).Int64("id", input.ID).Msg("failed to get ticket for update")
		return tickets.Ticket{}, fmt.Errorf("failed to get ticket: %w", err)
	}

	oldStatus := existing.Status

	// Обновляем поля
	if input.Status != nil {
		if !input.Status.IsValid() {
			return tickets.Ticket{}, ErrInvalidStatus
		}
		existing.Status = *input.Status
		s.sla.CheckResolved(&existing)
	}

	if input.Comment != nil {
		existing.Comment = *input.Comment
	}

	// Валидация
	if err := existing.Validate(); err != nil {
		s.logger.Warn().Err(err).Msg("invalid ticket data after update")
		return tickets.Ticket{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	// Сохранение
	updated, err := s.repo.Update(txCtx, existing)
	if err != nil {
		s.logger.Error().Err(err).Int64("id", input.ID).Msg("failed to update ticket")
		return tickets.Ticket{}, fmt.Errorf("failed to update ticket: %w", err)
	}

	// Добавление записи в историю при изменении статуса
	if input.Status != nil && oldStatus != *input.Status {
		history := tickets.History{
			TicketID: updated.ID,
			UserID:   updated.UserID,
			Action:   tickets.ActionStatusChanged,
			OldValue: oldStatus.String(),
			NewValue: input.Status.String(),
		}
		if err = s.repo.AddHistory(txCtx, history); err != nil {
			s.logger.Error().Err(err).Int64("ticket_id", updated.ID).Msg("failed to add history")
			return tickets.Ticket{}, fmt.Errorf("failed to add history: %w", err)
		}
		if updated.IsResolved() && oldStatus != tickets.StatusResolved && oldStatus != tickets.StatusClosed {
			if err = s.repo.AddHistory(txCtx, tickets.History{
				TicketID: updated.ID,
				UserID:   updated.UserID,
				Action:   tickets.ActionResolved,
				NewValue: updated.Status.String(),
			}); err != nil {
				return tickets.Ticket{}, fmt.Errorf("failed to add resolved history: %w", err)
			}
		}
	}

	// Добавление записи в историю при изменении комментария
	if input.Comment != nil {
		history := tickets.History{
			TicketID: updated.ID,
			UserID:   updated.UserID,
			Action:   tickets.ActionCommentAdded,
			NewValue: *input.Comment,
		}
		if err = s.repo.AddHistory(txCtx, history); err != nil {
			s.logger.Error().Err(err).Int64("ticket_id", updated.ID).Msg("failed to add history for comment")
			return tickets.Ticket{}, fmt.Errorf("failed to add history for comment: %w", err)
		}
	}

	// Коммит транзакции
	if err = tx.Commit(); err != nil {
		s.logger.Error().Err(err).Msg("failed to commit transaction")
		return tickets.Ticket{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Info().Int64("ticket_id", updated.ID).Msg("ticket updated")
	return updated, nil
}

// DeleteTicket удаляет тикет
func (s *service) DeleteTicket(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error().Err(err).Int64("id", id).Msg("failed to delete ticket")
		return fmt.Errorf("failed to delete ticket: %w", err)
	}

	s.logger.Info().Int64("ticket_id", id).Msg("ticket deleted")
	return nil
}

// ListTickets возвращает список тикетов
func (s *service) ListTickets(ctx context.Context, input ListTicketsInput) ([]tickets.Ticket, error) {
	filter := ListFilter(input)

	list, err := s.repo.List(ctx, filter)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to list tickets")
		return nil, fmt.Errorf("failed to list tickets: %w", err)
	}

	return list, nil
}

// GetTicketHistory возвращает историю изменений тикета
func (s *service) GetTicketHistory(ctx context.Context, ticketID int64, limit, offset int) ([]tickets.History, error) {
	history, err := s.repo.GetHistory(ctx, ticketID, limit, offset)
	if err != nil {
		s.logger.Error().Err(err).Int64("ticket_id", ticketID).Msg("failed to get ticket history")
		return nil, fmt.Errorf("failed to get ticket history: %w", err)
	}

	return history, nil
}

// GetAllStatuses возвращает все статусы тикетов
func (s *service) GetAllStatuses(ctx context.Context) ([]StatusInfo, error) {
	statuses, err := s.repo.GetAllStatuses(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get statuses")
		return nil, fmt.Errorf("failed to get statuses: %w", err)
	}

	return statuses, nil
}

// GetAllTopics возвращает все темы тикетов
func (s *service) GetAllTopics(ctx context.Context) ([]tickets.Topic, error) {
	topics, err := s.repo.GetAllTopics(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get topics")
		return nil, fmt.Errorf("failed to get topics: %w", err)
	}

	return topics, nil
}

// AddComment adds a ticket comment and records the first support response.
func (s *service) AddComment(ctx context.Context, input AddCommentInput) (tickets.Ticket, error) {
	if input.TicketID <= 0 || input.UserID <= 0 || input.Comment == "" {
		return tickets.Ticket{}, ErrInvalidInput
	}
	tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return tickets.Ticket{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := ContextWithTx(ctx, tx)
	ticket, err := s.repo.GetByID(txCtx, input.TicketID)
	if err != nil {
		return tickets.Ticket{}, fmt.Errorf("failed to get ticket: %w", err)
	}
	ticket.Comment = input.Comment
	firstResponse := s.sla.CheckFirstResponse(&ticket, input.IsSupport)
	updated, err := s.repo.Update(txCtx, ticket)
	if err != nil {
		return tickets.Ticket{}, fmt.Errorf("failed to add comment: %w", err)
	}
	if err = s.repo.AddHistory(txCtx, tickets.History{
		TicketID: updated.ID, UserID: input.UserID,
		Action: tickets.ActionCommentAdded, NewValue: input.Comment,
	}); err != nil {
		return tickets.Ticket{}, fmt.Errorf("failed to add comment history: %w", err)
	}
	if firstResponse {
		if err = s.repo.AddHistory(txCtx, tickets.History{
			TicketID: updated.ID, UserID: input.UserID,
			Action: tickets.ActionFirstResponse, NewValue: input.Comment,
		}); err != nil {
			return tickets.Ticket{}, fmt.Errorf("failed to add first response history: %w", err)
		}
	}
	if err = tx.Commit(); err != nil {
		return tickets.Ticket{}, fmt.Errorf("failed to commit transaction: %w", err)
	}
	return updated, nil
}

// GetSLAViolations returns tickets that have missed at least one SLA target.
func (s *service) GetSLAViolations(ctx context.Context) ([]tickets.Ticket, error) {
	result, err := s.repo.FindSLAViolations(ctx, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to find SLA violations: %w", err)
	}
	return result, nil
}
