package test

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestProxyLoad(t *testing.T) {
	const numWorkers = 50
	const totalRequests = 100000
	const requestsPerWorker = totalRequests / numWorkers

	proxyUrl := "http://localhost:8080"

	for i := 0; i < 10; i++ {
		res, err := http.Get(proxyUrl + "/keys")
		if err == nil {
			res.Body.Close()
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	var successCount int64

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        numWorkers,
			MaxIdleConnsPerHost: numWorkers,
			IdleConnTimeout:     90 * time.Second,
		},
		Timeout: 10 * time.Second,
	}

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	start := time.Now()

	for w := 0; w < numWorkers; w++ {
		go func(id int) {
			defer wg.Done()

			for i := 0; i < requestsPerWorker; i++ {
				key := fmt.Sprintf("key-%d-%d", id, rand.Intn(10000))
				value := fmt.Sprintf("value-%d", id)

				url := fmt.Sprintf("%s/set?key=%s", proxyUrl, key)
				payload := fmt.Sprintf(`{"key":"%s", "value":"%s"}`, key, value)

				var res *http.Response
				var err error

				for attempt := 0; attempt < 3; attempt++ {
					res, err = client.Post(url, "application/json", strings.NewReader(payload))
					if err == nil && res.StatusCode == http.StatusOK {
						break
					}
					if res != nil {
						io.Copy(io.Discard, res.Body)
						res.Body.Close()
					}
					time.Sleep(time.Duration(10*(attempt+1)) * time.Millisecond)
				}

				if err != nil {
					fmt.Printf("Request %d failed after retries: %v\n", id, err)
					continue
				}

				if res.StatusCode == http.StatusOK {
					atomic.AddInt64(&successCount, 1)
				} else {
					fmt.Printf("Request %d status: %v\n", id, res.Status)
				}
				if res != nil {
					io.Copy(io.Discard, res.Body)
					res.Body.Close()
				}
			}
		}(w)
	}

	wg.Wait()
	duration := time.Since(start)

	rps := float64(totalRequests) / duration.Seconds()
	successRate := (float64(successCount) / float64(totalRequests)) * 100

	fmt.Printf("\n--- Load Test Results ---\n")
	fmt.Printf("Total Requests: %d\n", totalRequests)
	fmt.Printf("Success Count:  %d\n", successCount)
	fmt.Printf("Success Rate:   %.2f%%\n", successRate)
	fmt.Printf("Total Time:     %v\n", duration)
	fmt.Printf("RPS:            %.2f\n", rps)
	fmt.Printf("-------------------------\n")
}
