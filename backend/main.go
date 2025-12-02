package main

import (
	"context"
	"log"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/swagger"

	"lambda-runner-server/handlers"
	"lambda-runner-server/services"

	_ "lambda-runner-server/docs"
)

// @title SoftGate API
// @version 1.0
// @description Serverless function execution platform (FaaS) API
// @host localhost:8080
// @BasePath /api
func main() {
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
	storagePath := getEnv("STORAGE_PATH", "/data/code")

	// Initialize services
	dbService, err := services.NewDBService(dbHost, dbPort, dbUser, dbPassword, dbName)
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

	// Initialize handlers
	functionHandler := handlers.NewFunctionHandler(functionService)

	// Fiber App
	app := fiber.New(fiber.Config{
		AppName: "SoftGate",
	})

	// Middleware
	app.Use(logger.New())
	app.Use(recover.New())
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
