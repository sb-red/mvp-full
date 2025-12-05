package middleware

import (
	"context"
	"fmt"

	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/gofiber/fiber/v2"
)

// XRayMiddleware wraps Fiber requests with AWS X-Ray tracing
func XRayMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Create a segment name from the route
		segmentName := fmt.Sprintf("%s %s", c.Method(), c.Path())

		// Start a new segment
		ctx, seg := xray.BeginSegment(context.Background(), segmentName)
		defer seg.Close(nil)

		// Add HTTP request metadata
		seg.GetHTTP().GetRequest().Method = c.Method()
		seg.GetHTTP().GetRequest().URL = c.OriginalURL()
		seg.GetHTTP().GetRequest().ClientIP = c.IP()
		seg.GetHTTP().GetRequest().UserAgent = c.Get("User-Agent")

		// Store X-Ray context in Fiber locals for downstream use
		c.Locals("xray-ctx", ctx)
		c.Locals("xray-seg", seg)

		// Process request
		err := c.Next()

		// Add HTTP response metadata
		seg.GetHTTP().GetResponse().Status = c.Response().StatusCode()

		if err != nil {
			// Mark segment as error
			seg.AddError(err)
			seg.GetHTTP().GetResponse().Status = fiber.StatusInternalServerError
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
