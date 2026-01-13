package api

import (
	"smart-load/internal/domain"
	"smart-load/internal/service"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App, optimizerService *service.OptimizerService) {
	app.Get("/healthz", HealthCheckHandler)
	app.Get("/actuator/health", HealthCheckHandler)
	
	v1 := app.Group("/api/v1")
	loadOptimizer := v1.Group("/load-optimizer")
	loadOptimizer.Post("/optimize", OptimizeHandler(optimizerService))
	loadOptimizer.Post("/pareto-solutions", ParetoHandler(optimizerService))
}

func HealthCheckHandler(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "UP",
		"service": "SmartLoad Optimizer API",
		"version": "1.0.0",
	})
}

func OptimizeHandler(optimizerService *service.OptimizerService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var request domain.OptimizeRequest
		if err := c.BodyParser(&request); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    fiber.StatusBadRequest,
					"message": "Invalid JSON format",
					"details": err.Error(),
				},
			})
		}
		
		response, err := optimizerService.OptimizeLoad(request)
		if err != nil {
			statusCode := fiber.StatusInternalServerError
			
			if strings.Contains(err.Error(), "validation") {
				statusCode = fiber.StatusBadRequest
			}
			
			return c.Status(statusCode).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    statusCode,
					"message": err.Error(),
				},
			})
		}
		
		return c.Status(fiber.StatusOK).JSON(response)
	}
}

func RequestSizeLimiter(maxBytes int) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Request().Header.ContentLength() > maxBytes {
			return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    fiber.StatusRequestEntityTooLarge,
					"message": "Request body too large",
				},
			})
		}
		return c.Next()
	}
}

func ParetoHandler(optimizerService *service.OptimizerService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var request domain.OptimizeRequest
		if err := c.BodyParser(&request); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    fiber.StatusBadRequest,
					"message": "Invalid JSON format",
				},
			})
		}
		
		if err := request.Validate(); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    fiber.StatusBadRequest,
					"message": err.Error(),
				},
			})
		}
		
		truck, orders, err := request.ToDomain()
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    fiber.StatusBadRequest,
					"message": err.Error(),
				},
			})
		}
		
		solutions := optimizerService.GetParetoOptimalSolutions(*truck, orders, 5)
		
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"truck_id":  truck.ID,
			"solutions": solutions,
			"count":     len(solutions),
		})
	}
}
