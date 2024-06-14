package common

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
)

type CacheOptions struct {
	MaxAge               time.Duration // Default: 1 hour
	StaleWhileRevalidate time.Duration // Default: 5 minutes
	StaleIfError         time.Duration // Default: 5 minutes
}

// SetCacheHeader sets the cache headers for the response based on the provided options.
// The function checks for negative durations and sets default values as follows:
//
// - MaxAge: 1 hour if the provided value is negative.
//
// - StaleWhileRevalidate: 5 minutes if the provided value is negative.
//
// - StaleIfError: 5 minutes if the provided value is negative.
//
// This configuration means that the response will be considered fresh for the duration of MaxAge,
// and may be revalidated in the background every StaleWhileRevalidate duration.
// In case of an error, the stale response may still be used for the duration of StaleIfError.
//
// The cache control header includes both 'max-age' for standard caches and 's-maxage' for shared caches,
// ensuring compatibility with browsers and CDNs.
func SetCacheHeader(c *fiber.Ctx, options CacheOptions) {
	// Set default values if not provided
	if options.MaxAge < 0 {
		options.MaxAge = time.Hour
	}
	if options.StaleWhileRevalidate < 0 {
		options.StaleWhileRevalidate = 5 * time.Minute
	}
	if options.StaleIfError < 0 {
		options.StaleIfError = 5 * time.Minute
	}

	// Convert the duration to seconds
	maxAge := options.MaxAge / time.Second
	staleWhileRevalidate := options.StaleWhileRevalidate / time.Second
	staleIfError := options.StaleIfError / time.Second

	// Construct the cache control header
	cacheControl := fmt.Sprintf("public, max-age=%d, s-maxage=%d, stale-while-revalidate=%d, stale-if-error=%d", maxAge, maxAge, staleWhileRevalidate, staleIfError)

	// Set the cache control header
	c.Set("Cache-Control", cacheControl)
}

// Returns the trueVal if the condition is true, otherwise it returns the falseVal.
func TernaryIf[T any](condition bool, trueVal, falseVal T) T {
	if condition {
		return trueVal
	}
	return falseVal
}
