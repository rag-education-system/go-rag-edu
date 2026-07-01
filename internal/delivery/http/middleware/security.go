package middleware

import (
	"time"

	"rag-api/pkg/config"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/gofiber/fiber/v2/middleware/timeout"
)

func ApplySecurity(app *fiber.App, cfg *config.Config) {
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))

	app.Use(requestid.New())

	app.Use(helmet.New(helmet.Config{
		XSSProtection:             "1; mode=block",
		ContentTypeNosniff:        "nosniff",
		XFrameOptions:             "DENY",
		ReferrerPolicy:            "no-referrer",
		CrossOriginEmbedderPolicy: "require-corp",
	}))

	if len(cfg.CORSOrigins) > 0 {
		app.Use(cors.New(cors.Config{
			AllowOrigins:     joinOrigins(cfg.CORSOrigins),
			AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
			AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
			AllowCredentials: true,
			MaxAge:           300,
		}))
	}

	requestTimeout := timeout.NewWithContext(func(c *fiber.Ctx) error {
		return c.Next()
	}, cfg.RequestTimeout)
	uploadTimeout := timeout.NewWithContext(func(c *fiber.Ctx) error {
		return c.Next()
	}, cfg.UploadRequestTimeout)
	streamTimeout := timeout.NewWithContext(func(c *fiber.Ctx) error {
		return c.Next()
	}, cfg.StreamRequestTimeout)
	app.Use(func(c *fiber.Ctx) error {
		switch c.Path() {
		case "/api/chat/stream":
			return streamTimeout(c)
		case "/api/documents/upload":
			return uploadTimeout(c)
		default:
			return requestTimeout(c)
		}
	})

	app.Use(GlobalRateLimit(cfg))
}

func GlobalRateLimit(cfg *config.Config) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        cfg.RateLimitGlobalMax,
		Expiration: cfg.RateLimitGlobalWindow,
		KeyGenerator: func(c *fiber.Ctx) string {
			return clientKey(c)
		},
		LimitReached: rateLimitResponse,
		SkipFailedRequests:     false,
		SkipSuccessfulRequests: false,
	})
}

func StrictRateLimit(max int, window time.Duration) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        max,
		Expiration: window,
		KeyGenerator: func(c *fiber.Ctx) string {
			return clientKey(c) + ":" + c.Path()
		},
		LimitReached: rateLimitResponse,
	})
}

func AuthRateLimit(cfg *config.Config) fiber.Handler {
	return StrictRateLimit(cfg.RateLimitAuthMax, cfg.RateLimitAuthWindow)
}

func QueryRateLimit(cfg *config.Config) fiber.Handler {
	return StrictRateLimit(cfg.RateLimitQueryMax, cfg.RateLimitQueryWindow)
}

func UploadRateLimit(cfg *config.Config) fiber.Handler {
	return StrictRateLimit(cfg.RateLimitUploadMax, cfg.RateLimitUploadWindow)
}

func AdminRateLimit(cfg *config.Config) fiber.Handler {
	return StrictRateLimit(cfg.RateLimitAdminMax, cfg.RateLimitAdminWindow)
}

func clientKey(c *fiber.Ctx) string {
	if userID, ok := c.Locals("userID").(string); ok && userID != "" {
		return "user:" + userID
	}
	return "ip:" + c.IP()
}

func rateLimitResponse(c *fiber.Ctx) error {
	return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
		"error": "Terlalu banyak permintaan. Coba lagi nanti.",
	})
}

func joinOrigins(origins []string) string {
	out := ""
	for i, origin := range origins {
		if i > 0 {
			out += ","
		}
		out += origin
	}
	return out
}

func FiberConfig(cfg *config.Config) fiber.Config {
	fiberCfg := fiber.Config{
		BodyLimit:    cfg.BodyLimitMB * 1024 * 1024,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
		Concurrency:  256 * 1024,
	}

	if len(cfg.TrustedProxies) > 0 {
		fiberCfg.EnableTrustedProxyCheck = true
		fiberCfg.TrustedProxies = cfg.TrustedProxies
	}

	return fiberCfg
}
