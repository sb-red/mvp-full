package main

import (
	"context"
	"log"
	"os"
	"strconv"

	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/swagger"

	"lambda-runner-server/handlers"
	customMiddleware "lambda-runner-server/middleware"
	"lambda-runner-server/services"

	_ "lambda-runner-server/docs"
)

// @title SoftGate API
// @version 1.0
// @description Serverless function execution platform (FaaS) API
// @host localhost:8080
// @BasePath /api
func main() {
	// Initialize X-Ray
	err := os.Setenv("AWS_XRAY_DAEMON_ADDRESS", getEnv("XRAY_DAEMON_ADDRESS", "127.0.0.1:2000"))
	if err != nil {
		log.Printf("Warning: Failed to set X-Ray daemon address: %v", err)
	}

	xray.Configure(xray.Config{
		DaemonAddr:     getEnv("XRAY_DAEMON_ADDRESS", "127.0.0.1:2000"),
		ServiceVersion: "1.0.0",
	})

	// Set X-Ray service name for tracing
	err = os.Setenv("AWS_XRAY_TRACING_NAME", "softgate-backend")
	if err != nil {
		log.Printf("Warning: Failed to set X-Ray service name: %v", err)
	}

	// Config
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort, _ := strconv.Atoi(getEnv("REDIS_PORT", "6379"))
	serverPort := getEnv("SERVER_PORT", "8080")

	// PostgreSQL Config
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort, _ := strconv.Atoi(getEnv("DB_PORT", "5432"))
	dbUser := getEnv("DB_USER", "softgate")
	dbPassword := getEnv("DB_PASSWORD", "softgate")
	dbName := getEnv("DB_NAME", "softgate")

	// Storage Config
	storageType := getEnv("STORAGE_TYPE", "local")
	storagePath := getEnv("STORAGE_BUCKET", getEnv("STORAGE_PATH", "/data/code"))

	// Initialize services
	dbService, dbErr := services.NewDBService(dbHost, dbPort, dbUser, dbPassword, dbName)
	err = dbErr
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbService.Close()

	// Initialize database schema
	if err := dbService.InitSchema(context.Background()); err != nil {
		log.Fatalf("Failed to initialize database schema: %v", err)
	}
	log.Println("Database schema initialized")

	// Initialize storage service
	storageService, err := services.NewStorageService(storageType, storagePath)
	if err != nil {
		log.Fatalf("Failed to initialize storage service: %v", err)
	}
	log.Printf("Storage service initialized: %s (%s)", storageType, storagePath)

	// Initialize Redis service
	redisService := services.NewRedisService(redisHost, redisPort)

	// Initialize function service
	functionService := services.NewFunctionService(dbService, storageService, redisService)

	// Initialize handlers/services
	functionHandler := handlers.NewFunctionHandler(functionService)
	scheduleService := services.NewScheduleService(dbService)
	scheduleHandler := handlers.NewScheduleHandler(scheduleService)

	// Start schedule runner
	scheduleRunner := services.NewScheduleRunner(scheduleService, functionService)
	scheduleRunner.Start()
	defer scheduleRunner.Stop()

	// Fiber App
	app := fiber.New(fiber.Config{
		AppName: "SoftGate",
	})

	// Middleware
	app.Use(logger.New())
	app.Use(recover.New())
	app.Use(customMiddleware.XRayMiddleware()) // X-Ray tracing
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept",
	}))

	// Swagger
	app.Get("/swagger/*", swagger.HandlerDefault)

	// Health endpoints
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "UP"})
	})

	// API routes
	api := app.Group("/api")

	// Function routes (PRD spec)
	api.Post("/functions", functionHandler.CreateFunction)
	api.Get("/functions", functionHandler.ListFunctions)
	api.Get("/functions/:id", functionHandler.GetFunction)
	api.Post("/functions/:id/invoke", functionHandler.InvokeFunction)
	api.Get("/functions/:id/invocations", functionHandler.ListInvocations)
	api.Get("/functions/:id/invocations/:invocationId", functionHandler.GetInvocationResult)
	api.Delete("/functions/:id", functionHandler.DeleteFunction)
	api.Post("/functions/:id/schedules", scheduleHandler.CreateSchedule)
	api.Get("/functions/:id/schedules", scheduleHandler.ListSchedules)
	api.Delete("/functions/:id/schedules/:scheduleId", scheduleHandler.DeleteSchedule)

	log.Printf("SoftGate Server starting on port %s", serverPort)
	log.Printf("Database: %s:%d/%s", dbHost, dbPort, dbName)
	log.Printf("Redis: %s:%d", redisHost, redisPort)
	log.Fatal(app.Listen(":" + serverPort))
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
