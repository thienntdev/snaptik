package main

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
)

func main() {
	urlstr := "https://www.tiktok.com/@tiktok/video/7376043144185253162"
	req, _ := http.NewRequest("GET", urlstr, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	
	client := &http.Client{}
	resp, _ := client.Do(req)
	body, _ := io.ReadAll(resp.Body)
	html := string(body)
	
	re := regexp.MustCompile(`<meta[^>]*property="og:video"[^>]*content="([^"]*)"`)
	matches := re.FindStringSubmatch(html)
	if len(matches) > 1 {
		fmt.Printf("og:video: %s\n", matches[1])
	} else {
		// Try reversed
		re2 := regexp.MustCompile(`<meta[^>]*content="([^"]*)"[^>]*property="og:video"`)
		m2 := re2.FindStringSubmatch(html)
		if len(m2) > 1 {
			fmt.Printf("og:video: %s\n", m2[1])
		} else {
			fmt.Println("No og:video found")
		}
	}
}
