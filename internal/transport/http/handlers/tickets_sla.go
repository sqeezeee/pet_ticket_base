package handlers

import (
	"strconv"

	"pet-ticket/internal/app/tickets"
	"pet-ticket/internal/transport/http/dto"
	mw "pet-ticket/internal/transport/http/middleware"

	"github.com/gofiber/fiber/v2"
)

// AddCommentRequest is the payload for a separate comment operation.
type AddCommentRequest struct {
	UserID    int64  `json:"userId,omitempty" validate:"omitempty,gte=1"`
	Comment   string `json:"comment" validate:"required,min=1,max=2000"`
	IsSupport bool   `json:"isSupport"`
}

func (h *TicketsHandler) getSLAViolations(c *fiber.Ctx) error {
	ticketList, err := h.service.GetSLAViolations(c.UserContext())
	if err != nil {
		return err
	}
	return c.JSON(dto.ToTicketWithSLAResponseList(ticketList))
}

func (h *TicketsHandler) addComment(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid ticket id")
	}
	req := mw.GetValidatedBody[AddCommentRequest](c)
	userID := req.UserID
	if userID == 0 {
		if value, ok := c.Locals("userID").(int64); ok {
			userID = value
		}
	}
	if userID <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "user id is required")
	}
	ticket, err := h.service.AddComment(c.UserContext(), tickets.AddCommentInput{
		TicketID: id, UserID: userID, Comment: req.Comment, IsSupport: req.IsSupport,
	})
	if err != nil {
		return err
	}
	return c.JSON(dto.ToTicketResponse(ticket))
}
