package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"pet-ticket/internal/app/tickets"
	domain "pet-ticket/internal/domain/tickets"
	"pet-ticket/internal/transport/http/dto"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

// mockTicketsService — мок сервиса для тестов handlers
type mockTicketsService struct {
	createTicketFunc     func(ctx context.Context, input tickets.CreateTicketInput) (domain.Ticket, error)
	getTicketFunc        func(ctx context.Context, id int64) (domain.Ticket, error)
	updateTicketFunc     func(ctx context.Context, input tickets.UpdateTicketInput) (domain.Ticket, error)
	updatePriorityFunc   func(ctx context.Context, ticketID int64, priority domain.Priority) (domain.Ticket, error)
	escalateTicketFunc   func(ctx context.Context, ticketID int64) (domain.Ticket, error)
	deleteTicketFunc     func(ctx context.Context, id int64) error
	listTicketsFunc      func(ctx context.Context, input tickets.ListTicketsInput) ([]domain.Ticket, error)
	getTicketHistoryFunc func(ctx context.Context, ticketID int64, limit, offset int) ([]domain.History, error)
	getAllStatusesFunc   func(ctx context.Context) ([]tickets.StatusInfo, error)
	getAllTopicsFunc     func(ctx context.Context) ([]domain.Topic, error)
	addCommentFunc       func(ctx context.Context, input tickets.AddCommentInput) (domain.Ticket, error)
	getSLAViolationsFunc func(ctx context.Context) ([]domain.Ticket, error)
}

func (m *mockTicketsService) CreateTicket(ctx context.Context, input tickets.CreateTicketInput) (domain.Ticket, error) {
	if m.createTicketFunc != nil {
		return m.createTicketFunc(ctx, input)
	}
	return domain.Ticket{}, errors.New("not implemented")
}

func (m *mockTicketsService) GetTicket(ctx context.Context, id int64) (domain.Ticket, error) {
	if m.getTicketFunc != nil {
		return m.getTicketFunc(ctx, id)
	}
	return domain.Ticket{}, errors.New("not implemented")
}

func (m *mockTicketsService) UpdateTicket(ctx context.Context, input tickets.UpdateTicketInput) (domain.Ticket, error) {
	if m.updateTicketFunc != nil {
		return m.updateTicketFunc(ctx, input)
	}
	return domain.Ticket{}, errors.New("not implemented")
}

func (m *mockTicketsService) UpdatePriority(
	ctx context.Context,
	ticketID int64,
	priority domain.Priority,
) (domain.Ticket, error) {
	if m.updatePriorityFunc != nil {
		return m.updatePriorityFunc(ctx, ticketID, priority)
	}
	return domain.Ticket{}, errors.New("not implemented")
}

func (m *mockTicketsService) EscalateTicket(ctx context.Context, ticketID int64) (domain.Ticket, error) {
	if m.escalateTicketFunc != nil {
		return m.escalateTicketFunc(ctx, ticketID)
	}
	return domain.Ticket{}, errors.New("not implemented")
}

func (m *mockTicketsService) DeleteTicket(ctx context.Context, id int64) error {
	if m.deleteTicketFunc != nil {
		return m.deleteTicketFunc(ctx, id)
	}
	return errors.New("not implemented")
}

func (m *mockTicketsService) ListTickets(ctx context.Context, input tickets.ListTicketsInput) ([]domain.Ticket, error) {
	if m.listTicketsFunc != nil {
		return m.listTicketsFunc(ctx, input)
	}
	return nil, errors.New("not implemented")
}

func (m *mockTicketsService) GetTicketHistory(ctx context.Context, ticketID int64, limit, offset int) ([]domain.History, error) {
	if m.getTicketHistoryFunc != nil {
		return m.getTicketHistoryFunc(ctx, ticketID, limit, offset)
	}
	return nil, errors.New("not implemented")
}

func (m *mockTicketsService) GetAllStatuses(ctx context.Context) ([]tickets.StatusInfo, error) {
	if m.getAllStatusesFunc != nil {
		return m.getAllStatusesFunc(ctx)
	}
	return nil, errors.New("not implemented")
}

func (m *mockTicketsService) GetAllTopics(ctx context.Context) ([]domain.Topic, error) {
	if m.getAllTopicsFunc != nil {
		return m.getAllTopicsFunc(ctx)
	}
	return nil, errors.New("not implemented")
}

func (m *mockTicketsService) AddComment(ctx context.Context, input tickets.AddCommentInput) (domain.Ticket, error) {
	if m.addCommentFunc != nil {
		return m.addCommentFunc(ctx, input)
	}
	return domain.Ticket{}, errors.New("not implemented")
}

func (m *mockTicketsService) GetSLAViolations(ctx context.Context) ([]domain.Ticket, error) {
	if m.getSLAViolationsFunc != nil {
		return m.getSLAViolationsFunc(ctx)
	}
	return nil, errors.New("not implemented")
}

