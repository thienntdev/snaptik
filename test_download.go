package main

import (
	"fmt"
	"time"

	"github.com/thienntdev/snaptiktok/internal/config"
	"github.com/thienntdev/snaptiktok/internal/services"
)

func main() {
	tiktokSvc := services.NewTikTokService()
	info, err := tiktokSvc.ParseVideo("https://www.tiktok.com/@nanq1914280319/video/7523537176777297159")
	if err != nil {
		fmt.Printf("Parse err: %v\n", err)
		return
	}
	
	_ = services.NewDownloadService(&config.Config{TempDir: "./tmp", CleanupInterval: time.Minute})
	
	fmt.Printf("Audio URL: %s\n", info.AudioURL)
}
