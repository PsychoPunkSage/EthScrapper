package utils

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

func Measurelatency(rpcEndpoints []string) []time.Duration {
	latencies := make([]time.Duration, len(rpcEndpoints))
	var wg sync.WaitGroup

	for _, endpoint := range rpcEndpoints {
		wg.Add(1)
		go func(endpoint string) {
			defer wg.Done()

			start := time.Now()
			client := &http.Client{Timeout: 2 * time.Second}
			req, err := http.NewRequest("GET", endpoint, nil)
			if err != nil {
				fmt.Printf("[ERROR | utils]		creating request for %s: %v\n", endpoint, err)
				latencies = append(latencies, time.Duration(10000*time.Millisecond))
				return
			}

			_, err = client.Do(req)
			if err != nil {
				fmt.Printf("[ERROR | utils]		pinging endpoint %s: %v\n", endpoint, err)
				latencies = append(latencies, time.Duration(10000*time.Millisecond))
				return
			}

			latency := time.Since(start)
			latencies = append(latencies, latency)
		}(endpoint)
	}

	wg.Wait()
	return latencies
}

func SelectFastestRPC(rpcEndpoints []string, latencies []time.Duration) string {
	min_for_now := latencies[0]
	minIndex := 0

	for i, latency := range latencies {
		if latency < min_for_now {
			min_for_now = latency
			minIndex = i
		}
	}

	return rpcEndpoints[minIndex]
}
