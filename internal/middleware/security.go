package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// SecurityHeaders adds security headers to all responses
func SecurityHeaders() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Security headers
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "SAMEORIGIN")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		// Only set HSTS in production
		if c.Protocol() == "https" {
			c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		return c.Next()
	}
}

// BotProtection provides basic bot detection
func BotProtection() fiber.Handler {
	// Known bad bots / scrapers
	blockedAgents := []string{
		"python-requests",
		"curl/",
		"wget/",
		"scrapy",
		"httpclient",
		"java/",
		"libwww",
	}

	return func(c *fiber.Ctx) error {
		// Only apply to API endpoints
		if !strings.HasPrefix(c.Path(), "/api/") {
			return c.Next()
		}

		ua := strings.ToLower(c.Get("User-Agent"))

		// Block empty user agents on API
		if ua == "" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"error":   "Access denied",
			})
		}

		// Block known bot user agents
		for _, blocked := range blockedAgents {
			if strings.Contains(ua, blocked) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"success": false,
					"error":   "Automated requests are not allowed",
				})
			}
		}

		return c.Next()
	}
}

// CORS sets up CORS headers
func CORS() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		c.Set("Access-Control-Allow-Headers", "Content-Type,Accept")

		if c.Method() == "OPTIONS" {
			return c.SendStatus(fiber.StatusNoContent)
		}

		return c.Next()
	}
}
