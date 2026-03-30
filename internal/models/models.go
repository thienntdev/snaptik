package models

import "time"

// Platform represents the source platform of the video
type Platform string

const (
	PlatformTikTok Platform = "tiktok"
	PlatformDouyin Platform = "douyin"
)

// VideoInfo contains all extracted information about a video
type VideoInfo struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Author       string    `json:"author"`
	AuthorAvatar string    `json:"author_avatar,omitempty"`
	CoverURL     string    `json:"cover_url"`
	VideoURL     string    `json:"video_url,omitempty"`
	VideoHDURL   string    `json:"video_hd_url,omitempty"`
	AudioURL     string    `json:"audio_url,omitempty"`
	Images       []string  `json:"images,omitempty"`
	Duration     int       `json:"duration,omitempty"` // Duration in seconds
	Music        string    `json:"music,omitempty"`
	Likes        int64     `json:"likes,omitempty"`
	Comments     int64     `json:"comments,omitempty"`
	Shares       int64     `json:"shares,omitempty"`
	Views        int64     `json:"views,omitempty"`
	Platform     Platform  `json:"platform"`
	OriginalURL  string    `json:"original_url"`
	CreatedAt    time.Time `json:"created_at"`
}
