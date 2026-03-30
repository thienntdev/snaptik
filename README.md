# 🎬 SnapTiktok

> Download TikTok & Douyin videos without watermark — HD quality, MP3 extraction, image downloads.

A production-ready, SEO-optimized web application built with Go (Fiber), Redis, TailwindCSS, and Nginx.

## ✨ Features

- **🎥 Video Download** — Download TikTok & Douyin videos without watermark in HD
- **🎵 MP3 Extraction** — Extract and download audio from any video
- **📷 Image Downloads** — Download all images from slideshow/photo posts
- **🇨🇳 Douyin Support** — Full support for Chinese TikTok (抖音) including all URL formats
- **⚡ Fast** — Smart Redis caching, response time < 3 seconds
- **🔍 SEO Optimized** — SSR pages, structured data, sitemap, optimized meta tags
- **💰 Ad Ready** — Smart ad placement slots (AdSense compatible)
- **🔒 Secure** — Rate limiting, bot protection, input validation
- **📱 Mobile First** — Responsive design, works on all devices

## 🏗️ Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.24+ / Fiber v2 |
| Frontend | HTML + TailwindCSS CDN + Vanilla JS |
| Cache | Redis 7 |
| Reverse Proxy | Nginx |
| Container | Docker + Docker Compose |

## 📂 Project Structure

```
SnapTiktok/
├── cmd/server/main.go          # Entry point
├── internal/
│   ├── config/config.go        # Configuration
│   ├── handlers/
│   │   ├── api.go              # API (parse, download, health)
│   │   └── pages.go            # SSR pages + SEO
│   ├── middleware/
│   │   ├── ratelimit.go        # Token bucket rate limiter
│   │   └── security.go         # Security headers, bot protection
│   ├── models/video.go         # Data models
│   ├── services/
│   │   ├── tiktok.go           # TikTok/Douyin extraction
│   │   ├── cache.go            # Redis cache
│   │   └── downloader.go       # File download & proxy
│   └── templates/              # HTML templates (SSR)
│       ├── layouts/base.html
│       ├── index.html
│       └── seo/                # SEO landing pages
├── static/                     # CSS, JS, images
├── nginx/nginx.conf            # Nginx config
├── Dockerfile                  # Multi-stage Docker build
├── docker-compose.yml          # Full stack deployment
└── .env.example                # Environment variables
```

## 🚀 Quick Start

### Development (Local)

```bash
# 1. Clone the repo
git clone https://github.com/thienntdev/snaptiktok.git
cd snaptiktok

# 2. Copy environment file
cp .env.example .env

# 3. Start Redis (optional, app works without it)
docker run -d --name redis -p 6379:6379 redis:7-alpine

# 4. Run the app
go run ./cmd/server

# App runs at http://localhost:3000
```

### Production (Docker)

```bash
# Build and start all services
docker-compose up -d --build

# View logs
docker-compose logs -f app

# App runs at http://localhost:80
```

## 🔌 API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/parse` | Parse TikTok/Douyin URL |
| `GET` | `/api/download` | Proxy download (video/audio/image) |
| `GET` | `/api/health` | Health check |

### Parse Example

```bash
curl -X POST http://localhost:3000/api/parse \
  -H "Content-Type: application/json" \
  -d '{"url": "https://www.tiktok.com/@user/video/1234567890"}'
```

### Response

```json
{
  "success": true,
  "data": {
    "id": "1234567890",
    "platform": "tiktok",
    "title": "Video description",
    "author": "username",
    "video_url": "https://...",
    "video_hd_url": "https://...",
    "audio_url": "https://...",
    "cover_url": "https://...",
    "images": [],
    "likes": 12345,
    "views": 98765
  }
}
```

## 🔍 SEO Pages

| URL | Purpose |
|-----|---------|
| `/` | Home page |
| `/tiktok-video-download` | TikTok video download landing |
| `/douyin-downloader` | Douyin downloader landing |
| `/download-tiktok-mp3` | MP3 extraction landing |
| `/sitemap.xml` | Auto-generated sitemap |
| `/robots.txt` | Search engine directives |

## 💰 Ads Setup

Replace the ad placeholders in the templates with your AdSense code:

1. In `layouts/base.html` — uncomment the AdSense script tag and add your `ca-pub-XXXX`
2. Search for `adsbygoogle` comments in template files
3. Ad slots available: above result, between buttons, sticky mobile footer

## ⚙️ Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `PORT` | 3000 | Server port |
| `ENV` | development | Environment (production/development) |
| `BASE_URL` | https://snaptiktok.com | Base URL for SEO |
| `REDIS_ADDR` | localhost:6379 | Redis address |
| `REDIS_PASSWORD` | (empty) | Redis password |
| `CACHE_TTL_HOURS` | 3 | Cache TTL in hours |
| `RATE_LIMIT_MAX` | 15 | Max requests per window |
| `RATE_LIMIT_WINDOW_SEC` | 60 | Rate limit window in seconds |
| `TEMP_DIR` | ./tmp/downloads | Temporary file storage |

## 📋 License

MIT License — See [LICENSE](LICENSE) for details.

---

Built with ❤️ by [thienntdev](https://github.com/thienntdev)
