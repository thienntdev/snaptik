package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"github.com/charmbracelet/log"


	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/template/html/v2"

	"github.com/thienntdev/snaptiktok/internal/config"
	"github.com/thienntdev/snaptiktok/internal/handlers"
	applogger "github.com/thienntdev/snaptiktok/internal/logger"
	"github.com/thienntdev/snaptiktok/internal/middleware"
	"github.com/thienntdev/snaptiktok/internal/services"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize global logger
	applogger.Init(cfg.IsProd)

	// Initialize template engine
	engine := html.New("./internal/templates", ".html")
	if !cfg.IsProd {
		engine.Reload(true) // Hot reload in development
	}

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		Views:                 engine,
		ViewsLayout:          "",
		DisableStartupMessage: cfg.IsProd,
		ServerHeader:          cfg.AppName,
		AppName:               cfg.AppName,
		BodyLimit:             1 * 1024 * 1024, // 1MB max body
		ReadBufferSize:        8192,
		Prefork:               false,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"success": false,
				"error":   err.Error(),
			})
		},
	})

	// ===== Initialize Services =====
	cacheSvc := services.NewCacheService(cfg)
	defer cacheSvc.Close()

	tiktokSvc := services.NewTikTokService()
	downloadSvc := services.NewDownloadService(cfg)

	// ===== Initialize Handlers =====
	apiHandler := handlers.NewAPIHandler(tiktokSvc, cacheSvc, downloadSvc)
	pageHandler := handlers.NewPageHandler(cfg)

	// ===== Middleware =====
	// Recovery middleware
	app.Use(recover.New())

	// Logger
	app.Use(logger.New(logger.Config{
		Format:     "${time} | ${status} | ${latency} | ${ip} | ${method} ${path}\n",
		TimeFormat: "2006-01-02 15:04:05",
	}))

	// Compression
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))

	// Security headers
	app.Use(middleware.SecurityHeaders())

	// CORS
	app.Use(middleware.CORS())

	// Bot protection (API only)
	app.Use(middleware.BotProtection())

	// ===== Static Files =====
	app.Static("/static", "./static", fiber.Static{
		Compress:      true,
		CacheDuration: 24 * 60 * 60, // 24h cache
		MaxAge:        86400,
	})

	// ===== Rate Limiter (API only) =====
	rateLimiter := middleware.NewRateLimiter(cfg)

	// ===== SEO Routes =====
	app.Get("/sitemap.xml", pageHandler.Sitemap)
	app.Get("/robots.txt", pageHandler.Robots)

	// ===== Page Routes =====
	app.Get("/", pageHandler.Index)
	app.Get("/tiktok-video-download", pageHandler.TikTokVideoDownload)
	app.Get("/douyin-downloader", pageHandler.DouyinDownloader)

	// ===== API Routes =====
	api := app.Group("/api")
	api.Use(rateLimiter.Middleware())

	api.Post("/parse", apiHandler.ParseVideo)
	api.Get("/download", apiHandler.DownloadProxy)
	api.Get("/health", apiHandler.HealthCheck)

	// ===== Start Server =====
	addr := fmt.Sprintf(":%s", cfg.Port)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-quit
		log.Info("🛑 Shutting down server...")
		if err := app.Shutdown(); err != nil {
			log.Printf("Error shutting down: %v", err)
		}
	}()

	log.Printf("🚀 %s starting on %s", cfg.AppName, addr)
	log.Printf("📦 Environment: %s", map[bool]string{true: "production", false: "development"}[cfg.IsProd])
	log.Printf("🌐 Base URL: %s", cfg.BaseURL)

	if err := app.Listen(addr); err != nil {
		log.Fatalf("❌ Server failed: %v", err)
	}
}
