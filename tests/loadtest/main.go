// Package main provides load testing for the Query Service via API Gateway
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Config holds load test configuration
type Config struct {
	URL             string
	Concurrency     int
	Requests        int
	Duration        time.Duration
	WarmupRequests  int
	ContractAddress string
}

// Result holds metrics for a single request
type Result struct {
	Duration time.Duration
	Success  bool
	Error    string
	CacheHit bool
}

// Stats holds aggregated metrics
type Stats struct {
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	Latencies       []time.Duration
	Errors          map[string]int
	CacheHits       int64
	StartTime       time.Time
	EndTime         time.Time
}

// GraphQL request/response structures
type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

type GraphQLResponse struct {
	Data   interface{} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

type SystemStatusData struct {
	SystemStatus struct {
		CacheHitRate float64 `json:"cacheHitRate"`
		IsHealthy    bool    `json:"isHealthy"`
	} `json:"systemStatus"`
}

func main() {
	// Parse flags
	url := flag.String("url", "http://localhost:8000/graphql", "API Gateway GraphQL endpoint")
	concurrency := flag.Int("c", 100, "Number of concurrent workers")
	requests := flag.Int("n", 1000, "Total number of requests (0 for duration-based)")
	duration := flag.Duration("d", 0, "Test duration (e.g., 30s, 1m). Overrides -n if set")
	warmup := flag.Int("warmup", 10, "Number of warmup requests before test")
	contract := flag.String("contract", "0x0000000000000000000000000000000000000000", "Contract address for queries")
	flag.Parse()

	config := &Config{
		URL:             *url,
		Concurrency:     *concurrency,
		Requests:        *requests,
		Duration:        *duration,
		WarmupRequests:  *warmup,
		ContractAddress: *contract,
	}

	fmt.Println("========================================")
	fmt.Println("Query Service Load Test")
	fmt.Println("========================================")
	fmt.Printf("Target URL:    %s\n", config.URL)
	fmt.Printf("Concurrency:   %d workers\n", config.Concurrency)
	if config.Duration > 0 {
		fmt.Printf("Duration:      %s\n", config.Duration)
	} else {
		fmt.Printf("Requests:      %d\n", config.Requests)
	}
	fmt.Printf("Contract:      %s\n", config.ContractAddress)
	fmt.Println("========================================")

	// Check if service is available
	if !checkHealth(config.URL) {
		fmt.Println("\n❌ API Gateway is not responding. Make sure services are running:")
		fmt.Println("   docker-compose up -d")
		os.Exit(1)
	}
	fmt.Println("✅ API Gateway is healthy")

	// Get initial cache stats
	initialStats := getSystemStatus(config.URL)
	fmt.Printf("📊 Initial cache hit rate: %.2f%%\n", initialStats.SystemStatus.CacheHitRate*100)

	// Warmup phase
	if config.WarmupRequests > 0 {
		fmt.Printf("\n🔥 Running %d warmup requests...\n", config.WarmupRequests)
		runWarmup(config)
	}

	// Run main test
	fmt.Println("\n🚀 Starting load test...")
	stats := runLoadTest(config)

	// Get final cache stats
	finalStats := getSystemStatus(config.URL)

	// Print results
	printResults(stats, initialStats, finalStats)
}

func checkHealth(baseURL string) bool {
	// Try health endpoint first
	resp, err := http.Get(baseURL[:len(baseURL)-8] + "/health") // Remove /graphql, add /health
	if err == nil && resp.StatusCode == 200 {
		resp.Body.Close()
		return true
	}

	// Try GraphQL endpoint
	query := GraphQLRequest{
		Query: `query { systemStatus { isHealthy } }`,
	}
	body, _ := json.Marshal(query)
	resp, err = http.Post(baseURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func getSystemStatus(baseURL string) SystemStatusData {
	query := GraphQLRequest{
		Query: `query { systemStatus { cacheHitRate isHealthy indexerLag totalContracts totalEvents } }`,
	}
	body, _ := json.Marshal(query)
	resp, err := http.Post(baseURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return SystemStatusData{}
	}
	defer resp.Body.Close()

	var result struct {
		Data SystemStatusData `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Data
}

func runWarmup(config *Config) {
	var wg sync.WaitGroup
	for i := 0; i < config.WarmupRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			executeQuery(config, getRandomQuery(config))
		}()
	}
	wg.Wait()
}

func runLoadTest(config *Config) *Stats {
	stats := &Stats{
		Latencies: make([]time.Duration, 0, config.Requests),
		Errors:    make(map[string]int),
		StartTime: time.Now(),
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	var totalRequests int64
	var successRequests int64
	var failedRequests int64

	// Create worker pool
	requestChan := make(chan struct{}, config.Concurrency*2)
	resultChan := make(chan Result, config.Concurrency*2)

	// Progress tracking
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				current := atomic.LoadInt64(&totalRequests)
				success := atomic.LoadInt64(&successRequests)
				failed := atomic.LoadInt64(&failedRequests)
				elapsed := time.Since(stats.StartTime).Seconds()
				rps := float64(current) / elapsed
				fmt.Printf("\r⏱  Progress: %d requests | ✅ %d success | ❌ %d failed | %.1f req/s",
					current, success, failed, rps)
			}
		}
	}()

	// Start workers
	for i := 0; i < config.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range requestChan {
				result := executeQuery(config, getRandomQuery(config))
				resultChan <- result
			}
		}()
	}

	// Collect results
	var collectWg sync.WaitGroup
	collectWg.Add(1)
	go func() {
		defer collectWg.Done()
		for result := range resultChan {
			atomic.AddInt64(&totalRequests, 1)
			if result.Success {
				atomic.AddInt64(&successRequests, 1)
			} else {
				atomic.AddInt64(&failedRequests, 1)
			}

			mu.Lock()
			stats.Latencies = append(stats.Latencies, result.Duration)
			if !result.Success {
				stats.Errors[result.Error]++
			}
			if result.CacheHit {
				stats.CacheHits++
			}
			mu.Unlock()
		}
	}()

	// Send requests
	if config.Duration > 0 {
		// Duration-based test
		deadline := time.After(config.Duration)
	DurationLoop:
		for {
			select {
			case <-deadline:
				break DurationLoop
			case requestChan <- struct{}{}:
			}
		}
	} else {
		// Request count-based test
		for i := 0; i < config.Requests; i++ {
			requestChan <- struct{}{}
		}
	}

	close(requestChan)
	wg.Wait()
	close(resultChan)
	collectWg.Wait()
	close(done)

	stats.EndTime = time.Now()
	stats.TotalRequests = atomic.LoadInt64(&totalRequests)
	stats.SuccessRequests = atomic.LoadInt64(&successRequests)
	stats.FailedRequests = atomic.LoadInt64(&failedRequests)

	fmt.Println() // New line after progress
	return stats
}

func getRandomQuery(config *Config) GraphQLRequest {
	queries := []GraphQLRequest{
		// Events query with pagination
		{
			Query: `query Events($filter: EventFilter, $pagination: PaginationInput) {
				events(filter: $filter, pagination: $pagination) {
					totalCount
					pageInfo { hasNextPage endCursor }
					edges { node { id eventName blockNumber } }
				}
			}`,
			Variables: map[string]interface{}{
				"filter":     map[string]interface{}{"contractAddress": config.ContractAddress},
				"pagination": map[string]interface{}{"first": 20},
			},
		},
		// Contract stats query
		{
			Query: `query Stats($address: Address!) {
				contractStats(address: $address) {
					totalEvents latestBlock indexerDelay uniqueAddresses
				}
			}`,
			Variables: map[string]interface{}{
				"address": config.ContractAddress,
			},
		},
		// System status query
		{
			Query: `query {
				systemStatus {
					indexerLag totalContracts totalEvents cacheHitRate isHealthy
				}
			}`,
		},
		// Contracts list query
		{
			Query: `query {
				contracts {
					id address name isActive currentBlock
				}
			}`,
		},
	}

	// Simple round-robin distribution weighted towards events query (60%)
	idx := time.Now().UnixNano() % 10
	if idx < 6 {
		return queries[0] // events query
	} else if idx < 8 {
		return queries[1] // stats query
	} else if idx < 9 {
		return queries[2] // system status
	}
	return queries[3] // contracts list
}

func executeQuery(config *Config, query GraphQLRequest) Result {
	start := time.Now()

	body, err := json.Marshal(query)
	if err != nil {
		return Result{
			Duration: time.Since(start),
			Success:  false,
			Error:    "marshal_error",
		}
	}

	resp, err := http.Post(config.URL, "application/json", bytes.NewReader(body))
	duration := time.Since(start)

	if err != nil {
		return Result{
			Duration: duration,
			Success:  false,
			Error:    "connection_error",
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return Result{
			Duration: duration,
			Success:  false,
			Error:    "read_error",
		}
	}

	if resp.StatusCode != 200 {
		return Result{
			Duration: duration,
			Success:  false,
			Error:    fmt.Sprintf("http_%d", resp.StatusCode),
		}
	}

	var graphqlResp GraphQLResponse
	if err := json.Unmarshal(respBody, &graphqlResp); err != nil {
		return Result{
			Duration: duration,
			Success:  false,
			Error:    "json_error",
		}
	}

	if len(graphqlResp.Errors) > 0 {
		return Result{
			Duration: duration,
			Success:  false,
			Error:    "graphql_error",
		}
	}

	// Check if response indicates cache hit (via header or fast response)
	cacheHit := resp.Header.Get("X-Cache") == "HIT" || duration < 10*time.Millisecond

	return Result{
		Duration: duration,
		Success:  true,
		CacheHit: cacheHit,
	}
}

func printResults(stats *Stats, initialStatus, finalStatus SystemStatusData) {
	duration := stats.EndTime.Sub(stats.StartTime)
	rps := float64(stats.TotalRequests) / duration.Seconds()

	// Sort latencies for percentile calculation
	sort.Slice(stats.Latencies, func(i, j int) bool {
		return stats.Latencies[i] < stats.Latencies[j]
	})

	fmt.Println("\n========================================")
	fmt.Println("📊 LOAD TEST RESULTS")
	fmt.Println("========================================")

	// Summary
	fmt.Println("\n📈 Summary:")
	fmt.Printf("   Total Requests:    %d\n", stats.TotalRequests)
	fmt.Printf("   Successful:        %d (%.1f%%)\n", stats.SuccessRequests,
		float64(stats.SuccessRequests)/float64(stats.TotalRequests)*100)
	fmt.Printf("   Failed:            %d (%.1f%%)\n", stats.FailedRequests,
		float64(stats.FailedRequests)/float64(stats.TotalRequests)*100)
	fmt.Printf("   Duration:          %s\n", duration.Round(time.Millisecond))
	fmt.Printf("   Requests/sec:      %.1f\n", rps)

	// Latency metrics
	if len(stats.Latencies) > 0 {
		fmt.Println("\n⏱  Latency Metrics:")
		fmt.Printf("   Min:               %s\n", stats.Latencies[0].Round(time.Microsecond))
		fmt.Printf("   Max:               %s\n", stats.Latencies[len(stats.Latencies)-1].Round(time.Microsecond))
		fmt.Printf("   Mean:              %s\n", calculateMean(stats.Latencies).Round(time.Microsecond))
		fmt.Printf("   P50 (Median):      %s\n", calculatePercentile(stats.Latencies, 50).Round(time.Microsecond))
		fmt.Printf("   P90:               %s\n", calculatePercentile(stats.Latencies, 90).Round(time.Microsecond))
		fmt.Printf("   P95:               %s\n", calculatePercentile(stats.Latencies, 95).Round(time.Microsecond))
		fmt.Printf("   P99:               %s\n", calculatePercentile(stats.Latencies, 99).Round(time.Microsecond))
	}

	// Cache metrics
	fmt.Println("\n💾 Cache Metrics:")
	fmt.Printf("   Initial Hit Rate:  %.2f%%\n", initialStatus.SystemStatus.CacheHitRate*100)
	fmt.Printf("   Final Hit Rate:    %.2f%%\n", finalStatus.SystemStatus.CacheHitRate*100)
	fmt.Printf("   Fast Responses:    %d (estimated cache hits)\n", stats.CacheHits)

	// Errors
	if len(stats.Errors) > 0 {
		fmt.Println("\n❌ Errors:")
		for errType, count := range stats.Errors {
			fmt.Printf("   %s: %d\n", errType, count)
		}
	}

	// Performance assessment
	fmt.Println("\n🎯 Performance Assessment:")
	p95 := calculatePercentile(stats.Latencies, 95)
	if p95 < 200*time.Millisecond {
		fmt.Printf("   ✅ P95 latency (%.0fms) is WITHIN target (<200ms)\n", p95.Seconds()*1000)
	} else {
		fmt.Printf("   ⚠️  P95 latency (%.0fms) EXCEEDS target (<200ms)\n", p95.Seconds()*1000)
	}

	successRate := float64(stats.SuccessRequests) / float64(stats.TotalRequests) * 100
	if successRate >= 99.0 {
		fmt.Printf("   ✅ Success rate (%.1f%%) is EXCELLENT (≥99%%)\n", successRate)
	} else if successRate >= 95.0 {
		fmt.Printf("   ⚠️  Success rate (%.1f%%) is ACCEPTABLE (≥95%%)\n", successRate)
	} else {
		fmt.Printf("   ❌ Success rate (%.1f%%) NEEDS IMPROVEMENT (<95%%)\n", successRate)
	}

	fmt.Println("\n========================================")
}

func calculateMean(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}
	var total time.Duration
	for _, l := range latencies {
		total += l
	}
	return total / time.Duration(len(latencies))
}

func calculatePercentile(latencies []time.Duration, p float64) time.Duration {
	if len(latencies) == 0 {
		return 0
	}
	idx := int(float64(len(latencies)) * p / 100)
	if idx >= len(latencies) {
		idx = len(latencies) - 1
	}
	return latencies[idx]
}