// TestGetTicket_Success — успешное получение тикета через HTTP
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

	mockSvc := &mockTicketsService{
		getTicketFunc: func(ctx context.Context, id int64) (domain.Ticket, error) {
			if id == 1 {
				return expectedTicket, nil
			}
			return domain.Ticket{}, tickets.ErrNotFound
		},
	}

	handler := NewTicketsHandler(mockSvc, testLogger())

	app := fiber.New()
	app.Get("/tickets/:id", handler.getTicket)

	// Act
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/tickets/1", nil)
	resp, err := app.Test(req)

	// Assert
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var ticketResp dto.TicketResponse
	if err := json.Unmarshal(body, &ticketResp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if ticketResp.ID != 1 {
		t.Errorf("expected ticket ID 1, got %d", ticketResp.ID)
	}
	if ticketResp.UserID != 100 {
		t.Errorf("expected user ID 100, got %d", ticketResp.UserID)
	}
}

// TestGetTicket_NotFound — тикет не найден (404)
func TestGetTicket_NotFound(t *testing.T) {
	// Arrange
	mockSvc := &mockTicketsService{
		getTicketFunc: func(ctx context.Context, id int64) (domain.Ticket, error) {
			return domain.Ticket{}, tickets.ErrNotFound
		},
	}

	handler := NewTicketsHandler(mockSvc, testLogger())

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if errors.Is(err, tickets.ErrNotFound) {
				code = fiber.StatusNotFound
			}
			return c.Status(code).JSON(fiber.Map{"error": err.Error()})
		},
	})
	app.Get("/tickets/:id", handler.getTicket)

	// Act
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/tickets/999", nil)
	resp, err := app.Test(req)

	// Assert
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("expected status 404, got %d", resp.StatusCode)
	}
}

