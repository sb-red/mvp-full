package middleware

import (
	"net/http"

	"github.com/aws/aws-xray-sdk-go/xray"
)

// GetXRayHTTPClient returns an HTTP client instrumented with X-Ray
// Use this client for all outbound HTTP requests to trace downstream services
func GetXRayHTTPClient() *http.Client {
	return xray.Client(&http.Client{})
}

// GetCustomXRayHTTPClient returns a custom HTTP client instrumented with X-Ray
// Useful when you need to customize the client (e.g., timeouts, transport)
func GetCustomXRayHTTPClient(client *http.Client) *http.Client {
	return xray.Client(client)
}
