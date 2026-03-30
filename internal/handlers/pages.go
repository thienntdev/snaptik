package handlers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/thienntdev/snaptiktok/internal/config"
)

// PageHandler handles SSR page rendering
type PageHandler struct {
	cfg *config.Config
}

// NewPageHandler creates a new page handler
func NewPageHandler(cfg *config.Config) *PageHandler {
	return &PageHandler{cfg: cfg}
}

// seoData holds common SEO data for templates
type seoData struct {
	Title       string
	Description string
	Keywords    string
	Canonical   string
	OGType      string
	OGImage     string
	AppName     string
	BaseURL     string
	Year        int
}

// Index renders the home page
func (h *PageHandler) Index(c *fiber.Ctx) error {
	return c.Render("index", fiber.Map{
		"SEO": seoData{
			Title:       "SnapTiktok - Download TikTok & Douyin Videos Without Watermark | HD Free",
			Description: "Download TikTok and Douyin videos, MP3 audio, and images without watermark in HD quality. Free, fast, and no registration required. Works on mobile and desktop.",
			Keywords:    "tiktok downloader, download tiktok video, tiktok no watermark, douyin downloader, tiktok mp3, download douyin video, tiktok video download hd",
			Canonical:   h.cfg.BaseURL,
			OGType:      "website",
			OGImage:     h.cfg.BaseURL + "/static/images/og-image.png",
			AppName:     h.cfg.AppName,
			BaseURL:     h.cfg.BaseURL,
			Year:        time.Now().Year(),
		},
	}, "layouts/base")
}

// TikTokVideoDownload renders the TikTok video download SEO page
func (h *PageHandler) TikTokVideoDownload(c *fiber.Ctx) error {
	return c.Render("seo/tiktok-video-download", fiber.Map{
		"SEO": seoData{
			Title:       "Download TikTok Video Without Watermark - HD Quality Free | SnapTiktok",
			Description: "Free TikTok video downloader. Save TikTok videos without watermark in full HD quality. No registration, no app install needed. Download unlimited TikTok videos online.",
			Keywords:    "download tiktok video, tiktok video downloader, tiktok no watermark, save tiktok video hd, tiktok download online free",
			Canonical:   h.cfg.BaseURL + "/tiktok-video-download",
			OGType:      "website",
			OGImage:     h.cfg.BaseURL + "/static/images/og-image.png",
			AppName:     h.cfg.AppName,
			BaseURL:     h.cfg.BaseURL,
			Year:        time.Now().Year(),
		},
	}, "layouts/base")
}

// DouyinDownloader renders the Douyin downloader SEO page
func (h *PageHandler) DouyinDownloader(c *fiber.Ctx) error {
	return c.Render("seo/douyin-downloader", fiber.Map{
		"SEO": seoData{
			Title:       "Douyin Video Downloader - Download Douyin (抖音) Videos Without Watermark | SnapTiktok",
			Description: "Download Douyin (抖音) videos without watermark for free. Save Chinese TikTok videos in HD quality. Fast, easy, and works on all devices.",
			Keywords:    "douyin downloader, download douyin video, 抖音下载, douyin video download, chinese tiktok downloader, douyin no watermark",
			Canonical:   h.cfg.BaseURL + "/douyin-downloader",
			OGType:      "website",
			OGImage:     h.cfg.BaseURL + "/static/images/og-image.png",
			AppName:     h.cfg.AppName,
			BaseURL:     h.cfg.BaseURL,
			Year:        time.Now().Year(),
		},
	}, "layouts/base")
}


// Sitemap generates the sitemap.xml
func (h *PageHandler) Sitemap(c *fiber.Ctx) error {
	c.Set("Content-Type", "application/xml; charset=utf-8")
	c.Set("Cache-Control", "public, max-age=86400")

	today := time.Now().Format("2006-01-02")

	xml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>%s</loc>
    <lastmod>%s</lastmod>
    <changefreq>daily</changefreq>
    <priority>1.0</priority>
  </url>
  <url>
    <loc>%s/tiktok-video-download</loc>
    <lastmod>%s</lastmod>
    <changefreq>weekly</changefreq>
    <priority>0.9</priority>
  </url>
  <url>
    <loc>%s/douyin-downloader</loc>
    <lastmod>%s</lastmod>
    <changefreq>weekly</changefreq>
    <priority>0.9</priority>
  </url>
</urlset>`, h.cfg.BaseURL, today, h.cfg.BaseURL, today, h.cfg.BaseURL, today)

	return c.SendString(xml)
}

// Robots serves robots.txt
func (h *PageHandler) Robots(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/plain; charset=utf-8")
	c.Set("Cache-Control", "public, max-age=86400")

	robots := fmt.Sprintf(`User-agent: *
Allow: /
Allow: /tiktok-video-download
Allow: /douyin-downloader

Disallow: /api/
Disallow: /tmp/

Sitemap: %s/sitemap.xml
`, h.cfg.BaseURL)

	return c.SendString(robots)
}
