package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"lambda-runner-server/models"
	"lambda-runner-server/services"
)

type ScheduleHandler struct {
	service *services.ScheduleService
}

func NewScheduleHandler(service *services.ScheduleService) *ScheduleHandler {
	return &ScheduleHandler{service: service}
}

// CreateSchedule godoc
// @Summary Create a scheduled execution for a function
// @Tags schedules
// @Accept json
// @Produce json
// @Param id path int true "Function ID"
// @Param schedule body models.CreateScheduleRequest true "Schedule request"
// @Success 200 {object} models.FunctionSchedule
// @Router /functions/{id}/schedules [post]
func (h *ScheduleHandler) CreateSchedule(c *fiber.Ctx) error {
	functionID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid function ID"})
	}

	var req models.CreateScheduleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	sched, err := h.service.CreateSchedule(c.Context(), functionID, &req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(sched)
}

// ListSchedules godoc
// @Summary List schedules for a function
// @Tags schedules
// @Produce json
// @Param id path int true "Function ID"
// @Success 200 {array} models.FunctionSchedule
// @Router /functions/{id}/schedules [get]
func (h *ScheduleHandler) ListSchedules(c *fiber.Ctx) error {
	functionID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid function ID"})
	}

	schedules, err := h.service.ListSchedules(c.Context(), functionID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(schedules)
}

// DeleteSchedule godoc
// @Summary Delete a function schedule
// @Tags schedules
// @Param id path int true "Function ID"
// @Param scheduleId path int true "Schedule ID"
// @Success 204
// @Router /functions/{id}/schedules/{scheduleId} [delete]
func (h *ScheduleHandler) DeleteSchedule(c *fiber.Ctx) error {
	functionID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid function ID"})
	}
	scheduleID, err := strconv.ParseInt(c.Params("scheduleId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid schedule ID"})
	}

	if err := h.service.DeleteSchedule(c.Context(), functionID, scheduleID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
