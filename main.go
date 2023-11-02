package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"
)

func main() {
	// Define command-line flags.
	url := flag.String("url", "", "URL to request")
	usernamesFile := flag.String("usernames", "", "File containing usernames")
	passwordsFile := flag.String("passwords", "", "File containing passwords")
	concurrency := flag.Int("concurrency", 10, "Number of concurrent requests")
	requestsPerSecond := flag.Float64("rps", 10.0, "Requests per second")
	flag.Parse()

	// Check if required flags are provided.
	if *url == "" || *usernamesFile == "" || *passwordsFile == "" {
		fmt.Println("Usage: go run main.go -url=<URL> -usernames=<usernames-file> -passwords=<passwords-file> -concurrency=<concurrency> -rps=<requests-per-second>")
		return
	}

	// Read the usernames from the file.
	usernames, err := readLines(*usernamesFile)
	if err != nil {
		fmt.Println("Error reading usernames file:", err)
		return
	}

	// Read the passwords from the file.
	passwords, err := readLines(*passwordsFile)
	if err != nil {
		fmt.Println("Error reading passwords file:", err)
		return
	}

	// Create a WaitGroup to wait for all goroutines to finish.
	var wg sync.WaitGroup

	// Create a channel for controlling concurrency.
	concurrencyCh := make(chan struct{}, *concurrency)

	// Calculate the interval duration for rate limiting.
	interval := time.Second / time.Duration(*requestsPerSecond)

	// Create a ticker for rate limiting.
	ticker := time.NewTicker(interval)

	// Send requests with each combination of username and password using goroutines.
	for _, username := range usernames {
		for _, password := range passwords {
			// Increment the WaitGroup counter.
			wg.Add(1)

			// Acquire a concurrency token.
			concurrencyCh <- struct{}{}

			go func(username, password string) {
				defer func() {
					// Release the concurrency token.
					<-concurrencyCh
					// Decrement the WaitGroup counter when the goroutine is done.
					wg.Done()
				}()

				// Wait for the next tick to control the rate.
				<-ticker.C

				// Create a new HTTP request.
				req, err := http.NewRequest("GET", *url, nil)
				if err != nil {
					fmt.Println("Error creating request:", err)
					return
				}

				// Set Basic Authentication header.
				req.SetBasicAuth(username, password)

				// Send the request.
				client := &http.Client{}
				resp, err := client.Do(req)
				if err != nil {
					fmt.Printf("Error sending request for username: %s, password: %s\n", username, password)
					return
				}
				defer resp.Body.Close()

				// Check the response status code.
				status := fmt.Sprintf("username: %s, password: %s - Status: %d", username, password, resp.StatusCode)
				fmt.Println(status)
			}(username, password)
		}
	}

	// Wait for all goroutines to finish.
	wg.Wait()
}

func readLines(filename string) ([]string, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	var nonEmptyLines []string
	for _, line := range lines {
		if line = strings.TrimSpace(line); line != "" {
			nonEmptyLines = append(nonEmptyLines, line)
		}
	}
	return nonEmptyLines, nil
}
