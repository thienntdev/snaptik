package handlers

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/log"

	"github.com/gofiber/fiber/v2"
	"github.com/thienntdev/snaptiktok/internal/services"
)

type APIHandler struct {
	tiktokSvc   *services.TikTokService
	cacheSvc    *services.CacheService
	downloadSvc *services.DownloadService
}

func NewAPIHandler(tiktokSvc *services.TikTokService, cacheSvc *services.CacheService, downloadSvc *services.DownloadService) *APIHandler {
	return &APIHandler{
		tiktokSvc:   tiktokSvc,
		cacheSvc:    cacheSvc,
		downloadSvc: downloadSvc,
	}
}

type parseRequest struct {
	URL string `json:"url"`
}

func (h *APIHandler) ParseVideo(c *fiber.Ctx) error {
	var req parseRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.URL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "URL is required",
		})
	}

	// Try cache first
	if cached, err := h.cacheSvc.Get(req.URL); err == nil && cached != nil {
		return c.JSON(fiber.Map{
			"success": true,
			"data":    cached,
		})
	}

	// Parse video
	info, err := h.tiktokSvc.ParseVideo(req.URL)
	if err != nil {
		log.Printf("Parse error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to parse video. The URL might be invalid or unsupported.",
		})
	}

	// Cache result
	h.cacheSvc.Set(req.URL, info)

	return c.JSON(fiber.Map{
		"success": true,
		"data":    info,
	})
}

func (h *APIHandler) DownloadProxy(c *fiber.Ctx) error {
	url := c.Query("url")
	if url == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Missing url parameter")
	}

	filename := c.Query("filename", "download")
	fileType := c.Query("type", "video")
	
	isAudio := fileType == "audio"
	
	if isAudio {
		if len(filename) < 4 || filename[len(filename)-4:] != ".mp3" {
			filename += ".mp3"
		}
		c.Set("Content-Type", "audio/mpeg")
	} else if fileType == "image" {
		if len(filename) < 5 || (filename[len(filename)-4:] != ".jpg" && filename[len(filename)-4:] != ".png" && filename[len(filename)-5:] != ".jpeg") {
			filename += ".jpeg"
		}
		c.Set("Content-Type", "image/jpeg")
	} else {
		if len(filename) < 4 || filename[len(filename)-4:] != ".mp4" {
			filename += ".mp4"
		}
		c.Set("Content-Type", "video/mp4")
	}
	
	c.Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Response().Header.Set("Cache-Control", "no-cache")

	stream, streamContentType, contentLength, err := h.downloadSvc.GetDownloadStream(url)
	if err != nil {
		log.Printf("Download proxy error: %v", err)
		c.Response().ResetBody() // Clear any partial data
		
		// [VERCEL FALLBACK]: If the CDN (Cloudflare or TikTok) blocks Vercel's IP with 403 Forbidden,
		// we fallback to doing a 302 Redirect directly to the target URL so the user's browser fetches it directly.
		if strings.Contains(err.Error(), "status 403") {
			log.Printf("Vercel IP Blocked (403), falling back to 302 Redirect...")
			c.Response().Header.Del("Content-Disposition")
			c.Response().Header.Del("Content-Type")
			return c.Redirect(url, fiber.StatusFound)
		}

		return c.Status(fiber.StatusBadGateway).SendString(fmt.Sprintf("Sorry, failed to download video from upstream server: %v", err))
	}
	
	// Check if upstream server returned an HTML error page (e.g. Captcha, 403 disguised as 200)
	if strings.Contains(streamContentType, "text/html") {
		stream.Close()
		c.Response().ResetBody()
		return c.Status(fiber.StatusBadGateway).SendString("Video blocked by upstream server (Captcha/Security check). Please try another video.")
	}
	
	if streamContentType != "" {
		c.Set("Content-Type", streamContentType)
	}

	return c.SendStream(stream, int(contentLength))
}

func (h *APIHandler) HealthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "ok",
	})
}
