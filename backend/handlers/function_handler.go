package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"lambda-runner-server/models"
	"lambda-runner-server/services"
)

type FunctionHandler struct {
	service *services.FunctionService
}

func NewFunctionHandler(svc *services.FunctionService) *FunctionHandler {
	return &FunctionHandler{service: svc}
}

// CreateFunction godoc
// @Summary Create a new function
// @Description Register a new serverless function with code and parameters
// @Tags functions
// @Accept json
// @Produce json
// @Param function body models.CreateFunctionRequest true "Function to create"
// @Success 200 {object} models.Function
// @Failure 400 {object} map[string]string
// @Router /functions [post]
func (h *FunctionHandler) CreateFunction(c *fiber.Ctx) error {
	var req models.CreateFunctionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validation
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "name is required",
		})
	}
	if req.Code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "code is required",
		})
	}
	if req.Runtime == "" {
		req.Runtime = "python3.11"
	}

	fn, err := h.service.CreateFunction(c.Context(), &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fn)
}

// ListFunctions godoc
// @Summary List all functions
// @Description Get a list of all registered functions
// @Tags functions
// @Produce json
// @Success 200 {array} models.FunctionListItem
// @Router /functions [get]
func (h *FunctionHandler) ListFunctions(c *fiber.Ctx) error {
	functions, err := h.service.ListFunctions(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if functions == nil {
		functions = []models.FunctionListItem{}
	}

	return c.JSON(functions)
}

// GetFunction godoc
// @Summary Get function details
// @Description Get detailed information about a specific function
// @Tags functions
// @Produce json
// @Param id path int true "Function ID"
// @Success 200 {object} models.Function
// @Failure 404 {object} map[string]string
// @Router /functions/{id} [get]
func (h *FunctionHandler) GetFunction(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid function ID",
		})
	}

	fn, err := h.service.GetFunction(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fn)
}

// InvokeFunction godoc
// @Summary Invoke a function
// @Description Execute a function with given parameters
// @Tags functions
// @Accept json
// @Produce json
// @Param id path int true "Function ID"
// @Param input body models.InvokeRequest true "Input parameters"
// @Success 200 {object} models.InvokeResponse
// @Failure 404 {object} map[string]string
// @Router /functions/{id}/invoke [post]
func (h *FunctionHandler) InvokeFunction(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid function ID",
		})
	}

	var req models.InvokeRequest
	if err := c.BodyParser(&req); err != nil {
		req.Params = make(map[string]interface{})
	}

	// Get client IP for invoked_by
	invokedBy := c.IP()
	if invokedBy == "" {
		invokedBy = "anonymous"
	}

	inv, err := h.service.InvokeFunction(c.Context(), id, req.Params, invokedBy)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Return initial response with invocation ID
	return c.JSON(fiber.Map{
		"status":        inv.Status,
		"function_id":   inv.FunctionID,
		"invocation_id": inv.ID,
		"input_event":   inv.InputEvent,
		"logged_at":     inv.InvokedAt,
	})
}

// GetInvocationResult godoc
// @Summary Get invocation result
// @Description Poll for the result of a function invocation
// @Tags functions
// @Produce json
// @Param id path int true "Function ID"
// @Param invocationId path int true "Invocation ID"
// @Success 200 {object} models.InvokeResponse
// @Failure 404 {object} map[string]string
// @Router /functions/{id}/invocations/{invocationId} [get]
func (h *FunctionHandler) GetInvocationResult(c *fiber.Ctx) error {
	invocationIdStr := c.Params("invocationId")
	invocationId, err := strconv.ParseInt(invocationIdStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid invocation ID",
		})
	}

	inv, err := h.service.GetInvocationResult(c.Context(), invocationId)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	response := models.InvokeResponse{
		Status:       inv.Status,
		FunctionID:   inv.FunctionID,
		InvocationID: inv.ID,
		InputEvent:   inv.InputEvent,
		DurationMs:   inv.DurationMs,
		LoggedAt:     inv.InvokedAt,
	}

	if inv.Status == models.StatusSuccess {
		response.Result = inv.OutputResult
	} else if inv.Status == models.StatusFail || inv.Status == models.StatusTimeout {
		response.ErrorMessage = inv.ErrorMessage
	}

	return c.JSON(response)
}

// ListInvocations godoc
// @Summary List function invocations
// @Description Get execution history for a function
// @Tags functions
// @Produce json
// @Param id path int true "Function ID"
// @Param limit query int false "Number of results to return" default(20)
// @Success 200 {array} models.InvocationListItem
// @Router /functions/{id}/invocations [get]
func (h *FunctionHandler) ListInvocations(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid function ID",
		})
	}

	limit := c.QueryInt("limit", 20)

	invocations, err := h.service.ListInvocations(c.Context(), id, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if invocations == nil {
		invocations = []models.InvocationListItem{}
	}

	return c.JSON(invocations)
}
