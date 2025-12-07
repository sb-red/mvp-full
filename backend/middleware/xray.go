package middleware

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/gofiber/fiber/v2"
)

// XRayMiddleware wraps Fiber requests with AWS X-Ray tracing
func XRayMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip tracing for health checks to reduce noise
		if c.Path() == "/health" {
			return c.Next()
		}

		// Create a segment name from the route
		segmentName := fmt.Sprintf("softgate-backend")

		// Start a new segment
		ctx, seg := xray.BeginSegment(context.Background(), segmentName)
		defer func() {
			if seg != nil {
				seg.Close(nil)
			}
		}()

		// Add HTTP request metadata
		if seg.GetHTTP() != nil {
			seg.GetHTTP().GetRequest().Method = c.Method()
			seg.GetHTTP().GetRequest().URL = c.OriginalURL()
			seg.GetHTTP().GetRequest().ClientIP = c.IP()
			seg.GetHTTP().GetRequest().UserAgent = c.Get("User-Agent")
		}

		// Add route as annotation
		seg.AddAnnotation("route", c.Path())
		seg.AddAnnotation("method", c.Method())

		// Store X-Ray context in Fiber locals for downstream use
		c.Locals("xray-ctx", ctx)
		c.Locals("xray-seg", seg)

		// Process request
		err := c.Next()

		// Add HTTP response metadata
		if seg.GetHTTP() != nil {
			seg.GetHTTP().GetResponse().Status = c.Response().StatusCode()
		}

		if err != nil {
			// Mark segment as error
			log.Printf("Request error: %v", err)
			seg.AddError(err)
			if seg.GetHTTP() != nil {
				seg.GetHTTP().GetResponse().Status = fiber.StatusInternalServerError
			}
		}

		return err
	}
}

// GetXRayContext retrieves X-Ray context from Fiber locals
func GetXRayContext(c *fiber.Ctx) context.Context {
	if ctx := c.Locals("xray-ctx"); ctx != nil {
		return ctx.(context.Context)
	}
	return context.Background()
}
