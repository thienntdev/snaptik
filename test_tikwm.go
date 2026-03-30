package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func main() {
	testURL := "https://www.tiktok.com/@nanq1914280319/video/7523537176777297159?is_from_webapp=1&sender_device=pc"
	fmt.Println("Testing TikWM...")
	api := "https://www.tikwm.com/api/"
	data := url.Values{}
	data.Set("url", testURL)
	data.Set("count", "12")
	data.Set("cursor", "0")
	data.Set("web", "1")
	data.Set("hd", "1")
	
	req, _ := http.NewRequest("POST", api, strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	var j map[string]interface{}
	json.Unmarshal(body, &j)
	if data, ok := j["data"].(map[string]interface{}); ok {
		fmt.Printf("id: %v\n", data["id"])
		fmt.Printf("title: %v\n", data["title"])
		fmt.Printf("play: %v\n", data["play"])
		fmt.Printf("wmplay: %v\n", data["wmplay"])
		fmt.Printf("hdplay: %v\n", data["hdplay"])
		fmt.Printf("cover: %v\n", data["cover"])
	}
}
