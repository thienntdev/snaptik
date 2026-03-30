package services

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/thienntdev/snaptiktok/internal/models"
)

// TikTokService handles video data extraction from TikTok and Douyin
type TikTokService struct {
	client *http.Client
}

// NewTikTokService creates a new TikTok extraction service
func NewTikTokService() *TikTokService {
	return &TikTokService{
		client: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
			},
			// Don't follow redirects automatically — we need the redirect URL
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 3 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
	}
}

// ValidateURL checks if the URL is a valid TikTok or Douyin URL
func (s *TikTokService) ValidateURL(rawURL string) (string, models.Platform, error) {
	rawURL = strings.TrimSpace(rawURL)

	// Normalize URL
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid URL format")
	}

	host := strings.ToLower(parsedURL.Host)

	// TikTok patterns
	tiktokPatterns := []string{
		"tiktok.com",
		"www.tiktok.com",
		"vm.tiktok.com",
		"vt.tiktok.com",
		"m.tiktok.com",
	}

	// Douyin patterns
	douyinPatterns := []string{
		"douyin.com",
		"www.douyin.com",
		"v.douyin.com",
		"m.douyin.com",
		"www.iesdouyin.com",
	}

	for _, p := range tiktokPatterns {
		if host == p || strings.HasSuffix(host, "."+p) {
			return rawURL, models.PlatformTikTok, nil
		}
	}

	for _, p := range douyinPatterns {
		if host == p || strings.HasSuffix(host, "."+p) {
			return rawURL, models.PlatformDouyin, nil
		}
	}

	return "", "", fmt.Errorf("URL must be from TikTok or Douyin")
}

// ParseVideo extracts video information from a TikTok or Douyin URL
func (s *TikTokService) ParseVideo(rawURL string) (*models.VideoInfo, error) {
	cleanURL, platform, err := s.ValidateURL(rawURL)
	if err != nil {
		return nil, err
	}

	switch platform {
	case models.PlatformTikTok:
		return s.parseTikTok(cleanURL)
	case models.PlatformDouyin:
		return s.parseDouyin(cleanURL)
	default:
		return nil, fmt.Errorf("unsupported platform")
	}
}

// parseTikTok handles TikTok URL extraction
func (s *TikTokService) parseTikTok(rawURL string) (*models.VideoInfo, error) {
	// Resolve short URLs (vm.tiktok.com, vt.tiktok.com)
	finalURL, err := s.resolveRedirects(rawURL)
	if err != nil {
		finalURL = rawURL
	}

	// Extract video ID from URL
	videoID := s.extractTikTokVideoID(finalURL)
	if videoID == "" {
		videoID = uuid.New().String()[:12]
	}

	// ALWAYS Use the web page scraping approach first to get the ACTUAL CDN MP4 video URL
	info, err := s.fetchViaWebScrape(finalURL, videoID)
	
	// If scraping fails or misses the video URL, maybe try oEmbed for metadata fallback
	if err != nil || info == nil || info.VideoURL == "" || info.VideoURL == finalURL {
		log.Printf("Scrape failed or incomplete, using oEmbed fallback")
		oembedInfo, oerr := s.fetchViaOEmbed(finalURL, videoID)
		if oerr == nil && oembedInfo != nil {
			if info == nil {
				info = oembedInfo
			} else {
				// Merge oembed metadata into scraped info
				if info.Title == "" || info.Title == "TikTok Video" || info.Title == "Douyin Video" { info.Title = oembedInfo.Title }
				if info.Author == "" || info.Author == "Unknown" { info.Author = oembedInfo.Author }
				if info.CoverURL == "" { info.CoverURL = oembedInfo.CoverURL }
			}
		}
	}
	
	// FINAL FALLBACK: Third party API if we still don't have the MP4 URL
	if info != nil && (info.VideoURL == "" || info.VideoURL == finalURL) {
		log.Printf("Still missing VideoURL, querying TikWM API ...")
		twInfo, twErr := s.fetchViaTikWM(finalURL)
		if twErr == nil && twInfo != nil {
			info.VideoURL = twInfo.VideoURL
			info.VideoHDURL = twInfo.VideoHDURL
			if info.Title == "" || info.Title == "TikTok Video" || info.Title == "Douyin Video" { info.Title = twInfo.Title }
			if info.Author == "" || info.Author == "Unknown" { info.Author = twInfo.Author }
			if info.CoverURL == "" { info.CoverURL = twInfo.CoverURL }
			if info.AudioURL == "" { info.AudioURL = twInfo.AudioURL }
			log.Printf("TikWM fallback succeeded!")
		}
	}

	if info == nil {
		return nil, fmt.Errorf("failed to extract video data from TikTok")
	}

	info.Platform = models.PlatformTikTok
	info.OriginalURL = rawURL
	return info, nil
}

