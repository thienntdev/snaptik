package main

import (
	"fmt"
	"net/http"
)

func main() {
	// A typical TikTok CDN URL
	urlstr := "https://v16-webapp-prime.tiktok.com/video/tos/useast2a/tos-useast2a-ve-0068c001-euttp/o4EBeGfgfEFAAAeI8BAeIQ8ACAfQOEAADeGegO/?a=1988&ch=tiktok_web&cr=1&dr=0&lr=tiktok&cd=0%7C0%7C1%7C3&cv=1&br=1716&bt=858&cs=0&ds=3&ft=4b~O9S5q8Zmo0XzO~64jVz4fppWrKecoT&mime_type=video_mp4&qs=0&rc=OTM3OTo3PDNlPGc5Nzk6aEBpamxxc3M5cnB2ZjMzNzczM0BjYTM1YzA2NmIxMjAtNGEzYSM0NjE1cjRvcWhgLS1kL2Nzcw%3D%3D&l=2024032512345601010101010101010101"
	
	for _, ref := range []string{"", "https://www.tiktok.com/", "https://www.douyin.com/"} {
		req, _ := http.NewRequest("GET", urlstr, nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		if ref != "" {
			req.Header.Set("Referer", ref)
		}
		
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Referer %q: error %v\n", ref, err)
		} else {
			fmt.Printf("Referer %q: status %d\n", ref, resp.StatusCode)
		}
	}
}