// TestCreateTicket_Success_WithValidation — создание тикета через handler с валидацией
func TestCreateTicket_Success_WithValidation(t *testing.T) {
	// Arrange
	createdTicket := domain.Ticket{
		ID:       42,
		UserID:   100,
		TopicID:  1,
		Status:   domain.StatusNew,
		Priority: domain.PriorityMedium,
		Comment:  "Test ticket from handler",
	}

	mockSvc := &mockTicketsService{
		createTicketFunc: func(ctx context.Context, input tickets.CreateTicketInput) (domain.Ticket, error) {
			return createdTicket, nil
		},
	}

	handler := NewTicketsHandler(mockSvc, testLogger())

	app := fiber.New()
	// Без middleware валидации — тестируем только handler
	app.Post("/tickets", func(c *fiber.Ctx) error {
		// Имитируем валидированное body
		req := CreateTicketRequest{
			UserID:  100,
			TopicID: 1,
			Comment: "Test ticket from handler",
		}
		c.Locals("validatedBody", req)
		return handler.createTicket(c)
	})

	// Act
	body := []byte(`{"userId": 100, "topicId": 1, "comment": "Test ticket from handler"}`)
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/tickets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	// Assert
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	var ticketResp dto.TicketResponse
	if err := json.Unmarshal(bodyBytes, &ticketResp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if ticketResp.ID != 42 {
		t.Errorf("expected ticket ID 42, got %d", ticketResp.ID)
	}
}

func TestCreateTicket_WithExplicitPriority(t *testing.T) {
	priorityID := int(domain.PriorityHigh)
	app := fiber.New()
	handler := NewTicketsHandler(&mockTicketsService{
		createTicketFunc: func(_ context.Context, input tickets.CreateTicketInput) (domain.Ticket, error) {
			if input.Priority != domain.PriorityHigh {
				t.Fatalf("priority input = %v, want high", input.Priority)
			}
			return domain.Ticket{
				ID: 1, UserID: input.UserID, TopicID: input.TopicID,
				Status: domain.StatusNew, Priority: input.Priority, Comment: input.Comment,
			}, nil
		},
	}, testLogger())
	app.Post("/tickets", func(c *fiber.Ctx) error {
		c.Locals("validatedBody", CreateTicketRequest{
			UserID: 1, TopicID: 1, PriorityID: &priorityID, Comment: "High priority ticket",
		})
		return handler.createTicket(c)
	})

	resp, err := app.Test(httptest.NewRequest("POST", "/tickets", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("got %d, want 201", resp.StatusCode)
	}
	var body dto.TicketResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.PriorityID != priorityID {
		t.Fatalf("priorityId = %d, want %d", body.PriorityID, priorityID)
	}
}

// TestListTickets_WithFilters — получение списка с фильтрами
func TestListTickets_WithFilters(t *testing.T) {
	// Arrange
	userID := int64(100)
	expectedTickets := []domain.Ticket{
		{ID: 1, UserID: 100, TopicID: 1, Status: domain.StatusNew, Priority: domain.PriorityMedium, Comment: "Ticket 1"},
		{ID: 2, UserID: 100, TopicID: 2, Status: domain.StatusInProgress, Priority: domain.PriorityHigh, Comment: "Ticket 2"},
	}

	var capturedFilter tickets.ListTicketsInput

	mockSvc := &mockTicketsService{
		listTicketsFunc: func(ctx context.Context, input tickets.ListTicketsInput) ([]domain.Ticket, error) {
			capturedFilter = input
			return expectedTickets, nil
		},
	}

	handler := NewTicketsHandler(mockSvc, testLogger())

	app := fiber.New()
	app.Get("/tickets", handler.listTickets)

	// Act
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/tickets?userId=100&limit=10&offset=5", nil)
	resp, err := app.Test(req)

	// Assert
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if capturedFilter.UserID == nil || *capturedFilter.UserID != userID {
		t.Errorf("expected userID filter 100, got %v", capturedFilter.UserID)
	}
	if capturedFilter.Limit != 10 {
		t.Errorf("expected limit 10, got %d", capturedFilter.Limit)
	}
	if capturedFilter.Offset != 5 {
		t.Errorf("expected offset 5, got %d", capturedFilter.Offset)
	}

	body, _ := io.ReadAll(resp.Body)
	var listResp dto.ListResponse
	if err := json.Unmarshal(body, &listResp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if listResp.Total != 2 {
		t.Errorf("expected total 2, got %d", listResp.Total)
	}
}

// testLogger возвращает no-op logger для тестов
func testLogger() zerolog.Logger {
	return zerolog.New(io.Discard)
}

func TestEscalateTicket_Success(t *testing.T) {
	app := fiber.New()
	h := NewTicketsHandler(&mockTicketsService{
		escalateTicketFunc: func(_ context.Context, ticketID int64) (domain.Ticket, error) {
			if ticketID != 1 {
				t.Fatalf("ticketID = %d, want 1", ticketID)
			}
			return domain.Ticket{
				ID: 1, UserID: 100, TopicID: 1, Status: domain.StatusNew,
				Priority: domain.PriorityHigh, Comment: "Escalated ticket",
			}, nil
		},
	}, zerolog.Nop())
	h.RegisterRoutes(app.Group("/api/v1"))
	resp, err := app.Test(httptest.NewRequest("POST", "/api/v1/tickets/1/escalate", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("got %d, want 200", resp.StatusCode)
	}
	var body dto.TicketResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.PriorityID != int(domain.PriorityHigh) || body.PriorityName != "high" {
		t.Fatalf("unexpected priority response: %+v", body)
	}
}

func TestListRejectsInvalidStatus(t *testing.T) {
	app := fiber.New()
	h := NewTicketsHandler(&mockTicketsService{}, zerolog.Nop())
	h.RegisterRoutes(app.Group("/api/v1"))
	for _, value := range []string{"", "bad", "0", "-1", "6"} {
		resp, err := app.Test(httptest.NewRequest("GET", "/api/v1/tickets?statusId="+value, nil))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != 400 {
			t.Fatalf("statusId=%q: got %d", value, resp.StatusCode)
		}
	}
}

func TestListTickets_WithPriorityFilterAndSort(t *testing.T) {
	var captured tickets.ListTicketsInput
	app := fiber.New()
	h := NewTicketsHandler(&mockTicketsService{
		listTicketsFunc: func(_ context.Context, input tickets.ListTicketsInput) ([]domain.Ticket, error) {
			captured = input
			return []domain.Ticket{}, nil
		},
	}, zerolog.Nop())
	h.RegisterRoutes(app.Group("/api/v1"))

	resp, err := app.Test(httptest.NewRequest(
		"GET", "/api/v1/tickets?priorityId=3&sortBy=priority_id&sortDesc=false", nil,
	))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("got %d, want 200", resp.StatusCode)
	}
	if captured.Priority == nil || *captured.Priority != domain.PriorityHigh {
		t.Fatalf("priority filter = %v, want high", captured.Priority)
	}
	if captured.SortBy != "priority_id" || captured.SortDesc {
		t.Fatalf("unexpected sort: %q desc=%v", captured.SortBy, captured.SortDesc)
	}
}

func TestListRejectsInvalidPriority(t *testing.T) {
	app := fiber.New()
	h := NewTicketsHandler(&mockTicketsService{}, zerolog.Nop())
	h.RegisterRoutes(app.Group("/api/v1"))
	for _, value := range []string{"", "bad", "0", "-1", "5"} {
		resp, err := app.Test(httptest.NewRequest("GET", "/api/v1/tickets?priorityId="+value, nil))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Fatalf("priorityId=%q: got %d, want 400", value, resp.StatusCode)
		}
	}
}