// parseDouyin handles Douyin (Chinese TikTok) URL extraction
func (s *TikTokService) parseDouyin(rawURL string) (*models.VideoInfo, error) {
	// Resolve short URLs (v.douyin.com)
	finalURL, err := s.resolveRedirects(rawURL)
	if err != nil {
		finalURL = rawURL
	}

	// Extract video ID
	videoID := s.extractDouyinVideoID(finalURL)
	if videoID == "" {
		videoID = uuid.New().String()[:12]
	}

	// Method 1: Try Douyin web scrape (Use rawURL so HTTP client handles GET redirects naturally)
	info, err := s.fetchDouyinData(rawURL, videoID)
	if err != nil {
		log.Printf("Douyin direct fetch failed: %v", err)
		// Fallback: return basic info
		info = &models.VideoInfo{
			ID:       videoID,
			Title:    "Douyin Video",
			Author:   "Unknown",
			CoverURL: "",
		}
	}

	info.Platform = models.PlatformDouyin
	info.OriginalURL = rawURL
	return info, nil
}

// fetchViaOEmbed uses TikTok's public oEmbed endpoint
func (s *TikTokService) fetchViaOEmbed(videoURL, videoID string) (*models.VideoInfo, error) {
	oembedURL := fmt.Sprintf("https://www.tiktok.com/oembed?url=%s", url.QueryEscape(videoURL))

	req, err := http.NewRequest("GET", oembedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("oembed returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var oembed struct {
		Title        string `json:"title"`
		AuthorName   string `json:"author_name"`
		AuthorURL    string `json:"author_url"`
		ThumbnailURL string `json:"thumbnail_url"`
		HTML         string `json:"html"`
	}

	if err := json.Unmarshal(body, &oembed); err != nil {
		return nil, err
	}

	return &models.VideoInfo{
		ID:       videoID,
		Title:    oembed.Title,
		Author:   oembed.AuthorName,
		CoverURL: oembed.ThumbnailURL,
		VideoURL: videoURL, // Will be resolved by download proxy
		CreatedAt: time.Now(),
	}, nil
}

// fetchViaTikWM uses the tikwm.com API as a final fallback
func (s *TikTokService) fetchViaTikWM(videoURL string) (*models.VideoInfo, error) {
	api := "https://www.tikwm.com/api/"
	data := url.Values{}
	data.Set("url", videoURL)
	data.Set("count", "12")
	data.Set("cursor", "0")
	data.Set("web", "1")
	data.Set("hd", "1")

	req, err := http.NewRequest("POST", api, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("tikwm api returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var j map[string]interface{}
	if err := json.Unmarshal(body, &j); err != nil {
		return nil, err
	}

	code, ok := j["code"].(float64)
	if !ok || code != 0 {
		return nil, fmt.Errorf("tikwm api returned error or non-zero code")
	}

	dataMap, ok := j["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("tikwm api returned invalid data")
	}

	info := &models.VideoInfo{
		CreatedAt: time.Now(),
	}

	if id, ok := dataMap["id"].(string); ok { info.ID = id }
	if title, ok := dataMap["title"].(string); ok { info.Title = title }

	// Ensure URLs are absolute
	if play, ok := dataMap["play"].(string); ok && play != "" {
		if strings.HasPrefix(play, "/") {
			info.VideoURL = "https://www.tikwm.com" + play
		} else {
			info.VideoURL = play
		}
	}
	if hdplay, ok := dataMap["hdplay"].(string); ok && hdplay != "" {
		if strings.HasPrefix(hdplay, "/") {
			info.VideoHDURL = "https://www.tikwm.com" + hdplay
		} else {
			info.VideoHDURL = hdplay
		}
	}
	if cover, ok := dataMap["cover"].(string); ok && cover != "" {
		if strings.HasPrefix(cover, "/") {
			info.CoverURL = "https://www.tikwm.com" + cover
		} else {
			info.CoverURL = cover
		}
	}
	if music, ok := dataMap["music"].(string); ok && music != "" {
		if strings.HasPrefix(music, "/") {
			info.AudioURL = "https://www.tikwm.com" + music
		} else {
			info.AudioURL = music
		}
	}
	
	if authorMap, ok := dataMap["author"].(map[string]interface{}); ok {
		if authorName, ok := authorMap["nickname"].(string); ok {
			info.Author = authorName
		}
		if authorAvatar, ok := authorMap["avatar"].(string); ok {
			info.AuthorAvatar = authorAvatar
		}
	}

	return info, nil
}

// fetchViaWebScrape extracts data by parsing the TikTok web page
func (s *TikTokService) fetchViaWebScrape(videoURL, videoID string) (*models.VideoInfo, error) {
	req, err := http.NewRequest("GET", videoURL, nil)
	if err != nil {
		return nil, err
	}

	// Set headers to mimic a browser request
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://www.tiktok.com/")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	htmlContent := string(body)

	// Try to extract SIGI_STATE or __UNIVERSAL_DATA_FOR_REHYDRATION__ JSON
	info := s.extractFromHTML(htmlContent, videoID)
	if info != nil && info.VideoURL != "" {
		return info, nil
	}

	// Try extracting from application/ld+json (Schema.org) VideoObject
	reLD := regexp.MustCompile(`(?s)<script type="application/ld\+json"[^>]*>(.*?)</script>`)
	matchesLD := reLD.FindAllStringSubmatch(htmlContent, -1)
	for _, m := range matchesLD {
		var ldData map[string]interface{}
		if err := json.Unmarshal([]byte(m[1]), &ldData); err == nil {
			if ldData["@type"] == "VideoObject" || ldData["@type"] == "Video" {
				if contentUrl, ok := ldData["contentUrl"].(string); ok && contentUrl != "" {
					if info == nil {
						info = &models.VideoInfo{ID: videoID, CreatedAt: time.Now()}
					}
					info.VideoURL = contentUrl
					if name, ok := ldData["name"].(string); ok { info.Title = name }
					if desc, ok := ldData["description"].(string); ok && info.Title == "" { info.Title = desc }
					if thumb, ok := ldData["thumbnailUrl"].([]interface{}); ok && len(thumb) > 0 { info.CoverURL, _ = thumb[0].(string) }
					if author, ok := ldData["creator"].(map[string]interface{}); ok {
						if authorName, ok := author["name"].(string); ok { info.Author = authorName }
					}
					// TikTok LD-JSON often contains actual MP4 links!
					return info, nil
				}
			}
		}
	}

	if info != nil {
		return info, nil // From HTML parser, even if missing VideoURL
	}

	// Basic fallback: extract what we can from meta tags
	return s.extractFromMetaTags(htmlContent, videoID), nil
}

// fetchDouyinData extracts data from Douyin web page  
func (s *TikTokService) fetchDouyinData(videoURL, videoID string) (*models.VideoInfo, error) {
	req, err := http.NewRequest("GET", videoURL, nil)
	if err != nil {
		return nil, err
	}

	// Douyin requires specific headers, and Mobile UA often works better for SSR extracting
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 16_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Mobile/15E148 Safari/604.1")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Referer", "https://www.douyin.com/")
	req.Header.Set("Cookie", "")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	htmlContent := string(body)

	// Try to extract RENDER_DATA or similar JSON blob
	info := s.extractDouyinFromHTML(htmlContent, videoID)
	if info != nil {
		return info, nil
	}

	return s.extractFromMetaTags(htmlContent, videoID), nil
}

// extractFromHTML tries to parse embedded JSON data from TikTok page
func (s *TikTokService) extractFromHTML(html, videoID string) *models.VideoInfo {
	// Look for __UNIVERSAL_DATA_FOR_REHYDRATION__ pattern
	patterns := []string{
		`(?s)<script id="__UNIVERSAL_DATA_FOR_REHYDRATION__"[^>]*>(.*?)</script>`,
		`(?s)<script id="SIGI_STATE"[^>]*>(.*?)</script>`,
		`(?s)<script id="__NEXT_DATA__"[^>]*>(.*?)</script>`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(html)
		if len(matches) < 2 {
			continue
		}

		var rawData map[string]interface{}
		if err := json.Unmarshal([]byte(matches[1]), &rawData); err != nil {
			continue
		}

		// Try to navigate the JSON structure to find video data
		info := s.navigateVideoData(rawData, videoID)
		if info != nil {
			return info
		}
	}

	return nil
}

// extractDouyinFromHTML tries to parse embedded JSON from Douyin page
func (s *TikTokService) extractDouyinFromHTML(html, videoID string) *models.VideoInfo {
	// Douyin uses RENDER_DATA or _SSR_DATA or _ROUTER_DATA
	patterns := []string{
		`(?s)<script id="RENDER_DATA"[^>]*>(.*?)</script>`,
		`(?s)window\._SSR_DATA\s*=\s*(\{.*?\})(?:;|</script>)`,
		`(?s)window\._ROUTER_DATA\s*=\s*(\{.*?\})(?:;|</script>)`,
	}

	for i, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(html)
		if len(matches) < 2 {
			continue
		}
		log.Printf("extractDouyinFromHTML: Matched pattern %d", i)

		var decoded string
		if pattern == `(?s)<script id="RENDER_DATA"[^>]*>(.*?)</script>` {
			var err error
			decoded, err = url.QueryUnescape(matches[1])
			if err != nil {
				decoded = matches[1]
			}
		} else {
			decoded = matches[1]
		}

		var rawData map[string]interface{}
		if err := json.Unmarshal([]byte(decoded), &rawData); err != nil {
			log.Printf("extractDouyinFromHTML: JSON unmarshal error for pattern %d: %v", i, err)
			continue
		}

		info := s.navigateDouyinVideoData(rawData, videoID)
		if info != nil {
			return info
		} else {
			log.Printf("extractDouyinFromHTML: navigateDouyinVideoData returned nil for pattern %d", i)
		}
	}

	return nil
}

// navigateVideoData traverses TikTok JSON to find video details
func (s *TikTokService) navigateVideoData(data map[string]interface{}, videoID string) *models.VideoInfo {
	info := &models.VideoInfo{
		ID:        videoID,
		CreatedAt: time.Now(),
	}

	// Try different JSON structures TikTok uses
	// Structure 1: __UNIVERSAL_DATA_FOR_REHYDRATION__
	if defaultScope, ok := data["__DEFAULT_SCOPE__"].(map[string]interface{}); ok {
		if webapp, ok := defaultScope["webapp.video-detail"].(map[string]interface{}); ok {
			if itemInfo, ok := webapp["itemInfo"].(map[string]interface{}); ok {
				if itemStruct, ok := itemInfo["itemStruct"].(map[string]interface{}); ok {
					s.populateFromItemStruct(info, itemStruct)
					return info
				}
			}
		}
	}

	// Structure 2: SIGI_STATE
	if itemModule, ok := data["ItemModule"].(map[string]interface{}); ok {
		for _, item := range itemModule {
			if itemMap, ok := item.(map[string]interface{}); ok {
				s.populateFromItemStruct(info, itemMap)
				return info
			}
		}
	}

	return nil
}

// navigateDouyinVideoData traverses Douyin JSON to find video details
func (s *TikTokService) navigateDouyinVideoData(data map[string]interface{}, videoID string) *models.VideoInfo {
	info := &models.VideoInfo{
		ID:        videoID,
		CreatedAt: time.Now(),
	}

	// Douyin data structure navigation
	// Check for direct loaderData inside the root
	if loaderData, ok := data["loaderData"].(map[string]interface{}); ok {
		for _, v := range loaderData {
			if routeData, ok := v.(map[string]interface{}); ok {
				if videoInfoRes, ok := routeData["videoInfoRes"].(map[string]interface{}); ok {
					if itemList, ok := videoInfoRes["item_list"].([]interface{}); ok && len(itemList) > 0 {
						if awemeDetail, ok := itemList[0].(map[string]interface{}); ok {
							s.populateFromDouyinAweme(info, awemeDetail)
							return info
						}
					}
				}
			}
		}
	}

	for _, value := range data {
		if section, ok := value.(map[string]interface{}); ok {
			if awemeDetail, ok := section["awemeDetail"].(map[string]interface{}); ok {
				s.populateFromDouyinAweme(info, awemeDetail)
				return info
			}
			// Try nested loaderData
			if loaderData, ok := section["loaderData"].(map[string]interface{}); ok {
				for _, ld := range loaderData {
					if ldMap, ok := ld.(map[string]interface{}); ok {
						if awemeDetail, ok := ldMap["awemeDetail"].(map[string]interface{}); ok {
							s.populateFromDouyinAweme(info, awemeDetail)
							return info
						}
					}
				}
			}
		}
	}

	return nil
}

// populateFromItemStruct fills VideoInfo from TikTok's itemStruct
func (s *TikTokService) populateFromItemStruct(info *models.VideoInfo, item map[string]interface{}) {
	if desc, ok := item["desc"].(string); ok {
		info.Title = desc
	}
	if author, ok := item["author"].(map[string]interface{}); ok {
		if nickname, ok := author["nickname"].(string); ok {
			info.Author = nickname
		}
		if avatar, ok := author["avatarThumb"].(string); ok {
			info.AuthorAvatar = avatar
		}
	}
	if video, ok := item["video"].(map[string]interface{}); ok {
		if playAddr, ok := video["playAddr"].(string); ok {
			info.VideoURL = playAddr
		}
		if downloadAddr, ok := video["downloadAddr"].(string); ok {
			info.VideoHDURL = downloadAddr
		}
		if cover, ok := video["cover"].(string); ok {
			info.CoverURL = cover
		}
		if originCover, ok := video["originCover"].(string); ok {
			if info.CoverURL == "" {
				info.CoverURL = originCover
			}
		}
		if duration, ok := video["duration"].(float64); ok {
			info.Duration = int(duration)
		}
	}
	if music, ok := item["music"].(map[string]interface{}); ok {
		if playUrl, ok := music["playUrl"].(string); ok {
			info.AudioURL = playUrl
		}
		if title, ok := music["title"].(string); ok {
			info.Music = title
		}
	}
	if stats, ok := item["stats"].(map[string]interface{}); ok {
		if likes, ok := stats["diggCount"].(float64); ok {
			info.Likes = int64(likes)
		}
		if comments, ok := stats["commentCount"].(float64); ok {
			info.Comments = int64(comments)
		}
		if shares, ok := stats["shareCount"].(float64); ok {
			info.Shares = int64(shares)
		}
		if views, ok := stats["playCount"].(float64); ok {
			info.Views = int64(views)
		}
	}
	// Check for image post (slideshow)
	if imagePost, ok := item["imagePost"].(map[string]interface{}); ok {
		if images, ok := imagePost["images"].([]interface{}); ok {
			for _, img := range images {
				if imgMap, ok := img.(map[string]interface{}); ok {
					if imageURL, ok := imgMap["imageURL"].(map[string]interface{}); ok {
						if urlList, ok := imageURL["urlList"].([]interface{}); ok && len(urlList) > 0 {
							if imgURL, ok := urlList[0].(string); ok {
								info.Images = append(info.Images, imgURL)
							}
						}
					}
				}
			}
		}
	}
}

// populateFromDouyinAweme fills VideoInfo from Douyin's aweme structure
func (s *TikTokService) populateFromDouyinAweme(info *models.VideoInfo, aweme map[string]interface{}) {
	if desc, ok := aweme["desc"].(string); ok {
		info.Title = desc
	}
	if author, ok := aweme["authorInfo"].(map[string]interface{}); ok {
		if nickname, ok := author["nickname"].(string); ok {
			info.Author = nickname
		}
	}
	// Douyin video structure
	if video, ok := aweme["video"].(map[string]interface{}); ok {
		if playAddr, ok := video["play_addr"].(map[string]interface{}); ok {
			// Get uri to construct unwatermarked url
			if uri, ok := playAddr["uri"].(string); ok && uri != "" {
				info.VideoURL = "https://www.douyin.com/aweme/v1/play/?video_id=" + uri + "&ratio=1080p&line=0"
			} else if urlList, ok := playAddr["url_list"].([]interface{}); ok && len(urlList) > 0 {
				if videoURL, ok := urlList[0].(string); ok {
					info.VideoURL = videoURL
				}
			}
		}
		if cover, ok := video["cover"].(map[string]interface{}); ok {
			if urlList, ok := cover["url_list"].([]interface{}); ok && len(urlList) > 0 {
				if coverURL, ok := urlList[0].(string); ok {
					info.CoverURL = coverURL
				}
			}
		}
		if duration, ok := video["duration"].(float64); ok {
			info.Duration = int(duration / 1000) // Douyin uses milliseconds
		}
	}
	// Douyin music
	if music, ok := aweme["music"].(map[string]interface{}); ok {
		if playUrl, ok := music["play_url"].(map[string]interface{}); ok {
			if urlList, ok := playUrl["url_list"].([]interface{}); ok && len(urlList) > 0 {
				if audioURL, ok := urlList[0].(string); ok {
					if strings.HasPrefix(audioURL, "//") {
						audioURL = "https:" + audioURL
					}
					info.AudioURL = audioURL
				}
			} else if uri, ok := playUrl["uri"].(string); ok && uri != "" {
				info.AudioURL = "https://v.douyin.com/" + uri // fallback if uri is there but no url_list
			}
		}
		if title, ok := music["title"].(string); ok {
			info.Music = title
		}
	}
	// Douyin images (for image posts)
	if images, ok := aweme["images"].([]interface{}); ok {
		for _, img := range images {
			if imgMap, ok := img.(map[string]interface{}); ok {
				if urlList, ok := imgMap["url_list"].([]interface{}); ok && len(urlList) > 0 {
					if imgURL, ok := urlList[0].(string); ok {
						info.Images = append(info.Images, imgURL)
					}
				}
			}
		}
	}
	// Stats
	if stats, ok := aweme["statistics"].(map[string]interface{}); ok {
		if likes, ok := stats["digg_count"].(float64); ok {
			info.Likes = int64(likes)
		}
		if comments, ok := stats["comment_count"].(float64); ok {
			info.Comments = int64(comments)
		}
		if shares, ok := stats["share_count"].(float64); ok {
			info.Shares = int64(shares)
		}
		if views, ok := stats["play_count"].(float64); ok {
			info.Views = int64(views)
		}
	}
}

// extractFromMetaTags extracts basic info from HTML meta tags
func (s *TikTokService) extractFromMetaTags(html, videoID string) *models.VideoInfo {
	info := &models.VideoInfo{
		ID:        videoID,
		CreatedAt: time.Now(),
	}

	// Extract og:title
	if title := s.extractMetaContent(html, `property="og:title"`); title != "" {
		info.Title = title
	}
	if info.Title == "" {
		if title := s.extractMetaContent(html, `name="title"`); title != "" {
			info.Title = title
		}
	}

	// Extract og:image (cover/thumbnail)
	if image := s.extractMetaContent(html, `property="og:image"`); image != "" {
		info.CoverURL = image
	}

	// Extract og:video (direct video URL)
	if video := s.extractMetaContent(html, `property="og:video"`); video != "" {
		info.VideoURL = video
	}
	if video := s.extractMetaContent(html, `property="og:video:secure_url"`); video != "" {
		if info.VideoURL == "" {
			info.VideoURL = video
		}
	}

	// Extract author from description
	if desc := s.extractMetaContent(html, `property="og:description"`); desc != "" {
		// TikTok descriptions often contain author info
		if info.Title == "" {
			info.Title = desc
		}
	}

	if info.Title == "" {
		info.Title = "TikTok Video"
	}
	if info.Author == "" {
		info.Author = "Unknown"
	}

	return info
}

// extractMetaContent extracts content attribute from a meta tag
func (s *TikTokService) extractMetaContent(html, attr string) string {
	pattern := fmt.Sprintf(`<meta[^>]*%s[^>]*content="([^"]*)"`, regexp.QuoteMeta(attr))
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(html)
	if len(matches) >= 2 {
		return matches[1]
	}

	// Try reversed attribute order
	pattern = fmt.Sprintf(`<meta[^>]*content="([^"]*)"[^>]*%s`, regexp.QuoteMeta(attr))
	re = regexp.MustCompile(pattern)
	matches = re.FindStringSubmatch(html)
	if len(matches) >= 2 {
		return matches[1]
	}

	return ""
}

// resolveRedirects follows redirects to get the final URL
func (s *TikTokService) resolveRedirects(rawURL string) (string, error) {
	req, err := http.NewRequest("HEAD", rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	finalURL := resp.Request.URL.String()
	return finalURL, nil
}

// extractTikTokVideoID extracts the video ID from a TikTok URL
func (s *TikTokService) extractTikTokVideoID(rawURL string) string {
	patterns := []string{
		`/video/(\d+)`,
		`/v/(\d+)`,
		`/photo/(\d+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(rawURL)
		if len(matches) >= 2 {
			return matches[1]
		}
	}

	return ""
}

// extractDouyinVideoID extracts the video ID from a Douyin URL
func (s *TikTokService) extractDouyinVideoID(rawURL string) string {
	patterns := []string{
		`/video/(\d+)`,
		`/note/(\d+)`,
		`modal_id=(\d+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(rawURL)
		if len(matches) >= 2 {
			return matches[1]
		}
	}

	return ""
}
