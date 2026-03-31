package services

import (
	"fmt"
	"io"
	"github.com/charmbracelet/log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/thienntdev/snaptiktok/internal/config"
)

// DownloadService handles file downloading and temporary storage
type DownloadService struct {
	tempDir string
	maxAge  time.Duration
	mu      sync.Mutex
	client  *http.Client
}

// NewDownloadService creates a new download service
func NewDownloadService(cfg *config.Config) *DownloadService {
	// Create temp directory if it doesn't exist
	if err := os.MkdirAll(cfg.TempDir, 0755); err != nil {
		log.Printf("Failed to create temp dir: %v", err)
	}

	ds := &DownloadService{
		tempDir: cfg.TempDir,
		maxAge:  cfg.FileMaxAge,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}

	// Start cleanup goroutine
	go ds.cleanupLoop(cfg.CleanupInterval)

	return ds
}

// GetDownloadStream returns a stream to a remote file
func (s *DownloadService) GetDownloadStream(remoteURL string) (io.ReadCloser, string, int64, error) {
	req, err := http.NewRequest("GET", remoteURL, nil)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	
	// Set correct referer based on the URL domain
	if strings.Contains(remoteURL, "douyinvod.com") || strings.Contains(remoteURL, "douyin.com") || strings.Contains(remoteURL, "iesdouyin.com") {
		req.Header.Set("Referer", "https://www.douyin.com/")
	} else if strings.Contains(remoteURL, "tiktok") || strings.Contains(remoteURL, "byte") {
		req.Header.Set("Referer", "https://www.tiktok.com/")
	} else if strings.Contains(remoteURL, "tikwm.com") {
		req.Header.Set("Referer", "https://www.tikwm.com/")
	}
	
	req.Header.Set("Accept", "*/*")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to download: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, "", 0, fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	return resp.Body, resp.Header.Get("Content-Type"), resp.ContentLength, nil
}

// ProxyDownload streams a remote file directly to the client writer
func (s *DownloadService) ProxyDownload(remoteURL string, w io.Writer) (int64, string, error) {
	req, err := http.NewRequest("GET", remoteURL, nil)
	if err != nil {
		return 0, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	
	// Set correct referer based on the URL domain
	if strings.Contains(remoteURL, "douyinvod.com") || strings.Contains(remoteURL, "douyin.com") || strings.Contains(remoteURL, "iesdouyin.com") {
		req.Header.Set("Referer", "https://www.douyin.com/")
	} else if strings.Contains(remoteURL, "tiktok") || strings.Contains(remoteURL, "byte") {
		req.Header.Set("Referer", "https://www.tiktok.com/")
	} else if strings.Contains(remoteURL, "tikwm.com") {
		req.Header.Set("Referer", "https://www.tikwm.com/")
	}
	
	req.Header.Set("Accept", "*/*")

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, "", fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	written, err := io.Copy(w, resp.Body)
	if err != nil {
		return written, contentType, fmt.Errorf("failed to stream: %w", err)
	}

	return written, contentType, nil
}

// DownloadToTemp downloads a remote file to temporary storage and returns the local path
func (s *DownloadService) DownloadToTemp(remoteURL, extension string) (string, error) {
	filename := fmt.Sprintf("%s.%s", uuid.New().String(), extension)
	localPath := filepath.Join(s.tempDir, filename)

	file, err := os.Create(localPath)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer file.Close()

	_, _, err = s.ProxyDownload(remoteURL, file)
	if err != nil {
		os.Remove(localPath)
		return "", err
	}

	return localPath, nil
}

// GetTempFilePath returns the absolute path for a temp file
func (s *DownloadService) GetTempFilePath(filename string) string {
	return filepath.Join(s.tempDir, filename)
}

// cleanupLoop periodically removes old temporary files
func (s *DownloadService) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanup()
	}
}

// cleanup removes temporary files older than maxAge
func (s *DownloadService) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.tempDir)
	if err != nil {
		log.Printf("Cleanup error reading dir: %v", err)
		return
	}

	now := time.Now()
	removed := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if now.Sub(info.ModTime()) > s.maxAge {
			path := filepath.Join(s.tempDir, entry.Name())
			if err := os.Remove(path); err != nil {
				log.Printf("Cleanup: failed to remove %s: %v", path, err)
			} else {
				removed++
			}
		}
	}

	if removed > 0 {
		log.Printf("🧹 Cleanup: removed %d expired files", removed)
	}
}
